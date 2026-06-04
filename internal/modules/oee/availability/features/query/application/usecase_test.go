package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/query/application"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/query/application/mocks"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

var (
	qT0  = time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	qWin = ooedomain.Window{From: qT0, To: qT0.Add(420 * time.Minute)}
	qErr = errors.New("store: read failed")
)

func tsPtr(t time.Time) *time.Time { return &t }

func TestQueryUseCase_Execute(t *testing.T) {
	tests := []struct {
		name        string
		machineID   string
		window      ooedomain.Window
		setupMocks  func(ir *mocks.MockIntervalReader, sr *mocks.MockShiftReader)
		expectError bool
		validate    func(t *testing.T, out *application.Output)
	}{
		{
			name:      "happy path - §7 oracle end-to-end: planned 420 min, stop 47 min → availability ≈ 0.8881",
			machineID: "CNC-01",
			window:    qWin,
			setupMocks: func(ir *mocks.MockIntervalReader, sr *mocks.MockShiftReader) {
				intervals := []avdomain.StateInterval{
					{MachineID: "CNC-01", State: ooedomain.Stopped, StartedAt: qT0, EndedAt: tsPtr(qT0.Add(47 * time.Minute))},
					{MachineID: "CNC-01", State: ooedomain.Running, StartedAt: qT0.Add(47 * time.Minute), EndedAt: tsPtr(qT0.Add(420 * time.Minute))},
				}
				ir.EXPECT().ByMachineRange(gomock.Any(), "CNC-01", qWin.From, qWin.To).Return(intervals, nil).Times(1)
				sr.EXPECT().ForMachineWindow(gomock.Any(), "CNC-01", qWin).Return(420*time.Minute, true, nil).Times(1)
			},
			validate: func(t *testing.T, out *application.Output) {
				require.NotNil(t, out)
				assert.True(t, out.HasData)
				assert.InDelta(t, 373.0/420.0, out.Availability, 0.0001, "availability should match §7 oracle")
				assert.Equal(t, 373*time.Minute, out.RunTime)
				assert.Equal(t, 47*time.Minute, out.UnplannedDowntime)
				assert.Equal(t, time.Duration(0), out.PlannedDowntime)
				assert.Equal(t, 2, out.IntervalCount)
				assert.False(t, out.Flags.NoData)
				assert.False(t, out.Flags.ShiftConfigMissing)
				assert.False(t, out.Flags.OpenIntervalClipped)
			},
		},
		{
			name:      "happy path - empty interval slice returns output with no_data flag and availability 0",
			machineID: "CNC-01",
			window:    qWin,
			setupMocks: func(ir *mocks.MockIntervalReader, sr *mocks.MockShiftReader) {
				ir.EXPECT().ByMachineRange(gomock.Any(), "CNC-01", qWin.From, qWin.To).Return(nil, nil).Times(1)
				sr.EXPECT().ForMachineWindow(gomock.Any(), "CNC-01", qWin).Return(420*time.Minute, true, nil).Times(1)
			},
			validate: func(t *testing.T, out *application.Output) {
				require.NotNil(t, out)
				assert.False(t, out.HasData)
				assert.True(t, out.Flags.NoData)
				assert.Equal(t, 0.0, out.Availability)
				assert.Equal(t, 0, out.IntervalCount)
			},
		},
		{
			name:      "business rule - open interval is clipped to window.To and open_interval_clipped flag is set",
			machineID: "CNC-01",
			window:    qWin,
			setupMocks: func(ir *mocks.MockIntervalReader, sr *mocks.MockShiftReader) {
				intervals := []avdomain.StateInterval{
					{MachineID: "CNC-01", State: ooedomain.Running, StartedAt: qT0, EndedAt: nil},
				}
				ir.EXPECT().ByMachineRange(gomock.Any(), "CNC-01", qWin.From, qWin.To).Return(intervals, nil).Times(1)
				sr.EXPECT().ForMachineWindow(gomock.Any(), "CNC-01", qWin).Return(420*time.Minute, true, nil).Times(1)
			},
			validate: func(t *testing.T, out *application.Output) {
				require.NotNil(t, out)
				assert.True(t, out.Flags.OpenIntervalClipped)
				assert.True(t, out.HasData)
				assert.InDelta(t, 1.0, out.Availability, 0.0001)
			},
		},
		{
			name:      "business rule - shift not found: shift_config_missing flag set and wall-clock duration used as planned",
			machineID: "CNC-01",
			window:    qWin,
			setupMocks: func(ir *mocks.MockIntervalReader, sr *mocks.MockShiftReader) {
				intervals := []avdomain.StateInterval{
					{MachineID: "CNC-01", State: ooedomain.Running, StartedAt: qT0, EndedAt: tsPtr(qT0.Add(420 * time.Minute))},
				}
				ir.EXPECT().ByMachineRange(gomock.Any(), "CNC-01", qWin.From, qWin.To).Return(intervals, nil).Times(1)
				sr.EXPECT().ForMachineWindow(gomock.Any(), "CNC-01", qWin).Return(time.Duration(0), false, nil).Times(1)
			},
			validate: func(t *testing.T, out *application.Output) {
				require.NotNil(t, out)
				assert.True(t, out.Flags.ShiftConfigMissing)
				assert.Equal(t, 420*time.Minute, out.PlannedTime, "wall-clock window used as planned time")
				assert.True(t, out.HasData)
			},
		},
		{
			name:      "business rule - no_data flag set when planned time is zero (window fully inside break)",
			machineID: "CNC-01",
			window:    qWin,
			setupMocks: func(ir *mocks.MockIntervalReader, sr *mocks.MockShiftReader) {
				intervals := []avdomain.StateInterval{
					{MachineID: "CNC-01", State: ooedomain.Running, StartedAt: qT0, EndedAt: tsPtr(qT0.Add(420 * time.Minute))},
				}
				ir.EXPECT().ByMachineRange(gomock.Any(), "CNC-01", qWin.From, qWin.To).Return(intervals, nil).Times(1)
				sr.EXPECT().ForMachineWindow(gomock.Any(), "CNC-01", qWin).Return(time.Duration(0), true, nil).Times(1)
			},
			validate: func(t *testing.T, out *application.Output) {
				require.NotNil(t, out)
				assert.False(t, out.HasData)
				assert.True(t, out.Flags.NoData)
			},
		},
		{
			name:      "failure - IntervalReader error is propagated, ShiftReader never called",
			machineID: "CNC-01",
			window:    qWin,
			setupMocks: func(ir *mocks.MockIntervalReader, sr *mocks.MockShiftReader) {
				ir.EXPECT().ByMachineRange(gomock.Any(), "CNC-01", qWin.From, qWin.To).Return(nil, qErr).Times(1)
				sr.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			expectError: true,
		},
		{
			name:      "failure - ShiftReader error is propagated",
			machineID: "CNC-01",
			window:    qWin,
			setupMocks: func(ir *mocks.MockIntervalReader, sr *mocks.MockShiftReader) {
				ir.EXPECT().ByMachineRange(gomock.Any(), "CNC-01", qWin.From, qWin.To).Return(nil, nil).Times(1)
				sr.EXPECT().ForMachineWindow(gomock.Any(), "CNC-01", qWin).Return(time.Duration(0), false, qErr).Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ir := mocks.NewMockIntervalReader(ctrl)
			sr := mocks.NewMockShiftReader(ctrl)
			tc.setupMocks(ir, sr)

			uc := application.NewUseCase(ir, sr, observability.NewNopLogger())
			out, err := uc.Execute(context.Background(), tc.machineID, tc.window)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, out)
				return
			}
			require.NoError(t, err)
			if tc.validate != nil {
				tc.validate(t, out)
			}
		})
	}
}
