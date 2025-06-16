package mqtt // parse MQTT binary (CONNECT, PUBLISH, etc.)

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

func ReadPacket(reader *bufio.Reader) (byte, int, error) {
	header, err := reader.ReadByte()
	if err != nil {
		return 0, 0, fmt.Errorf("read header: %w", err)
	}

	remLen, err := decodeRemainingLength(reader)
	if err != nil {
		return 0, 0, fmt.Errorf("read remaining length: %w", err)
	}

	return header, remLen, nil
}

func ReadPublish(reader *bufio.Reader, remLen int, header byte) (string, string, error) {
	buf := make([]byte, remLen)
	n, err := io.ReadFull(reader, buf)
	if err != nil {
		return "", "", fmt.Errorf("read publish payload: %w (got %d bytes)", err, n)
	}

	msgReader := bytes.NewReader(buf)

	var topicLen uint16
	if err := binary.Read(msgReader, binary.BigEndian, &topicLen); err != nil {
		return "", "", fmt.Errorf("read topic length: %w", err)
	}

	topic := make([]byte, topicLen)
	if _, err := msgReader.Read(topic); err != nil {
		return "", "", fmt.Errorf("read topic: %w", err)
	}

	// If QoS > 0, jump to packet ID (2 bytes)
	qos := (header >> 1) & 0x03
	if qos > 0 {
		var packetID uint16
		if err := binary.Read(msgReader, binary.BigEndian, &packetID); err != nil {
			return "", "", fmt.Errorf("read packet ID: %w", err)
		}
	}

	payload := make([]byte, msgReader.Len())
	if _, err := msgReader.Read(payload); err != nil {
		return "", "", fmt.Errorf("read payload: %w", err)
	}

	return string(topic), string(payload), nil
}

func decodeRemainingLength(r *bufio.Reader) (int, error) {
	multiplier := 1
	value := 0

	for i := 0; i < 4; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, fmt.Errorf("error reading remaining length byte: %w", err)
		}
		value += int(b&127) * multiplier
		if b&128 == 0 {
			break
		}
		multiplier *= 128
	}

	return value, nil
}
