package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

var (
	baseTime = time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	baseWin  = ooedomain.Window{From: baseTime, To: baseTime.Add(8 * time.Hour)}
)

func timePtr(t time.Time) *time.Time { return &t }

func TestAggregate(t *testing.T) {
	tests := []struct {
		name              string
		window            ooedomain.Window
		intervals         []avdomain.StateInterval
		planned           time.Duration
		wantPlannedDown   time.Duration
		wantUnplannedDown time.Duration
		wantCount         int
	}{
		{
			name:   "clips interval that starts before window.From to window.From",
			window: baseWin,
			intervals: []avdomain.StateInterval{
				{State: ooedomain.Stopped, StartedAt: baseTime.Add(-30 * time.Minute), EndedAt: timePtr(baseTime.Add(30 * time.Minute))},
			},
			planned:           8 * time.Hour,
			wantUnplannedDown: 30 * time.Minute,
			wantCount:         1,
		},
		{
			name:   "clips open interval (nil EndedAt) to window.To",
			window: baseWin,
			intervals: []avdomain.StateInterval{
				{State: ooedomain.Stopped, StartedAt: baseTime.Add(7 * time.Hour), EndedAt: nil},
			},
			planned:           8 * time.Hour,
			wantUnplannedDown: 1 * time.Hour,
			wantCount:         1,
		},
		{
			name:   "skips interval whose EndedAt falls entirely before window.From",
			window: baseWin,
			intervals: []avdomain.StateInterval{
				{State: ooedomain.Stopped, StartedAt: baseTime.Add(-2 * time.Hour), EndedAt: timePtr(baseTime.Add(-1 * time.Hour))},
			},
			planned:   8 * time.Hour,
			wantCount: 0,
		},
		{
			name:   "skips interval that starts on or after window.To",
			window: baseWin,
			intervals: []avdomain.StateInterval{
				{State: ooedomain.Stopped, StartedAt: baseTime.Add(8 * time.Hour), EndedAt: timePtr(baseTime.Add(9 * time.Hour))},
			},
			planned:   8 * time.Hour,
			wantCount: 0,
		},
		{
			name:   "running state does not contribute to downtime totals",
			window: baseWin,
			intervals: []avdomain.StateInterval{
				{State: ooedomain.Running, StartedAt: baseTime, EndedAt: timePtr(baseTime.Add(8 * time.Hour))},
			},
			planned:   8 * time.Hour,
			wantCount: 1,
		},
		{
			name:   "stopped (unplanned) contributes to UnplannedDowntime only",
			window: baseWin,
			intervals: []avdomain.StateInterval{
				{State: ooedomain.Stopped, StartedAt: baseTime, EndedAt: timePtr(baseTime.Add(47 * time.Minute))},
			},
			planned:           420 * time.Minute,
			wantUnplannedDown: 47 * time.Minute,
			wantCount:         1,
		},
		{
			name:   "setup/maintenance (planned) contributes to PlannedDowntime only",
			window: baseWin,
			intervals: []avdomain.StateInterval{
				{State: ooedomain.Setup, StartedAt: baseTime, EndedAt: timePtr(baseTime.Add(30 * time.Minute))},
				{State: ooedomain.Maintenance, StartedAt: baseTime.Add(time.Hour), EndedAt: timePtr(baseTime.Add(time.Hour + 15*time.Minute))},
			},
			planned:         8 * time.Hour,
			wantPlannedDown: 45 * time.Minute,
			wantCount:       2,
		},
		{
			name:   "mixed states — sums planned and unplanned downtime independently",
			window: baseWin,
			intervals: []avdomain.StateInterval{
				{State: ooedomain.Stopped, StartedAt: baseTime, EndedAt: timePtr(baseTime.Add(30 * time.Minute))},
				{State: ooedomain.Setup, StartedAt: baseTime.Add(time.Hour), EndedAt: timePtr(baseTime.Add(time.Hour + 20*time.Minute))},
				{State: ooedomain.Running, StartedAt: baseTime.Add(2 * time.Hour), EndedAt: timePtr(baseTime.Add(8 * time.Hour))},
			},
			planned:           8 * time.Hour,
			wantUnplannedDown: 30 * time.Minute,
			wantPlannedDown:   20 * time.Minute,
			wantCount:         3,
		},
		{
			name:      "empty interval slice — returns zero downtime, IntervalCount 0",
			window:    baseWin,
			intervals: nil,
			planned:   8 * time.Hour,
			wantCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := avdomain.Aggregate(tc.window, tc.intervals, tc.planned)

			assert.Equal(t, tc.planned, got.PlannedProductionTime)
			assert.Equal(t, tc.wantPlannedDown, got.PlannedDowntime, "PlannedDowntime")
			assert.Equal(t, tc.wantUnplannedDown, got.UnplannedDowntime, "UnplannedDowntime")
			assert.Equal(t, tc.wantCount, got.IntervalCount, "IntervalCount")
		})
	}
}

func TestAvailability(t *testing.T) {
	tests := []struct {
		name             string
		planned          time.Duration
		plannedDown      time.Duration
		unplannedDown    time.Duration
		wantHasData      bool
		wantAvailability float64
		wantRunTime      time.Duration
	}{
		{
			name:             "research §7 oracle — planned 420 min, stop 47 min → availability ≈ 0.8881",
			planned:          420 * time.Minute,
			unplannedDown:    47 * time.Minute,
			wantHasData:      true,
			wantAvailability: 373.0 / 420.0,
			wantRunTime:      373 * time.Minute,
		},
		{
			name:        "PlannedProductionTime == 0 → HasData false, no divide-by-zero",
			planned:     0,
			wantHasData: false,
		},
		{
			name:             "no downtime at all → availability 1.0, HasData true",
			planned:          8 * time.Hour,
			wantHasData:      true,
			wantAvailability: 1.0,
			wantRunTime:      8 * time.Hour,
		},
		{
			name:             "full downtime equals planned → availability 0.0, RunTime 0",
			planned:          8 * time.Hour,
			unplannedDown:    8 * time.Hour,
			wantHasData:      true,
			wantAvailability: 0.0,
			wantRunTime:      0,
		},
		{
			name:             "downtime exceeds planned → clamps availability to 0.0, RunTime never negative",
			planned:          4 * time.Hour,
			unplannedDown:    6 * time.Hour,
			wantHasData:      true,
			wantAvailability: 0.0,
			wantRunTime:      0,
		},
		{
			name:             "planned downtime only (no unplanned) — RunTime and Result fields correct",
			planned:          8 * time.Hour,
			plannedDown:      2 * time.Hour,
			wantHasData:      true,
			wantAvailability: 6.0 / 8.0,
			wantRunTime:      6 * time.Hour,
		},
		{
			name:             "unplanned downtime only (no planned stops) — PlannedDowntime stays 0",
			planned:          8 * time.Hour,
			unplannedDown:    1 * time.Hour,
			wantHasData:      true,
			wantAvailability: 7.0 / 8.0,
			wantRunTime:      7 * time.Hour,
		},
		{
			name:             "mixed planned + unplanned — availability, RunTime, both downtime fields all correct",
			planned:          8 * time.Hour,
			plannedDown:      1 * time.Hour,
			unplannedDown:    30 * time.Minute,
			wantHasData:      true,
			wantAvailability: 6.5 / 8.0,
			wantRunTime:      6*time.Hour + 30*time.Minute,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := avdomain.Input{
				PlannedProductionTime: tc.planned,
				PlannedDowntime:       tc.plannedDown,
				UnplannedDowntime:     tc.unplannedDown,
			}
			got := avdomain.Availability(in)

			assert.Equal(t, tc.wantHasData, got.HasData)
			if !tc.wantHasData {
				return
			}
			assert.InDelta(t, tc.wantAvailability, got.Availability, 0.0001, "Availability")
			assert.Equal(t, tc.wantRunTime, got.RunTime, "RunTime")
			assert.Equal(t, tc.planned, got.PlannedTime, "PlannedTime")
			assert.Equal(t, tc.plannedDown, got.PlannedDowntime, "PlannedDowntime")
			assert.Equal(t, tc.unplannedDown, got.UnplannedDowntime, "UnplannedDowntime")
		})
	}
}
