package attendance

import "time"

// Window is one confirmation step of a shift. Intervals are half-open: [Start, End).
// Start and End are minutes since midnight (Asia/Tashkent wall-clock).
type Window struct {
	Shift int `json:"shift"`
	Step  int `json:"step"`
	Start int `json:"-"`
	End   int `json:"-"`
}

func hm(h, m int) int { return h*60 + m }

// Table is THE shift/step schedule. It lives only here.
//
//	shift 1: 08:00-10:00, 10:00-12:00, 12:00-13:00
//	shift 2: 13:00-14:00, 14:00-16:00, 16:00-18:00
var Table = [...]Window{
	{Shift: 1, Step: 1, Start: hm(8, 0), End: hm(10, 0)},
	{Shift: 1, Step: 2, Start: hm(10, 0), End: hm(12, 0)},
	{Shift: 1, Step: 3, Start: hm(12, 0), End: hm(13, 0)},
	{Shift: 2, Step: 1, Start: hm(13, 0), End: hm(14, 0)},
	{Shift: 2, Step: 2, Start: hm(14, 0), End: hm(16, 0)},
	{Shift: 2, Step: 3, Start: hm(16, 0), End: hm(18, 0)},
}

// DayStart and DayEnd bound all windows.
const (
	DayStart = 8 * 60
	DayEnd   = 18 * 60
)

// minutesOfDay is the wall-clock minute of t in t's own location. Callers pass business-time values.
func minutesOfDay(t time.Time) int { return t.Hour()*60 + t.Minute() }

// WindowAt returns the window containing t (business time), or false outside 08:00-18:00.
func WindowAt(t time.Time) (Window, bool) {
	m := minutesOfDay(t)
	for _, w := range Table {
		if m >= w.Start && m < w.End {
			return w, true
		}
	}
	return Window{}, false
}

// ShiftWindows lists the three windows of a shift in step order.
func ShiftWindows(shift int) []Window {
	var out []Window
	for _, w := range Table {
		if w.Shift == shift {
			out = append(out, w)
		}
	}
	return out
}

// StepState is the state of one step of a shift at a given moment.
type StepState string

const (
	StepDone    StepState = "done"
	StepPending StepState = "pending" // current or upcoming and not yet confirmed
	StepMissed  StepState = "missed"  // window is over without a successful confirmation
)

// StateOf classifies a step: done wins; otherwise it is missed once its window has ended.
func StateOf(w Window, success bool, t time.Time) StepState {
	switch {
	case success:
		return StepDone
	case minutesOfDay(t) >= w.End:
		return StepMissed
	default:
		return StepPending
	}
}

// Contains reports whether t (business time) is inside the window.
func (w Window) Contains(t time.Time) bool {
	m := minutesOfDay(t)
	return m >= w.Start && m < w.End
}

// Clock formats minutes since midnight as HH:MM.
func Clock(min int) string {
	return time.Date(2000, 1, 1, min/60, min%60, 0, 0, time.UTC).Format("15:04")
}
