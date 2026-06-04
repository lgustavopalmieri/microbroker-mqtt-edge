package application

import (
	"context"
	"time"

	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// Apply updates the in-memory accumulator for a machine. Safe for concurrent callers.
func (e *Engine) Apply(t avdomain.StateTransition) {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, ok := e.machines[t.MachineID]
	if !ok {
		data = &machineData{windowStart: t.Timestamp}
		e.machines[t.MachineID] = data
	}

	// Close the last open interval.
	if n := len(data.intervals); n > 0 {
		last := &data.intervals[n-1]
		if last.IsOpen() {
			ts := t.Timestamp
			last.EndedAt = &ts
		}
	}

	// Open the new interval.
	data.intervals = append(data.intervals, avdomain.StateInterval{
		MachineID: t.MachineID,
		State:     t.State,
		StartedAt: t.Timestamp,
	})
}

// Start rehydrates open intervals for tracked machines then runs the tick loop until ctx is cancelled.
func (e *Engine) Start(ctx context.Context, machineIDs []string) {
	e.rehydrate(ctx, machineIDs)

	ticker := time.NewTicker(e.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tick(ctx)
		}
	}
}

// rehydrate seeds in-memory state from the last open interval in the persistent store.
func (e *Engine) rehydrate(ctx context.Context, machineIDs []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, id := range machineIDs {
		if _, ok := e.machines[id]; ok {
			continue
		}
		iv, found, err := e.intervals.LastOpen(ctx, id)
		if err != nil {
			e.logger.Error("live: rehydration failed", "machine_id", id, "error", err)
			continue
		}
		if found {
			e.machines[id] = &machineData{
				intervals:   []avdomain.StateInterval{iv},
				windowStart: iv.StartedAt,
			}
		}
	}
}

// tick computes a snapshot for every tracked machine and pushes it to the sink.
func (e *Engine) tick(ctx context.Context) {
	e.mu.Lock()
	snapshot := make(map[string]*machineData, len(e.machines))
	for id, data := range e.machines {
		// Shallow-copy intervals so we can clip the open one without mutating the accumulator.
		cp := &machineData{
			windowStart: data.windowStart,
			intervals:   make([]avdomain.StateInterval, len(data.intervals)),
		}
		copy(cp.intervals, data.intervals)
		snapshot[id] = cp
	}
	e.mu.Unlock()

	now := e.now()

	for id, data := range snapshot {
		// Clip any open interval to now.
		if n := len(data.intervals); n > 0 {
			last := &data.intervals[n-1]
			if last.IsOpen() {
				ts := now
				last.EndedAt = &ts
			}
		}

		window := ooedomain.Window{From: data.windowStart, To: now}

		planned, _, err := e.shifts.ForMachineWindow(ctx, id, window)
		if err != nil {
			e.logger.Error("live: shift read failed", "machine_id", id, "error", err)
			continue
		}

		in := avdomain.Aggregate(window, data.intervals, planned)
		result := avdomain.Availability(in)

		s := avdomain.AvailabilitySnapshot{
			MachineID:         id,
			WindowFrom:        window.From,
			WindowTo:          now,
			Availability:      result.Availability,
			PlannedTime:       result.PlannedTime,
			RunTime:           result.RunTime,
			PlannedDowntime:   result.PlannedDowntime,
			UnplannedDowntime: result.UnplannedDowntime,
			IntervalCount:     in.IntervalCount,
			HasData:           result.HasData,
			ComputedAt:        now,
		}
		if !result.HasData {
			s.Flags.NoData = true
		}

		if err := e.sink.Update(ctx, s); err != nil {
			e.logger.Error("live: sink update failed", "machine_id", id, "error", err)
		}
	}
}
