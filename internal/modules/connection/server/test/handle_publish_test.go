package test

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/testutil"
)

func TestHandlePublish_QoS0(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "pub0")
	defer conn.Close()

	conn.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"temp":42}`), 0, 0))

	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/status", msg.Topic)
		assert.Equal(t, "pub0", msg.ClientID)
		assert.Equal(t, []byte(`{"temp":42}`), msg.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestHandlePublish_QoS1(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "pub1")
	defer conn.Close()

	conn.Write(testutil.BuildPublishPacket("machine/alarm", []byte(`{"alarm":true}`), 1, 42))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	pubackBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, pubackBuf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x40), pubackBuf[0])
	assert.Equal(t, byte(0x00), pubackBuf[2])
	assert.Equal(t, byte(0x2A), pubackBuf[3])

	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/alarm", msg.Topic)
		assert.Equal(t, []byte(`{"alarm":true}`), msg.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestHandlePublish_DisallowedTopic(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "pub-bad")
	defer conn.Close()

	conn.Write(testutil.BuildPublishPacket("not/allowed", []byte("data"), 0, 0))

	select {
	case <-msgChan:
		t.Fatal("message should not have been forwarded")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestHandlePublish_DisallowedTopicQoS1_SendsPuback(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "pub-bad-q1")
	defer conn.Close()

	conn.Write(testutil.BuildPublishPacket("not/allowed", []byte("data"), 1, 99))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	pubackBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, pubackBuf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x40), pubackBuf[0])
	assert.Equal(t, byte(0x00), pubackBuf[2])
	assert.Equal(t, byte(0x63), pubackBuf[3])

	select {
	case <-msgChan:
		t.Fatal("message should not have been forwarded")
	case <-time.After(200 * time.Millisecond):
	}
}
