package service

import (
	"time"

	"pc/internal/attendance"
)

// Weekdays flags Monday..Saturday (index 0 = Monday).
type Weekdays = attendance.Days

// SchedulePeriod is the part of a schedule relevant for overlap detection.
type SchedulePeriod struct {
	Start, End time.Time // inclusive dates
	Days       Weekdays
}

// Overlaps reports whether two schedules of the same target collide: their date ranges
// intersect AND they share at least one weekday.
func Overlaps(a, b SchedulePeriod) bool {
	if a.Start.After(b.End) || b.Start.After(a.End) {
		return false
	}
	for i := range a.Days {
		if a.Days[i] && b.Days[i] {
			return true
		}
	}
	return false
}
