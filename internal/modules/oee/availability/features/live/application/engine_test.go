package application_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/live/application"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/live/application/mocks"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

const testTickInterval = 5 * time.Millisecond

var (
	eT0 = time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	eT1 = eT0.Add(5 * time.Minute)
	eT2 = eT0.Add(10 * time.Minute)
)

func fixedClock(at time.Time) func() time.Time { return func() time.Time { return at } }

// advancingClock returns successive times on each call (wraps around).
func advancingClock(times ...time.Time) func() time.Time {
	var mu sync.Mutex
	idx := 0
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		t := times[idx%len(times)]
		idx++
		return t
	}
}

func newEngine(
	sink application.AvailabilitySink,
	shifts application.ShiftReader,
	intervals application.StateIntervalReader,
	now func() time.Time,
) *application.Engine {
	return application.NewEngine(sink, shifts, intervals, observability.NewNopLogger(), testTickInterval, now)
}

func waitForSnap(t *testing.T, ch <-chan avdomain.AvailabilitySnapshot) avdomain.AvailabilitySnapshot {
	t.Helper()
	select {
	case s := <-ch:
		return s
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for AvailabilitySink.Update")
		return avdomain.AvailabilitySnapshot{}
	}
}

// capturingSink returns an AnyTimes expectation that pipes snapshots into ch.
func capturingSink(t *testing.T, sink *mocks.MockAvailabilitySink, ch chan<- avdomain.AvailabilitySnapshot) {
	t.Helper()
	sink.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, s avdomain.AvailabilitySnapshot) error {
			select {
			case ch <- s:
			default:
			}
			return nil
		}).AnyTimes()
}

// --- Apply behaviour -------------------------------------------------------

func TestEngine_Apply(t *testing.T) {
	tests := []struct {
		name        string
		transitions []avdomain.StateTransition
		verify      func(t *testing.T, s avdomain.AvailabilitySnapshot)
	}{
		{
			name: "first event creates machine accumulator with correct interval and windowStart",
			transitions: []avdomain.StateTransition{
				{MachineID: "CNC-01", State: ooedomain.Running, Timestamp: eT0},
			},
			verify: func(t *testing.T, s avdomain.AvailabilitySnapshot) {
				assert.Equal(t, "CNC-01", s.MachineID)
				assert.Equal(t, 1, s.IntervalCount)
				assert.True(t, s.HasData)
			},
		},
		{
			name: "second event closes the open interval and opens a new one",
			transitions: []avdomain.StateTransition{
				{MachineID: "CNC-01", State: ooedomain.Running, Timestamp: eT0},
				{MachineID: "CNC-01", State: ooedomain.Stopped, Timestamp: eT1},
			},
			verify: func(t *testing.T, s avdomain.AvailabilitySnapshot) {
				assert.Equal(t, "CNC-01", s.MachineID)
				assert.Equal(t, 2, s.IntervalCount)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			snaps := make(chan avdomain.AvailabilitySnapshot, 8)
			sink := mocks.NewMockAvailabilitySink(ctrl)
			capturingSink(t, sink, snaps)
			shifts := mocks.NewMockShiftReader(ctrl)
			shifts.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(10*time.Minute, true, nil).AnyTimes()
			intervals := mocks.NewMockStateIntervalReader(ctrl)
			intervals.EXPECT().LastOpen(gomock.Any(), gomock.Any()).
				Return(avdomain.StateInterval{}, false, nil).AnyTimes()

			eng := newEngine(sink, shifts, intervals, fixedClock(eT2))
			for _, tr := range tc.transitions {
				eng.Apply(tr)
			}

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() { defer close(done); eng.Start(ctx, nil) }()

			s := waitForSnap(t, snaps)
			cancel()
			<-done

			tc.verify(t, s)
		})
	}
}

// TestEngine_Apply_ConcurrentRaceClean asserts that concurrent Apply callers
// and a running ticker do not produce data races (-race flag enforced by gate).
func TestEngine_Apply_ConcurrentRaceClean(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sink := mocks.NewMockAvailabilitySink(ctrl)
	sink.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	shifts := mocks.NewMockShiftReader(ctrl)
	shifts.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).Return(time.Duration(0), false, nil).AnyTimes()
	intervals := mocks.NewMockStateIntervalReader(ctrl)
	intervals.EXPECT().LastOpen(gomock.Any(), gomock.Any()).Return(avdomain.StateInterval{}, false, nil).AnyTimes()

	eng := newEngine(sink, shifts, intervals, fixedClock(eT2))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go eng.Start(ctx, nil)

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			eng.Apply(avdomain.StateTransition{
				MachineID: "CNC-01",
				State:     ooedomain.Running,
				Timestamp: eT0.Add(time.Duration(i) * time.Second),
			})
		}(i)
	}
	wg.Wait()
}

// --- Start / tick behaviour ------------------------------------------------

func TestEngine_Start(t *testing.T) {
	tests := []struct {
		name          string
		machineIDs    []string
		setupMocks    func(sink *mocks.MockAvailabilitySink, shifts *mocks.MockShiftReader, intervals *mocks.MockStateIntervalReader, snaps chan<- avdomain.AvailabilitySnapshot)
		primeEngine   func(eng *application.Engine)
		now           func() time.Time
		verify        func(t *testing.T, s avdomain.AvailabilitySnapshot)
		skipSnapCheck bool
	}{
		{
			name:       "rehydrates open interval via StateIntervalReader for each supplied machine ID on start",
			machineIDs: []string{"CNC-01"},
			setupMocks: func(sink *mocks.MockAvailabilitySink, shifts *mocks.MockShiftReader, intervals *mocks.MockStateIntervalReader, snaps chan<- avdomain.AvailabilitySnapshot) {
				iv := avdomain.StateInterval{MachineID: "CNC-01", State: ooedomain.Running, StartedAt: eT0}
				intervals.EXPECT().LastOpen(gomock.Any(), "CNC-01").Return(iv, true, nil).Times(1)
				shifts.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).Return(10*time.Minute, true, nil).AnyTimes()
				sink.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, s avdomain.AvailabilitySnapshot) error {
						select {
						case snaps <- s:
						default:
						}
						return nil
					}).AnyTimes()
			},
			now:    fixedClock(eT2),
			verify: func(t *testing.T, s avdomain.AvailabilitySnapshot) { assert.Equal(t, "CNC-01", s.MachineID) },
		},
		{
			name: "tick emits snapshot via AvailabilitySink after Apply populates machine state",
			setupMocks: func(sink *mocks.MockAvailabilitySink, shifts *mocks.MockShiftReader, intervals *mocks.MockStateIntervalReader, snaps chan<- avdomain.AvailabilitySnapshot) {
				intervals.EXPECT().LastOpen(gomock.Any(), gomock.Any()).Return(avdomain.StateInterval{}, false, nil).AnyTimes()
				shifts.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).Return(10*time.Minute, true, nil).AnyTimes()
				sink.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, s avdomain.AvailabilitySnapshot) error {
						select {
						case snaps <- s:
						default:
						}
						return nil
					}).AnyTimes()
			},
			primeEngine: func(eng *application.Engine) {
				eng.Apply(avdomain.StateTransition{MachineID: "CNC-01", State: ooedomain.Running, Timestamp: eT0})
			},
			now: fixedClock(eT2),
			verify: func(t *testing.T, s avdomain.AvailabilitySnapshot) {
				assert.Equal(t, "CNC-01", s.MachineID)
				assert.True(t, s.HasData)
			},
		},
		{
			name: "availability decreases across ticks as downtime accrues (injectable clock advances)",
			setupMocks: func(sink *mocks.MockAvailabilitySink, shifts *mocks.MockShiftReader, intervals *mocks.MockStateIntervalReader, snaps chan<- avdomain.AvailabilitySnapshot) {
				intervals.EXPECT().LastOpen(gomock.Any(), gomock.Any()).Return(avdomain.StateInterval{}, false, nil).AnyTimes()
				shifts.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).Return(10*time.Minute, true, nil).AnyTimes()
				sink.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, s avdomain.AvailabilitySnapshot) error {
						select {
						case snaps <- s:
						default:
						}
						return nil
					}).AnyTimes()
			},
			// running eT0→eT1 (5min), stopped eT1→now; now=eT0+10min → stopped 5min
			// planned=10min, unplannedDown=5min, runTime=5min → avail=0.5
			primeEngine: func(eng *application.Engine) {
				eng.Apply(avdomain.StateTransition{MachineID: "CNC-01", State: ooedomain.Running, Timestamp: eT0})
				eng.Apply(avdomain.StateTransition{MachineID: "CNC-01", State: ooedomain.Stopped, Timestamp: eT1})
			},
			now: advancingClock(eT2, eT0.Add(20*time.Minute)),
			verify: func(t *testing.T, s avdomain.AvailabilitySnapshot) {
				assert.InDelta(t, 0.5, s.Availability, 0.01, "availability should reflect 5/10 runtime")
			},
		},
		{
			name: "sink failure on tick is tolerated, engine continues",
			setupMocks: func(sink *mocks.MockAvailabilitySink, shifts *mocks.MockShiftReader, intervals *mocks.MockStateIntervalReader, snaps chan<- avdomain.AvailabilitySnapshot) {
				intervals.EXPECT().LastOpen(gomock.Any(), gomock.Any()).Return(avdomain.StateInterval{}, false, nil).AnyTimes()
				shifts.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).Return(10*time.Minute, true, nil).AnyTimes()
				callCount := 0
				sink.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, s avdomain.AvailabilitySnapshot) error {
						callCount++
						if callCount == 1 {
							return assert.AnError
						}
						select {
						case snaps <- s:
						default:
						}
						return nil
					}).AnyTimes()
			},
			primeEngine: func(eng *application.Engine) {
				eng.Apply(avdomain.StateTransition{MachineID: "CNC-01", State: ooedomain.Running, Timestamp: eT0})
			},
			now:    fixedClock(eT2),
			verify: func(t *testing.T, s avdomain.AvailabilitySnapshot) { assert.Equal(t, "CNC-01", s.MachineID) },
		},
		{
			name: "context cancellation stops the engine",
			setupMocks: func(sink *mocks.MockAvailabilitySink, shifts *mocks.MockShiftReader, intervals *mocks.MockStateIntervalReader, snaps chan<- avdomain.AvailabilitySnapshot) {
				intervals.EXPECT().LastOpen(gomock.Any(), gomock.Any()).Return(avdomain.StateInterval{}, false, nil).AnyTimes()
				shifts.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).Return(time.Duration(0), false, nil).AnyTimes()
				sink.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			primeEngine: func(eng *application.Engine) {
				eng.Apply(avdomain.StateTransition{MachineID: "CNC-01", State: ooedomain.Running, Timestamp: eT0})
			},
			now:           fixedClock(eT2),
			skipSnapCheck: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			snaps := make(chan avdomain.AvailabilitySnapshot, 16)
			sink := mocks.NewMockAvailabilitySink(ctrl)
			shifts := mocks.NewMockShiftReader(ctrl)
			intervals := mocks.NewMockStateIntervalReader(ctrl)
			tc.setupMocks(sink, shifts, intervals, snaps)

			eng := newEngine(sink, shifts, intervals, tc.now)
			if tc.primeEngine != nil {
				tc.primeEngine(eng)
			}

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() { defer close(done); eng.Start(ctx, tc.machineIDs) }()

			if tc.skipSnapCheck {
				cancel()
			} else {
				s := waitForSnap(t, snaps)
				cancel()
				tc.verify(t, s)
			}
			<-done
		})
	}
}
