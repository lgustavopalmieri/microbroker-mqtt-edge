package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
)

// --- Mock Store for Pipeline Tests ---

type mockStore struct {
	mu    sync.Mutex
	saved []message.Message
}

func (m *mockStore) SaveRawData(_ context.Context, msg message.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saved = append(m.saved, msg)
	return nil
}

func (m *mockStore) Close() error { return nil }

func (m *mockStore) getSaved() []message.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]message.Message, len(m.saved))
	copy(cp, m.saved)
	return cp
}

// --- Tests ---

func TestPipeline_RoutesToCorrectQueue(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan message.Message, 20)
	topics := []string{"topic/a", "topic/b"}

	p := NewPipeline(topics, store, dispatchChan, 10, observability.NopLogger{})
	assert.Equal(t, 2, p.QueueCount())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan message.Message, 10)
	go p.Start(ctx, inputChan)

	inputChan <- message.Message{ClientID: "c1", Topic: "topic/a", Payload: []byte("a1"), Timestamp: time.Now()}
	inputChan <- message.Message{ClientID: "c1", Topic: "topic/b", Payload: []byte("b1"), Timestamp: time.Now()}
	inputChan <- message.Message{ClientID: "c1", Topic: "topic/a", Payload: []byte("a2"), Timestamp: time.Now()}

	var dispatched []message.Message
	for i := 0; i < 3; i++ {
		select {
		case msg := <-dispatchChan:
			dispatched = append(dispatched, msg)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for message %d", i)
		}
	}

	require.Len(t, dispatched, 3)
	saved := store.getSaved()
	assert.Len(t, saved, 3)
}

func TestPipeline_UnknownTopicDiscarded(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan message.Message, 10)
	topics := []string{"topic/a"}

	p := NewPipeline(topics, store, dispatchChan, 10, observability.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan message.Message, 10)
	go p.Start(ctx, inputChan)

	inputChan <- message.Message{ClientID: "c1", Topic: "unknown/topic", Payload: []byte("data"), Timestamp: time.Now()}

	select {
	case <-dispatchChan:
		t.Fatal("message for unknown topic should not be dispatched")
	case <-time.After(300 * time.Millisecond):
	}

	assert.Empty(t, store.getSaved())
}

func TestPipeline_MultipleTopicsSimultaneously(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan message.Message, 50)
	topics := []string{"t/1", "t/2", "t/3"}

	p := NewPipeline(topics, store, dispatchChan, 20, observability.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan message.Message, 50)
	go p.Start(ctx, inputChan)

	total := 0
	for _, topic := range topics {
		for i := 0; i < 5; i++ {
			inputChan <- message.Message{
				ClientID:  "c1",
				Topic:     topic,
				Payload:   []byte("data"),
				Timestamp: time.Now(),
			}
			total++
		}
	}

	for i := 0; i < total; i++ {
		select {
		case <-dispatchChan:
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for message %d of %d", i, total)
		}
	}

	assert.Len(t, store.getSaved(), total)
}

func TestPipeline_GracefulShutdown(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan message.Message, 10)
	topics := []string{"topic/a"}

	p := NewPipeline(topics, store, dispatchChan, 10, observability.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	inputChan := make(chan message.Message, 10)

	done := make(chan struct{})
	go func() {
		p.Start(ctx, inputChan)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pipeline did not stop after context cancel")
	}
}

func TestPipeline_MessagePersistedBeforeDispatch(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan message.Message, 10)
	topics := []string{"topic/a"}

	p := NewPipeline(topics, store, dispatchChan, 10, observability.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan message.Message, 10)
	go p.Start(ctx, inputChan)

	msg := message.Message{ClientID: "c1", Topic: "topic/a", Payload: []byte("critical"), Timestamp: time.Now()}
	inputChan <- msg

	select {
	case <-dispatchChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	saved := store.getSaved()
	require.Len(t, saved, 1)
	assert.Equal(t, []byte("critical"), saved[0].Payload)
}
