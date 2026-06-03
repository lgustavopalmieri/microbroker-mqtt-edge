package domain_test

import (
	"testing"
	"time"

	"microbroker-mqtt-edge/internal/modules/oee/domain"

	"github.com/stretchr/testify/assert"
)

// date builds a UTC time at minute resolution for table-driven cases.
func date(y, mo, d, h, m int) time.Time {
	return time.Date(y, time.Month(mo), d, h, m, 0, 0, time.UTC)
}

// ── MachineState ────────────────────────────────────────────────────────────

func TestMachineState_IsDowntime(t *testing.T) {
	tests := []struct {
		name  string
		state domain.MachineState
		want  bool
	}{
		{"running is not downtime", domain.Running, false},
		{"stopped is downtime", domain.Stopped, true},
		{"setup is downtime", domain.Setup, true},
		{"idle is not downtime", domain.Idle, false},
		{"maintenance is downtime", domain.Maintenance, true},
		{"off is not downtime", domain.Off, false},
		{"unknown value is not downtime (defensive)", domain.MachineState("unknown"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.state.IsDowntime())
		})
	}
}

func TestMachineState_IsPlannedStop(t *testing.T) {
	tests := []struct {
		name  string
		state domain.MachineState
		want  bool
	}{
		{"setup is a planned stop", domain.Setup, true},
		{"maintenance is a planned stop", domain.Maintenance, true},
		{"stopped is not a planned stop", domain.Stopped, false},
		{"running is not a planned stop", domain.Running, false},
		{"idle is not a planned stop", domain.Idle, false},
		{"off is not a planned stop", domain.Off, false},
		{"unknown value is not a planned stop (defensive)", domain.MachineState("unknown"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.state.IsPlannedStop())
		})
	}
}

func TestMachineState_Valid(t *testing.T) {
	tests := []struct {
		name  string
		state domain.MachineState
		want  bool
	}{
		{"running is valid", domain.Running, true},
		{"stopped is valid", domain.Stopped, true},
		{"setup is valid", domain.Setup, true},
		{"idle is valid", domain.Idle, true},
		{"maintenance is valid", domain.Maintenance, true},
		{"off is valid", domain.Off, true},
		{"empty string is invalid", domain.MachineState(""), false},
		{"unknown literal is invalid", domain.MachineState("paused"), false},
		{"uppercase RUNNING is invalid (case-sensitive)", domain.MachineState("RUNNING"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.state.Valid())
		})
	}
}

// ── PlannedProductionTime ───────────────────────────────────────────────────

func TestShift_PlannedProductionTime(t *testing.T) {
	tests := []struct {
		name   string
		shift  domain.Shift
		window domain.Window
		want   time.Duration
	}{
		// --- happy path / clipping ---
		{
			name:   "single-day window exactly covering the shift returns full shift duration",
			shift:  domain.Shift{StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC"},
			window: domain.Window{From: date(2026, 6, 1, 7, 0), To: date(2026, 6, 1, 15, 0)},
			want:   8 * time.Hour,
		},
		{
			name:   "window narrower than the shift returns only the overlapping span",
			shift:  domain.Shift{StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC"},
			window: domain.Window{From: date(2026, 6, 1, 9, 0), To: date(2026, 6, 1, 11, 0)},
			want:   2 * time.Hour,
		},
		{
			name:   "window wider than the shift on both sides clips to the shift duration",
			shift:  domain.Shift{StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC"},
			window: domain.Window{From: date(2026, 6, 1, 0, 0), To: date(2026, 6, 1, 23, 0)},
			want:   8 * time.Hour,
		},

		// --- multi-day & weekday mask ---
		{
			name:   "multi-day window, shift every day (empty weekdays) sums each day",
			shift:  domain.Shift{StartMin: 6 * 60, EndMin: 14 * 60, TZ: "UTC"},
			window: domain.Window{From: date(2026, 6, 1, 6, 0), To: date(2026, 6, 3, 14, 0)},
			want:   3 * 8 * time.Hour,
		},
		{
			// 2026-06-01 = Mon, 2026-06-07 = Sun.
			name: "full-week window with Mon-Fri mask returns 5 shift-days",
			shift: domain.Shift{
				StartMin: 8 * 60, EndMin: 16 * 60, TZ: "UTC",
				Weekdays: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
			},
			window: domain.Window{From: date(2026, 6, 1, 8, 0), To: date(2026, 6, 7, 16, 0)},
			want:   5 * 8 * time.Hour,
		},
		{
			// Mon 12:00→16:00 (4h) + Tue full (8h) + Wed 08:00→10:00 (2h) = 14h.
			name: "multi-day window with weekday mask and partial first/last day counts only masked, clipped days",
			shift: domain.Shift{
				StartMin: 8 * 60, EndMin: 16 * 60, TZ: "UTC",
				Weekdays: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
			},
			window: domain.Window{From: date(2026, 6, 1, 12, 0), To: date(2026, 6, 3, 10, 0)},
			want:   14 * time.Hour,
		},
		{
			// 2026-06-01 is a Monday; mask only covers the weekend.
			name: "window on a weekday excluded by the mask returns zero",
			shift: domain.Shift{
				StartMin: 8 * 60, EndMin: 16 * 60, TZ: "UTC",
				Weekdays: []time.Weekday{time.Saturday, time.Sunday},
			},
			window: domain.Window{From: date(2026, 6, 1, 8, 0), To: date(2026, 6, 1, 16, 0)},
			want:   0,
		},

		// --- breaks ---
		{
			name: "single break inside the window is deducted",
			shift: domain.Shift{
				StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC",
				Breaks: []domain.Break{{StartMin: 12 * 60, EndMin: 12*60 + 30, Type: "meal"}},
			},
			window: domain.Window{From: date(2026, 6, 1, 7, 0), To: date(2026, 6, 1, 15, 0)},
			want:   8*time.Hour - 30*time.Minute,
		},
		{
			name: "multiple breaks inside the window are all deducted",
			shift: domain.Shift{
				StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC",
				Breaks: []domain.Break{
					{StartMin: 10 * 60, EndMin: 10*60 + 15, Type: "rest"},
					{StartMin: 12 * 60, EndMin: 12*60 + 30, Type: "meal"},
				},
			},
			window: domain.Window{From: date(2026, 6, 1, 7, 0), To: date(2026, 6, 1, 15, 0)},
			want:   7*time.Hour + 15*time.Minute,
		},
		{
			// Break 14:00–14:30 is inside the shift but outside the [07:00,13:00] window.
			name: "break entirely outside the window is not deducted",
			shift: domain.Shift{
				StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC",
				Breaks: []domain.Break{{StartMin: 14 * 60, EndMin: 14*60 + 30, Type: "rest"}},
			},
			window: domain.Window{From: date(2026, 6, 1, 7, 0), To: date(2026, 6, 1, 13, 0)},
			want:   6 * time.Hour,
		},
		{
			// Window 07:00–12:30; break 12:00–13:00 overlaps only 12:00–12:30 (30m).
			name: "break partially overlapping the window deducts only the overlapping portion",
			shift: domain.Shift{
				StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC",
				Breaks: []domain.Break{{StartMin: 12 * 60, EndMin: 13 * 60, Type: "meal"}},
			},
			window: domain.Window{From: date(2026, 6, 1, 7, 0), To: date(2026, 6, 1, 12, 30)},
			want:   5 * time.Hour,
		},

		// --- timezones ---
		{
			name:   "empty TZ defaults to UTC (same result as explicit UTC)",
			shift:  domain.Shift{StartMin: 7 * 60, EndMin: 15 * 60, TZ: ""},
			window: domain.Window{From: date(2026, 6, 1, 7, 0), To: date(2026, 6, 1, 15, 0)},
			want:   8 * time.Hour,
		},
		{
			name:   "invalid TZ string falls back to UTC without panicking",
			shift:  domain.Shift{StartMin: 7 * 60, EndMin: 15 * 60, TZ: "Not/AZone"},
			window: domain.Window{From: date(2026, 6, 1, 7, 0), To: date(2026, 6, 1, 15, 0)},
			want:   8 * time.Hour,
		},

		// --- overnight & guards ---
		{
			// 22:00 → 06:00 next day, EndMin = 30*60 = 1800.
			name:   "overnight shift spanning midnight (end_minute > 1440) is counted across days",
			shift:  domain.Shift{StartMin: 22 * 60, EndMin: 30 * 60, TZ: "UTC"},
			window: domain.Window{From: date(2026, 6, 1, 22, 0), To: date(2026, 6, 2, 6, 0)},
			want:   8 * time.Hour,
		},
		{
			name:   "window entirely outside shift hours returns zero",
			shift:  domain.Shift{StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC"},
			window: domain.Window{From: date(2026, 6, 1, 16, 0), To: date(2026, 6, 1, 18, 0)},
			want:   0,
		},
		{
			name:   "zero-duration window (To == From) returns zero",
			shift:  domain.Shift{StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC"},
			window: domain.Window{From: date(2026, 6, 1, 7, 0), To: date(2026, 6, 1, 7, 0)},
			want:   0,
		},
		{
			name:   "reversed window (To before From) returns zero",
			shift:  domain.Shift{StartMin: 7 * 60, EndMin: 15 * 60, TZ: "UTC"},
			window: domain.Window{From: date(2026, 6, 1, 15, 0), To: date(2026, 6, 1, 7, 0)},
			want:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.shift.PlannedProductionTime(tc.window))
		})
	}
}
