package protocol

import "io"

// ReadPacket reads a complete MQTT Control Packet from the reader.
// It returns the fixed header, the remaining bytes (variable header + payload),
// and any error encountered.
func ReadPacket(reader io.Reader) (FixedHeader, []byte, error) {
	// Read first byte: packet type (bits 7-4) + flags (bits 3-0)
	firstByte := make([]byte, 1)
	if _, err := io.ReadFull(reader, firstByte); err != nil {
		return FixedHeader{}, nil, err
	}

	packetType := PacketType(firstByte[0] >> 4)
	flags := firstByte[0] & 0x0F

	// Validate packet type
	if packetType == Reserved1 || packetType == Reserved15 {
		return FixedHeader{}, nil, ErrInvalidPacketType
	}

	// Read remaining length
	remainingLength, err := DecodeRemainingLength(reader)
	if err != nil {
		return FixedHeader{}, nil, err
	}

	header := FixedHeader{
		PacketType:      packetType,
		Flags:           flags,
		RemainingLength: remainingLength,
	}

	// Read remaining bytes
	var data []byte
	if remainingLength > 0 {
		data = make([]byte, remainingLength)
		if _, err := io.ReadFull(reader, data); err != nil {
			return header, nil, err
		}
	}

	return header, data, nil
}
