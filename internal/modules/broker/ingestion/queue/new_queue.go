package queue

import "microbroker-mqtt-edge/internal/common/message"

// Queue is a FIFO queue for a single topic, backed by a Go channel.
// Each queue has exactly one consumer goroutine that processes messages sequentially.
type Queue struct {
	topic    string
	messages chan message.Message
}

// NewQueue creates a new FIFO queue for the given topic with the specified buffer size.
func NewQueue(topic string, bufferSize int) *Queue {
	return &Queue{
		topic:    topic,
		messages: make(chan message.Message, bufferSize),
	}
}
