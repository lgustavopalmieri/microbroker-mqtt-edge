package session

// Logger defines the logging interface used by the session package.
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
