package connection

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/auth"
	"microbroker-mqtt-edge/internal/modules/protocol"
	topicdomain "microbroker-mqtt-edge/internal/modules/topic/domain"
)

// --- Test helpers ---

func buildFullConnectPacket(clientID, username, password string, keepAlive uint16) []byte {
	var payload []byte

	// Protocol Name
	payload = append(payload, 0x00, 0x04, 'M', 'Q', 'T', 'T')
	// Protocol Level
	payload = append(payload, 0x04)
	// Connect Flags: Username + Password + CleanSession
	payload = append(payload, 0xC2)
	// Keep Alive
	payload = append(payload, byte(keepAlive>>8), byte(keepAlive&0xFF))
	// Client ID
	payload = append(payload, protocol.WriteUTF8String(clientID)...)
	// Username
	payload = append(payload, protocol.WriteUTF8String(username)...)
	// Password
	passBytes := []byte(password)
	payload = append(payload, byte(len(passBytes)>>8), byte(len(passBytes)&0xFF))
	payload = append(payload, passBytes...)

	// Build full packet: fixed header + payload
	var pkt []byte
	pkt = append(pkt, 0x10) // CONNECT type
	pkt = append(pkt, protocol.EncodeRemainingLength(len(payload))...)
	pkt = append(pkt, payload...)
	return pkt
}

func buildTestPublishPacket(topic string, payload []byte, qos byte, packetID uint16) []byte {
	var data []byte
	data = append(data, protocol.WriteUTF8String(topic)...)
	if qos > 0 {
		data = append(data, byte(packetID>>8), byte(packetID&0xFF))
	}
	data = append(data, payload...)

	flags := qos << 1
	var pkt []byte
	pkt = append(pkt, 0x30|flags) // PUBLISH type + flags
	pkt = append(pkt, protocol.EncodeRemainingLength(len(data))...)
	pkt = append(pkt, data...)
	return pkt
}

func buildSubscribePacket(packetID uint16, topics []string, qos []byte) []byte {
	var data []byte
	data = append(data, byte(packetID>>8), byte(packetID&0xFF))
	for i, t := range topics {
		data = append(data, protocol.WriteUTF8String(t)...)
		data = append(data, qos[i])
	}

	var pkt []byte
	pkt = append(pkt, 0x82) // SUBSCRIBE type + reserved flags
	pkt = append(pkt, protocol.EncodeRemainingLength(len(data))...)
	pkt = append(pkt, data...)
	return pkt
}

func setupTestServer(t *testing.T) (*Server, chan message.Message, context.CancelFunc) {
	t.Helper()
	topics, err := topicdomain.NewTopicRegistry([]string{"machine/status", "machine/alarm"})
	require.NoError(t, err)

	msgChan := make(chan message.Message, 100)
	connMgr := NewClientManager(5)
	authenticator := auth.NewEnvAuthenticator("admin", "secret")
	ctx, cancel := context.WithCancel(context.Background())

	server := NewServer("127.0.0.1:0", connMgr, authenticator, topics, msgChan, "UTC", observability.NopLogger{})

	go func() {
		server.ListenAndServe(ctx)
	}()

	// Wait for listener to be ready
	select {
	case <-server.ready:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start in time")
	}
	require.NotNil(t, server.Addr(), "server did not start")

	return server, msgChan, cancel
}

func dialServer(t *testing.T, server *Server) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", server.Addr().String(), time.Second)
	require.NoError(t, err)
	return conn
}

func readConnack(t *testing.T, conn net.Conn) (bool, protocol.ConnackReturnCode) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4)
	_, err := io.ReadFull(conn, buf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x20), buf[0]) // CONNACK type
	assert.Equal(t, byte(0x02), buf[1]) // remaining length
	sessionPresent := buf[2]&0x01 != 0
	returnCode := protocol.ConnackReturnCode(buf[3])
	return sessionPresent, returnCode
}

// --- Tests ---

func TestHandler_ConnectValid(t *testing.T) {
	server, _, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, server)
	defer conn.Close()

	conn.Write(buildFullConnectPacket("client1", "admin", "secret", 60))

	_, rc := readConnack(t, conn)
	assert.Equal(t, protocol.ConnAccepted, rc)
}

func TestHandler_ConnectBadAuth(t *testing.T) {
	server, _, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, server)
	defer conn.Close()

	conn.Write(buildFullConnectPacket("client1", "admin", "wrong", 60))

	_, rc := readConnack(t, conn)
	assert.Equal(t, protocol.ConnRefusedBadAuth, rc)
}

func TestHandler_PublishQoS0(t *testing.T) {
	server, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, server)
	defer conn.Close()

	// Connect
	conn.Write(buildFullConnectPacket("pub0", "admin", "secret", 60))
	readConnack(t, conn)

	// Publish QoS 0
	conn.Write(buildTestPublishPacket("machine/status", []byte(`{"temp":42}`), 0, 0))

	// Read message from channel
	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/status", msg.Topic)
		assert.Equal(t, "pub0", msg.ClientID)
		assert.Equal(t, []byte(`{"temp":42}`), msg.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestHandler_PublishQoS1(t *testing.T) {
	server, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, server)
	defer conn.Close()

	// Connect
	conn.Write(buildFullConnectPacket("pub1", "admin", "secret", 60))
	readConnack(t, conn)

	// Publish QoS 1
	conn.Write(buildTestPublishPacket("machine/alarm", []byte(`{"alarm":true}`), 1, 42))

	// Read PUBACK
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	pubackBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, pubackBuf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x40), pubackBuf[0]) // PUBACK type
	assert.Equal(t, byte(0x00), pubackBuf[2]) // PacketID MSB
	assert.Equal(t, byte(0x2A), pubackBuf[3]) // PacketID LSB = 42

	// Read message from channel
	select {
	case msg := <-msgChan:
		assert.Equal(t, "machine/alarm", msg.Topic)
		assert.Equal(t, []byte(`{"alarm":true}`), msg.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestHandler_PublishDisallowedTopic(t *testing.T) {
	server, msgChan, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, server)
	defer conn.Close()

	conn.Write(buildFullConnectPacket("pub-bad", "admin", "secret", 60))
	readConnack(t, conn)

	// Publish to disallowed topic
	conn.Write(buildTestPublishPacket("not/allowed", []byte("data"), 0, 0))

	// Should NOT appear in channel
	select {
	case <-msgChan:
		t.Fatal("message should not have been forwarded")
	case <-time.After(200 * time.Millisecond):
		// expected: no message
	}
}

func TestHandler_PingReqResp(t *testing.T) {
	server, _, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, server)
	defer conn.Close()

	conn.Write(buildFullConnectPacket("pinger", "admin", "secret", 60))
	readConnack(t, conn)

	// Send PINGREQ
	conn.Write([]byte{0xC0, 0x00})

	// Read PINGRESP
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 2)
	_, err := io.ReadFull(conn, buf)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xD0, 0x00}, buf)
}

func TestHandler_Disconnect(t *testing.T) {
	server, _, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, server)
	defer conn.Close()

	conn.Write(buildFullConnectPacket("disconnector", "admin", "secret", 60))
	readConnack(t, conn)

	// Send DISCONNECT
	conn.Write([]byte{0xE0, 0x00})

	// Connection should be closed by server
	time.Sleep(100 * time.Millisecond)
	_, ok := server.connMgr.Get("disconnector")
	assert.False(t, ok, "client should be removed after DISCONNECT")
}

func TestHandler_FirstPacketNotConnect(t *testing.T) {
	server, _, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, server)
	defer conn.Close()

	// Send PINGREQ as first packet (not CONNECT)
	conn.Write([]byte{0xC0, 0x00})

	// Server should close connection — read should fail
	conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 1)
	_, err := conn.Read(buf)
	assert.Error(t, err) // EOF or connection reset
}

func TestHandler_Subscribe(t *testing.T) {
	server, _, cancel := setupTestServer(t)
	defer cancel()

	conn := dialServer(t, server)
	defer conn.Close()

	conn.Write(buildFullConnectPacket("subscriber", "admin", "secret", 60))
	readConnack(t, conn)

	// Send SUBSCRIBE
	conn.Write(buildSubscribePacket(10, []string{"machine/status", "not/allowed"}, []byte{0x01, 0x00}))

	// Read SUBACK
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Read fixed header
	headerBuf := make([]byte, 2)
	_, err := io.ReadFull(conn, headerBuf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x90), headerBuf[0]) // SUBACK type

	remainingLen := int(headerBuf[1])
	body := make([]byte, remainingLen)
	_, err = io.ReadFull(conn, body)
	require.NoError(t, err)

	// Parse: PacketID (2 bytes) + return codes
	reader := bytes.NewReader(body)
	pidBuf := make([]byte, 2)
	io.ReadFull(reader, pidBuf)
	assert.Equal(t, byte(0x00), pidBuf[0])
	assert.Equal(t, byte(0x0A), pidBuf[1]) // PacketID = 10

	rc1, _ := reader.ReadByte()
	rc2, _ := reader.ReadByte()
	assert.Equal(t, byte(0x01), rc1) // machine/status → QoS 1 granted
	assert.Equal(t, byte(0x80), rc2) // not/allowed → failure
}
