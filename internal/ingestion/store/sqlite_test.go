package store

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/ingestion/domain"
)

// --- Test Helpers ---

// newTestStore creates an in-memory SQLite store with migrations applied.
// Returns the store and a cleanup function.
func newTestStore(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	require.NoError(t, err, "failed to create test store")

	err = store.Migrate(context.Background())
	require.NoError(t, err, "failed to migrate test store")

	cleanup := func() {
		store.Close()
	}
	return store, cleanup
}

func msgFactory(overrides ...func(*domain.Message)) domain.Message {
	msg := domain.Message{
		ClientID:  "test-client",
		Topic:     "machine/status",
		Payload:   []byte(`{"temp":42}`),
		Timezone:  "UTC",
		Timestamp: time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC),
	}
	for _, fn := range overrides {
		fn(&msg)
	}
	return msg
}

// --- Tests ---

func TestSQLiteStore_NewAndClose(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.NoError(t, store.Close())
}

func TestSQLiteStore_Migrate(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Verify table exists by inserting directly
	ctx := context.Background()
	_, err := store.db.ExecContext(ctx,
		`INSERT INTO raw_data (client, topic, timezone, timestamp, payload)
		 VALUES ('c', 't', 'UTC', '2026-01-01T00:00:00Z', '{}')`)
	assert.NoError(t, err, "table should exist after migration")
}

func TestSQLiteStore_MigrateIdempotent(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Running migrate again should not fail
	err := store.Migrate(context.Background())
	assert.NoError(t, err, "migrate should be idempotent")
}

func TestSQLiteStore_SaveRawData(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	msg := msgFactory()

	err := store.SaveRawData(ctx, msg)
	require.NoError(t, err)

	// Verify via GetByTopic
	messages, err := store.GetByTopic(ctx, "machine/status")
	require.NoError(t, err)
	require.Len(t, messages, 1)

	got := messages[0]
	assert.Equal(t, "test-client", got.ClientID)
	assert.Equal(t, "machine/status", got.Topic)
	assert.Equal(t, "UTC", got.Timezone)
	assert.Equal(t, []byte(`{"temp":42}`), got.Payload)
	assert.Equal(t, msg.Timestamp.Format(time.RFC3339Nano), got.Timestamp.Format(time.RFC3339Nano))
}

func TestSQLiteStore_SaveRawData_MultipleInserts_PreserveOrder(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	topic := "machine/production"

	for i := 0; i < 10; i++ {
		msg := msgFactory(func(m *domain.Message) {
			m.Topic = topic
			m.Payload = []byte(fmt.Sprintf(`{"seq":%d}`, i))
			m.Timestamp = time.Date(2026, 4, 21, 12, 0, i, 0, time.UTC)
		})
		err := store.SaveRawData(ctx, msg)
		require.NoError(t, err, "insert %d failed", i)
	}

	messages, err := store.GetByTopic(ctx, topic)
	require.NoError(t, err)
	require.Len(t, messages, 10)

	// Verify order
	for i, msg := range messages {
		expected := fmt.Sprintf(`{"seq":%d}`, i)
		assert.Equal(t, expected, string(msg.Payload), "message %d out of order", i)
	}
}

func TestSQLiteStore_SaveRawData_ConcurrentWrites(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	var wg sync.WaitGroup
	errChan := make(chan error, 50)

	// 5 goroutines writing 10 messages each (simulates 5 topic queues)
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				msg := msgFactory(func(m *domain.Message) {
					m.Topic = fmt.Sprintf("topic/%d", goroutineID)
					m.Payload = []byte(fmt.Sprintf(`{"g":%d,"i":%d}`, goroutineID, i))
					m.Timestamp = time.Now()
				})
				if err := store.SaveRawData(ctx, msg); err != nil {
					errChan <- err
				}
			}
		}(g)
	}

	wg.Wait()
	close(errChan)

	// No errors should have occurred
	for err := range errChan {
		t.Errorf("concurrent write error: %v", err)
	}

	// Verify total count: 5 goroutines * 10 messages = 50
	for g := 0; g < 5; g++ {
		messages, err := store.GetByTopic(ctx, fmt.Sprintf("topic/%d", g))
		require.NoError(t, err)
		assert.Len(t, messages, 10, "goroutine %d should have 10 messages", g)
	}
}

func TestSQLiteStore_GetByTopic_Empty(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	messages, err := store.GetByTopic(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestSQLiteStore_GetByTopic_FiltersByTopic(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Insert messages for different topics
	store.SaveRawData(ctx, msgFactory(func(m *domain.Message) { m.Topic = "topic/a"; m.Payload = []byte("a1") }))
	store.SaveRawData(ctx, msgFactory(func(m *domain.Message) { m.Topic = "topic/b"; m.Payload = []byte("b1") }))
	store.SaveRawData(ctx, msgFactory(func(m *domain.Message) { m.Topic = "topic/a"; m.Payload = []byte("a2") }))

	messagesA, err := store.GetByTopic(ctx, "topic/a")
	require.NoError(t, err)
	assert.Len(t, messagesA, 2)
	assert.Equal(t, []byte("a1"), messagesA[0].Payload)
	assert.Equal(t, []byte("a2"), messagesA[1].Payload)

	messagesB, err := store.GetByTopic(ctx, "topic/b")
	require.NoError(t, err)
	assert.Len(t, messagesB, 1)
}

func TestSQLiteStore_SaveRawData_BinaryPayload(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	// Non-JSON payload should be saved as-is
	msg := msgFactory(func(m *domain.Message) {
		m.Payload = []byte("this is not json, just raw bytes 0xFF")
	})

	err := store.SaveRawData(ctx, msg)
	require.NoError(t, err)

	messages, err := store.GetByTopic(ctx, msg.Topic)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, msg.Payload, messages[0].Payload)
}

func TestSQLiteStore_SaveRawData_ContextCancelled(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	msg := msgFactory()
	err := store.SaveRawData(ctx, msg)
	assert.Error(t, err, "should fail with cancelled context")
}
