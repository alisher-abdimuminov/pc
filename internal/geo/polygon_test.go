package geo

import (
	"math"
	"testing"
)

// A ~111 m x ~85 m rectangle near Samarkand (lat 39.65 -> 0.001 deg lat = 111.2 m).
var rect = []Point{{39.650, 66.950}, {39.651, 66.950}, {39.651, 66.951}, {39.650, 66.951}}

// metersEast returns a point `m` meters east of p.
func metersEast(p Point, m float64) Point {
	return Point{p[0], p[1] + m/(earthRadiusM*math.Cos(p[0]*math.Pi/180))*180/math.Pi}
}

func metersNorth(p Point, m float64) Point {
	return Point{p[0] + m/earthRadiusM*180/math.Pi, p[1]}
}

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestHaversine(t *testing.T) {
	if d := Haversine(Point{0, 0}, Point{0, 0}); d != 0 {
		t.Fatalf("zero distance, got %v", d)
	}
	// one degree of latitude is ~111.2 km
	if d := Haversine(Point{39, 66}, Point{40, 66}); !near(d, 111195, 100) {
		t.Fatalf("1 deg lat = %v m", d)
	}
	// Tashkent -> Samarkand is roughly 266 km
	if d := Haversine(Point{41.2995, 69.2401}, Point{39.6542, 66.9597}); d < 260e3 || d > 275e3 {
		t.Fatalf("Tashkent-Samarkand = %v m", d)
	}
}

func TestContains(t *testing.T) {
	in := Point{39.6505, 66.9505}
	cases := []struct {
		name string
		p    Point
		want bool
	}{
		{"center", in, true},
		{"just inside the corner", Point{39.65001, 66.95001}, true},
		{"north of the rectangle", Point{39.6515, 66.9505}, false},
		{"south", Point{39.6495, 66.9505}, false},
		{"east", Point{39.6505, 66.9515}, false},
		{"west", Point{39.6505, 66.9495}, false},
		{"far away", Point{41.3, 69.24}, false},
		{"on the south edge", Point{39.650, 66.9505}, true},
		{"on a vertex", Point{39.651, 66.951}, true},
	}
	for _, c := range cases {
		if got := Contains(rect, c.p); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
	if Contains(rect[:2], in) || Contains(nil, in) {
		t.Fatal("fewer than 3 points is never a polygon")
	}
}

func TestContainsConcavePolygon(t *testing.T) {
	// An "L" shape: the notch at the top right is outside.
	l := []Point{{0, 0}, {0, 2}, {1, 2}, {1, 1}, {2, 1}, {2, 0}}
	for p, want := range map[Point]bool{
		{0.5, 0.5}: true, {0.5, 1.5}: true, {1.5, 0.5}: true,
		{1.5, 1.5}: false, // inside the notch
		{2.5, 0.5}: false,
	} {
		if got := Contains(l, p); got != want {
			t.Errorf("%v: got %v want %v", p, got, want)
		}
	}
}

func TestDistanceToEdge(t *testing.T) {
	// 40 m east of the east edge, mid-height
	p := metersEast(Point{39.6505, 66.951}, 40)
	if d := DistanceToEdgeM(rect, p); !near(d, 40, 0.5) {
		t.Fatalf("east: %v", d)
	}
	// 25 m north of the north edge
	p = metersNorth(Point{39.651, 66.9505}, 25)
	if d := DistanceToEdgeM(rect, p); !near(d, 25, 0.5) {
		t.Fatalf("north: %v", d)
	}
	// diagonal off a corner: nearest point is the vertex, 3-4-5 style -> 50 m
	p = metersNorth(metersEast(Point{39.651, 66.951}, 30), 40)
	if d := DistanceToEdgeM(rect, p); !near(d, 50, 0.5) {
		t.Fatalf("corner: %v", d)
	}
	// inside points still report the distance to the nearest edge
	if d := DistanceToEdgeM(rect, Point{39.65005, 66.9505}); !near(d, 5.6, 0.5) {
		t.Fatalf("inside: %v", d)
	}
}

func TestCheckTolerance(t *testing.T) {
	east := Point{39.6505, 66.951}
	cases := []struct {
		name      string
		p         Point
		tolerance float64
		ok, in    bool
	}{
		{"inside, zero tolerance", Point{39.6505, 66.9505}, 0, true, true},
		{"20 m outside within 30 m tolerance", metersEast(east, 20), 30, true, false},
		{"29.5 m outside within 30 m tolerance", metersEast(east, 29.5), 30, true, false},
		{"31 m outside beyond 30 m tolerance", metersEast(east, 31), 30, false, false},
		{"5 m outside, zero tolerance", metersEast(east, 5), 0, false, false},
		{"far away", Point{41.3, 69.24}, 100, false, false},
	}
	for _, c := range cases {
		r := Check(rect, c.p, c.tolerance)
		if r.OK != c.ok || r.Inside != c.in {
			t.Errorf("%s: got %+v want ok=%v inside=%v", c.name, r, c.ok, c.in)
		}
	}
	if r := Check(rect, metersEast(east, 20), 30); !near(r.OutsideM, 20, 0.5) {
		t.Fatalf("OutsideM = %v", r.OutsideM)
	}
	if r := Check(rect[:2], Point{39.6505, 66.9505}, 100); r.OK {
		t.Fatal("degenerate polygon must never pass")
	}
}
