package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	gorilla "github.com/gorilla/websocket"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
)

const writeTimeout = 5 * time.Second

// Hub manages per-machine WebSocket connections. All operations are safe for
// concurrent use. Broadcast copies the connection list under the lock and
// writes outside it so slow writes never block Register/Unregister callers.
type Hub struct {
	mu    sync.Mutex
	conns map[string][]*gorilla.Conn
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{conns: make(map[string][]*gorilla.Conn)}
}

// Register adds conn to the subscriber set for machineID.
func (h *Hub) Register(machineID string, conn *gorilla.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[machineID] = append(h.conns[machineID], conn)
}

// Unregister removes conn from machineID's set and closes it.
// Safe to call for an already-closed connection.
func (h *Hub) Unregister(machineID string, conn *gorilla.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeUnsafe(machineID, conn)
	conn.Close() //nolint:errcheck
}

func (h *Hub) removeUnsafe(machineID string, conn *gorilla.Conn) {
	conns := h.conns[machineID]
	for i, c := range conns {
		if c == conn {
			h.conns[machineID] = append(conns[:i], conns[i+1:]...)
			return
		}
	}
}

// Broadcast writes payload to every connection subscribed to machineID.
// Connections that fail the write (dead or slow past writeTimeout) are
// unregistered so they never block future broadcasts.
func (h *Hub) Broadcast(machineID string, payload []byte) {
	h.mu.Lock()
	src := h.conns[machineID]
	conns := make([]*gorilla.Conn, len(src))
	copy(conns, src)
	h.mu.Unlock()

	var dead []*gorilla.Conn
	for _, conn := range conns {
		conn.SetWriteDeadline(time.Now().Add(writeTimeout)) //nolint:errcheck
		if err := conn.WriteMessage(gorilla.TextMessage, payload); err != nil {
			dead = append(dead, conn)
		}
	}
	for _, conn := range dead {
		h.Unregister(machineID, conn)
	}
}

// ConnCount returns the number of active connections for machineID.
func (h *Hub) ConnCount(machineID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns[machineID])
}

// WebsocketSink implements application.AvailabilitySink by broadcasting
// JSON-encoded snapshots to all clients subscribed to the snapshot's machine.
type WebsocketSink struct {
	hub    *Hub
	logger observability.Logger
}

// NewWebsocketSink creates a WebsocketSink backed by the given hub.
func NewWebsocketSink(hub *Hub, logger observability.Logger) *WebsocketSink {
	return &WebsocketSink{hub: hub, logger: logger}
}

// Update JSON-encodes the snapshot and broadcasts it to subscribed clients.
func (s *WebsocketSink) Update(_ context.Context, snap avdomain.AvailabilitySnapshot) error {
	payload, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("ws sink: marshal: %w", err)
	}
	s.hub.Broadcast(snap.MachineID, payload)
	return nil
}
