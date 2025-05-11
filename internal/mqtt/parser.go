package mqtt // parse binário MQTT (CONNECT, PUBLISH, etc.)

import (
	"bufio"
	"fmt"
	"net"
)

func ReadPacket(conn net.Conn) (packetType byte, remaining int, err error) {
	reader := bufio.NewReader(conn)

	// First byte
	header, err := reader.ReadByte()
	if err != nil {
		return 0, 0, fmt.Errorf("read header: %w", err)
	}

	packetType = header >> 4

	// Remaining length (simplificado, max 1 byte por enquanto)
	remainingByte, err := reader.ReadByte()
	if err != nil {
		return 0, 0, fmt.Errorf("read remaining length: %w", err)
	}
	remaining = int(remainingByte)

	// Descarta payload (não tratamos ainda)
	buf := make([]byte, remaining)
	_, err = reader.Read(buf)
	if err != nil {
		return 0, 0, fmt.Errorf("read payload: %w", err)
	}

	return packetType, remaining, nil
}
