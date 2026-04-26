package test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
	clientmanager "microbroker-mqtt-edge/internal/modules/broker/connection/client_manager"
	"microbroker-mqtt-edge/internal/modules/broker/connection/server"
	topicdomain "microbroker-mqtt-edge/internal/modules/topic"
)

func TestServer_ListenAndServe_Ready(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	assert.NotNil(t, ts.Server.Addr())
}

func TestServer_ListenAndServe_InvalidAddress(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	// Bind to the same port to force an error
	addr := ts.Server.Addr().String()
	topics, _ := topicdomain.NewTopicRegistry([]string{"t/1"})
	msgChan := make(chan message.Message, 1)
	connMgr := clientmanager.NewClientManager(5)

	badSrv := server.NewServer(addr, connMgr, stubAuth{true}, topics, msgChan, "UTC", observability.NopLogger{})

	err := badSrv.ListenAndServe(context.Background())
	assert.Error(t, err)
}

func TestServer_Close_NilListener(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	cancel()
	time.Sleep(50 * time.Millisecond)

	err := ts.Server.Close()
	_ = err
}

func TestServer_Addr_BeforeStart(t *testing.T) {
	topics, _ := topicdomain.NewTopicRegistry([]string{"t/1"})
	msgChan := make(chan message.Message, 1)
	connMgr := clientmanager.NewClientManager(5)

	srv := server.NewServer("127.0.0.1:0", connMgr, stubAuth{true}, topics, msgChan, "UTC", observability.NopLogger{})
	assert.Nil(t, srv.Addr())
}

func TestServer_RejectsWhenMaxClients(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conns := make([]net.Conn, 5)
	for i := range 5 {
		conns[i] = connectClient(t, ts, string(rune('a'+i)))
		defer conns[i].Close()
	}

	conn6, err := net.DialTimeout("tcp", ts.Server.Addr().String(), time.Second)
	require.NoError(t, err)
	defer conn6.Close()

	conn6.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 1)
	_, err = conn6.Read(buf)
	assert.Error(t, err)
}

func TestServer_GracefulShutdown(t *testing.T) {
	ts, _, cancel := setupTestServer(t)

	conn := connectClient(t, ts, "shutdown-client")
	defer conn.Close()

	cancel()
	time.Sleep(100 * time.Millisecond)

	_, err := net.DialTimeout("tcp", ts.Server.Addr().String(), 500*time.Millisecond)
	assert.Error(t, err)
}
