package http_handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/observability"
	handler "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/inbound/http_handler"
	wssink "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/websocket"
)

func newTestServer(t *testing.T) (*httptest.Server, *wssink.Hub) {
	t.Helper()
	hub := wssink.NewHub()
	h := handler.NewHandler(hub, observability.NewNopLogger())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, hub
}

func dialWS(t *testing.T, srv *httptest.Server, machine string) *websocket.Conn {
	t.Helper()
	u := "ws" + strings.TrimPrefix(srv.URL, "http") + "/availability/" + machine + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestHandler_ServeWS_UpgradesAndReceivesBroadcast(t *testing.T) {
	srv, hub := newTestServer(t)

	client := dialWS(t, srv, "m1")

	// Give the handler goroutine time to register the conn
	assert.Eventually(t, func() bool {
		return hub.ConnCount("m1") == 1
	}, time.Second, 10*time.Millisecond, "conn should be registered")

	payload := []byte(`{"machine_id":"m1"}`)
	hub.Broadcast("m1", payload)

	client.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := client.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, payload, msg)
}

func TestHandler_ServeWS_UnregistersOnClientDisconnect(t *testing.T) {
	srv, hub := newTestServer(t)

	client := dialWS(t, srv, "m1")

	assert.Eventually(t, func() bool {
		return hub.ConnCount("m1") == 1
	}, time.Second, 10*time.Millisecond, "conn should be registered")

	// Client disconnects
	client.Close()

	// Read loop should detect the close and unregister
	assert.Eventually(t, func() bool {
		return hub.ConnCount("m1") == 0
	}, time.Second, 10*time.Millisecond, "conn should be unregistered after client disconnect")
}
