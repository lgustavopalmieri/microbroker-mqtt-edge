package domain

import "time"

type SnapshotFlags struct {
	ShiftConfigMissing  bool `json:"shift_config_missing,omitempty"`
	OpenIntervalClipped bool `json:"open_interval_clipped,omitempty"`
	NoData              bool `json:"no_data,omitempty"`
}

type AvailabilitySnapshot struct {
	MachineID         string        `json:"machine_id"`
	WindowFrom        time.Time     `json:"window_from"`
	WindowTo          time.Time     `json:"window_to"`
	Availability      float64       `json:"availability"`
	PlannedTime       time.Duration `json:"planned_time_ns"`
	RunTime           time.Duration `json:"run_time_ns"`
	PlannedDowntime   time.Duration `json:"planned_downtime_ns"`
	UnplannedDowntime time.Duration `json:"unplanned_downtime_ns"`
	IntervalCount     int           `json:"interval_count"`
	HasData           bool          `json:"has_data"`
	ComputedAt        time.Time     `json:"computed_at"`
	Flags             SnapshotFlags `json:"flags"`
}
