package domain

import (
	"context"

	ingestiondomain "microbroker-mqtt-edge/internal/modules/ingestion/domain"
)

// Message is a type alias for the ingestion domain Message,
// used throughout the dispatch module to avoid tight coupling.
type Message = ingestiondomain.Message

// Worker defines the contract for any forwarding/processing plugin.
// Implementations must be safe for concurrent use.
type Worker interface {
	// Name returns a human-readable identifier for the worker.
	Name() string

	// Process handles a single persisted message.
	// Implementations should not block indefinitely.
	Process(ctx context.Context, msg Message) error

	// Close releases any resources held by the worker.
	// Called once during graceful shutdown.
	Close() error
}
