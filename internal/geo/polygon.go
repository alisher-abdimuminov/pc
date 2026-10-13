package geo

import "math"

const earthRadiusM = 6371008.8

// Haversine returns the great-circle distance in meters between two lat/lng points.
func Haversine(a, b Point) float64 {
	la1, la2 := a[0]*math.Pi/180, b[0]*math.Pi/180
	dla := (b[0] - a[0]) * math.Pi / 180
	dlo := (b[1] - a[1]) * math.Pi / 180
	h := math.Sin(dla/2)*math.Sin(dla/2) + math.Cos(la1)*math.Cos(la2)*math.Sin(dlo/2)*math.Sin(dlo/2)
	return 2 * earthRadiusM * math.Asin(math.Min(1, math.Sqrt(h)))
}

// Contains reports whether p lies inside the polygon (ray casting). Points exactly on an edge
// count as inside. Polygons are small (a building or yard), so plain lat/lng is used as the plane.
func Contains(poly []Point, p Point) bool {
	if len(poly) < 3 {
		return false
	}
	if onBoundary(poly, p) {
		return true
	}
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		yi, xi := poly[i][0], poly[i][1]
		yj, xj := poly[j][0], poly[j][1]
		if (yi > p[0]) != (yj > p[0]) && p[1] < (xj-xi)*(p[0]-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside
}

func onBoundary(poly []Point, p Point) bool {
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		if segmentDistanceM(poly[j], poly[i], p) < 1e-6 {
			return true
		}
	}
	return false
}

// local projects q into meters on a plane tangent at origin (equirectangular; exact enough at this scale).
func local(origin, q Point) (x, y float64) {
	x = (q[1] - origin[1]) * math.Pi / 180 * earthRadiusM * math.Cos(origin[0]*math.Pi/180)
	y = (q[0] - origin[0]) * math.Pi / 180 * earthRadiusM
	return
}

// segmentDistanceM is the distance in meters from p to the segment a-b.
func segmentDistanceM(a, b, p Point) float64 {
	ax, ay := local(p, a)
	bx, by := local(p, b)
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = math.Max(0, math.Min(1, (-ax*dx-ay*dy)/l2)) // projection of the origin (p) onto the segment
	}
	return math.Hypot(ax+t*dx, ay+t*dy)
}

// DistanceToEdgeM is the distance in meters from p to the nearest polygon edge,
// regardless of whether p is inside or outside.
func DistanceToEdgeM(poly []Point, p Point) float64 {
	best := math.Inf(1)
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		if d := segmentDistanceM(poly[j], poly[i], p); d < best {
			best = d
		}
	}
	return best
}

// Result of a location check.
type Result struct {
	Inside bool
	// OutsideM is how far outside the polygon the point is (0 when inside).
	OutsideM float64
	// OK is true when the point is inside or within toleranceM of the polygon.
	OK bool
}

// Check decides whether p is acceptable for a location polygon with the given tolerance in meters.
func Check(poly []Point, p Point, toleranceM float64) Result {
	if len(poly) < 3 {
		return Result{}
	}
	if Contains(poly, p) {
		return Result{Inside: true, OK: true}
	}
	d := DistanceToEdgeM(poly, p)
	return Result{OutsideM: d, OK: d <= toleranceM}
}
