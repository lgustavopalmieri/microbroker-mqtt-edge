package domain

import "time"

// Window is a closed time interval [From, To).
type Window struct {
	From time.Time
	To   time.Time
}

// Break is a scheduled break within a shift, expressed as minutes from midnight.
type Break struct {
	StartMin int
	EndMin   int
	Type     string // rest | meal | cleaning
}

// Shift describes a recurring production shift.
// StartMin and EndMin are minutes from midnight; EndMin may exceed 1440 for overnight shifts.
// Weekdays lists the days on which the shift applies; empty means every day.
type Shift struct {
	ID        string
	Name      string
	MachineID string // empty/"*" = applies to all machines
	StartMin  int
	EndMin    int
	Weekdays  []time.Weekday
	Breaks    []Break
	TZ        string // IANA timezone; empty defaults to UTC
}

// PlannedProductionTime returns the total duration within w that falls inside the shift
// schedule after subtracting break periods. Returns 0 for an empty or invalid window.
func (s Shift) PlannedProductionTime(w Window) time.Duration {
	if !w.To.After(w.From) {
		return 0
	}

	loc := time.UTC
	if s.TZ != "" {
		if l, err := time.LoadLocation(s.TZ); err == nil {
			loc = l
		}
	}

	weekdaySet := make(map[time.Weekday]bool, len(s.Weekdays))
	for _, wd := range s.Weekdays {
		weekdaySet[wd] = true
	}
	allDays := len(s.Weekdays) == 0

	var total time.Duration

	// Start one day early to catch overnight shifts that began on the previous calendar day.
	fromLocal := w.From.In(loc)
	startDay := time.Date(fromLocal.Year(), fromLocal.Month(), fromLocal.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -1)

	toLocal := w.To.In(loc)
	endDay := time.Date(toLocal.Year(), toLocal.Month(), toLocal.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		if !allDays && !weekdaySet[d.Weekday()] {
			continue
		}

		shiftStart := d.Add(time.Duration(s.StartMin) * time.Minute)
		shiftEnd := d.Add(time.Duration(s.EndMin) * time.Minute)

		intFrom := maxTime(shiftStart, w.From)
		intTo := minTime(shiftEnd, w.To)
		if !intTo.After(intFrom) {
			continue
		}

		workDur := intTo.Sub(intFrom)

		for _, b := range s.Breaks {
			bStart := d.Add(time.Duration(b.StartMin) * time.Minute)
			bEnd := d.Add(time.Duration(b.EndMin) * time.Minute)
			bIntFrom := maxTime(bStart, intFrom)
			bIntTo := minTime(bEnd, intTo)
			if bIntTo.After(bIntFrom) {
				workDur -= bIntTo.Sub(bIntFrom)
			}
		}

		if workDur > 0 {
			total += workDur
		}
	}

	return total
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
