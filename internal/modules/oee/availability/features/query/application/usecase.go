package application

import (
	"context"

	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// Execute reads intervals and planned time for machineID in window w, then returns the
// computed availability together with data-quality flags.
func (uc *UseCase) Execute(ctx context.Context, machineID string, w ooedomain.Window) (*Output, error) {
	intervals, err := uc.intervals.ByMachineRange(ctx, machineID, w.From, w.To)
	if err != nil {
		uc.logger.Error("query: failed to read intervals", "machine_id", machineID, "error", err)
		return nil, err
	}

	planned, found, err := uc.shifts.ForMachineWindow(ctx, machineID, w)
	if err != nil {
		uc.logger.Error("query: failed to read shift", "machine_id", machineID, "error", err)
		return nil, err
	}

	var flags OutputFlags
	if !found {
		flags.ShiftConfigMissing = true
		planned = w.To.Sub(w.From)
	}

	// Detect and clip open intervals to window.To.
	clipped := make([]avdomain.StateInterval, len(intervals))
	copy(clipped, intervals)
	for i := range clipped {
		if clipped[i].IsOpen() {
			flags.OpenIntervalClipped = true
			end := w.To
			clipped[i].EndedAt = &end
		}
	}

	in := avdomain.Aggregate(w, clipped, planned)
	result := avdomain.Availability(in)

	hasData := result.HasData && in.IntervalCount > 0
	if !hasData {
		flags.NoData = true
	}

	availability := result.Availability
	if !hasData {
		availability = 0
	}

	return &Output{
		MachineID:         machineID,
		WindowFrom:        w.From,
		WindowTo:          w.To,
		Availability:      availability,
		PlannedTime:       result.PlannedTime,
		RunTime:           result.RunTime,
		PlannedDowntime:   result.PlannedDowntime,
		UnplannedDowntime: result.UnplannedDowntime,
		IntervalCount:     in.IntervalCount,
		HasData:           hasData,
		Flags:             flags,
	}, nil
}
