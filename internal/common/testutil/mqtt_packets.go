package testutil

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/modules/protocol"
)

// BuildConnectPacket builds a full MQTT CONNECT packet with username+password+clean session.
func BuildConnectPacket(clientID, username, password string, keepAlive uint16) []byte {
	var payload []byte

	payload = append(payload, 0x00, 0x04, 'M', 'Q', 'T', 'T')
	payload = append(payload, 0x04)
	payload = append(payload, 0xC2)
	payload = append(payload, byte(keepAlive>>8), byte(keepAlive&0xFF))
	payload = append(payload, protocol.WriteUTF8String(clientID)...)
	payload = append(payload, protocol.WriteUTF8String(username)...)
	passBytes := []byte(password)
	payload = append(payload, byte(len(passBytes)>>8), byte(len(passBytes)&0xFF))
	payload = append(payload, passBytes...)

	var pkt []byte
	pkt = append(pkt, 0x10)
	pkt = append(pkt, protocol.EncodeRemainingLength(len(payload))...)
	pkt = append(pkt, payload...)
	return pkt
}

// BuildPublishPacket builds an MQTT PUBLISH packet.
func BuildPublishPacket(topic string, payload []byte, qos byte, packetID uint16) []byte {
	var data []byte
	data = append(data, protocol.WriteUTF8String(topic)...)
	if qos > 0 {
		data = append(data, byte(packetID>>8), byte(packetID&0xFF))
	}
	data = append(data, payload...)

	flags := qos << 1
	var pkt []byte
	pkt = append(pkt, 0x30|flags)
	pkt = append(pkt, protocol.EncodeRemainingLength(len(data))...)
	pkt = append(pkt, data...)
	return pkt
}

// BuildSubscribePacket builds an MQTT SUBSCRIBE packet.
func BuildSubscribePacket(packetID uint16, topics []string, qos []byte) []byte {
	var data []byte
	data = append(data, byte(packetID>>8), byte(packetID&0xFF))
	for i, t := range topics {
		data = append(data, protocol.WriteUTF8String(t)...)
		data = append(data, qos[i])
	}

	var pkt []byte
	pkt = append(pkt, 0x82)
	pkt = append(pkt, protocol.EncodeRemainingLength(len(data))...)
	pkt = append(pkt, data...)
	return pkt
}

// ReadConnack reads and validates a CONNACK packet from the connection.
func ReadConnack(t *testing.T, conn net.Conn) (bool, protocol.ConnackReturnCode) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 4)
	_, err := io.ReadFull(conn, buf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x20), buf[0])
	assert.Equal(t, byte(0x02), buf[1])
	sessionPresent := buf[2]&0x01 != 0
	returnCode := protocol.ConnackReturnCode(buf[3])
	return sessionPresent, returnCode
}

// ReadPuback reads and validates a PUBACK packet from the connection.
func ReadPuback(t *testing.T, conn net.Conn, expectedID uint16) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 4)
	_, err := io.ReadFull(conn, buf)
	require.NoError(t, err)
	require.Equal(t, byte(0x40), buf[0])
	gotID := uint16(buf[2])<<8 | uint16(buf[3])
	require.Equal(t, expectedID, gotID)
}
