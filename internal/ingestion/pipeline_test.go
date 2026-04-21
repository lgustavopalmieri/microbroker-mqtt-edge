package ingestion

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/ingestion/domain"
)

func TestPipeline_RoutesToCorrectQueue(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan domain.Message, 20)
	topics := []string{"topic/a", "topic/b"}

	pipeline := NewPipeline(topics, store, dispatchChan, 10, NopLogger{})
	assert.Equal(t, 2, pipeline.QueueCount())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan domain.Message, 10)
	go pipeline.Start(ctx, inputChan)

	// Send messages to different topics
	inputChan <- domain.Message{ClientID: "c1", Topic: "topic/a", Payload: []byte("a1"), Timestamp: time.Now()}
	inputChan <- domain.Message{ClientID: "c1", Topic: "topic/b", Payload: []byte("b1"), Timestamp: time.Now()}
	inputChan <- domain.Message{ClientID: "c1", Topic: "topic/a", Payload: []byte("a2"), Timestamp: time.Now()}

	// Collect dispatched messages
	var dispatched []domain.Message
	for i := 0; i < 3; i++ {
		select {
		case msg := <-dispatchChan:
			dispatched = append(dispatched, msg)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for message %d", i)
		}
	}

	require.Len(t, dispatched, 3)

	// All 3 should have been saved
	saved := store.getSaved()
	assert.Len(t, saved, 3)
}

func TestPipeline_UnknownTopicDiscarded(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan domain.Message, 10)
	topics := []string{"topic/a"}

	pipeline := NewPipeline(topics, store, dispatchChan, 10, NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan domain.Message, 10)
	go pipeline.Start(ctx, inputChan)

	// Send to unknown topic
	inputChan <- domain.Message{ClientID: "c1", Topic: "unknown/topic", Payload: []byte("data"), Timestamp: time.Now()}

	// Should NOT appear in dispatch
	select {
	case <-dispatchChan:
		t.Fatal("message for unknown topic should not be dispatched")
	case <-time.After(300 * time.Millisecond):
		// expected
	}

	// Should NOT be saved
	assert.Empty(t, store.getSaved())
}

func TestPipeline_MultipleTopicsSimultaneously(t *testing.T) {
	store := &mockStore{}
	dispatchChan := make(chan domain.Message, 50)
	topics := []string{"t/1", "t/2", "t/3"}

	pipeline := NewPipeline(topics, store, dispatchChan, 20, NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan domain.Message, 50)
	go pipeline.Start(ctx, inputChan)

	// Send 5 messages per topic
	total := 0
	for _, topic := range topics {
		for i := 0; i < 5; i++ {
			inputChan <- domain.Message{
				ClientID:  "c1",
				Topic:     topic,
				Payload:   []byte("data"),
				Timestamp: time.Now(),
			}
			total++
		}
	}

	// Collect all dispatched
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
	dispatchChan := make(chan domain.Message, 10)
	topics := []string{"topic/a"}

	pipeline := NewPipeline(topics, store, dispatchChan, 10, NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	inputChan := make(chan domain.Message, 10)

	done := make(chan struct{})
	go func() {
		pipeline.Start(ctx, inputChan)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// pipeline stopped
	case <-time.After(2 * time.Second):
		t.Fatal("pipeline did not stop after context cancel")
	}
}

func TestPipeline_MessagePersistedBeforeDispatch(t *testing.T) {
	// This test verifies the critical invariant: save THEN forward
	store := &mockStore{}
	dispatchChan := make(chan domain.Message, 10)
	topics := []string{"topic/a"}

	pipeline := NewPipeline(topics, store, dispatchChan, 10, NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan domain.Message, 10)
	go pipeline.Start(ctx, inputChan)

	msg := domain.Message{ClientID: "c1", Topic: "topic/a", Payload: []byte("critical"), Timestamp: time.Now()}
	inputChan <- msg

	// Wait for dispatch
	select {
	case <-dispatchChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	// At this point, the message MUST have been saved
	saved := store.getSaved()
	require.Len(t, saved, 1)
	assert.Equal(t, []byte("critical"), saved[0].Payload)
}
