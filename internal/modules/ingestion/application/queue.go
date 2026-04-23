package application

import (
	"context"

	"microbroker-mqtt-edge/internal/modules/ingestion/domain"
)

// Queue is a FIFO queue for a single topic, backed by a Go channel.
// Each queue has exactly one consumer goroutine that processes messages sequentially.
type Queue struct {
	topic    string
	messages chan domain.Message
}

// NewQueue creates a new FIFO queue for the given topic with the specified buffer size.
func NewQueue(topic string, bufferSize int) *Queue {
	return &Queue{
		topic:    topic,
		messages: make(chan domain.Message, bufferSize),
	}
}

// Enqueue adds a message to the queue. Blocks if the buffer is full (backpressure).
func (q *Queue) Enqueue(msg domain.Message) {
	q.messages <- msg
}

// Topic returns the topic name this queue handles.
func (q *Queue) Topic() string {
	return q.topic
}

// StartConsumer runs the sequential consumer loop for this queue.
// For each message: persist via Store, then forward to dispatchChan.
// If Store fails, the message is logged and skipped (not forwarded).
// Blocks until context is cancelled or the queue channel is closed.
func (q *Queue) StartConsumer(ctx context.Context, store Store, dispatchChan chan<- domain.Message, logger Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-q.messages:
			if !ok {
				return
			}

			// 1. Persist
			if err := store.SaveRawData(ctx, msg); err != nil {
				logger.Error("failed to save message",
					"topic", q.topic,
					"client", msg.ClientID,
					"error", err,
				)
				continue // skip forwarding, process next message
			}

			// 2. Forward to dispatch (blocks if dispatch channel is full = backpressure)
			select {
			case dispatchChan <- msg:
			case <-ctx.Done():
				return
			}
		}
	}
}
