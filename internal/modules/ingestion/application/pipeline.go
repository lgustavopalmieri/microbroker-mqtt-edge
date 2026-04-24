package application

import (
	"context"

	"microbroker-mqtt-edge/internal/common/message"
)

// Pipeline orchestrates the FIFO queues and routes messages from the input channel
// to the correct topic queue. Each queue has its own sequential consumer.
type Pipeline struct {
	queues       map[string]*Queue
	store        Store
	dispatchChan chan<- message.Message
	bufSize      int
	logger       Logger
}

// NewPipeline creates a Pipeline with one FIFO queue per topic.
func NewPipeline(topics []string, store Store, dispatchChan chan<- message.Message, bufSize int, logger Logger) *Pipeline {
	queues := make(map[string]*Queue, len(topics))
	for _, topic := range topics {
		queues[topic] = NewQueue(topic, bufSize)
	}

	return &Pipeline{
		queues:       queues,
		store:        store,
		dispatchChan: dispatchChan,
		bufSize:      bufSize,
		logger:       logger,
	}
}

// Start launches all queue consumers and begins routing messages from inputChan
// to the appropriate topic queue. Blocks until context is cancelled or inputChan is closed.
func (p *Pipeline) Start(ctx context.Context, inputChan <-chan message.Message) {
	// Start one consumer goroutine per topic queue
	for _, q := range p.queues {
		go q.StartConsumer(ctx, p.store, p.dispatchChan, p.logger)
	}

	// Route messages to the correct queue
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-inputChan:
			if !ok {
				return
			}
			q, exists := p.queues[msg.Topic]
			if !exists {
				p.logger.Warn("message for unknown topic, discarding",
					"topic", msg.Topic,
					"client", msg.ClientID,
				)
				continue
			}
			q.Enqueue(msg)
		}
	}
}

// QueueCount returns the number of topic queues (for testing).
func (p *Pipeline) QueueCount() int {
	return len(p.queues)
}
