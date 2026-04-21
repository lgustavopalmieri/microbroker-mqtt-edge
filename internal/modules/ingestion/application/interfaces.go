package application

import (
	"context"

	"microbroker-mqtt-edge/internal/modules/ingestion/domain"
)

// Store defines the persistence contract for the ingestion pipeline.
// Implementations must be safe for concurrent use from multiple queue consumers.
type Store interface {
	// SaveRawData persists a single message atomically.
	// Must be safe for concurrent calls (implementations should serialize internally).
	SaveRawData(ctx context.Context, msg domain.Message) error

	// GetByTopic retrieves all messages for a given topic, ordered by insertion.
	// Primarily used for testing and debugging.
	GetByTopic(ctx context.Context, topic string) ([]domain.Message, error)

	// Close releases adapter-specific resources (may be no-op if connection is shared).
	Close() error
}

// Logger defines the logging interface used by the ingestion package.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
}

// NopLogger discards all log output. Useful for tests.
type NopLogger struct{}

func (NopLogger) Info(string, ...any)  {}
func (NopLogger) Error(string, ...any) {}
func (NopLogger) Warn(string, ...any)  {}
func (NopLogger) Debug(string, ...any) {}
