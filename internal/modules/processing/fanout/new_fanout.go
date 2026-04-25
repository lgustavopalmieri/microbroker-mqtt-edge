package fanout

import (
	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
)

// FanOut reads messages from an input channel and fans them out
// to all registered workers concurrently. It waits for every worker
// to finish before processing the next message.
type FanOut struct {
	workers []Worker
	input   <-chan message.Message
	logger  observability.Logger
}

// NewFanOut creates a FanOut that reads from input and
// distributes each message to every worker.
func NewFanOut(input <-chan message.Message, workers []Worker, logger observability.Logger) *FanOut {
	return &FanOut{
		workers: workers,
		input:   input,
		logger:  logger,
	}
}
