package protocol

// PacketType represents the MQTT Control Packet type (4-bit value in byte 1, bits 7-4).
type PacketType byte

const (
	Reserved1   PacketType = 0
	CONNECT     PacketType = 1
	CONNACK     PacketType = 2
	PUBLISH     PacketType = 3
	PUBACK      PacketType = 4
	PUBREC      PacketType = 5
	PUBREL      PacketType = 6
	PUBCOMP     PacketType = 7
	SUBSCRIBE   PacketType = 8
	SUBACK      PacketType = 9
	UNSUBSCRIBE PacketType = 10
	UNSUBACK    PacketType = 11
	PINGREQ     PacketType = 12
	PINGRESP    PacketType = 13
	DISCONNECT  PacketType = 14
	Reserved15  PacketType = 15
)

// String returns the human-readable name of the packet type.
func (p PacketType) String() string {
	switch p {
	case CONNECT:
		return "CONNECT"
	case CONNACK:
		return "CONNACK"
	case PUBLISH:
		return "PUBLISH"
	case PUBACK:
		return "PUBACK"
	case PUBREC:
		return "PUBREC"
	case PUBREL:
		return "PUBREL"
	case PUBCOMP:
		return "PUBCOMP"
	case SUBSCRIBE:
		return "SUBSCRIBE"
	case SUBACK:
		return "SUBACK"
	case UNSUBSCRIBE:
		return "UNSUBSCRIBE"
	case UNSUBACK:
		return "UNSUBACK"
	case PINGREQ:
		return "PINGREQ"
	case PINGRESP:
		return "PINGRESP"
	case DISCONNECT:
		return "DISCONNECT"
	default:
		return "RESERVED"
	}
}

// ConnackReturnCode represents the return code in a CONNACK packet.
type ConnackReturnCode byte

const (
	ConnAccepted           ConnackReturnCode = 0x00
	ConnRefusedProtocol    ConnackReturnCode = 0x01
	ConnRefusedIdentifier  ConnackReturnCode = 0x02
	ConnRefusedUnavailable ConnackReturnCode = 0x03
	ConnRefusedBadAuth     ConnackReturnCode = 0x04
	ConnRefusedNotAuth     ConnackReturnCode = 0x05
)

// FixedHeader is present in every MQTT Control Packet.
type FixedHeader struct {
	PacketType      PacketType
	Flags           byte
	RemainingLength int
}

// ConnectPacket represents a parsed MQTT CONNECT packet.
type ConnectPacket struct {
	ProtocolName  string
	ProtocolLevel byte
	ConnectFlags  byte
	KeepAlive     uint16
	ClientID      string
	WillTopic     string
	WillMessage   []byte
	Username      string
	Password      []byte
}

// HasUsername returns true if the Username Flag is set.
func (p *ConnectPacket) HasUsername() bool { return p.ConnectFlags&0x80 != 0 }

// HasPassword returns true if the Password Flag is set.
func (p *ConnectPacket) HasPassword() bool { return p.ConnectFlags&0x40 != 0 }

// WillRetain returns true if the Will Retain flag is set.
func (p *ConnectPacket) WillRetain() bool { return p.ConnectFlags&0x20 != 0 }

// WillQoS returns the Will QoS level (0, 1, or 2).
func (p *ConnectPacket) WillQoS() byte { return (p.ConnectFlags >> 3) & 0x03 }

// HasWill returns true if the Will Flag is set.
func (p *ConnectPacket) HasWill() bool { return p.ConnectFlags&0x04 != 0 }

// IsCleanSession returns true if the Clean Session flag is set.
func (p *ConnectPacket) IsCleanSession() bool { return p.ConnectFlags&0x02 != 0 }

// PublishPacket represents a parsed MQTT PUBLISH packet.
type PublishPacket struct {
	DUP       bool
	QoS       byte
	Retain    bool
	TopicName string
	PacketID  uint16
	Payload   []byte
}

// SubscribePacket represents a parsed MQTT SUBSCRIBE packet.
type SubscribePacket struct {
	PacketID      uint16
	Subscriptions []Subscription
}

// Subscription represents a single topic filter + QoS pair in a SUBSCRIBE packet.
type Subscription struct {
	TopicFilter string
	QoS         byte
}
