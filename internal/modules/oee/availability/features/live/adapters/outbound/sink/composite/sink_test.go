package composite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/composite"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/live/application/mocks"
)

// blockingSink blocks its Update call until the context is cancelled.
// Used to verify CompositeSink does not block the caller on a slow sink.
type blockingSink struct{}

func (b *blockingSink) Update(ctx context.Context, _ avdomain.AvailabilitySnapshot) error {
	<-ctx.Done()
	return ctx.Err()
}

func snapshotFactory(overrides ...func(*avdomain.AvailabilitySnapshot)) avdomain.AvailabilitySnapshot {
	s := avdomain.AvailabilitySnapshot{MachineID: "m1", Availability: 0.9, HasData: true}
	for _, fn := range overrides {
		fn(&s)
	}
	return s
}

func TestCompositeSink_Update_DeliversToAllSinks(t *testing.T) {
	ctrl := gomock.NewController(t)
	sink1 := mocks.NewMockAvailabilitySink(ctrl)
	sink2 := mocks.NewMockAvailabilitySink(ctrl)

	received := make(chan struct{}, 2)
	sink1.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, _ avdomain.AvailabilitySnapshot) error {
			received <- struct{}{}
			return nil
		})
	sink2.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, _ avdomain.AvailabilitySnapshot) error {
			received <- struct{}{}
			return nil
		})

	c := composite.NewCompositeSink(observability.NewNopLogger(), sink1, sink2)
	defer c.Close()

	assert.NoError(t, c.Update(context.Background(), snapshotFactory()))

	for range 2 {
		select {
		case <-received:
		case <-time.After(time.Second):
			t.Fatal("sink did not receive update within timeout")
		}
	}
}

func TestCompositeSink_Update_DoesNotBlockOnSlowSink(t *testing.T) {
	c := composite.NewCompositeSink(observability.NewNopLogger(), &blockingSink{})
	defer c.Close()

	start := time.Now()
	assert.NoError(t, c.Update(context.Background(), snapshotFactory()))
	assert.Less(t, time.Since(start), 50*time.Millisecond, "Update must return immediately regardless of sink speed")
}

func TestCompositeSink_Update_ContinuesDeliveryWhenOneSinkErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	failingSink := mocks.NewMockAvailabilitySink(ctrl)
	successSink := mocks.NewMockAvailabilitySink(ctrl)

	received := make(chan struct{}, 2)
	failingSink.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, _ avdomain.AvailabilitySnapshot) error {
			received <- struct{}{}
			return errors.New("sink failure")
		})
	successSink.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, _ avdomain.AvailabilitySnapshot) error {
			received <- struct{}{}
			return nil
		})

	c := composite.NewCompositeSink(observability.NewNopLogger(), failingSink, successSink)
	defer c.Close()

	assert.NoError(t, c.Update(context.Background(), snapshotFactory()))

	for range 2 {
		select {
		case <-received:
		case <-time.After(time.Second):
			t.Fatal("sink did not receive update within timeout")
		}
	}
}

func TestCompositeSink_Close_StopsDrainGoroutines(t *testing.T) {
	// blockingSink goroutine is cleanly stopped when Close cancels the context.
	c := composite.NewCompositeSink(observability.NewNopLogger(), &blockingSink{})

	c.Update(context.Background(), snapshotFactory()) //nolint:errcheck

	assert.NoError(t, c.Close())

	// After Close, Update still returns promptly — drops non-blocking.
	start := time.Now()
	c.Update(context.Background(), snapshotFactory()) //nolint:errcheck
	assert.Less(t, time.Since(start), 50*time.Millisecond)
}
