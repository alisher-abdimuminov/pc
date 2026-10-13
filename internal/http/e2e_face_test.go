package http_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pc/internal/config"
)

// TestE2ERealFaceService runs the whole flow against the REAL face-service (real DeepFace models):
// enroll a student from a photo through the admin API, then confirm a step with a matching photo,
// a different person's photo and a photo-of-a-screen. It only runs with E2E_FACE=1 (make test-e2e)
// and needs the photos from face-service/tests/fetch_photos.sh.
func TestE2ERealFaceService(t *testing.T) {
	if os.Getenv("E2E_FACE") != "1" {
		t.Skip("set E2E_FACE=1 (make test-e2e) to run against the real face-service")
	}
	dir := os.Getenv("E2E_PHOTOS")
	if dir == "" {
		dir = "../../face-service/tests/data"
	}
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Skipf("photo %s missing (run face-service/tests/fetch_photos.sh): %v", name, err)
		}
		return b
	}
	enroll, same, other, replay := read("face_a1.jpg"), read("face_a2.jpg"), read("face_b.jpg"), read("replay.jpg")

	te := newTeacherEnv(t, func(c *config.Config) {
		c.FaceServiceURL, c.FaceServiceToken = os.Getenv("FACE_SERVICE_URL"), os.Getenv("FACE_SERVICE_TOKEN")
		c.FaceTimeout, c.MaxAttemptsPerStep, c.GPSMaxAccuracyM = 60*time.Second, 5, 100
	})
	if te.cfg.FaceServiceURL == "" {
		t.Skip("FACE_SERVICE_URL not set")
	}
	ae := &aEnv{tEnv: te, mock: nil, day: te.lastWorkday()}
	ctx := context.Background()
	var loc string
	te.pool.QueryRow(ctx, `INSERT INTO locations (name, polygon, tolerance_meters) VALUES ('Maktab','[[39.650,66.950],[39.651,66.950],[39.651,66.951],[39.650,66.951]]',30) RETURNING id`).Scan(&loc)
	te.pool.QueryRow(ctx, `INSERT INTO schedules (start_date, end_date, group_id, location_id, monday, tuesday, wednesday, thursday, friday, saturday)
		VALUES ($1,$1,$2,$3,true,true,true,true,true,true) RETURNING id`, ds(ae.day), te.groupA, loc).Scan(&ae.sched)
	ae.stu = te.mustLogin("alpha1")
	ae.at(9, 0)

	// without enrollment the attempt is refused outright
	if r := ae.attemptAs(ae.stu, inLat, inLng, goodAcc, same); r.Status != 409 || r.Code != "no_face_enrolled" {
		t.Fatalf("before enrollment: %+v", r)
	}

	// enroll through the admin API: the real /embed produces a 512-d Facenet512 vector
	u := te.putPhoto("/api/admin/users/"+te.studentA1+"/photo", te.admin, enroll, "image", 200)
	if u["face_enrolled"] != true {
		t.Fatalf("enroll: %v", u)
	}
	var dims int
	te.pool.QueryRow(ctx, `SELECT array_length(face_embedding,1) FROM users WHERE id=$1`, te.studentA1).Scan(&dims)
	if dims != 512 {
		t.Fatalf("embedding has %d dims, want 512", dims)
	}

	// 1) a different person is rejected on identity, but liveness passes
	r := ae.attemptAs(ae.stu, inLat, inLng, goodAcc, other)
	t.Logf("other person: %+v", r.Body)
	if r.Status != 200 || r.Body["success"] != false || r.Body["liveness_verified"] != true || r.Body["face_verified"] != false {
		t.Fatalf("other person must fail on identity: %+v", r)
	}

	// 2) a photo of a screen is rejected by anti-spoofing
	r = ae.attemptAs(ae.stu, inLat, inLng, goodAcc, replay)
	t.Logf("replay: %+v", r.Body)
	if r.Body["success"] != false || r.Body["liveness_verified"] != false {
		t.Fatalf("replayed photo must fail liveness: %+v", r)
	}

	// 3) the same person with a fresh photo succeeds, and the step is confirmed
	r = ae.attemptAs(ae.stu, inLat, inLng, goodAcc, same)
	t.Logf("same person: %+v", r.Body)
	if r.Status != 200 || r.Body["success"] != true || r.Body["steps_done"] != 1.0 {
		t.Fatalf("matching photo must succeed: %+v", r)
	}
	var dist, thr, score float64
	te.pool.QueryRow(ctx, `SELECT face_distance, face_threshold, antispoof_score FROM attendance_attempts WHERE success`).Scan(&dist, &thr, &score)
	if dist <= 0 || dist > thr || thr <= 0 {
		t.Fatalf("distance %v / threshold %v", dist, thr)
	}
	if ae.attempts() != 3 {
		t.Fatalf("three attempts expected, got %d", ae.attempts())
	}
}
