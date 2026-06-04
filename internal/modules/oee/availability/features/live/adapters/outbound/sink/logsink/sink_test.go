package logsink_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/logsink"
)

func snapshotFactory(overrides ...func(*avdomain.AvailabilitySnapshot)) avdomain.AvailabilitySnapshot {
	s := avdomain.AvailabilitySnapshot{MachineID: "m1", Availability: 0.9, HasData: true}
	for _, fn := range overrides {
		fn(&s)
	}
	return s
}

func TestLogSink_Update_StoresLatestSnapshotPerMachine(t *testing.T) {
	s := logsink.NewLogSink(observability.NewNopLogger())

	snap := snapshotFactory(func(s *avdomain.AvailabilitySnapshot) { s.Availability = 0.85 })
	require.NoError(t, s.Update(context.Background(), snap))

	got, found := s.Latest("m1")
	assert.True(t, found)
	assert.Equal(t, "m1", got.MachineID)
	assert.InDelta(t, 0.85, got.Availability, 0.001)
}

func TestLogSink_Latest_NotFoundForUnknownMachine(t *testing.T) {
	s := logsink.NewLogSink(observability.NewNopLogger())

	got, found := s.Latest("unknown")
	assert.False(t, found)
	assert.Empty(t, got.MachineID)
}

func TestLogSink_ConcurrentUpdates_AreRaceClean(t *testing.T) {
	s := logsink.NewLogSink(observability.NewNopLogger())

	const n = 10
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			snap := snapshotFactory(func(s *avdomain.AvailabilitySnapshot) {
				s.MachineID = fmt.Sprintf("m%d", id)
				s.Availability = float64(id) / float64(n)
			})
			require.NoError(t, s.Update(context.Background(), snap))
		}(i)
	}
	wg.Wait()

	for i := range n {
		machineID := fmt.Sprintf("m%d", i)
		got, found := s.Latest(machineID)
		assert.True(t, found, "machine %s should have a snapshot", machineID)
		assert.Equal(t, machineID, got.MachineID)
	}
}
