package audit

import "context"

// Record represents a single persisted message returned by the audit query.
type Record struct {
	ClientID  string `json:"client_id"`
	Topic     string `json:"topic"`
	Timezone  string `json:"timezone"`
	Timestamp string `json:"timestamp"`
	Payload   string `json:"payload"`
}

// Reader defines the port for querying persisted messages.
// Implementations must be safe for concurrent use.
type Reader interface {
	GetByTopic(ctx context.Context, topic string) ([]Record, error)
	CountByTopic(ctx context.Context, topic string) (int64, error)
}

// Logger defines the logging interface used by the audit package.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}
