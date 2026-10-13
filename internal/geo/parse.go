// Package geo holds polygon parsing (and, in phase 5, point-in-polygon checks).
package geo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Point is [lat, lng].
type Point [2]float64

var ErrTooFewPoints = errors.New("a polygon needs at least 3 points")

// ParsePoint parses "39.667365,66.878348".
func ParsePoint(s string) (Point, error) {
	parts := strings.Split(strings.TrimSpace(s), ",")
	if len(parts) != 2 {
		return Point{}, fmt.Errorf("expected lat,lng")
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return Point{}, fmt.Errorf("lat and lng must be numbers")
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return Point{}, fmt.Errorf("coordinates out of range")
	}
	return Point{lat, lng}, nil
}

// PointError reports which input row was invalid.
type PointError struct {
	Index int
	Err   error
}

func (e *PointError) Error() string { return fmt.Sprintf("point %d: %v", e.Index+1, e.Err) }
func (e *PointError) Unwrap() error { return e.Err }

// ParsePoints parses a list of "lat,lng" strings; blank rows are rejected.
func ParsePoints(in []string) ([]Point, error) {
	pts := make([]Point, 0, len(in))
	for i, s := range in {
		p, err := ParsePoint(s)
		if err != nil {
			return nil, &PointError{Index: i, Err: err}
		}
		pts = append(pts, p)
	}
	if len(pts) < 3 {
		return nil, ErrTooFewPoints
	}
	return pts, nil
}
