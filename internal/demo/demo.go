// Package demo fills a development database with realistic attendance attempts.
// Everything it creates is tagged with the user agent "seed-demo" so it can be recognised and reset.
package demo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/attendance"
	"pc/internal/db"
	"pc/internal/service"
	"pc/internal/storage"
)

const (
	Marker       = "seed-demo"
	locationName = "Namuna maktab (seed)"
	scheduleName = "Namuna jadval (seed)"
)

type Options struct {
	Days  int  // how many past days (including Sundays, which are skipped) to generate
	Reset bool // remove earlier demo data first
	Now   time.Time
	Loc   *time.Location
}

type Summary struct {
	Skipped     bool
	Attendances int
	Attempts    int
}

// kinds of attempt outcomes.
type kind int

const (
	kOK kind = iota
	kLocation
	kAccuracy
	kFace
	kLiveness
	kNoFace
)

// Rectangle used for the demo location (about 110 m x 85 m) and a point inside / far away.
var (
	polygon = `[[39.650,66.950],[39.651,66.950],[39.651,66.951],[39.650,66.951]]`
	inside  = [2]float64{39.6505, 66.9505}
	far     = [2]float64{41.30, 69.24}
)

// Run generates demo attempts for the seeded students student1..student3.
func Run(ctx context.Context, pool *pgxpool.Pool, store storage.Storage, opt Options) (Summary, error) {
	if opt.Days <= 0 {
		opt.Days = 10
	}
	if opt.Loc == nil {
		opt.Loc = time.UTC
	}
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}
	q := db.New(pool)

	var students []db.User
	for _, name := range []string{"student1", "student2", "student3"} {
		u, err := q.GetUserByUsername(ctx, pgtype.Text{String: name, Valid: true})
		if err != nil {
			return Summary{}, fmt.Errorf("student %q not found: run `make seed` first", name)
		}
		students = append(students, u)
	}
	if !students[0].GroupID.Valid {
		return Summary{}, errors.New("the seeded students have no group: run `make seed` first")
	}
	groupID := students[0].GroupID.UUID

	existing, err := q.CountDemoAttempts(ctx)
	if err != nil {
		return Summary{}, err
	}
	if existing > 0 {
		if !opt.Reset {
			return Summary{Skipped: true}, nil
		}
		if err := reset(ctx, q, store); err != nil {
			return Summary{}, err
		}
	}

	n := opt.Now.In(opt.Loc)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)

	locationID, err := ensureLocation(ctx, pool)
	if err != nil {
		return Summary{}, err
	}
	sched, err := ensureSchedule(ctx, q, groupID, locationID, today, opt.Days)
	if err != nil {
		return Summary{}, err
	}

	rng := rand.New(rand.NewSource(42))
	var sum Summary
	for dayIdx := 1; dayIdx <= opt.Days; dayIdx++ {
		day := today.AddDate(0, 0, -dayIdx)
		if day.Weekday() == time.Sunday || day.Before(sched.StartDate.Time) || day.After(sched.EndDate.Time) {
			continue
		}
		for si, stu := range students {
			plan := planFor(si, dayIdx)
			if len(plan) == 0 {
				continue // absent that day
			}
			shift := 1 + dayIdx%2
			err := service.InTx(ctx, pool, func(tx pgx.Tx, q *db.Queries) error {
				pgDay := pgtype.Date{Time: day, Valid: true}
				if err := q.EnsureAttendance(ctx, db.EnsureAttendanceParams{StudentID: stu.ID, ScheduleID: sched.ID, Shift: int16(shift), Date: pgDay}); err != nil {
					return err
				}
				att, err := q.LockAttendance(ctx, db.LockAttendanceParams{StudentID: stu.ID, ScheduleID: sched.ID, Date: pgDay})
				if err != nil {
					return err
				}
				sum.Attendances++
				for _, sp := range plan {
					win := attendance.ShiftWindows(shift)[sp.step-1]
					if err := q.EnsureStep(ctx, db.EnsureStepParams{AttendanceID: att.ID, Step: int16(sp.step)}); err != nil {
						return err
					}
					st, err := q.GetStep(ctx, db.GetStepParams{AttendanceID: att.ID, Step: int16(sp.step)})
					if err != nil {
						return err
					}
					when := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, opt.Loc).Add(time.Duration(win.Start+3) * time.Minute)
					for _, k := range sp.kinds {
						when = when.Add(time.Duration(2+rng.Intn(9)) * time.Minute)
						if err := insertAttempt(ctx, q, store, st.ID, locationID, k, when, rng, si+dayIdx); err != nil {
							return err
						}
						sum.Attempts++
						if k == kOK {
							if err := q.MarkStepSuccess(ctx, st.ID); err != nil {
								return err
							}
						}
					}
				}
				return nil
			})
			if err != nil {
				return sum, err
			}
		}
	}
	return sum, nil
}

type stepPlan struct {
	step  int
	kinds []kind
}

// planFor decides what a student did on a day: nil = no attendance (absent).
func planFor(student, dayIdx int) []stepPlan {
	switch student {
	case 0: // reliable: every step done, sometimes after a wrong-location try
		first := []kind{kOK}
		if dayIdx%3 == 0 {
			first = []kind{kLocation, kOK}
		}
		return []stepPlan{{1, first}, {2, []kind{kOK}}, {3, []kind{kOK}}}
	case 1: // partial: third step keeps failing on the face
		first := []kind{kOK}
		if dayIdx%4 == 1 {
			first = []kind{kLiveness, kOK}
		}
		return []stepPlan{{1, first}, {2, []kind{kOK}}, {3, []kind{kFace, kFace}}}
	default: // flaky: absent every third day, otherwise needs several tries
		if dayIdx%3 == 0 {
			return nil
		}
		plan := []stepPlan{{1, []kind{kNoFace, kOK}}, {2, []kind{kAccuracy, kOK}}}
		if dayIdx%2 == 1 {
			plan = append(plan, stepPlan{3, []kind{kOK}})
		}
		return plan
	}
}

func insertAttempt(ctx context.Context, q *db.Queries, store storage.Storage, stepID, locationID uuid.UUID, k kind, at time.Time, rng *rand.Rand, hue int) error {
	lat, lng := inside[0]+rng.Float64()*0.0003-0.00015, inside[1]+rng.Float64()*0.0003-0.00015
	acc := 6 + rng.Float64()*14
	var faceOK, locOK, liveOK = true, true, true
	var dist, score *float64
	f := func(v float64) *float64 { return &v }
	threshold := 0.3
	dist, score = f(0.1+rng.Float64()*0.12), f(0.96+rng.Float64()*0.039)
	var codes []string
	switch k {
	case kLocation:
		lat, lng, locOK = far[0], far[1], false
		codes = []string{service.FailLocation}
	case kAccuracy:
		acc, locOK = 150+rng.Float64()*80, false
		codes = []string{service.FailLocationAccuracy}
	case kFace:
		faceOK, dist = false, f(0.55+rng.Float64()*0.4)
		codes = []string{service.FailFace}
	case kLiveness:
		faceOK, liveOK, dist, score = false, false, nil, f(0.9+rng.Float64()*0.09)
		codes = []string{service.FailLiveness}
	case kNoFace:
		faceOK, liveOK, dist, score = false, false, nil, nil
		codes = []string{service.FailNoFace}
	}
	success := faceOK && locOK && liveOK

	path, err := store.Save(ctx, storage.KindSelfies, "png", bytes.NewReader(placeholderPNG(hue)))
	if err != nil {
		return err
	}
	ip := netip.MustParseAddr("127.0.0.1")
	_, err = q.CreateSeedAttempt(ctx, db.CreateSeedAttemptParams{
		StepID: stepID, Image: path, IpAddress: &ip, UserAgent: Marker, Latitude: lat, Longitude: lng, GpsAccuracy: acc,
		LocationID: uuid.NullUUID{UUID: locationID, Valid: true}, FaceVerified: faceOK, LocationVerified: locOK,
		LivenessVerified: liveOK, Success: success, Error: service.FailureSummary(codes),
		FaceDistance: pgFloat(dist), FaceThreshold: pgFloat(&threshold), AntispoofScore: pgFloat(score),
		AttemptedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return err
}

func pgFloat(v *float64) pgtype.Float8 {
	if v == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *v, Valid: true}
}

// placeholderPNG draws a simple head-and-shoulders silhouette; the colour varies with seed.
func placeholderPNG(seed int) []byte {
	const w, h = 240, 320
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := color.RGBA{uint8(200 + (seed*13)%40), uint8(215 + (seed*7)%30), uint8(230 + (seed*3)%20), 255}
	fg := color.RGBA{uint8(90 + (seed*11)%50), uint8(100 + (seed*5)%50), uint8(120 + (seed*9)%50), 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := bg
			dx, dy := float64(x-w/2), float64(y-120)
			if dx*dx/(55*55)+dy*dy/(70*70) <= 1 { // head
				c = fg
			}
			sx, sy := float64(x-w/2), float64(y-330)
			if sx*sx/(110*110)+sy*sy/(140*140) <= 1 { // shoulders
				c = fg
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func reset(ctx context.Context, q *db.Queries, store storage.Storage) error {
	images, err := q.ListDemoAttemptImages(ctx)
	if err != nil {
		return err
	}
	if _, err := q.DeleteDemoAttendances(ctx); err != nil { // cascades to steps and attempts
		return err
	}
	for _, p := range images {
		_ = store.Delete(ctx, p)
	}
	return nil
}

func ensureLocation(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM locations WHERE name = $1`, locationName).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx, `INSERT INTO locations (name, polygon, tolerance_meters, location_url)
			VALUES ($1, $2::jsonb, 30, 'https://www.openstreetmap.org/?mlat=39.6505&mlon=66.9505#map=18/39.6505/66.9505') RETURNING id`,
			locationName, polygon).Scan(&id)
	}
	return id, err
}

func ensureSchedule(ctx context.Context, q *db.Queries, groupID, locationID uuid.UUID, today time.Time, days int) (db.Schedule, error) {
	var sched db.Schedule
	rows, err := q.ListSchedules(ctx, db.ListSchedulesParams{GroupID: uuid.NullUUID{UUID: groupID, Valid: true}, Search: scheduleName, Lim: 1})
	if err != nil {
		return sched, err
	}
	need := today.AddDate(0, 0, -days-3)
	if len(rows) > 0 && !rows[0].StartDate.Time.After(need) && !rows[0].EndDate.Time.Before(today.AddDate(0, 0, 14)) {
		return q.GetSchedule(ctx, rows[0].ID)
	}
	if len(rows) > 0 { // an older demo schedule that is too short: replace it
		if err := q.DeleteSchedule(ctx, rows[0].ID); err != nil {
			return sched, err
		}
	}
	return q.CreateSchedule(ctx, db.CreateScheduleParams{
		Name: scheduleName, StartDate: pgtype.Date{Time: need, Valid: true}, EndDate: pgtype.Date{Time: today.AddDate(0, 0, 14), Valid: true},
		GroupID: uuid.NullUUID{UUID: groupID, Valid: true}, LocationID: uuid.NullUUID{UUID: locationID, Valid: true},
		Monday: true, Tuesday: true, Wednesday: true, Thursday: true, Friday: true, Saturday: true,
	})
}
