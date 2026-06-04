package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/adapters/inbound/worker"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/adapters/inbound/worker/mocks"
)

const stateTopic = "machine/state"

func newWorker(t *testing.T) (*worker.StateChangeWorker, *mocks.MockStateIngester) {
	t.Helper()
	ctrl := gomock.NewController(t)
	ingester := mocks.NewMockStateIngester(ctrl)
	w := worker.NewStateChangeWorker(stateTopic, ingester, observability.NewNopLogger())
	return w, ingester
}

func validStatePayload() []byte {
	return []byte(`{
		"type": "state_change",
		"machine_id": "m1",
		"state": "stopped",
		"timestamp": "2026-06-01T08:00:00Z"
	}`)
}

func msgFactory(overrides ...func(*message.Message)) message.Message {
	m := message.Message{
		Topic:     stateTopic,
		Payload:   validStatePayload(),
		Timestamp: time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC),
	}
	for _, fn := range overrides {
		fn(&m)
	}
	return m
}

func TestStateChangeWorker_Name(t *testing.T) {
	w, _ := newWorker(t)
	assert.Equal(t, "oee-state-change", w.Name())
}

func TestStateChangeWorker_Close_ReturnsNil(t *testing.T) {
	w, _ := newWorker(t)
	assert.NoError(t, w.Close())
}

func TestStateChangeWorker_Process_IgnoresNonStateTopic(t *testing.T) {
	w, ingester := newWorker(t)
	ingester.EXPECT().Apply(gomock.Any(), gomock.Any()).Times(0)

	msg := msgFactory(func(m *message.Message) { m.Topic = "machine/production" })
	err := w.Process(context.Background(), msg)
	assert.NoError(t, err)
}

func TestStateChangeWorker_Process_DecodesAndDelegatesValidPayload(t *testing.T) {
	w, ingester := newWorker(t)
	ingester.EXPECT().Apply(gomock.Any(), gomock.Any()).Times(1).Return(nil)

	err := w.Process(context.Background(), msgFactory())
	require.NoError(t, err)
}

func TestStateChangeWorker_Process_SkipsMalformedPayload(t *testing.T) {
	w, ingester := newWorker(t)
	ingester.EXPECT().Apply(gomock.Any(), gomock.Any()).Times(0)

	msg := msgFactory(func(m *message.Message) { m.Payload = []byte(`not json`) })
	err := w.Process(context.Background(), msg)
	assert.NoError(t, err)
}

func TestStateChangeWorker_Process_PropagatesApplyError(t *testing.T) {
	w, ingester := newWorker(t)
	applyErr := errors.New("store failure")
	ingester.EXPECT().Apply(gomock.Any(), gomock.Any()).Times(1).Return(applyErr)

	err := w.Process(context.Background(), msgFactory())
	assert.ErrorIs(t, err, applyErr)
}
