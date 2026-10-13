package geo

import (
	"errors"
	"testing"
)

func TestParsePoint(t *testing.T) {
	p, err := ParsePoint(" 39.667365 , 66.878348 ")
	if err != nil || p != (Point{39.667365, 66.878348}) {
		t.Fatalf("got %v %v", p, err)
	}
	for _, bad := range []string{"", "39.6", "a,b", "1,2,3", "91,0", "0,181", "-91,0"} {
		if _, err := ParsePoint(bad); err == nil {
			t.Errorf("%q should fail", bad)
		}
	}
}

func TestParsePoints(t *testing.T) {
	pts, err := ParsePoints([]string{"39.1,66.1", "39.2,66.1", "39.2,66.2"})
	if err != nil || len(pts) != 3 {
		t.Fatalf("got %v %v", pts, err)
	}
	if _, err := ParsePoints([]string{"39.1,66.1", "39.2,66.1"}); !errors.Is(err, ErrTooFewPoints) {
		t.Fatalf("want ErrTooFewPoints, got %v", err)
	}
	_, err = ParsePoints([]string{"39.1,66.1", "oops", "39.2,66.2"})
	var pe *PointError
	if !errors.As(err, &pe) || pe.Index != 1 {
		t.Fatalf("want PointError at index 1, got %v", err)
	}
}
