package domain

import "time"

// Window is a half-open time interval [From, To).
type Window struct {
	From time.Time
	To   time.Time
}

type Break struct {
	StartMin int    // minutes from midnight
	EndMin   int    // minutes from midnight
	Type     string // rest | meal | cleaning
}

type Shift struct {
	ID        string
	Name      string
	MachineID string         // empty/"*" = applies to all machines
	StartMin  int            // minutes from midnight
	EndMin    int            // minutes from midnight; may exceed 1440 for overnight shifts
	Weekdays  []time.Weekday // days the shift applies; empty = every day
	Breaks    []Break
	TZ        string // IANA timezone; empty defaults to UTC
}

func (s Shift) PlannedProductionTime(w Window) time.Duration {
	if !w.To.After(w.From) {
		return 0
	}

	loc := s.location()
	weekdays := s.weekdaySet()
	everyDay := len(s.Weekdays) == 0

	// Start a day early so overnight shifts that began on the previous day are counted.
	firstDay := startOfDay(w.From.In(loc)).AddDate(0, 0, -1)
	lastDay := startOfDay(w.To.In(loc)).AddDate(0, 0, 1)

	var total time.Duration
	for day := firstDay; !day.After(lastDay); day = day.AddDate(0, 0, 1) {
		if everyDay || weekdays[day.Weekday()] {
			total += s.plannedTimeOnDay(day, w)
		}
	}
	return total
}

func (s Shift) plannedTimeOnDay(day time.Time, w Window) time.Duration {
	shiftStart := day.Add(time.Duration(s.StartMin) * time.Minute)
	shiftEnd := day.Add(time.Duration(s.EndMin) * time.Minute)

	workedFrom := maxTime(shiftStart, w.From)
	workedTo := minTime(shiftEnd, w.To)
	if !workedTo.After(workedFrom) {
		return 0
	}

	worked := workedTo.Sub(workedFrom)
	for _, b := range s.Breaks {
		breakStart := day.Add(time.Duration(b.StartMin) * time.Minute)
		breakEnd := day.Add(time.Duration(b.EndMin) * time.Minute)
		worked -= overlap(breakStart, breakEnd, workedFrom, workedTo)
	}

	if worked < 0 {
		return 0
	}
	return worked
}

func (s Shift) location() *time.Location {
	if s.TZ == "" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(s.TZ); err == nil {
		return loc
	}
	return time.UTC
}

func (s Shift) weekdaySet() map[time.Weekday]bool {
	set := make(map[time.Weekday]bool, len(s.Weekdays))
	for _, wd := range s.Weekdays {
		set[wd] = true
	}
	return set
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func overlap(aStart, aEnd, bStart, bEnd time.Time) time.Duration {
	start := maxTime(aStart, bStart)
	end := minTime(aEnd, bEnd)
	if end.After(start) {
		return end.Sub(start)
	}
	return 0
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
