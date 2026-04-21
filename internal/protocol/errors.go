package protocol

import "errors"

var (
	// ErrMalformedRemainingLength indicates the remaining length field uses more than 4 bytes.
	ErrMalformedRemainingLength = errors.New("protocol: malformed remaining length")

	// ErrInvalidProtocol indicates the protocol name is not "MQTT" or the level is not 4.
	ErrInvalidProtocol = errors.New("protocol: invalid protocol name or level")

	// ErrInvalidPacketType indicates a reserved or unknown packet type (0 or 15).
	ErrInvalidPacketType = errors.New("protocol: invalid packet type")

	// ErrInvalidQoS indicates a QoS value of 3, which is forbidden by the spec.
	ErrInvalidQoS = errors.New("protocol: invalid QoS level")

	// ErrEmptyPayload indicates a packet that requires a payload received none.
	ErrEmptyPayload = errors.New("protocol: empty payload")

	// ErrTruncatedData indicates the packet data ended before all fields could be read.
	ErrTruncatedData = errors.New("protocol: truncated data")

	// ErrInvalidReservedBit indicates the reserved bit in CONNECT flags is not zero.
	ErrInvalidReservedBit = errors.New("protocol: reserved bit must be zero")

	// ErrEmptyTopicName indicates a PUBLISH packet with an empty topic name.
	ErrEmptyTopicName = errors.New("protocol: empty topic name")
)
