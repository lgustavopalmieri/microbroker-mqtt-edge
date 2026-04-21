package protocol

import (
	"encoding/binary"
	"io"
)

// DecodeRemainingLength reads and decodes the MQTT variable-length encoding
// for the Remaining Length field from the given reader.
// Returns the decoded integer value or an error if the encoding is malformed.
func DecodeRemainingLength(reader io.Reader) (int, error) {
	multiplier := 1
	value := 0
	buf := make([]byte, 1)

	for {
		if _, err := io.ReadFull(reader, buf); err != nil {
			return 0, err
		}
		encodedByte := buf[0]
		value += int(encodedByte&0x7F) * multiplier
		multiplier *= 128

		if multiplier > 128*128*128*128 {
			return 0, ErrMalformedRemainingLength
		}
		if encodedByte&0x80 == 0 {
			break
		}
	}
	return value, nil
}

// EncodeRemainingLength encodes an integer into the MQTT variable-length
// encoding used for the Remaining Length field.
func EncodeRemainingLength(length int) []byte {
	var encoded []byte
	for {
		encodedByte := byte(length % 128)
		length /= 128
		if length > 0 {
			encodedByte |= 0x80
		}
		encoded = append(encoded, encodedByte)
		if length == 0 {
			break
		}
	}
	return encoded
}

// ReadUTF8String reads an MQTT UTF-8 encoded string (2-byte length prefix + content).
func ReadUTF8String(reader io.Reader) (string, error) {
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(reader, lenBuf); err != nil {
		return "", err
	}
	length := int(binary.BigEndian.Uint16(lenBuf))
	if length == 0 {
		return "", nil
	}
	strBuf := make([]byte, length)
	if _, err := io.ReadFull(reader, strBuf); err != nil {
		return "", err
	}
	return string(strBuf), nil
}

// WriteUTF8String encodes a string into the MQTT UTF-8 format (2-byte length prefix + content).
func WriteUTF8String(s string) []byte {
	length := len(s)
	buf := make([]byte, 2+length)
	binary.BigEndian.PutUint16(buf, uint16(length))
	copy(buf[2:], s)
	return buf
}

// ReadBinaryData reads MQTT binary data (2-byte length prefix + raw bytes).
// Used for Password and Will Message fields.
func ReadBinaryData(reader io.Reader) ([]byte, error) {
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(reader, lenBuf); err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint16(lenBuf))
	if length == 0 {
		return []byte{}, nil
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}
	return data, nil
}
