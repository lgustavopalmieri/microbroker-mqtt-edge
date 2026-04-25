package logger

import (
	"context"

	"microbroker-mqtt-edge/internal/common/message"
)

// Logger defines the logging interface used by workers in this package.
type Logger interface {
	Info(msg string, args ...any)
}

// LoggerWorker is a reference Worker implementation that logs every
// message it receives. Useful for debugging and validating the pipeline.
type LoggerWorker struct {
	logger Logger
}

// NewLoggerWorker creates a LoggerWorker that writes to the given logger.
func NewLoggerWorker(logger Logger) *LoggerWorker {
	return &LoggerWorker{logger: logger}
}

// Name returns "logger".
func (w *LoggerWorker) Name() string { return "logger" }

// Process logs the message topic, client ID, payload size and timestamp.
func (w *LoggerWorker) Process(_ context.Context, msg message.Message) error {
	w.logger.Info("message received",
		"topic", msg.Topic,
		"clientID", msg.ClientID,
		"payloadSize", len(msg.Payload),
		"timestamp", msg.Timestamp,
	)
	return nil
}

// Close is a no-op — the logger worker holds no resources.
func (w *LoggerWorker) Close() error { return nil }
