package ingestion

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/ingestion/domain"
)

// --- Mock Store for Queue Tests ---

type mockStore struct {
	mu      sync.Mutex
	saved   []domain.Message
	saveErr error
	saveFn  func(domain.Message) error // optional custom behavior
}

func (m *mockStore) Migrate(_ context.Context) error { return nil }

func (m *mockStore) SaveRawData(_ context.Context, msg domain.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveFn != nil {
		return m.saveFn(msg)
	}
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, msg)
	return nil
}

func (m *mockStore) GetByTopic(_ context.Context, _ string) ([]domain.Message, error) {
	return nil, nil
}

func (m *mockStore) Close() error { return nil }

func (m *mockStore) getSaved() []domain.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]domain.Message, len(m.saved))
	copy(cp, m.saved)
	return cp
}

// --- Tests ---

func TestQueue_FIFOOrder(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan domain.Message, 10)
	q := NewQueue("test/topic", 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go q.StartConsumer(ctx, store, dispatchChan, NopLogger{})

	// Enqueue A, B, C
	msgs := []domain.Message{
		{ClientID: "c1", Topic: "test/topic", Payload: []byte("A"), Timestamp: time.Now()},
		{ClientID: "c1", Topic: "test/topic", Payload: []byte("B"), Timestamp: time.Now()},
		{ClientID: "c1", Topic: "test/topic", Payload: []byte("C"), Timestamp: time.Now()},
	}
	for _, m := range msgs {
		q.Enqueue(m)
	}

	// Read from dispatch in order
	for i, expected := range msgs {
		select {
		case got := <-dispatchChan:
			assert.Equal(t, string(expected.Payload), string(got.Payload), "message %d out of order", i)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for message %d", i)
		}
	}

	// Verify store received in order too
	saved := store.getSaved()
	require.Len(t, saved, 3)
	assert.Equal(t, []byte("A"), saved[0].Payload)
	assert.Equal(t, []byte("B"), saved[1].Payload)
	assert.Equal(t, []byte("C"), saved[2].Payload)
}

func TestQueue_ConsumerStopsOnContextCancel(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan domain.Message, 10)
	q := NewQueue("test/topic", 10)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		q.StartConsumer(ctx, store, dispatchChan, NopLogger{})
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// consumer stopped
	case <-time.After(2 * time.Second):
		t.Fatal("consumer did not stop after context cancel")
	}
}

func TestQueue_MessageForwardedAfterSave(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan domain.Message, 10)
	q := NewQueue("test/topic", 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go q.StartConsumer(ctx, store, dispatchChan, NopLogger{})

	msg := domain.Message{ClientID: "c1", Topic: "test/topic", Payload: []byte("data"), Timestamp: time.Now()}
	q.Enqueue(msg)

	select {
	case got := <-dispatchChan:
		assert.Equal(t, []byte("data"), got.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for dispatched message")
	}

	// Verify it was saved first
	saved := store.getSaved()
	require.Len(t, saved, 1)
	assert.Equal(t, []byte("data"), saved[0].Payload)
}

func TestQueue_StoreError_SkipsForwarding_ContinuesProcessing(t *testing.T) {
	callCount := 0
	store := &mockStore{
		saveFn: func(msg domain.Message) error {
			callCount++
			if string(msg.Payload) == "fail" {
				return errors.New("disk full")
			}
			return nil
		},
	}
	dispatchChan := make(chan domain.Message, 10)
	q := NewQueue("test/topic", 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go q.StartConsumer(ctx, store, dispatchChan, NopLogger{})

	// Enqueue: good, fail, good
	q.Enqueue(domain.Message{Payload: []byte("good1"), Timestamp: time.Now()})
	q.Enqueue(domain.Message{Payload: []byte("fail"), Timestamp: time.Now()})
	q.Enqueue(domain.Message{Payload: []byte("good2"), Timestamp: time.Now()})

	// Should receive good1 and good2, NOT fail
	var received []string
	for i := 0; i < 2; i++ {
		select {
		case got := <-dispatchChan:
			received = append(received, string(got.Payload))
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for message %d", i)
		}
	}

	assert.Equal(t, []string{"good1", "good2"}, received)

	// "fail" should NOT be in dispatch channel
	select {
	case msg := <-dispatchChan:
		t.Fatalf("unexpected message in dispatch: %s", string(msg.Payload))
	case <-time.After(200 * time.Millisecond):
		// expected: no more messages
	}
}

func TestQueue_Backpressure_EnqueueBlocks(t *testing.T) {
	// Buffer size 1 — second enqueue should block
	q := NewQueue("test/topic", 1)

	// Fill the buffer
	q.Enqueue(domain.Message{Payload: []byte("first"), Timestamp: time.Now()})

	// Second enqueue should block
	blocked := make(chan struct{})
	go func() {
		q.Enqueue(domain.Message{Payload: []byte("second"), Timestamp: time.Now()})
		close(blocked)
	}()

	select {
	case <-blocked:
		t.Fatal("enqueue should have blocked but didn't")
	case <-time.After(200 * time.Millisecond):
		// expected: blocked
	}

	// Drain one message to unblock
	<-q.messages

	select {
	case <-blocked:
		// unblocked after drain
	case <-time.After(2 * time.Second):
		t.Fatal("enqueue should have unblocked after drain")
	}
}

func TestQueue_Topic(t *testing.T) {
	q := NewQueue("machine/status", 10)
	assert.Equal(t, "machine/status", q.Topic())
}
