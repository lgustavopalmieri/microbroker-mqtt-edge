package pipeline

import (
	"context"

	"microbroker-mqtt-edge/internal/common/message"
)

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
