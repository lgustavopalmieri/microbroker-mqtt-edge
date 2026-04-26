package fanout_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/modules/processing/fanout"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ----------------------------------------------------------------

type spyLogger struct {
	mu   sync.Mutex
	logs []string
}

func (l *spyLogger) Info(msg string, _ ...any)  { l.append(msg) }
func (l *spyLogger) Error(msg string, _ ...any) { l.append(msg) }
func (l *spyLogger) Warn(msg string, _ ...any)  { l.append(msg) }
func (l *spyLogger) Debug(msg string, _ ...any) { l.append(msg) }

func (l *spyLogger) append(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, msg)
}

func (l *spyLogger) messages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	cp := make([]string, len(l.logs))
	copy(cp, l.logs)
	return cp
}

type mockWorker struct {
	name      string
	processFn func(ctx context.Context, msg message.Message) error
	closeFn   func() error
	mu        sync.Mutex
	received  []message.Message
}

func newMockWorker(name string) *mockWorker {
	return &mockWorker{name: name}
}

func (w *mockWorker) Name() string { return w.name }

func (w *mockWorker) Process(ctx context.Context, msg message.Message) error {
	w.mu.Lock()
	w.received = append(w.received, msg)
	w.mu.Unlock()
	if w.processFn != nil {
		return w.processFn(ctx, msg)
	}
	return nil
}

func (w *mockWorker) Close() error {
	if w.closeFn != nil {
		return w.closeFn()
	}
	return nil
}

func (w *mockWorker) messages() []message.Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	cp := make([]message.Message, len(w.received))
	copy(cp, w.received)
	return cp
}

func sampleMsg(topic string) message.Message {
	return message.Message{
		ClientID:  "client-1",
		Topic:     topic,
		Payload:   []byte(`{"temp":42}`),
		Timezone:  "UTC",
		Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// --- tests ------------------------------------------------------------------

func TestFanOut_AllWorkersReceiveMessage(t *testing.T) {
	ch := make(chan message.Message, 1)
	w1 := newMockWorker("w1")
	w2 := newMockWorker("w2")
	w3 := newMockWorker("w3")

	f := fanout.NewFanOut(ch, []fanout.Worker{w1, w2, w3}, &spyLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { f.Start(ctx); close(done) }()

	msg := sampleMsg("machine/status")
	ch <- msg
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	assert.Len(t, w1.messages(), 1)
	assert.Len(t, w2.messages(), 1)
	assert.Len(t, w3.messages(), 1)
	assert.Equal(t, msg, w1.messages()[0])
}

func TestFanOut_WorkerError_OthersContinue(t *testing.T) {
	ch := make(chan message.Message, 1)
	failing := newMockWorker("failing")
	failing.processFn = func(context.Context, message.Message) error {
		return errors.New("boom")
	}
	healthy := newMockWorker("healthy")
	logger := &spyLogger{}

	f := fanout.NewFanOut(ch, []fanout.Worker{failing, healthy}, logger)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { f.Start(ctx); close(done) }()

	ch <- sampleMsg("t")
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	assert.Len(t, healthy.messages(), 1, "healthy worker should still receive the message")
	assert.Contains(t, logger.messages(), "worker process failed")
}

func TestFanOut_ContextCancelled_Stops(t *testing.T) {
	ch := make(chan message.Message, 10)
	f := fanout.NewFanOut(ch, nil, &spyLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { f.Start(ctx); close(done) }()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("fan-out did not stop after context cancellation")
	}
}

func TestFanOut_NoWorkers_ConsumesWithoutError(t *testing.T) {
	ch := make(chan message.Message, 3)
	logger := &spyLogger{}
	f := fanout.NewFanOut(ch, nil, logger)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { f.Start(ctx); close(done) }()

	ch <- sampleMsg("a")
	ch <- sampleMsg("b")
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	for _, m := range logger.messages() {
		assert.NotContains(t, m, "error")
	}
}

func TestFanOut_ChannelClosed_Stops(t *testing.T) {
	ch := make(chan message.Message, 1)
	f := fanout.NewFanOut(ch, nil, &spyLogger{})

	done := make(chan struct{})
	go func() { f.Start(context.Background()); close(done) }()

	close(ch)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("fan-out did not stop after channel close")
	}
}

func TestFanOut_MessagesProcessedInOrder(t *testing.T) {
	ch := make(chan message.Message, 10)
	var order []string
	var mu sync.Mutex

	w := newMockWorker("ordered")
	w.processFn = func(_ context.Context, msg message.Message) error {
		mu.Lock()
		order = append(order, msg.Topic)
		mu.Unlock()
		return nil
	}

	f := fanout.NewFanOut(ch, []fanout.Worker{w}, &spyLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { f.Start(ctx); close(done) }()

	topics := []string{"a", "b", "c", "d", "e"}
	for _, topic := range topics {
		ch <- sampleMsg(topic)
	}
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, topics, order)
}

func TestFanOut_WorkerPanic_Recovered(t *testing.T) {
	ch := make(chan message.Message, 1)
	panicker := newMockWorker("panicker")
	panicker.processFn = func(context.Context, message.Message) error {
		panic("unexpected")
	}
	healthy := newMockWorker("healthy")
	logger := &spyLogger{}

	f := fanout.NewFanOut(ch, []fanout.Worker{panicker, healthy}, logger)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { f.Start(ctx); close(done) }()

	ch <- sampleMsg("t")
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	assert.Len(t, healthy.messages(), 1, "healthy worker should still receive the message")
	assert.Contains(t, logger.messages(), "worker panicked")
}

func TestFanOut_Close_CallsAllWorkers(t *testing.T) {
	ch := make(chan message.Message)
	var closed atomic.Int32

	makeWorker := func(name string) *mockWorker {
		w := newMockWorker(name)
		w.closeFn = func() error { closed.Add(1); return nil }
		return w
	}

	f := fanout.NewFanOut(ch, []fanout.Worker{
		makeWorker("a"), makeWorker("b"), makeWorker("c"),
	}, &spyLogger{})

	require.NoError(t, f.Close())
	assert.Equal(t, int32(3), closed.Load())
}
