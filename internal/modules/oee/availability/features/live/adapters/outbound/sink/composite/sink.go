package composite

import (
	"context"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/live/application"
)

// CompositeSink fans out AvailabilitySnapshot updates to multiple sinks.
// Each sink gets a dedicated drain goroutine backed by a buffered channel of
// size 1. Update is non-blocking — if a sink is busy the update is dropped
// for that sink rather than blocking the engine ticker.
type CompositeSink struct {
	chans  []chan avdomain.AvailabilitySnapshot
	cancel context.CancelFunc
	logger observability.Logger
}

// NewCompositeSink creates a CompositeSink that fans out to the given sinks
// and starts one drain goroutine per sink. Call Close to stop them.
func NewCompositeSink(logger observability.Logger, sinks ...application.AvailabilitySink) *CompositeSink {
	ctx, cancel := context.WithCancel(context.Background())
	c := &CompositeSink{
		chans:  make([]chan avdomain.AvailabilitySnapshot, len(sinks)),
		cancel: cancel,
		logger: logger,
	}
	for i, sink := range sinks {
		ch := make(chan avdomain.AvailabilitySnapshot, 1)
		c.chans[i] = ch
		go c.drain(ctx, sink, ch)
	}
	return c
}

// Update fans out the snapshot to all sinks non-blocking.
// A sink whose channel is full (busy processing the previous update) is
// skipped — the engine ticker is never delayed by a slow sink.
func (c *CompositeSink) Update(_ context.Context, s avdomain.AvailabilitySnapshot) error {
	for _, ch := range c.chans {
		select {
		case ch <- s:
		default:
		}
	}
	return nil
}

// Close cancels the shared context, signalling all drain goroutines to stop.
func (c *CompositeSink) Close() error {
	c.cancel()
	return nil
}

func (c *CompositeSink) drain(ctx context.Context, sink application.AvailabilitySink, ch <-chan avdomain.AvailabilitySnapshot) {
	for {
		select {
		case <-ctx.Done():
			return
		case s := <-ch:
			if err := sink.Update(ctx, s); err != nil {
				c.logger.Error("composite sink: update failed", "error", err)
			}
		}
	}
}
