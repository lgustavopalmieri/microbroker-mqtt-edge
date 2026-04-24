package processing

import (
	"context"
	"fmt"
	"sync"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/modules/processing/domain"
)

// FanOut reads messages from an input channel and fans them out
// to all registered workers concurrently. It waits for every worker
// to finish before processing the next message.
type FanOut struct {
	workers []domain.Worker
	input   <-chan message.Message
	logger  Logger
}

// NewFanOut creates a FanOut that reads from input and
// distributes each message to every worker.
func NewFanOut(input <-chan message.Message, workers []domain.Worker, logger Logger) *FanOut {
	return &FanOut{
		workers: workers,
		input:   input,
		logger:  logger,
	}
}

// Start blocks, reading messages from the input channel and fanning
// out to workers until the channel is closed or the context is cancelled.
func (f *FanOut) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-f.input:
			if !ok {
				return
			}
			f.fanOut(ctx, msg)
		}
	}
}

// fanOut sends msg to every registered worker concurrently and waits
// for all of them to complete. A panic in any single worker is
// recovered so the fan-out keeps running.
func (f *FanOut) fanOut(ctx context.Context, msg message.Message) {
	var wg sync.WaitGroup
	for _, w := range f.workers {
		wg.Add(1)
		go func(worker domain.Worker) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					f.logger.Error("worker panicked",
						"worker", worker.Name(),
						"panic", fmt.Sprintf("%v", r),
					)
				}
			}()
			if err := worker.Process(ctx, msg); err != nil {
				f.logger.Error("worker process failed",
					"worker", worker.Name(),
					"topic", msg.Topic,
					"error", err,
				)
			}
		}(w)
	}
	wg.Wait()
}

// Close calls Close on every registered worker, collecting errors.
func (f *FanOut) Close() error {
	var firstErr error
	for _, w := range f.workers {
		if err := w.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
