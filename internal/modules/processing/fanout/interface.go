package fanout

import (
	"context"

	"microbroker-mqtt-edge/internal/common/message"
)

// Worker defines the contract for any message processing plugin.
// Implementations must be safe for concurrent use.
type Worker interface {
	// Name returns a human-readable identifier for the worker.
	Name() string

	// Process handles a single persisted message.
	// Implementations should not block indefinitely.
	Process(ctx context.Context, msg message.Message) error

	// Close releases any resources held by the worker.
	// Called once during graceful shutdown.
	Close() error
}
