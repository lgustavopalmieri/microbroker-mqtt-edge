package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"microbroker-mqtt-edge/internal/common/testutil"
	"microbroker-mqtt-edge/internal/modules/broker/protocol"
)

func TestHandleConnection_ConnectValid(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, ts)
	defer conn.Close()

	conn.Write(testutil.BuildConnectPacket("client1", "admin", "secret", 60))

	_, rc := readConnack(t, conn)
	assert.Equal(t, protocol.ConnAccepted, rc)
}

func TestHandleConnection_ConnectBadAuth(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, ts)
	defer conn.Close()

	conn.Write(testutil.BuildConnectPacket("client1", "admin", "wrong", 60))

	_, rc := readConnack(t, conn)
	assert.Equal(t, protocol.ConnRefusedBadAuth, rc)
}

func TestHandleConnection_FirstPacketNotConnect(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, ts)
	defer conn.Close()

	conn.Write([]byte{0xC0, 0x00})

	conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 1)
	_, err := conn.Read(buf)
	assert.Error(t, err)
}

func TestHandleConnection_Disconnect(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "disconnector")
	defer conn.Close()

	conn.Write([]byte{0xE0, 0x00})

	time.Sleep(100 * time.Millisecond)
	_, ok := ts.ConnMgr.Get("disconnector")
	assert.False(t, ok, "client should be removed after DISCONNECT")
}
