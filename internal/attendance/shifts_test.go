package attendance

import (
	"testing"
	"time"
)

func at(h, m int) time.Time {
	return time.Date(2026, 9, 14, h, m, 0, 0, time.FixedZone("UTC+5", 5*3600))
}

func TestWindowAtBoundaries(t *testing.T) {
	cases := []struct {
		name        string
		h, m        int
		shift, step int
		ok          bool
	}{
		{"before the day 07:59", 7, 59, 0, 0, false},
		{"08:00 starts shift 1 step 1", 8, 0, 1, 1, true},
		{"09:59 last minute of 1/1", 9, 59, 1, 1, true},
		{"10:00 starts 1/2", 10, 0, 1, 2, true},
		{"11:59 last minute of 1/2", 11, 59, 1, 2, true},
		{"12:00 starts 1/3", 12, 0, 1, 3, true},
		{"12:59 last minute of 1/3", 12, 59, 1, 3, true},
		{"13:00 starts shift 2 step 1", 13, 0, 2, 1, true},
		{"13:59 last minute of 2/1", 13, 59, 2, 1, true},
		{"14:00 starts 2/2", 14, 0, 2, 2, true},
		{"15:59 last minute of 2/2", 15, 59, 2, 2, true},
		{"16:00 starts 2/3", 16, 0, 2, 3, true},
		{"17:59 last minute of 2/3", 17, 59, 2, 3, true},
		{"18:00 is closed", 18, 0, 0, 0, false},
		{"midnight", 0, 0, 0, 0, false},
		{"23:59", 23, 59, 0, 0, false},
	}
	for _, c := range cases {
		w, ok := WindowAt(at(c.h, c.m))
		if ok != c.ok || w.Shift != c.shift || w.Step != c.step {
			t.Errorf("%s: got shift=%d step=%d ok=%v, want %d/%d ok=%v", c.name, w.Shift, w.Step, ok, c.shift, c.step, c.ok)
		}
	}
}

func TestWindowAtIgnoresSecondsAndUsesWallClock(t *testing.T) {
	// 09:59:59 is still step 1; the wall-clock of the passed location is what counts.
	if w, ok := WindowAt(time.Date(2026, 9, 14, 9, 59, 59, 0, time.FixedZone("x", 5*3600))); !ok || w.Step != 1 {
		t.Fatal("09:59:59 must be 1/1")
	}
}

func TestTableIsContiguousAndComplete(t *testing.T) {
	if len(Table) != 6 {
		t.Fatalf("want 6 windows, got %d", len(Table))
	}
	if Table[0].Start != DayStart || Table[len(Table)-1].End != DayEnd {
		t.Fatal("table must span 08:00-18:00")
	}
	for i := 1; i < len(Table); i++ {
		if Table[i].Start != Table[i-1].End {
			t.Errorf("gap or overlap between window %d and %d", i-1, i)
		}
	}
	for shift := 1; shift <= 2; shift++ {
		ws := ShiftWindows(shift)
		if len(ws) != 3 {
			t.Fatalf("shift %d has %d steps", shift, len(ws))
		}
		for i, w := range ws {
			if w.Step != i+1 {
				t.Errorf("shift %d step order", shift)
			}
		}
	}
}

func TestStateOf(t *testing.T) {
	w := ShiftWindows(1)[0] // 08:00-10:00
	cases := []struct {
		name    string
		success bool
		h, m    int
		want    StepState
	}{
		{"done early", true, 8, 30, StepDone},
		{"done even after the window", true, 15, 0, StepDone},
		{"pending before the window", false, 7, 0, StepPending},
		{"pending inside the window", false, 9, 59, StepPending},
		{"missed at the end boundary", false, 10, 0, StepMissed},
		{"missed later", false, 17, 0, StepMissed},
	}
	for _, c := range cases {
		if got := StateOf(w, c.success, at(c.h, c.m)); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestClock(t *testing.T) {
	if Clock(8*60) != "08:00" || Clock(17*60+5) != "17:05" {
		t.Fatal("Clock")
	}
	if !ShiftWindows(2)[1].Contains(at(14, 0)) || ShiftWindows(2)[1].Contains(at(16, 0)) {
		t.Fatal("Contains is half-open")
	}
}
