package service

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestOverlaps(t *testing.T) {
	monWed := Weekdays{true, false, true}
	tueThu := Weekdays{false, true, false, true}
	base := SchedulePeriod{Start: d("2026-09-01"), End: d("2026-09-30"), Days: monWed}

	cases := []struct {
		name string
		b    SchedulePeriod
		want bool
	}{
		{"same days, same range", SchedulePeriod{d("2026-09-01"), d("2026-09-30"), monWed}, true},
		{"disjoint weekdays", SchedulePeriod{d("2026-09-01"), d("2026-09-30"), tueThu}, false},
		{"share one weekday", SchedulePeriod{d("2026-09-10"), d("2026-10-10"), Weekdays{true}}, true},
		{"touching on last day (inclusive)", SchedulePeriod{d("2026-09-30"), d("2026-10-31"), monWed}, true},
		{"starts day after end", SchedulePeriod{d("2026-10-01"), d("2026-10-31"), monWed}, false},
		{"ends day before start", SchedulePeriod{d("2026-08-01"), d("2026-08-31"), monWed}, false},
		{"contained", SchedulePeriod{d("2026-09-10"), d("2026-09-12"), monWed}, true},
		{"no weekdays at all", SchedulePeriod{d("2026-09-01"), d("2026-09-30"), Weekdays{}}, false},
	}
	for _, c := range cases {
		if got := Overlaps(base, c.b); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
		if got := Overlaps(c.b, base); got != c.want {
			t.Errorf("%s (swapped): got %v want %v", c.name, got, c.want)
		}
	}
}

func TestWeekdaysHas(t *testing.T) {
	w := Weekdays{true, false, false, false, false, true}
	if !w.Has(time.Monday) || w.Has(time.Tuesday) || !w.Has(time.Saturday) || w.Has(time.Sunday) {
		t.Fatal("Has misbehaves")
	}
}
