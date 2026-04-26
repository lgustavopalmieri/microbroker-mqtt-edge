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

// --- Topic ownership integration tests ---

func TestHandlePublish_OwnershipClaimed(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "owner-A")
	defer conn.Close()

	// First publish claims ownership
	conn.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"v":1}`), 0, 0))

	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/status", msg.Topic)
		assert.Equal(t, "owner-A", msg.ClientID)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	// Same client can keep publishing (idempotent)
	conn.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"v":2}`), 0, 0))

	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/status", msg.Topic)
		assert.Equal(t, []byte(`{"v":2}`), msg.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for second message")
	}
}

func TestHandlePublish_OwnershipRejectedForDifferentClient(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	// Client A claims "machine/status"
	connA := connectClient(t, ts, "owner-A")
	defer connA.Close()

	connA.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"v":1}`), 0, 0))

	select {
	case <-msgChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for owner-A message")
	}

	// Client B tries to publish to the same topic — should be rejected
	connB := connectClient(t, ts, "owner-B")
	defer connB.Close()

	connB.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"v":2}`), 0, 0))

	select {
	case <-msgChan:
		t.Fatal("message from non-owner should not have been forwarded")
	case <-time.After(200 * time.Millisecond):
		// expected: message rejected
	}
}

func TestHandlePublish_OwnershipRejectedQoS1_SendsPuback(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	// Client A claims "machine/alarm"
	connA := connectClient(t, ts, "owner-A")
	defer connA.Close()

	connA.Write(testutil.BuildPublishPacket("machine/alarm", []byte(`{"v":1}`), 0, 0))

	select {
	case <-msgChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for owner-A message")
	}

	// Client B publishes QoS 1 to same topic — rejected but gets PUBACK
	connB := connectClient(t, ts, "owner-B")
	defer connB.Close()

	connB.Write(testutil.BuildPublishPacket("machine/alarm", []byte(`{"v":2}`), 1, 77))

	connB.SetReadDeadline(time.Now().Add(2 * time.Second))
	pubackBuf := make([]byte, 4)
	_, err := io.ReadFull(connB, pubackBuf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x40), pubackBuf[0]) // PUBACK type
	assert.Equal(t, byte(0x00), pubackBuf[2]) // packet ID MSB
	assert.Equal(t, byte(0x4D), pubackBuf[3]) // packet ID LSB (77)

	select {
	case <-msgChan:
		t.Fatal("message from non-owner should not have been forwarded")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestHandlePublish_OwnershipReleasedOnDisconnect(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	// Client A claims "machine/status"
	connA := connectClient(t, ts, "owner-A")

	connA.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"v":1}`), 0, 0))

	select {
	case <-msgChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for owner-A message")
	}

	// Client A disconnects
	connA.Write([]byte{0xE0, 0x00}) // DISCONNECT
	connA.Close()
	time.Sleep(100 * time.Millisecond) // wait for server to process disconnect

	// Client B should now be able to claim the same topic
	connB := connectClient(t, ts, "owner-B")
	defer connB.Close()

	connB.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"v":2}`), 0, 0))

	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/status", msg.Topic)
		assert.Equal(t, "owner-B", msg.ClientID)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: owner-B should be able to publish after owner-A disconnected")
	}
}

func TestHandlePublish_ClientOwnsMultipleTopics(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "multi-owner")
	defer conn.Close()

	// Publish to both allowed topics
	conn.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"s":1}`), 0, 0))

	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/status", msg.Topic)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first topic message")
	}

	conn.Write(testutil.BuildPublishPacket("machine/alarm", []byte(`{"a":1}`), 0, 0))

	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/alarm", msg.Topic)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for second topic message")
	}
}

func TestHandlePublish_DifferentClientsOwnDifferentTopics(t *testing.T) {
	ts, msgChan, cancel := setupTestServer(t)
	defer cancel()

	// Client A owns "machine/status"
	connA := connectClient(t, ts, "client-A")
	defer connA.Close()

	connA.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"s":1}`), 0, 0))

	select {
	case <-msgChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	// Client B owns "machine/alarm"
	connB := connectClient(t, ts, "client-B")
	defer connB.Close()

	connB.Write(testutil.BuildPublishPacket("machine/alarm", []byte(`{"a":1}`), 0, 0))

	select {
	case <-msgChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	// Client B tries "machine/status" — rejected (owned by A)
	connB.Write(testutil.BuildPublishPacket("machine/status", []byte(`{"s":2}`), 0, 0))

	select {
	case <-msgChan:
		t.Fatal("client-B should not publish to machine/status owned by client-A")
	case <-time.After(200 * time.Millisecond):
	}

	// Client A tries "machine/alarm" — rejected (owned by B)
	connA.Write(testutil.BuildPublishPacket("machine/alarm", []byte(`{"a":2}`), 0, 0))

	select {
	case <-msgChan:
		t.Fatal("client-A should not publish to machine/alarm owned by client-B")
	case <-time.After(200 * time.Millisecond):
	}
}
