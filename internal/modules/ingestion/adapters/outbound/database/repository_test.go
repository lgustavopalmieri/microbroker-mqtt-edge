package database

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/platform/database"
)

// --- Test Helpers ---

// newTestRepository creates an in-memory SQLite connection, runs migrations,
// and returns a repository + cleanup function.
// This mirrors how bootstrap will work: platform creates DB → migrates → injects into adapter.
func newTestRepository(t *testing.T) (*SQLiteRepository, func()) {
	t.Helper()

	db, err := database.NewSQLiteConnection(":memory:")
	require.NoError(t, err, "failed to create test db connection")

	migrator := database.NewMigrator(db)
	err = migrator.Run(context.Background())
	require.NoError(t, err, "failed to run migrations")

	repo := NewSQLiteRepository(db)

	cleanup := func() {
		db.Close()
	}
	return repo, cleanup
}

func msgFactory(overrides ...func(*message.Message)) message.Message {
	msg := message.Message{
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

func TestSQLiteRepository_SaveRawData(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	msg := msgFactory()

	err := repo.SaveRawData(ctx, msg)
	require.NoError(t, err)

	messages, err := repo.GetByTopic(ctx, "machine/status")
	require.NoError(t, err)
	require.Len(t, messages, 1)

	got := messages[0]
	assert.Equal(t, "test-client", got.ClientID)
	assert.Equal(t, "machine/status", got.Topic)
	assert.Equal(t, "UTC", got.Timezone)
	assert.Equal(t, []byte(`{"temp":42}`), got.Payload)
	assert.Equal(t, msg.Timestamp.Format(time.RFC3339Nano), got.Timestamp.Format(time.RFC3339Nano))
}

func TestSQLiteRepository_MultipleInserts_PreserveOrder(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	topic := "machine/production"

	for i := 0; i < 10; i++ {
		msg := msgFactory(func(m *message.Message) {
			m.Topic = topic
			m.Payload = []byte(fmt.Sprintf(`{"seq":%d}`, i))
			m.Timestamp = time.Date(2026, 4, 21, 12, 0, i, 0, time.UTC)
		})
		err := repo.SaveRawData(ctx, msg)
		require.NoError(t, err, "insert %d failed", i)
	}

	messages, err := repo.GetByTopic(ctx, topic)
	require.NoError(t, err)
	require.Len(t, messages, 10)

	for i, msg := range messages {
		expected := fmt.Sprintf(`{"seq":%d}`, i)
		assert.Equal(t, expected, string(msg.Payload), "message %d out of order", i)
	}
}

func TestSQLiteRepository_ConcurrentWrites(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	var wg sync.WaitGroup
	errChan := make(chan error, 50)

	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				msg := msgFactory(func(m *message.Message) {
					m.Topic = fmt.Sprintf("topic/%d", goroutineID)
					m.Payload = []byte(fmt.Sprintf(`{"g":%d,"i":%d}`, goroutineID, i))
					m.Timestamp = time.Now()
				})
				if err := repo.SaveRawData(ctx, msg); err != nil {
					errChan <- err
				}
			}
		}(g)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent write error: %v", err)
	}

	for g := 0; g < 5; g++ {
		messages, err := repo.GetByTopic(ctx, fmt.Sprintf("topic/%d", g))
		require.NoError(t, err)
		assert.Len(t, messages, 10, "goroutine %d should have 10 messages", g)
	}
}

func TestSQLiteRepository_GetByTopic_Empty(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	messages, err := repo.GetByTopic(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestSQLiteRepository_GetByTopic_FiltersByTopic(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	repo.SaveRawData(ctx, msgFactory(func(m *message.Message) { m.Topic = "topic/a"; m.Payload = []byte("a1") }))
	repo.SaveRawData(ctx, msgFactory(func(m *message.Message) { m.Topic = "topic/b"; m.Payload = []byte("b1") }))
	repo.SaveRawData(ctx, msgFactory(func(m *message.Message) { m.Topic = "topic/a"; m.Payload = []byte("a2") }))

	messagesA, err := repo.GetByTopic(ctx, "topic/a")
	require.NoError(t, err)
	assert.Len(t, messagesA, 2)
	assert.Equal(t, []byte("a1"), messagesA[0].Payload)
	assert.Equal(t, []byte("a2"), messagesA[1].Payload)

	messagesB, err := repo.GetByTopic(ctx, "topic/b")
	require.NoError(t, err)
	assert.Len(t, messagesB, 1)
}

func TestSQLiteRepository_BinaryPayload(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	msg := msgFactory(func(m *message.Message) {
		m.Payload = []byte("this is not json, just raw bytes 0xFF")
	})

	err := repo.SaveRawData(ctx, msg)
	require.NoError(t, err)

	messages, err := repo.GetByTopic(ctx, msg.Topic)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, msg.Payload, messages[0].Payload)
}

func TestSQLiteRepository_ContextCancelled(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msg := msgFactory()
	err := repo.SaveRawData(ctx, msg)
	assert.Error(t, err, "should fail with cancelled context")
}

func TestSQLiteRepository_Close_IsNoop(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	// Close on repository is a no-op (platform owns the connection)
	err := repo.Close()
	assert.NoError(t, err)
}

func TestMigrator_Idempotent(t *testing.T) {
	db, err := database.NewSQLiteConnection(":memory:")
	require.NoError(t, err)
	defer db.Close()

	migrator := database.NewMigrator(db)

	// Run twice — should not fail
	err = migrator.Run(context.Background())
	require.NoError(t, err)

	err = migrator.Run(context.Background())
	require.NoError(t, err)
}
