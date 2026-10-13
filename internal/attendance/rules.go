// Package attendance holds the attendance business rules.
package attendance

import "time"

// Days flags Monday..Saturday (index 0 = Monday). Sunday is never a working day.
type Days [6]bool

func (d Days) Any() bool {
	for _, v := range d {
		if v {
			return true
		}
	}
	return false
}

func (d Days) Has(w time.Weekday) bool {
	if w == time.Sunday {
		return false
	}
	return d[int(w)-1]
}

type Status string

const (
	StatusFull    Status = "full"
	StatusPartial Status = "partial"
	StatusAbsent  Status = "absent"
	StatusNone    Status = "" // not a scheduled day (or still in the future)
)

// StatusFromSteps maps the number of successful steps to the daily status:
// 3/3 = full, 1-2 = partial, 0 = absent.
func StatusFromSteps(done int) Status {
	switch {
	case done >= 3:
		return StatusFull
	case done >= 1:
		return StatusPartial
	default:
		return StatusAbsent
	}
}

// Schedule is the part of a schedule needed to decide whether a day is expected.
// Start and End are inclusive calendar dates.
type Schedule struct {
	Start, End time.Time
	Days       Days
}

func (s Schedule) covers(date time.Time) bool {
	return !date.Before(s.Start) && !date.After(s.End)
}

// Expected reports whether the student must attend on date. The student's schedule for a
// date is a personal one first, otherwise the group's; it counts when it covers the date
// and has the weekday enabled.
func Expected(date time.Time, personal, group []Schedule) bool {
	for _, s := range personal {
		if s.covers(date) {
			return s.Days.Has(date.Weekday())
		}
	}
	for _, s := range group {
		if s.covers(date) {
			return s.Days.Has(date.Weekday())
		}
	}
	return false
}

// DateRange lists every calendar date in [from, to].
func DateRange(from, to time.Time) []time.Time {
	var out []time.Time
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		out = append(out, d)
	}
	return out
}

// Totals summarises one student's period. Only days up to today count as "expected".
type Totals struct {
	Expected int `json:"expected"`
	Full     int `json:"full"`
	Partial  int `json:"partial"`
	Absent   int `json:"absent"`
	// Percent is the share of expected days attended in full (0-100, rounded).
	Percent int `json:"percent"`
}

func (t *Totals) Add(s Status) {
	switch s {
	case StatusFull:
		t.Full++
	case StatusPartial:
		t.Partial++
	case StatusAbsent:
		t.Absent++
	default:
		return
	}
	t.Expected++
}

func (t *Totals) Finish() {
	if t.Expected > 0 {
		t.Percent = (t.Full*100 + t.Expected/2) / t.Expected
	}
}
