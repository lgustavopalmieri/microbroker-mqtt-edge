package application

import "time"

// OutputFlags carries data-quality indicators for the query result.
type OutputFlags struct {
	ShiftConfigMissing  bool `json:"shift_config_missing,omitempty"`
	OpenIntervalClipped bool `json:"open_interval_clipped,omitempty"`
	NoData              bool `json:"no_data,omitempty"`
}

// Output is the result of an availability query.
type Output struct {
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
	Flags             OutputFlags   `json:"flags"`
}
