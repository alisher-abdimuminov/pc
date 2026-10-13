package attendance

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestStatusFromSteps(t *testing.T) {
	for done, want := range map[int]Status{0: StatusAbsent, 1: StatusPartial, 2: StatusPartial, 3: StatusFull} {
		if got := StatusFromSteps(done); got != want {
			t.Errorf("steps=%d got %q want %q", done, got, want)
		}
	}
}

func TestExpectedWeekdaysAndBoundaries(t *testing.T) {
	// 2026-09-01 is a Tuesday. Schedule Mon/Wed/Sat from Tue 09-01 through Sat 09-12 (inclusive).
	s := []Schedule{{Start: date("2026-09-01"), End: date("2026-09-12"), Days: Days{true, false, true, false, false, true}}}
	cases := map[string]bool{
		"2026-08-31": false, // Monday, the day before the range
		"2026-09-01": false, // first day, Tuesday (not enabled)
		"2026-09-02": true,  // Wednesday
		"2026-09-05": true,  // Saturday
		"2026-09-06": false, // Sunday never
		"2026-09-07": true,  // Monday
		"2026-09-12": true,  // last day, Saturday: inclusive upper bound
		"2026-09-14": false, // Monday after the range
	}
	for d, want := range cases {
		if got := Expected(date(d), nil, s); got != want {
			t.Errorf("%s: got %v want %v", d, got, want)
		}
	}
}

func TestExpectedSundayNeverEvenIfFlagged(t *testing.T) {
	all := Days{true, true, true, true, true, true}
	s := []Schedule{{Start: date("2026-09-01"), End: date("2026-09-30"), Days: all}}
	if Expected(date("2026-09-06"), nil, s) { // Sunday
		t.Fatal("Sunday must never be expected")
	}
}

func TestPersonalScheduleOverridesGroup(t *testing.T) {
	mon := Days{true}
	tue := Days{false, true}
	group := []Schedule{{Start: date("2026-09-01"), End: date("2026-09-30"), Days: mon}}
	personal := []Schedule{{Start: date("2026-09-14"), End: date("2026-09-20"), Days: tue}}

	// Monday 09-14 is inside the personal range, which only has Tuesday: the group's Monday is overridden.
	if Expected(date("2026-09-14"), personal, group) {
		t.Fatal("personal schedule must win over the group's on covered dates")
	}
	if !Expected(date("2026-09-15"), personal, group) { // Tuesday
		t.Fatal("personal Tuesday expected")
	}
	// Outside the personal range the group schedule applies again.
	if !Expected(date("2026-09-07"), personal, group) || !Expected(date("2026-09-21"), personal, group) {
		t.Fatal("group Mondays outside the personal range expected")
	}
}

func TestDateRangeAndTotals(t *testing.T) {
	r := DateRange(date("2026-09-01"), date("2026-09-03"))
	if len(r) != 3 || !r[2].Equal(date("2026-09-03")) {
		t.Fatalf("range: %v", r)
	}
	if len(DateRange(date("2026-09-03"), date("2026-09-01"))) != 0 {
		t.Fatal("reversed range must be empty")
	}

	var tot Totals
	for _, s := range []Status{StatusFull, StatusFull, StatusPartial, StatusAbsent, StatusNone} {
		tot.Add(s)
	}
	tot.Finish()
	if tot.Expected != 4 || tot.Full != 2 || tot.Partial != 1 || tot.Absent != 1 || tot.Percent != 50 {
		t.Fatalf("totals: %+v", tot)
	}
	var empty Totals
	empty.Finish()
	if empty.Percent != 0 {
		t.Fatal("empty percent")
	}
}
