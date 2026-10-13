package demo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/service"
	"pc/internal/storage"
	"pc/internal/testutil"
)

var runner *pgxpool.Pool

var loc = time.FixedZone("UTC+5", 5*3600)

func setup(t *testing.T) (context.Context, func(sql string, args ...any) int, storage.Storage, string, *service.AttemptsAdminService, *service.ReportService) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	dir := t.TempDir()
	var gid string
	pool.QueryRow(ctx, `INSERT INTO groups (name) VALUES ('TEST-101') RETURNING id`).Scan(&gid)
	for _, u := range []string{"student1", "student2", "student3"} {
		pool.Exec(ctx, `INSERT INTO users (username, role, full_name, group_id) VALUES ($1,'student',$1,$2)`, u, gid)
	}
	count := func(sql string, args ...any) int {
		var n int
		if err := pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// keep the pool alive for the whole test through the closure above
	t.Cleanup(pool.Close)
	runner = pool
	return ctx, count, storage.NewLocal(dir), dir, service.NewAttemptsAdminService(pool, loc), service.NewReportService(pool, loc)
}

func files(dir string) int {
	n := 0
	filepath.Walk(filepath.Join(dir, "selfies"), func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func TestRunCreatesVariedAttemptsForTheSeededStudents(t *testing.T) {
	ctx, count, store, dir, admin, reports := setup(t)
	now := time.Now()
	sum, err := Run(ctx, runner, store, Options{Days: 10, Now: now, Loc: loc})
	if err != nil || sum.Skipped || sum.Attempts == 0 || sum.Attendances == 0 {
		t.Fatalf("%+v %v", sum, err)
	}
	if got := count(`SELECT count(*) FROM attendance_attempts WHERE user_agent='seed-demo'`); got != sum.Attempts {
		t.Fatalf("db has %d, summary says %d", got, sum.Attempts)
	}
	if files(dir) != sum.Attempts {
		t.Fatalf("%d selfie files for %d attempts", files(dir), sum.Attempts)
	}

	// every outcome the admin page can filter on is represented
	list := func(f service.AttemptFilter) int {
		p, err := admin.List(ctx, service.ListParams{PageSize: 100}, f)
		if err != nil {
			t.Fatal(err)
		}
		return int(p.Total)
	}
	yes, no := true, false
	for name, f := range map[string]service.AttemptFilter{
		"success": {Success: &yes}, "failed": {Success: &no}, "location": {Kind: "location"}, "face": {Kind: "face"},
		"liveness": {Kind: "liveness"}, "no_face": {Kind: "no_face"},
	} {
		if list(f) == 0 {
			t.Errorf("no %s attempts in the demo data", name)
		}
	}
	if list(service.AttemptFilter{Success: &yes})+list(service.AttemptFilter{Success: &no}) != sum.Attempts {
		t.Fatal("success + failed must add up")
	}

	// consistency of the stored rows
	if n := count(`SELECT count(*) FROM attendance_attempts WHERE user_agent='seed-demo' AND success AND NOT (face_verified AND location_verified AND liveness_verified)`); n != 0 {
		t.Fatalf("%d successful attempts with a failed check", n)
	}
	if n := count(`SELECT count(*) FROM attendance_attempts WHERE user_agent='seed-demo' AND NOT success AND error = ''`); n != 0 {
		t.Fatalf("%d failed attempts without an error text", n)
	}
	if n := count(`SELECT count(*) FROM attendance_attempts WHERE user_agent='seed-demo' AND face_verified AND NOT liveness_verified`); n != 0 {
		t.Fatalf("%d attempts verified a face without liveness", n)
	}
	if n := count(`SELECT count(*) FROM (SELECT step_id FROM attendance_attempts GROUP BY step_id HAVING count(*) > 5) x`); n != 0 {
		t.Fatal("no step may exceed the 5-attempt limit")
	}
	if n := count(`SELECT count(*) FROM attendance_steps s WHERE success <> EXISTS (SELECT 1 FROM attendance_attempts t WHERE t.step_id=s.id AND t.success)`); n != 0 {
		t.Fatalf("%d steps whose success flag disagrees with their attempts", n)
	}

	// attempts sit inside their shift's windows, on working days in the past
	rows, _ := runner.Query(ctx, `SELECT (t.attempted_at AT TIME ZONE 'Asia/Tashkent')::time, a.date, a.shift, st.step
		FROM attendance_attempts t JOIN attendance_steps st ON st.id=t.step_id JOIN attendances a ON a.id=st.attendance_id`)
	defer rows.Close()
	today := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), 0, 0, 0, 0, time.UTC)
	for rows.Next() {
		var tm time.Time
		var date time.Time
		var shift, step int
		if err := rows.Scan(&tm, &date, &shift, &step); err != nil {
			t.Fatal(err)
		}
		if !date.Before(today) || date.Weekday() == time.Sunday {
			t.Errorf("attempt on %s (today %s)", date.Format("2006-01-02 Mon"), today.Format("2006-01-02"))
		}
		min := tm.Hour()*60 + tm.Minute()
		var start, end int
		switch {
		case shift == 1 && step == 1:
			start, end = 8*60, 10*60
		case shift == 1 && step == 2:
			start, end = 10*60, 12*60
		case shift == 1 && step == 3:
			start, end = 12*60, 13*60
		case shift == 2 && step == 1:
			start, end = 13*60, 14*60
		case shift == 2 && step == 2:
			start, end = 14*60, 16*60
		default:
			start, end = 16*60, 18*60
		}
		if min < start || min >= end {
			t.Errorf("shift %d step %d attempt at %s is outside %d-%d", shift, step, tm.Format("15:04"), start, end)
		}
	}

	// the three students have the intended profiles in the attendance report
	rep, err := reports.Build(ctx, nil, nil, today.AddDate(0, 0, -10), today.AddDate(0, 0, -1), today)
	if err != nil {
		t.Fatal(err)
	}
	totals := map[string]struct{ full, partial, absent int }{}
	for _, s := range rep.Students {
		totals[s.FullName] = struct{ full, partial, absent int }{s.Totals.Full, s.Totals.Partial, s.Totals.Absent}
	}
	if s1 := totals["student1"]; s1.full == 0 || s1.partial != 0 {
		t.Errorf("student1 should be reliable: %+v", s1)
	}
	if s2 := totals["student2"]; s2.partial == 0 || s2.full != 0 {
		t.Errorf("student2 should be partial: %+v", s2)
	}
	if s3 := totals["student3"]; s3.absent == 0 || s3.full+s3.partial == 0 {
		t.Errorf("student3 should be flaky: %+v", s3)
	}
}

func TestRunIsIdempotentAndResettable(t *testing.T) {
	ctx, count, store, dir, _, _ := setup(t)
	first, err := Run(ctx, runner, store, Options{Days: 8, Loc: loc})
	if err != nil || first.Attempts == 0 {
		t.Fatalf("%+v %v", first, err)
	}
	again, err := Run(ctx, runner, store, Options{Days: 8, Loc: loc})
	if err != nil || !again.Skipped {
		t.Fatalf("a second run must be skipped: %+v %v", again, err)
	}
	if count(`SELECT count(*) FROM attendance_attempts`) != first.Attempts {
		t.Fatal("a skipped run must not add rows")
	}

	reset, err := Run(ctx, runner, store, Options{Days: 8, Loc: loc, Reset: true})
	if err != nil || reset.Skipped || reset.Attempts != first.Attempts || reset.Attendances != first.Attendances {
		t.Fatalf("reset run: %+v vs %+v (%v)", reset, first, err)
	}
	if count(`SELECT count(*) FROM attendance_attempts`) != first.Attempts || files(dir) != first.Attempts {
		t.Fatalf("after reset: %d rows, %d files, want %d", count(`SELECT count(*) FROM attendance_attempts`), files(dir), first.Attempts)
	}
	if count(`SELECT count(*) FROM locations WHERE name='Namuna maktab (seed)'`) != 1 || count(`SELECT count(*) FROM schedules WHERE name='Namuna jadval (seed)'`) != 1 {
		t.Fatal("the location and schedule must be reused, not duplicated")
	}
}

func TestRunNeedsTheSeededStudents(t *testing.T) {
	pool := testutil.Pool(t)
	if _, err := Run(context.Background(), pool, storage.NewLocal(t.TempDir()), Options{Loc: loc}); err == nil {
		t.Fatal("expected an error without students")
	}
}

func TestPlaceholderIsAValidPNG(t *testing.T) {
	b := placeholderPNG(3)
	if len(b) < 100 || string(b[1:4]) != "PNG" {
		t.Fatal("not a PNG")
	}
	if ext, err := storage.Validate(storage.KindSelfies, "x.png", b[:512]); err != nil || ext != "png" {
		t.Fatalf("storage must accept it: %v %q", err, ext)
	}
}
