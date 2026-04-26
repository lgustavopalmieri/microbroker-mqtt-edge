package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// DecodeConnect parses the variable header and payload of a CONNECT packet.
// The data parameter should contain everything after the fixed header (remaining length bytes).
func DecodeConnect(data []byte) (*ConnectPacket, error) {
	reader := bytes.NewReader(data)
	pkt := &ConnectPacket{}

	// Protocol Name
	name, err := ReadUTF8String(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: reading protocol name: %v", ErrTruncatedData, err)
	}
	if name != "MQTT" {
		return nil, fmt.Errorf("%w: got %q", ErrInvalidProtocol, name)
	}
	pkt.ProtocolName = name

	// Protocol Level
	levelBuf := make([]byte, 1)
	if _, err := io.ReadFull(reader, levelBuf); err != nil {
		return nil, fmt.Errorf("%w: reading protocol level: %v", ErrTruncatedData, err)
	}
	if levelBuf[0] != 4 {
		return nil, fmt.Errorf("%w: level %d", ErrInvalidProtocol, levelBuf[0])
	}
	pkt.ProtocolLevel = levelBuf[0]

	// Connect Flags
	flagsBuf := make([]byte, 1)
	if _, err := io.ReadFull(reader, flagsBuf); err != nil {
		return nil, fmt.Errorf("%w: reading connect flags: %v", ErrTruncatedData, err)
	}
	pkt.ConnectFlags = flagsBuf[0]

	// Validate reserved bit (bit 0 must be 0)
	if pkt.ConnectFlags&0x01 != 0 {
		return nil, ErrInvalidReservedBit
	}

	// Keep Alive
	keepAliveBuf := make([]byte, 2)
	if _, err := io.ReadFull(reader, keepAliveBuf); err != nil {
		return nil, fmt.Errorf("%w: reading keep alive: %v", ErrTruncatedData, err)
	}
	pkt.KeepAlive = binary.BigEndian.Uint16(keepAliveBuf)

	// --- Payload ---

	// Client ID (always present)
	clientID, err := ReadUTF8String(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: reading client id: %v", ErrTruncatedData, err)
	}
	pkt.ClientID = clientID

	// Will Topic + Will Message (if Will Flag set)
	if pkt.HasWill() {
		willTopic, err := ReadUTF8String(reader)
		if err != nil {
			return nil, fmt.Errorf("%w: reading will topic: %v", ErrTruncatedData, err)
		}
		pkt.WillTopic = willTopic

		willMsg, err := ReadBinaryData(reader)
		if err != nil {
			return nil, fmt.Errorf("%w: reading will message: %v", ErrTruncatedData, err)
		}
		pkt.WillMessage = willMsg
	}

	// Username (if Username Flag set)
	if pkt.HasUsername() {
		username, err := ReadUTF8String(reader)
		if err != nil {
			return nil, fmt.Errorf("%w: reading username: %v", ErrTruncatedData, err)
		}
		pkt.Username = username
	}

	// Password (if Password Flag set)
	if pkt.HasPassword() {
		password, err := ReadBinaryData(reader)
		if err != nil {
			return nil, fmt.Errorf("%w: reading password: %v", ErrTruncatedData, err)
		}
		pkt.Password = password
	}

	return pkt, nil
}

// DecodePublish parses the variable header and payload of a PUBLISH packet.
// The header parameter provides the fixed header (for DUP, QoS, Retain flags).
// The data parameter contains everything after the fixed header.
func DecodePublish(header FixedHeader, data []byte) (*PublishPacket, error) {
	pkt := &PublishPacket{}

	// Extract flags from fixed header
	pkt.DUP = header.Flags&0x08 != 0
	pkt.QoS = (header.Flags >> 1) & 0x03
	pkt.Retain = header.Flags&0x01 != 0

	// Validate QoS
	if pkt.QoS == 3 {
		return nil, ErrInvalidQoS
	}

	reader := bytes.NewReader(data)

	// Topic Name
	topicName, err := ReadUTF8String(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: reading topic name: %v", ErrTruncatedData, err)
	}
	if topicName == "" {
		return nil, ErrEmptyTopicName
	}
	pkt.TopicName = topicName

	// Packet ID (only if QoS > 0)
	if pkt.QoS > 0 {
		pidBuf := make([]byte, 2)
		if _, err := io.ReadFull(reader, pidBuf); err != nil {
			return nil, fmt.Errorf("%w: reading packet id: %v", ErrTruncatedData, err)
		}
		pkt.PacketID = binary.BigEndian.Uint16(pidBuf)
	}

	// Payload (remaining bytes)
	remaining := reader.Len()
	if remaining > 0 {
		pkt.Payload = make([]byte, remaining)
		if _, err := io.ReadFull(reader, pkt.Payload); err != nil {
			return nil, fmt.Errorf("%w: reading payload: %v", ErrTruncatedData, err)
		}
	}

	return pkt, nil
}

// DecodeSubscribe parses the variable header and payload of a SUBSCRIBE packet.
// The data parameter contains everything after the fixed header.
func DecodeSubscribe(data []byte) (*SubscribePacket, error) {
	if len(data) < 5 { // minimum: 2 (packet id) + 2 (topic length) + 1 (qos)
		return nil, ErrEmptyPayload
	}

	reader := bytes.NewReader(data)
	pkt := &SubscribePacket{}

	// Packet ID
	pidBuf := make([]byte, 2)
	if _, err := io.ReadFull(reader, pidBuf); err != nil {
		return nil, fmt.Errorf("%w: reading packet id: %v", ErrTruncatedData, err)
	}
	pkt.PacketID = binary.BigEndian.Uint16(pidBuf)

	// Topic Filter + QoS pairs
	for reader.Len() > 0 {
		topicFilter, err := ReadUTF8String(reader)
		if err != nil {
			return nil, fmt.Errorf("%w: reading topic filter: %v", ErrTruncatedData, err)
		}

		qosBuf := make([]byte, 1)
		if _, err := io.ReadFull(reader, qosBuf); err != nil {
			return nil, fmt.Errorf("%w: reading qos: %v", ErrTruncatedData, err)
		}

		qos := qosBuf[0] & 0x03
		if qosBuf[0] > 2 {
			return nil, fmt.Errorf("%w: subscription qos %d", ErrInvalidQoS, qosBuf[0])
		}

		pkt.Subscriptions = append(pkt.Subscriptions, Subscription{
			TopicFilter: topicFilter,
			QoS:         qos,
		})
	}

	if len(pkt.Subscriptions) == 0 {
		return nil, ErrEmptyPayload
	}

	return pkt, nil
}
