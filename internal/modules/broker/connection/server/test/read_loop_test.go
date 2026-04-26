package test

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/testutil"
)

func TestReadLoop_PingReqResp(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "pinger")
	defer conn.Close()

	conn.Write([]byte{0xC0, 0x00})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 2)
	_, err := io.ReadFull(conn, buf)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xD0, 0x00}, buf)
}

func TestReadLoop_MultiplePings(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "multi-pinger")
	defer conn.Close()

	for range 3 {
		conn.Write([]byte{0xC0, 0x00})

		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 2)
		_, err := io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, []byte{0xD0, 0x00}, buf)
	}
}

func TestReadLoop_DisconnectStopsLoop(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "loop-disc")
	defer conn.Close()

	conn.Write([]byte{0xE0, 0x00})

	time.Sleep(100 * time.Millisecond)
	_, ok := ts.ConnMgr.Get("loop-disc")
	assert.False(t, ok, "client should be removed after DISCONNECT")
}

func TestReadLoop_PublishThenPing(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "pub-ping")
	defer conn.Close()

	conn.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"v":1}`), 0, 0))

	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/status", msg.Topic)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	conn.Write([]byte{0xC0, 0x00})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 2)
	_, err := io.ReadFull(conn, buf)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xD0, 0x00}, buf)
}
