package websocket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	wssink "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/websocket"
)

var testUpgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

// newUpgradeServer starts an httptest.Server that upgrades every connection and
// sends the server-side *websocket.Conn to the returned channel.
func newUpgradeServer(t *testing.T) (*httptest.Server, <-chan *websocket.Conn) {
	t.Helper()
	ch := make(chan *websocket.Conn, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		ch <- conn
	}))
	t.Cleanup(srv.Close)
	return srv, ch
}

// dialClient dials the given httptest.Server URL as a WebSocket client.
func dialClient(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func snapshotFactory(machineID string) avdomain.AvailabilitySnapshot {
	return avdomain.AvailabilitySnapshot{MachineID: machineID, Availability: 0.9, HasData: true}
}

func TestWebsocketSink_Update_BroadcastsJSONToSubscribedClients(t *testing.T) {
	hub := wssink.NewHub()
	sink := wssink.NewWebsocketSink(hub, observability.NewNopLogger())

	srv, serverConns := newUpgradeServer(t)

	client1 := dialClient(t, srv)
	hub.Register("m1", <-serverConns)

	client2 := dialClient(t, srv)
	hub.Register("m1", <-serverConns)

	snap := snapshotFactory("m1")
	require.NoError(t, sink.Update(context.Background(), snap))

	for _, client := range []*websocket.Conn{client1, client2} {
		client.SetReadDeadline(time.Now().Add(time.Second))
		_, msg, err := client.ReadMessage()
		require.NoError(t, err)
		var got avdomain.AvailabilitySnapshot
		require.NoError(t, json.Unmarshal(msg, &got))
		assert.Equal(t, "m1", got.MachineID)
	}
}

func TestWebsocketSink_Update_DoesNotDeliverToOtherMachineClients(t *testing.T) {
	hub := wssink.NewHub()
	sink := wssink.NewWebsocketSink(hub, observability.NewNopLogger())

	srv, serverConns := newUpgradeServer(t)

	clientM1 := dialClient(t, srv)
	hub.Register("m1", <-serverConns)

	clientM2 := dialClient(t, srv)
	hub.Register("m2", <-serverConns)

	require.NoError(t, sink.Update(context.Background(), snapshotFactory("m1")))

	// m1 client receives the snapshot
	clientM1.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := clientM1.ReadMessage()
	require.NoError(t, err)
	var got avdomain.AvailabilitySnapshot
	require.NoError(t, json.Unmarshal(msg, &got))
	assert.Equal(t, "m1", got.MachineID)

	// m2 client receives nothing — short deadline expires
	clientM2.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = clientM2.ReadMessage()
	assert.Error(t, err, "m2 client should not receive a message intended for m1")
}

func TestWebsocketSink_Update_DropsDeadClientWithoutBlockingBroadcast(t *testing.T) {
	hub := wssink.NewHub()
	sink := wssink.NewWebsocketSink(hub, observability.NewNopLogger())

	srv, serverConns := newUpgradeServer(t)

	// Dead client: register server conn then immediately close it
	deadClient := dialClient(t, srv)
	deadServerConn := <-serverConns
	hub.Register("m1", deadServerConn)
	deadClient.Close()
	deadServerConn.Close() // force write failure immediately

	// Live client
	liveClient := dialClient(t, srv)
	hub.Register("m1", <-serverConns)

	start := time.Now()
	require.NoError(t, sink.Update(context.Background(), snapshotFactory("m1")))
	assert.Less(t, time.Since(start), 500*time.Millisecond, "Broadcast must not block on dead conn")

	liveClient.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := liveClient.ReadMessage()
	require.NoError(t, err)
	var got avdomain.AvailabilitySnapshot
	require.NoError(t, json.Unmarshal(msg, &got))
	assert.Equal(t, "m1", got.MachineID)
}
