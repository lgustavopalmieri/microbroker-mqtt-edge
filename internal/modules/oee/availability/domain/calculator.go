package domain

import (
	"time"

	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

type Input struct {
	PlannedProductionTime time.Duration
	PlannedDowntime       time.Duration
	UnplannedDowntime     time.Duration
	IntervalCount         int
}

type Result struct {
	Availability      float64
	PlannedTime       time.Duration
	RunTime           time.Duration
	PlannedDowntime   time.Duration
	UnplannedDowntime time.Duration
	HasData           bool
}

func Aggregate(w ooedomain.Window, intervals []StateInterval, planned time.Duration) Input {
	var plannedDown, unplannedDown time.Duration
	count := 0

	for _, iv := range intervals {
		start := iv.StartedAt
		if start.Before(w.From) {
			start = w.From
		}

		end := w.To
		if iv.EndedAt != nil && iv.EndedAt.Before(w.To) {
			end = *iv.EndedAt
		}

		if !end.After(start) {
			continue
		}

		dur := end.Sub(start)
		if iv.State.IsDowntime() {
			if iv.State.IsPlannedStop() {
				plannedDown += dur
			} else {
				unplannedDown += dur
			}
		}
		count++
	}

	return Input{
		PlannedProductionTime: planned,
		PlannedDowntime:       plannedDown,
		UnplannedDowntime:     unplannedDown,
		IntervalCount:         count,
	}
}

func Availability(in Input) Result {
	if in.PlannedProductionTime == 0 {
		return Result{HasData: false}
	}

	downtime := in.PlannedDowntime + in.UnplannedDowntime
	runTime := in.PlannedProductionTime - downtime
	runTime = max(runTime, 0)

	avail := float64(runTime) / float64(in.PlannedProductionTime)
	if avail > 1.0 {
		avail = 1.0
	}

	return Result{
		Availability:      avail,
		PlannedTime:       in.PlannedProductionTime,
		RunTime:           runTime,
		PlannedDowntime:   in.PlannedDowntime,
		UnplannedDowntime: in.UnplannedDowntime,
		HasData:           true,
	}
}
