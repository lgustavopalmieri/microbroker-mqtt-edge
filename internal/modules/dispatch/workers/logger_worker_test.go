package workers_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"microbroker-mqtt-edge/internal/modules/dispatch/domain"
	"microbroker-mqtt-edge/internal/modules/dispatch/workers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- spy logger -------------------------------------------------------------

type logEntry struct {
	msg  string
	args []any
}

type spyLogger struct {
	mu      sync.Mutex
	entries []logEntry
}

func (l *spyLogger) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, logEntry{msg: msg, args: args})
}

func (l *spyLogger) getEntries() []logEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	cp := make([]logEntry, len(l.entries))
	copy(cp, l.entries)
	return cp
}

// --- tests ------------------------------------------------------------------

func TestLoggerWorker_Name(t *testing.T) {
	w := workers.NewLoggerWorker(&spyLogger{})
	assert.Equal(t, "logger", w.Name())
}

func TestLoggerWorker_Process_LogsCorrectFields(t *testing.T) {
	logger := &spyLogger{}
	w := workers.NewLoggerWorker(logger)

	msg := domain.Message{
		ClientID:  "device-01",
		Topic:     "machine/status",
		Payload:   []byte(`{"rpm":1500}`),
		Timezone:  "America/Sao_Paulo",
		Timestamp: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
	}

	err := w.Process(context.Background(), msg)
	require.NoError(t, err)

	entries := logger.getEntries()
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, "message received", e.msg)

	// Verify key-value pairs in args.
	argsMap := make(map[string]any)
	for i := 0; i+1 < len(e.args); i += 2 {
		key, ok := e.args[i].(string)
		if ok {
			argsMap[key] = e.args[i+1]
		}
	}

	assert.Equal(t, "machine/status", argsMap["topic"])
	assert.Equal(t, "device-01", argsMap["clientID"])
	assert.Equal(t, len(msg.Payload), argsMap["payloadSize"])
	assert.Equal(t, msg.Timestamp, argsMap["timestamp"])
}

func TestLoggerWorker_Close_ReturnsNil(t *testing.T) {
	w := workers.NewLoggerWorker(&spyLogger{})
	assert.NoError(t, w.Close())
}
