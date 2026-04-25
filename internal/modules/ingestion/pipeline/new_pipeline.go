package pipeline

import (
	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/modules/ingestion/queue"
)

// Pipeline orchestrates the FIFO queues and routes messages from the input channel
// to the correct topic queue. Each queue has its own sequential consumer.
type Pipeline struct {
	queues       map[string]*queue.Queue
	store        Store
	dispatchChan chan<- message.Message
	bufSize      int
	logger       Logger
}

// NewPipeline creates a Pipeline with one FIFO queue per topic.
func NewPipeline(topics []string, store Store, dispatchChan chan<- message.Message, bufSize int, logger Logger) *Pipeline {
	queues := make(map[string]*queue.Queue, len(topics))
	for _, topic := range topics {
		queues[topic] = queue.NewQueue(topic, bufSize)
	}

	return &Pipeline{
		queues:       queues,
		store:        store,
		dispatchChan: dispatchChan,
		bufSize:      bufSize,
		logger:       logger,
	}
}
