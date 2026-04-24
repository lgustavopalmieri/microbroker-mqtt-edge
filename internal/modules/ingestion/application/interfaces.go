package application

import (
	"context"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
)

// Store defines the persistence contract for the ingestion pipeline.
// Implementations must be safe for concurrent use from multiple queue consumers.
type Store interface {
	// SaveRawData persists a single message atomically.
	// Must be safe for concurrent calls (implementations should serialize internally).
	SaveRawData(ctx context.Context, msg message.Message) error

	// GetByTopic retrieves all messages for a given topic, ordered by insertion.
	// Primarily used for testing and debugging.
	GetByTopic(ctx context.Context, topic string) ([]message.Message, error)

	// Close releases adapter-specific resources (may be no-op if connection is shared).
	Close() error
}

// Logger is the observability contract used by the ingestion module.
type Logger = observability.Logger
