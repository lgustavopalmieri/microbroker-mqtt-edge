package test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/common/testutil"
	"microbroker-mqtt-edge/internal/modules/auth"
	clientmanager "microbroker-mqtt-edge/internal/modules/broker/connection/client_manager"
	"microbroker-mqtt-edge/internal/modules/broker/connection/server"
	"microbroker-mqtt-edge/internal/modules/broker/protocol"
	topicdomain "microbroker-mqtt-edge/internal/modules/topic"
)

// testServer bundles the server with its dependencies for test access.
type testServer struct {
	Server  *server.Server
	ConnMgr *clientmanager.ClientManager
}

func setupTestServer(t *testing.T) (*testServer, chan message.Message, context.CancelFunc) {
	t.Helper()
	topics, err := topicdomain.NewTopicRegistry([]string{"machine/status", "machine/alarm"})
	require.NoError(t, err)

	msgChan := make(chan message.Message, 100)
	connMgr := clientmanager.NewClientManager(5)
	authenticator := auth.NewEnvAuthenticator("admin", "secret")
	ctx, cancel := context.WithCancel(context.Background())

	srv := server.NewServer("127.0.0.1:0", connMgr, authenticator, topics, msgChan, "UTC", observability.NopLogger{})

	go func() {
		srv.ListenAndServe(ctx)
	}()

	select {
	case <-srv.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start in time")
	}
	require.NotNil(t, srv.Addr(), "server did not start")

	return &testServer{Server: srv, ConnMgr: connMgr}, msgChan, cancel
}

func dialServer(t *testing.T, ts *testServer) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", ts.Server.Addr().String(), time.Second)
	require.NoError(t, err)
	return conn
}

func connectClient(t *testing.T, ts *testServer, clientID string) net.Conn {
	t.Helper()
	conn := dialServer(t, ts)
	conn.Write(testutil.BuildConnectPacket(clientID, "admin", "secret", 60))
	testutil.ReadConnack(t, conn)
	return conn
}

func readConnack(t *testing.T, conn net.Conn) (bool, protocol.ConnackReturnCode) {
	t.Helper()
	return testutil.ReadConnack(t, conn)
}
