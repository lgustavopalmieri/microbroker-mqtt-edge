package dispatch

import (
	"context"
	"fmt"
	"sync"

	"microbroker-mqtt-edge/internal/modules/dispatch/domain"
)

// Dispatcher reads messages from an input channel and fans them out
// to all registered workers concurrently. It waits for every worker
// to finish before processing the next message.
type Dispatcher struct {
	workers []domain.Worker
	input   <-chan domain.Message
	logger  Logger
}

// NewDispatcher creates a Dispatcher that reads from input and
// distributes each message to every worker.
func NewDispatcher(input <-chan domain.Message, workers []domain.Worker, logger Logger) *Dispatcher {
	return &Dispatcher{
		workers: workers,
		input:   input,
		logger:  logger,
	}
}

// Start blocks, reading messages from the input channel and fanning
// out to workers until the channel is closed or the context is cancelled.
func (d *Dispatcher) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-d.input:
			if !ok {
				return
			}
			d.fanOut(ctx, msg)
		}
	}
}

// fanOut sends msg to every registered worker concurrently and waits
// for all of them to complete. A panic in any single worker is
// recovered so the dispatcher keeps running.
func (d *Dispatcher) fanOut(ctx context.Context, msg domain.Message) {
	var wg sync.WaitGroup
	for _, w := range d.workers {
		wg.Add(1)
		go func(worker domain.Worker) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					d.logger.Error("worker panicked",
						"worker", worker.Name(),
						"panic", fmt.Sprintf("%v", r),
					)
				}
			}()
			if err := worker.Process(ctx, msg); err != nil {
				d.logger.Error("worker process failed",
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
func (d *Dispatcher) Close() error {
	var firstErr error
	for _, w := range d.workers {
		if err := w.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
