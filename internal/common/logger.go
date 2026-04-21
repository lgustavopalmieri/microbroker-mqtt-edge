package common

import "log/slog"

// Logger defines the logging interface shared across all modules.
// Implementations must be safe for concurrent use.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
}

// slogLogger adapts *slog.Logger to the Logger interface.
type slogLogger struct {
	l *slog.Logger
}

// NewSlogLogger creates a Logger backed by the given *slog.Logger.
// If nil is passed, slog.Default() is used.
func NewSlogLogger(l *slog.Logger) Logger {
	if l == nil {
		l = slog.Default()
	}
	return &slogLogger{l: l}
}

func (s *slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogLogger) Error(msg string, args ...any) { s.l.Error(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s *slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }

// NopLogger discards all log output. Useful for tests.
type NopLogger struct{}

// NewNopLogger creates a Logger that discards all output.
func NewNopLogger() Logger { return NopLogger{} }

func (NopLogger) Info(string, ...any)  {}
func (NopLogger) Error(string, ...any) {}
func (NopLogger) Warn(string, ...any)  {}
func (NopLogger) Debug(string, ...any) {}
