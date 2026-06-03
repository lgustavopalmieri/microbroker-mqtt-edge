package domain_test

import (
	"testing"
	"time"

	"microbroker-mqtt-edge/internal/modules/oee/domain"

	"github.com/stretchr/testify/assert"
)

// ── MachineState ────────────────────────────────────────────────────────────

func TestMachineState_IsDowntime(t *testing.T) {
	cases := []struct {
		state domain.MachineState
		want  bool
	}{
		{domain.Stopped, true},
		{domain.Setup, true},
		{domain.Maintenance, true},
		{domain.Running, false},
		{domain.Idle, false},
		{domain.Off, false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.state.IsDowntime(), "IsDowntime(%s)", tc.state)
	}
}

func TestMachineState_IsPlannedStop(t *testing.T) {
	cases := []struct {
		state domain.MachineState
		want  bool
	}{
		{domain.Setup, true},
		{domain.Maintenance, true},
		{domain.Stopped, false},
		{domain.Running, false},
		{domain.Idle, false},
		{domain.Off, false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.state.IsPlannedStop(), "IsPlannedStop(%s)", tc.state)
	}
}

func TestMachineState_Valid(t *testing.T) {
	valid := []domain.MachineState{
		domain.Running, domain.Stopped, domain.Setup,
		domain.Idle, domain.Maintenance, domain.Off,
	}
	for _, s := range valid {
		assert.True(t, s.Valid(), "expected %s to be valid", s)
	}
	assert.False(t, domain.MachineState("unknown").Valid())
	assert.False(t, domain.MachineState("").Valid())
}

// ── PlannedProductionTime ───────────────────────────────────────────────────

func date(y, mo, d, h, m int) time.Time {
	return time.Date(y, time.Month(mo), d, h, m, 0, 0, time.UTC)
}

func TestShift_PlannedProductionTime(t *testing.T) {
	// Weekday helpers — 2026-06-01 is a Monday.
	mon := date(2026, 6, 1, 0, 0)
	_ = mon

	cases := []struct {
		name   string
		shift  domain.Shift
		window domain.Window
		want   time.Duration
	}{
		{
			// Window exactly covers the shift: 07:00–15:00, no breaks.
			name: "single_day_full_shift",
			shift: domain.Shift{
				StartMin: 7 * 60,  // 07:00
				EndMin:   15 * 60, // 15:00
				TZ:       "UTC",
			},
			window: domain.Window{
				From: date(2026, 6, 1, 7, 0),
				To:   date(2026, 6, 1, 15, 0),
			},
			want: 8 * time.Hour,
		},
		{
			// Window is narrower than the shift: 09:00–11:00 inside 07:00–15:00.
			name: "window_narrower_than_shift",
			shift: domain.Shift{
				StartMin: 7 * 60,
				EndMin:   15 * 60,
				TZ:       "UTC",
			},
			window: domain.Window{
				From: date(2026, 6, 1, 9, 0),
				To:   date(2026, 6, 1, 11, 0),
			},
			want: 2 * time.Hour,
		},
		{
			// Three-day window, shift every day, no breaks.
			name: "multi_day_no_weekday_filter",
			shift: domain.Shift{
				StartMin: 6 * 60,  // 06:00
				EndMin:   14 * 60, // 14:00
				TZ:       "UTC",
			},
			window: domain.Window{
				From: date(2026, 6, 1, 6, 0),
				To:   date(2026, 6, 3, 14, 0),
			},
			want: 3 * 8 * time.Hour,
		},
		{
			// 7-day window Mon-Sun, shift only Mon-Fri (5 days × 8 h = 40 h).
			// 2026-06-01 = Mon, 2026-06-07 = Sun.
			name: "weekday_mask_mon_to_fri",
			shift: domain.Shift{
				StartMin: 8 * 60,
				EndMin:   16 * 60,
				Weekdays: []time.Weekday{
					time.Monday, time.Tuesday, time.Wednesday,
					time.Thursday, time.Friday,
				},
				TZ: "UTC",
			},
			window: domain.Window{
				From: date(2026, 6, 1, 8, 0),  // Mon 08:00
				To:   date(2026, 6, 7, 16, 0), // Sun 16:00
			},
			want: 5 * 8 * time.Hour,
		},
		{
			// Shift 07:00–15:00, 30-min break 12:00–12:30, full-day window.
			name: "break_overlap",
			shift: domain.Shift{
				StartMin: 7 * 60,
				EndMin:   15 * 60,
				Breaks: []domain.Break{
					{StartMin: 12 * 60, EndMin: 12*60 + 30, Type: "meal"},
				},
				TZ: "UTC",
			},
			window: domain.Window{
				From: date(2026, 6, 1, 7, 0),
				To:   date(2026, 6, 1, 15, 0),
			},
			want: 8*time.Hour - 30*time.Minute,
		},
		{
			// Window falls entirely outside the shift hours (after 15:00).
			name: "no_overlap_outside_shift",
			shift: domain.Shift{
				StartMin: 7 * 60,
				EndMin:   15 * 60,
				TZ:       "UTC",
			},
			window: domain.Window{
				From: date(2026, 6, 1, 16, 0),
				To:   date(2026, 6, 1, 18, 0),
			},
			want: 0,
		},
		{
			// Overnight shift: 22:00–06:00 next day (EndMin = 30*60).
			// Window covers the full overnight shift from 22:00 to 06:00.
			name: "overnight_shift",
			shift: domain.Shift{
				StartMin: 22 * 60, // 22:00
				EndMin:   30 * 60, // 06:00 next day (1800 min)
				TZ:       "UTC",
			},
			window: domain.Window{
				From: date(2026, 6, 1, 22, 0),
				To:   date(2026, 6, 2, 6, 0),
			},
			want: 8 * time.Hour,
		},
		{
			// Zero-duration window returns 0.
			name: "invalid_window_same_time",
			shift: domain.Shift{
				StartMin: 7 * 60,
				EndMin:   15 * 60,
				TZ:       "UTC",
			},
			window: domain.Window{
				From: date(2026, 6, 1, 7, 0),
				To:   date(2026, 6, 1, 7, 0),
			},
			want: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.shift.PlannedProductionTime(tc.window)
			assert.Equal(t, tc.want, got)
		})
	}
}
