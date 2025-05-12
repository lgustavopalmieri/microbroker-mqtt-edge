package queue

// InboundMessage represents a decoded MQTT PUBLISH message
type InboundMessage struct {
	Topic   string
	Payload []byte
}
