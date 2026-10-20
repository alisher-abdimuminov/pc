package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pc/internal/config"
	"pc/internal/service"
)

// ---- a mock of the private face-service (speaks its real HTTP API) ----

type mockFace struct {
	srv   *httptest.Server
	calls atomic.Int32

	mu     sync.Mutex
	status int           // 0 = 200
	body   string        // JSON body for /verify; "" = default match
	delay  time.Duration // simulated inference time
	busy   bool          // /health reports an overloaded service
	health int           // /health status; 0 = 200
}

const matchBody = `{"face_found":true,"is_real":true,"antispoof_score":0.98,"distance":0.12,"threshold":0.3,"verified":true}`

func newMockFace(t *testing.T) *mockFace {
	m := &mockFace{}
	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Token") != "tok" {
			w.WriteHeader(401)
			return
		}
		m.mu.Lock()
		status, body, delay := m.status, m.body, m.delay
		m.mu.Unlock()
		switch r.URL.Path {
		case "/health":
			m.mu.Lock()
			busy, health := m.busy, m.health
			m.mu.Unlock()
			if health != 0 {
				w.WriteHeader(health)
				return
			}
			fmt.Fprintf(w, `{"status":"ok","busy":%t}`, busy)
		case "/verify":
			m.calls.Add(1)
			if delay > 0 {
				time.Sleep(delay)
			}
			if status != 0 {
				w.WriteHeader(status)
				w.Write([]byte(body))
				return
			}
			if body == "" {
				body = matchBody
			}
			w.Write([]byte(body))
		case "/embed":
			m.calls.Add(1)
			if status != 0 {
				w.WriteHeader(status)
				w.Write([]byte(body))
				return
			}
			w.Write([]byte(`{"embedding":[0.1,0.2,0.3],"model":"Facenet512","face_count":1}`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(m.srv.Close)
	return m
}

func (m *mockFace) set(status int, body string) {
	m.mu.Lock()
	m.status, m.body = status, body
	m.mu.Unlock()
}

func (m *mockFace) reset() {
	m.set(0, "")
	m.mu.Lock()
	m.delay, m.busy, m.health = 0, false, 0
	m.mu.Unlock()
}

func (m *mockFace) setBusy(b bool) { m.mu.Lock(); m.busy = b; m.mu.Unlock() }

func (m *mockFace) setDelay(d time.Duration) { m.mu.Lock(); m.delay = d; m.mu.Unlock() }

// ---- fixture: student alpha1 with an embedding, a schedule today and a rectangular location ----

var tinyPNG = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)

type aEnv struct {
	*tEnv
	mock  *mockFace
	day   time.Time
	stu   string // alpha1 session cookie
	sched string
}

// Inside the rectangle (39.650-39.651, 66.950-66.951); 20 m east of its east edge; far away.
const (
	inLat, inLng     = 39.6505, 66.9505
	nearLat, nearLng = 39.6505, 66.95122 // ~ 20 m outside the east edge (tolerance 30 m)
	farLat, farLng   = 41.30, 69.24
	goodAcc          = 12.0
)

func newAttemptEnv(t *testing.T, extra ...func(*config.Config)) *aEnv {
	mock := newMockFace(t)
	te := newTeacherEnv(t, append([]func(*config.Config){func(c *config.Config) {
		c.FaceServiceURL, c.FaceServiceToken = mock.srv.URL, "tok"
		c.FaceTimeout, c.MaxAttemptsPerStep, c.GPSMaxAccuracyM = 5*time.Second, 5, 100
		c.FaceMismatchLimit = 3
	}}, extra...)...)
	ae := &aEnv{tEnv: te, mock: mock, day: te.lastWorkday()}
	ctx := context.Background()
	var loc string
	te.pool.QueryRow(ctx, `INSERT INTO locations (name, polygon, tolerance_meters) VALUES ('Maktab','[[39.650,66.950],[39.651,66.950],[39.651,66.951],[39.650,66.951]]',30) RETURNING id`).Scan(&loc)
	te.pool.QueryRow(ctx, `INSERT INTO schedules (start_date, end_date, group_id, location_id, monday, tuesday, wednesday, thursday, friday, saturday)
		VALUES ($1,$1,$2,$3,true,true,true,true,true,true) RETURNING id`, ds(ae.day), te.groupA, loc).Scan(&ae.sched)
	te.pool.Exec(ctx, `UPDATE users SET face_embedding='{1,0,0}', face_model='Facenet512' WHERE id=$1`, te.studentA1)
	ae.stu = te.mustLogin("alpha1")
	ae.at(9, 0)
	return ae
}

func (a *aEnv) at(h, m int) { a.setNow(a.tEnv.at(a.day, h, m)) }

type attemptResp struct {
	Status int
	Code   string // error code when Status >= 400
	Body   map[string]any
}

func (a *aEnv) attemptAs(cookie string, lat, lng, acc float64, image []byte) attemptResp {
	a.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if image != nil {
		fw, _ := w.CreateFormFile("image", "selfie.png")
		fw.Write(image)
	}
	w.WriteField("latitude", strconv.FormatFloat(lat, 'f', -1, 64))
	w.WriteField("longitude", strconv.FormatFloat(lng, 'f', -1, 64))
	w.WriteField("accuracy", strconv.FormatFloat(acc, 'f', -1, 64))
	w.Close()
	req := httptest.NewRequest("POST", "/api/student/attendance/attempt", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Cookie", "session="+cookie)
	resp, err := a.app.Test(req, -1)
	if err != nil {
		a.t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	out := attemptResp{Status: resp.StatusCode}
	_ = json.Unmarshal(b, &out.Body)
	if resp.StatusCode >= 400 {
		out.Code = errCode(a.t, string(b))
	}
	return out
}

func (a *aEnv) attempt(lat, lng, acc float64) attemptResp {
	a.t.Helper()
	return a.attemptAs(a.stu, lat, lng, acc, tinyPNG)
}

func (a *aEnv) good() attemptResp { return a.attempt(inLat, inLng, goodAcc) }

func (a *aEnv) count(sql string, args ...any) int {
	var n int
	if err := a.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		a.t.Fatal(err)
	}
	return n
}

func (a *aEnv) attempts() int { return a.count(`SELECT count(*) FROM attendance_attempts`) }

func (a *aEnv) selfieFiles() int {
	n := 0
	filepath.Walk(filepath.Join(a.cfg.UploadDir, "selfies"), func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func strs(v any) []string {
	var out []string
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}

func (a *aEnv) expectNothingRecorded(what string) {
	a.t.Helper()
	if n := a.attempts(); n != 0 {
		a.t.Fatalf("%s: %d attempts recorded, want 0", what, n)
	}
	if n := a.count(`SELECT count(*) FROM attendances`); n != 0 {
		a.t.Fatalf("%s: %d attendances created, want 0", what, n)
	}
	if n := a.selfieFiles(); n != 0 {
		a.t.Fatalf("%s: %d selfie files left behind", what, n)
	}
}

// ---- tests ----

func TestAttemptSuccessAcrossTheDay(t *testing.T) {
	a := newAttemptEnv(t)
	ctx := context.Background()

	r := a.good()
	if r.Status != 200 || r.Body["success"] != true || r.Body["shift"] != 1.0 || r.Body["step"] != 1.0 || r.Body["steps_done"] != 1.0 {
		t.Fatalf("step 1: %+v", r)
	}
	if r.Body["face_verified"] != true || r.Body["location_verified"] != true || r.Body["liveness_verified"] != true || r.Body["attempts_used"] != 1.0 {
		t.Fatalf("flags: %+v", r.Body)
	}
	var shift int
	var date string
	a.pool.QueryRow(ctx, `SELECT shift, date::text FROM attendances`).Scan(&shift, &date)
	if shift != 1 || date != ds(a.day) {
		t.Fatalf("attendance: shift=%d date=%s", shift, date)
	}
	var fd, ft, sp float64
	var img string
	var okStep bool
	a.pool.QueryRow(ctx, `SELECT t.face_distance, t.face_threshold, t.antispoof_score, t.image, s.success FROM attendance_attempts t JOIN attendance_steps s ON s.id=t.step_id`).Scan(&fd, &ft, &sp, &img, &okStep)
	if fd != 0.12 || ft != 0.3 || sp != 0.98 || !okStep || img == "" {
		t.Fatalf("stored: %v %v %v %q %v", fd, ft, sp, img, okStep)
	}
	if _, err := os.Stat(filepath.Join(a.cfg.UploadDir, img)); err != nil {
		t.Fatal("selfie not saved")
	}
	if a.audits("attendance_attempt", "attendance_attempt") != 1 {
		t.Fatal("attempt not audited")
	}

	a.at(10, 30)
	if r := a.good(); r.Status != 200 || r.Body["step"] != 2.0 || r.Body["steps_done"] != 2.0 {
		t.Fatalf("step 2: %+v", r)
	}
	a.at(12, 59)
	if r := a.good(); r.Status != 200 || r.Body["step"] != 3.0 || r.Body["steps_done"] != 3.0 {
		t.Fatalf("step 3: %+v", r)
	}

	// the day shows as "full" in the teacher's report, with the shift
	rep := jsonMap(t, a.get("/api/teacher/groups/"+a.groupA+"/report?from="+ds(a.day)+"&to="+ds(a.day), a.cookieA, 200))
	for _, s := range rep["students"].([]any) {
		m := s.(map[string]any)
		if m["full_name"] == "alpha1" {
			c := m["cells"].(map[string]any)[ds(a.day)].(map[string]any)
			if c["status"] != "full" || c["shift"] != 1.0 {
				t.Fatalf("report cell: %v", c)
			}
		}
	}
	// attempt details and selfies are visible to admins only
	at := jsonMap(t, a.get("/api/admin/students/"+a.studentA1+"/attempts?from="+ds(a.day)+"&to="+ds(a.day), a.admin, 200))
	first := at["days"].([]any)[0].(map[string]any)["steps"].([]any)[0].(map[string]any)["attempts"].([]any)[0].(map[string]any)
	if first["has_selfie"] != true || first["face_distance"] != 0.12 || first["antispoof_score"] != 0.98 {
		t.Fatalf("attempt details: %v", first)
	}
	id := first["id"].(string)
	a.get("/api/files/selfies/"+id, a.admin, 200)
	a.get("/api/files/selfies/"+id, a.cookieA, 404) // not even the student's own teacher
	a.get("/api/files/selfies/"+id, a.cookieB, 404)
	a.get("/api/files/selfies/"+id, a.stu, 404) // and the student cannot fetch it back
	// a teacher's list holds only attempts to review; this normal one is not in it
	if days := jsonMap(t, a.get("/api/teacher/students/"+a.studentA1+"/attempts", a.cookieA, 200))["days"].([]any); len(days) != 0 {
		t.Fatalf("teacher sees a normal attempt: %v", days)
	}
	a.get("/api/teacher/students/"+a.studentA1+"/attempts", a.cookieB, 404)

	// only an admin can delete an attempt; doing so removes the selfie and un-succeeds the step
	if resp, _ := a.do("DELETE", "/api/admin/attempts/"+id, "", a.cookieA); resp.StatusCode != 403 {
		t.Fatalf("teacher delete: %d", resp.StatusCode)
	}
	if resp, _ := a.do("DELETE", "/api/admin/attempts/"+id, "", a.admin); resp.StatusCode != 204 {
		t.Fatalf("admin delete: %d", resp.StatusCode)
	}
	if resp, _ := a.do("DELETE", "/api/admin/attempts/"+id, "", a.admin); resp.StatusCode != 404 {
		t.Fatalf("second delete: %d", resp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(a.cfg.UploadDir, img)); !os.IsNotExist(err) {
		t.Fatal("selfie file must be removed")
	}
	var done int
	a.pool.QueryRow(ctx, `SELECT count(*) FROM attendance_steps WHERE success AND step = 1`).Scan(&done)
	if done != 0 {
		t.Fatal("step 1 must no longer be successful")
	}
	if a.audits("delete", "attendance_attempt") != 1 {
		t.Fatal("delete not audited")
	}
}

func TestAttemptLocationFailures(t *testing.T) {
	a := newAttemptEnv(t)

	// far away: recorded, not successful, step stays open, other checks still reported
	r := a.attempt(farLat, farLng, goodAcc)
	if r.Status != 200 || r.Body["success"] != false || r.Body["location_verified"] != false || r.Body["face_verified"] != true {
		t.Fatalf("far: %+v", r)
	}
	if f := strs(r.Body["failures"]); len(f) != 1 || f[0] != "location" || r.Body["attempts_left"] != 4.0 || r.Body["steps_done"] != 0.0 {
		t.Fatalf("far failures: %+v", r.Body)
	}
	if a.attempts() != 1 || a.count(`SELECT count(*) FROM attendance_steps WHERE success`) != 0 {
		t.Fatal("failed attempt must be recorded and not complete the step")
	}

	// ~20 m outside is within the 30 m tolerance
	if r := a.attempt(nearLat, nearLng, goodAcc); r.Body["success"] != true {
		t.Fatalf("within tolerance should pass: %+v", r)
	}
}

func TestAttemptPoorGPSAccuracy(t *testing.T) {
	a := newAttemptEnv(t)
	r := a.attempt(inLat, inLng, 150) // limit is 100 m
	if r.Body["success"] != false || r.Body["location_verified"] != false {
		t.Fatalf("%+v", r)
	}
	if f := strs(r.Body["failures"]); len(f) != 1 || f[0] != "location_accuracy" {
		t.Fatalf("failures: %v", f)
	}
	if r := a.attempt(inLat, inLng, 100); r.Body["success"] != true { // exactly at the limit is fine
		t.Fatalf("accuracy at the limit: %+v", r)
	}
}

func TestAttemptLivenessFaceAndNoFace(t *testing.T) {
	a := newAttemptEnv(t)

	a.mock.set(0, `{"face_found":true,"is_real":false,"antispoof_score":0.03,"distance":null,"threshold":0.3,"verified":false}`)
	r := a.good()
	if r.Body["success"] != false || r.Body["liveness_verified"] != false || r.Body["face_verified"] != false || r.Body["location_verified"] != true {
		t.Fatalf("spoof: %+v", r)
	}
	if f := strs(r.Body["failures"]); len(f) != 1 || f[0] != "liveness" {
		t.Fatalf("spoof failures: %v", f)
	}
	var score float64
	var dist *float64
	a.pool.QueryRow(context.Background(), `SELECT antispoof_score, face_distance FROM attendance_attempts`).Scan(&score, &dist)
	if score != 0.03 || dist != nil {
		t.Fatalf("stored spoof data: %v %v", score, dist)
	}

	a.mock.set(0, `{"face_found":true,"is_real":true,"antispoof_score":0.9,"distance":0.8,"threshold":0.3,"verified":false}`)
	r = a.good()
	if f := strs(r.Body["failures"]); r.Body["liveness_verified"] != true || r.Body["face_verified"] != false || len(f) != 1 || f[0] != "face" {
		t.Fatalf("mismatch: %+v", r)
	}

	a.mock.set(422, `{"code":"no_face"}`)
	r = a.good()
	if f := strs(r.Body["failures"]); r.Status != 200 || r.Body["success"] != false || len(f) != 1 || f[0] != "no_face" {
		t.Fatalf("no face: %+v", r)
	}
	a.mock.set(422, `{"code":"multiple_faces"}`)
	if f := strs(a.good().Body["failures"]); len(f) != 1 || f[0] != "multiple_faces" {
		t.Fatalf("multiple faces: %v", f)
	}
	if a.attempts() != 4 {
		t.Fatalf("all four failures must be recorded, got %d", a.attempts())
	}

	// combined failures are all reported
	a.mock.set(0, `{"face_found":true,"is_real":false,"antispoof_score":0.1,"distance":null,"threshold":0.3,"verified":false}`)
	if f := strs(a.attempt(farLat, farLng, goodAcc).Body["failures"]); len(f) != 2 {
		t.Fatalf("expected location+liveness, got %v", f)
	}
}

func TestAttemptFaceServiceDownConsumesNothingWhenLocationFails(t *testing.T) {
	a := newAttemptEnv(t)
	for _, status := range []int{500, 503, 401} {
		a.mock.set(status, `x`)
		r := a.attempt(farLat, farLng, goodAcc) // the location would fail anyway, so there is nothing to defer
		if r.Status != 503 || r.Code != "face_unavailable" {
			t.Fatalf("status %d -> %+v", status, r)
		}
	}
	a.expectNothingRecorded("service down")
}

func TestAttemptAcceptedForReviewWhenFaceServiceIsBusyOrDown(t *testing.T) {
	cases := map[string]func(a *aEnv){
		"busy":       func(a *aEnv) { a.mock.setBusy(true) },
		"verify 503": func(a *aEnv) { a.mock.set(503, `x`) },
		"health 500": func(a *aEnv) { a.mock.health = 500 },
		"closed":     func(a *aEnv) { a.mock.srv.Close() },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			a := newAttemptEnv(t)
			setup(a)
			r := a.good()
			if r.Status != 200 || r.Body["success"] != true || r.Body["message"] != "Tasdiqlandi" {
				t.Fatalf("student must be told it passed: %+v", r)
			}
			if got := a.count(`SELECT count(*) FROM attendance_attempts WHERE review='pending' AND success AND NOT face_verified`); got != 1 {
				t.Fatalf("pending attempts = %d", got)
			}
			if got := a.count(`SELECT count(*) FROM attendance_steps WHERE review='pending' AND success`); got != 1 {
				t.Fatalf("pending steps = %d", got)
			}
			// the same step cannot be submitted twice
			if r := a.good(); r.Code != "step_done" {
				t.Fatalf("repeat: %+v", r)
			}
		})
	}
}

func TestBusyServiceStillRejectsWrongLocation(t *testing.T) {
	a := newAttemptEnv(t)
	a.mock.setBusy(true)
	r := a.attempt(farLat, farLng, goodAcc)
	if r.Body["success"] != false || !strings.Contains(strings.Join(strs(r.Body["failures"]), ","), "location") {
		t.Fatalf("%+v", r)
	}
	if a.count(`SELECT count(*) FROM attendance_attempts WHERE review<>''`) != 0 {
		t.Fatal("a failed location must not be queued for review")
	}
}

func TestReviewApproveAndRejectPendingAttempt(t *testing.T) {
	a := newAttemptEnv(t)
	a.mock.setBusy(true)
	a.good()
	var id string
	a.pool.QueryRow(context.Background(), `SELECT id FROM attendance_attempts WHERE review='pending'`).Scan(&id)

	// report + teacher group page show it as pending
	day := ds(a.day)
	rep := jsonMap(t, a.get("/api/teacher/groups/"+a.groupA+"/report?from="+day+"&to="+day, a.cookieA, 200))
	cell := rep["students"].([]any)[0].(map[string]any)["cells"].(map[string]any)[day].(map[string]any)
	if cell["pending"] != float64(1) {
		t.Fatalf("report cell: %v", cell)
	}
	// the teacher sees it and may open its selfie; another teacher may not
	list := jsonMap(t, a.get("/api/teacher/students/"+a.studentA1+"/attempts?from="+day+"&to="+day, a.cookieA, 200))
	if len(list["days"].([]any)) != 1 {
		t.Fatalf("teacher list: %v", list)
	}
	a.get("/api/files/selfies/"+id, a.cookieA, 200)
	a.get("/api/files/selfies/"+id, a.cookieB, 404)
	if r, _ := a.do("PUT", "/api/teacher/attempts/"+id+"/review", `{"approved":true}`, a.cookieB); r.StatusCode != 404 {
		t.Fatalf("other teacher: %d", r.StatusCode)
	}

	// approve
	if r, b := a.do("PUT", "/api/teacher/attempts/"+id+"/review", `{"approved":true}`, a.cookieA); r.StatusCode != 204 {
		t.Fatalf("approve: %d %s", r.StatusCode, b)
	}
	if a.count(`SELECT count(*) FROM attendance_steps WHERE review='approved' AND success`) != 1 {
		t.Fatal("step not approved")
	}
	// admin rejects: the step is lost and cannot be retried
	if r, b := a.do("PUT", "/api/admin/attempts/"+id+"/review", `{"approved":false}`, a.admin); r.StatusCode != 204 {
		t.Fatalf("reject: %d %s", r.StatusCode, b)
	}
	if a.count(`SELECT count(*) FROM attendance_steps WHERE review='rejected' AND NOT success`) != 1 {
		t.Fatal("step not rejected")
	}
	a.mock.reset()
	if r := a.good(); r.Status != 409 || r.Code != "step_rejected" {
		t.Fatalf("retry after reject: %+v", r)
	}
	// a normal attempt has nothing to review
	if a.count(`SELECT count(*) FROM audit_logs WHERE entity_type='attendance_attempt' AND action='update'`) != 2 {
		t.Fatal("review not audited")
	}
}

func TestAttemptOutsideHoursAndPreconditions(t *testing.T) {
	a := newAttemptEnv(t)
	for _, c := range []struct {
		h, m int
		code string
	}{{7, 59, "outside_hours"}, {18, 0, "outside_hours"}, {23, 59, "outside_hours"}, {0, 0, "outside_hours"}} {
		a.at(c.h, c.m)
		if r := a.good(); r.Status != 409 || r.Code != c.code {
			t.Fatalf("%02d:%02d -> %+v", c.h, c.m, r)
		}
	}
	// the boundaries that are inside the day
	a.at(8, 0)
	if r := a.good(); r.Status != 200 {
		t.Fatalf("08:00: %+v", r)
	}
	a.mock.reset()

	// no schedule on that date
	b := newAttemptEnv(t)
	b.setNow(b.tEnv.at(b.day.AddDate(0, 0, 1), 9, 0)) // the schedule covers one day only
	if r := b.good(); r.Status != 409 || r.Code != "no_schedule" {
		t.Fatalf("no schedule: %+v", r)
	}
	b.expectNothingRecorded("no schedule")

	// a student without an enrolled face
	c := newAttemptEnv(t)
	c.pool.Exec(context.Background(), `UPDATE users SET face_embedding=NULL WHERE id=$1`, c.studentA1)
	if r := c.good(); r.Status != 409 || r.Code != "no_face_enrolled" {
		t.Fatalf("not enrolled: %+v", r)
	}
	if c.mock.calls.Load() != 0 {
		t.Fatal("the face-service must not be called without an enrolled face")
	}

	// a schedule without a location
	d := newAttemptEnv(t)
	d.pool.Exec(context.Background(), `UPDATE schedules SET location_id=NULL`)
	if r := d.good(); r.Status != 409 || r.Code != "no_location" {
		t.Fatalf("no location: %+v", r)
	}

	// only students may attempt
	if r := d.attemptAs(d.cookieA, inLat, inLng, goodAcc, tinyPNG); r.Status != 403 {
		t.Fatalf("teacher: %d", r.Status)
	}
	if r := d.attemptAs("", inLat, inLng, goodAcc, tinyPNG); r.Status != 401 {
		t.Fatalf("anonymous: %d", r.Status)
	}
}

func TestAttemptRejectsInvalidInput(t *testing.T) {
	a := newAttemptEnv(t)
	cases := map[string]attemptResp{
		"no image":       a.attemptAs(a.stu, inLat, inLng, goodAcc, nil),
		"not an image":   a.attemptAs(a.stu, inLat, inLng, goodAcc, []byte("%PDF-1.4 not a selfie")),
		"latitude range": a.attemptAs(a.stu, 123, inLng, goodAcc, tinyPNG),
		"negative acc":   a.attemptAs(a.stu, inLat, inLng, -5, tinyPNG),
		"too large":      a.attemptAs(a.stu, inLat, inLng, goodAcc, append(append([]byte{}, tinyPNG...), make([]byte, 5<<20)...)),
	}
	for name, r := range cases {
		if r.Status != 422 {
			t.Errorf("%s: got %d (%s)", name, r.Status, r.Code)
		}
	}
	a.expectNothingRecorded("invalid input")
	if a.mock.calls.Load() != 0 {
		t.Fatal("invalid input must not reach the face-service")
	}
}

func TestAttemptShiftSwitchIsRejected(t *testing.T) {
	a := newAttemptEnv(t)
	a.at(9, 0)
	if r := a.attempt(farLat, farLng, goodAcc); r.Status != 200 { // even a failed attempt fixes the day's shift
		t.Fatalf("%+v", r)
	}
	calls := a.mock.calls.Load()
	a.at(14, 0) // shift 2 hours
	r := a.good()
	if r.Status != 409 || r.Code != "shift_locked" {
		t.Fatalf("shift switch: %+v", r)
	}
	if a.mock.calls.Load() != calls || a.attempts() != 1 || a.selfieFiles() != 1 {
		t.Fatalf("rejection must not call the face-service or record anything (calls %d, attempts %d, files %d)", a.mock.calls.Load(), a.attempts(), a.selfieFiles())
	}
	// the student's home screen says so
	today := jsonMap(t, a.get("/api/student/today", a.stu, 200))
	if today["shift_locked"] != true {
		t.Fatalf("today: %v", today)
	}
	// back in shift 1 hours the next step still works
	a.at(11, 0)
	if r := a.good(); r.Status != 200 || r.Body["step"] != 2.0 {
		t.Fatalf("%+v", r)
	}
}

func TestAttemptLimitPerStep(t *testing.T) {
	a := newAttemptEnv(t)
	for i := 1; i <= 5; i++ {
		r := a.attempt(farLat, farLng, goodAcc)
		if r.Status != 200 || r.Body["attempts_used"] != float64(i) || r.Body["attempts_left"] != float64(5-i) {
			t.Fatalf("attempt %d: %+v", i, r)
		}
	}
	calls := a.mock.calls.Load()
	r := a.good() // a perfect attempt, but the budget is gone
	if r.Status != 409 || r.Code != "attempts_exhausted" {
		t.Fatalf("6th attempt: %+v", r)
	}
	if a.mock.calls.Load() != calls || a.attempts() != 5 || a.selfieFiles() != 5 {
		t.Fatalf("calls=%d attempts=%d files=%d", a.mock.calls.Load(), a.attempts(), a.selfieFiles())
	}
	// the budget is per step: the next window starts fresh
	a.at(10, 0)
	if r := a.good(); r.Status != 200 || r.Body["success"] != true || r.Body["attempts_used"] != 1.0 {
		t.Fatalf("next step: %+v", r)
	}
}

func TestAttemptStepAlreadyDone(t *testing.T) {
	a := newAttemptEnv(t)
	if r := a.good(); r.Body["success"] != true {
		t.Fatal("first attempt should succeed")
	}
	calls := a.mock.calls.Load()
	if r := a.good(); r.Status != 409 || r.Code != "step_done" {
		t.Fatalf("repeat: %+v", r)
	}
	if a.mock.calls.Load() != calls || a.attempts() != 1 {
		t.Fatal("a repeat for a finished step must be rejected before the face check")
	}
}

func TestAttemptConcurrentSuccessIsAppliedOnce(t *testing.T) {
	a := newAttemptEnv(t)
	a.mock.setDelay(150 * time.Millisecond) // both requests pass the pre-check before either commits
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = a.good().Status
		}(i)
	}
	wg.Wait()
	sort.Ints(codes)
	if codes[0] != 200 || codes[1] != 409 {
		t.Fatalf("statuses %v, want one 200 and one 409", codes)
	}
	if a.attempts() != 1 || a.count(`SELECT count(*) FROM attendance_steps WHERE success`) != 1 ||
		a.count(`SELECT count(*) FROM attendances`) != 1 || a.count(`SELECT count(*) FROM attendance_steps`) != 1 {
		t.Fatal("concurrent requests must produce exactly one attendance, one step and one attempt")
	}
	if a.selfieFiles() != 1 {
		t.Fatalf("the rejected request's selfie must be removed, %d files", a.selfieFiles())
	}
}

func TestAttemptConcurrentFailuresRespectTheLimit(t *testing.T) {
	a := newAttemptEnv(t)
	a.mock.setDelay(100 * time.Millisecond)
	const n = 9
	var wg sync.WaitGroup
	var ok, rejected atomic.Int32
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch r := a.attempt(farLat, farLng, goodAcc); r.Status {
			case 200:
				ok.Add(1)
			case 409:
				rejected.Add(1)
			default:
				t.Errorf("unexpected %+v", r)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 5 || rejected.Load() != n-5 || a.attempts() != 5 || a.selfieFiles() != 5 {
		t.Fatalf("ok=%d rejected=%d recorded=%d files=%d, want 5 / %d / 5 / 5", ok.Load(), rejected.Load(), a.attempts(), a.selfieFiles(), n-5)
	}
}

// ---- 5.1 / 5.3: student photo enrollment through the admin UI ----

func (te *tEnv) putPhoto(path, cookie string, content []byte, field string, want int) map[string]any {
	te.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if content != nil {
		fw, _ := w.CreateFormFile(field, "photo.png")
		fw.Write(content)
	}
	w.Close()
	req := httptest.NewRequest("PUT", path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Cookie", "session="+cookie)
	resp, err := te.app.Test(req, -1)
	if err != nil {
		te.t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		te.t.Fatalf("PUT %s: got %d want %d: %s", path, resp.StatusCode, want, b)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func TestAdminEnrollsStudentPhoto(t *testing.T) {
	mock := newMockFace(t)
	te := newTeacherEnv(t, func(c *config.Config) {
		c.FaceServiceURL, c.FaceServiceToken, c.FaceTimeout = mock.srv.URL, "tok", 5*time.Second
	})
	ctx := context.Background()
	url := "/api/admin/users/" + te.studentA2 + "/photo"

	u := te.putPhoto(url, te.admin, tinyPNG, "image", 200)
	if u["face_enrolled"] != true || u["has_photo"] != true {
		t.Fatalf("user: %v", u)
	}
	var emb []float32
	var model, img string
	te.pool.QueryRow(ctx, `SELECT face_embedding, face_model, image FROM users WHERE id=$1`, te.studentA2).Scan(&emb, &model, &img)
	if len(emb) != 3 || emb[1] != 0.2 || model != "Facenet512" || img == "" {
		t.Fatalf("stored: %v %q %q", emb, model, img)
	}
	if _, err := os.Stat(filepath.Join(te.cfg.UploadDir, img)); err != nil {
		t.Fatal("photo not stored")
	}
	// the audit entry records the fact, never the embedding
	var ch string
	te.pool.QueryRow(ctx, `SELECT changes::text FROM audit_logs WHERE action='update' AND entity_type='user' AND entity_id=$1`, te.studentA2).Scan(&ch)
	if !jsonHas(ch, "face_enrolled") || jsonHas(ch, "0.2") || jsonHas(ch, "face_embedding") {
		t.Fatalf("audit: %s", ch)
	}
	// the student sees their own photo; replacing it removes the old file
	te.get("/api/files/avatars/"+te.studentA2, te.admin, 200)
	te.putPhoto(url, te.admin, tinyPNG, "image", 200)
	if _, err := os.Stat(filepath.Join(te.cfg.UploadDir, img)); !os.IsNotExist(err) {
		t.Fatal("old photo must be deleted")
	}

	// failures leave the previous enrollment untouched
	te.putPhoto(url, te.admin, nil, "image", 422)
	te.putPhoto(url, te.admin, []byte("not an image"), "image", 422)
	mock.set(422, `{"code":"no_face"}`)
	if m := te.putPhoto(url, te.admin, tinyPNG, "image", 422); jsonHas(toJSON(m), "yuz topilmadi") == false {
		t.Fatalf("no face message: %v", m)
	}
	mock.set(422, `{"code":"multiple_faces"}`)
	te.putPhoto(url, te.admin, tinyPNG, "image", 422)
	mock.set(503, ``)
	if m := te.putPhoto(url, te.admin, tinyPNG, "image", 503); jsonHas(toJSON(m), "vaqtincha") == false {
		t.Fatalf("unavailable message: %v", m)
	}
	var after []float32
	te.pool.QueryRow(ctx, `SELECT face_embedding FROM users WHERE id=$1`, te.studentA2).Scan(&after)
	if len(after) != 3 {
		t.Fatal("a failed enrollment must not wipe the old embedding")
	}
	// staff and unauthenticated users cannot, and only students can be enrolled
	te.putPhoto(url, te.cookieA, tinyPNG, "image", 403)
	te.putPhoto("/api/admin/users/"+te.teachA+"/photo", te.admin, tinyPNG, "image", 422)
	te.putPhoto("/api/admin/users/00000000-0000-0000-0000-000000000000/photo", te.admin, tinyPNG, "image", 404)
}

func toJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func jsonHas(s, sub string) bool {
	return bytes.Contains(bytes.ToLower([]byte(s)), bytes.ToLower([]byte(sub)))
}

// ---- rate limiter: repeated face mismatches within one attendance block the student ----

const mismatchBody = `{"face_found":true,"is_real":true,"antispoof_score":0.9,"distance":0.8,"threshold":0.3,"verified":false}`
const spoofBody = `{"face_found":true,"is_real":false,"antispoof_score":0.9,"distance":null,"threshold":0.3,"verified":false}`

func (a *aEnv) userActive() bool {
	var active bool
	a.pool.QueryRow(context.Background(), `SELECT is_active FROM users WHERE id=$1`, a.studentA1).Scan(&active)
	return active
}

func TestThreeFaceMismatchesInOneAttendanceBlockAndLogOutTheStudent(t *testing.T) {
	a := newAttemptEnv(t)
	a.mock.set(0, mismatchBody)

	// two mismatches (in different steps of the same attendance) are only warnings
	for _, h := range []int{9, 10} {
		a.at(h, 30)
		r := a.good()
		if r.Status != 200 || r.Body["blocked"] != false || r.Body["success"] != false {
			t.Fatalf("mismatch at %d:30: %+v", h, r)
		}
		if f := strs(r.Body["failures"]); len(f) != 1 || f[0] != "face" {
			t.Fatalf("failures: %v", f)
		}
	}
	if !a.userActive() {
		t.Fatal("student must still be active after two mismatches")
	}

	// the third, in a third step of the same attendance, blocks
	a.at(12, 30)
	r := a.good()
	if r.Status != 200 || r.Body["blocked"] != true || r.Body["success"] != false || r.Body["message"] != service.AccountInactiveMessage {
		t.Fatalf("third mismatch: %+v", r)
	}
	if a.userActive() {
		t.Fatal("student must be deactivated")
	}
	if a.count(`SELECT count(*) FROM sessions WHERE user_id=$1`, a.studentA1) != 0 {
		t.Fatal("every session must be ended")
	}
	if a.attempts() != 3 {
		t.Fatalf("the blocking attempt must still be recorded, got %d", a.attempts())
	}

	// the old cookie no longer works anywhere
	a.get("/api/student/today", a.stu, 401)
	if r := a.attempt(inLat, inLng, goodAcc); r.Status != 401 {
		t.Fatalf("attempt after the block: %d", r.Status)
	}
	// logging in again explains why
	resp, out, _ := a.login("alpha1", testPass)
	if resp.StatusCode != 403 || errCode(t, out) != "account_inactive" || !strings.Contains(out, "Hisobingiz faol emas") {
		t.Fatalf("login after block: %d %s", resp.StatusCode, out)
	}
	// audited with its reason, and the response cleared the cookie
	var ch string
	a.pool.QueryRow(context.Background(), `SELECT changes::text FROM audit_logs WHERE action='update' AND entity_type='user' AND entity_id=$1`, a.studentA1).Scan(&ch)
	if !strings.Contains(ch, "face_mismatch_limit") || !strings.Contains(ch, `"is_active": false`) && !strings.Contains(ch, `"is_active":false`) {
		t.Fatalf("block not audited: %s", ch)
	}
}

func TestOnlyFaceMismatchesCountTowardsTheBlock(t *testing.T) {
	a := newAttemptEnv(t)

	// many other failures never block: wrong place, bad GPS, spoof, no face, several faces
	a.mock.set(0, matchBody)
	a.attempt(farLat, farLng, goodAcc)
	a.attempt(inLat, inLng, 500)
	a.mock.set(0, spoofBody)
	a.good()
	a.mock.set(422, `{"code":"no_face"}`)
	a.good()
	if a.attempts() != 4 || !a.userActive() {
		t.Fatalf("attempts=%d active=%v", a.attempts(), a.userActive())
	}

	// a mismatch caused only by the face (location also wrong) still counts as a face mismatch
	a.mock.set(0, mismatchBody)
	a.at(10, 5)
	a.attempt(farLat, farLng, goodAcc) // face mismatch + wrong place
	a.at(11, 0)
	a.good()
	if !a.userActive() {
		t.Fatal("two mismatches must not block")
	}
	a.at(12, 10)
	if r := a.good(); r.Body["blocked"] != true {
		t.Fatalf("third mismatch should block: %+v", r)
	}
}

func TestMismatchesAreCountedPerAttendanceNotAcrossDays(t *testing.T) {
	a := newAttemptEnv(t)
	ctx := context.Background()
	a.mock.set(0, mismatchBody)
	a.good()
	a.at(10, 5)
	a.good() // two mismatches today

	// a second working day with its own schedule: a fresh attendance, a fresh count
	next := a.day.AddDate(0, 0, 1)
	if next.Weekday() == time.Sunday {
		t.Skip("Sunday has no schedule")
	}
	var loc string
	a.pool.QueryRow(ctx, `SELECT location_id FROM schedules WHERE id=$1`, a.sched).Scan(&loc)
	a.pool.Exec(ctx, `INSERT INTO schedules (start_date, end_date, group_id, location_id, monday, tuesday, wednesday, thursday, friday, saturday)
		VALUES ($1,$1,$2,$3,true,true,true,true,true,true)`, ds(next), a.groupA, loc)
	a.setNow(a.tEnv.at(next, 9, 0))
	if r := a.good(); r.Body["blocked"] != false || !a.userActive() {
		t.Fatalf("a new day starts counting from zero: %+v", r)
	}
}

func TestReactivatedStudentGetsAnotherRoundOfTries(t *testing.T) {
	a := newAttemptEnv(t)
	ctx := context.Background()
	a.mock.set(0, mismatchBody)
	for _, h := range []int{9, 10, 12} {
		a.at(h, 5)
		a.good()
	}
	if a.userActive() {
		t.Fatal("should be blocked after three")
	}

	// the admin reactivates the account; the student logs in again
	a.pool.Exec(ctx, `UPDATE users SET is_active=true WHERE id=$1`, a.studentA1)
	a.stu = a.mustLogin("alpha1")

	// attempts 4 and 5 are fine (the fourth step window of the day is shift 1 step 3 / limits aside)
	a.at(12, 30)
	if r := a.good(); r.Body["blocked"] != false {
		t.Fatalf("4th mismatch must not block: %+v", r)
	}
	if r := a.good(); r.Body["blocked"] != false {
		t.Fatalf("5th mismatch must not block: %+v", r)
	}
	if r := a.good(); r.Body["blocked"] != true { // 6th
		t.Fatalf("6th mismatch must block again: %+v", r)
	}
}

func TestMismatchLimitIsConfigurable(t *testing.T) {
	a := newAttemptEnv(t, func(c *config.Config) { c.FaceMismatchLimit = 2 })
	a.mock.set(0, mismatchBody)
	if r := a.good(); r.Body["blocked"] != false {
		t.Fatalf("first mismatch must not block: %+v", r)
	}
	if r := a.good(); r.Body["blocked"] != true {
		t.Fatalf("second mismatch must block with FACE_MISMATCH_LIMIT=2: %+v", r)
	}
}

// ---- admin "Urinishlar": every attempt of every student, filterable ----

func TestAdminAttemptsListAndFilters(t *testing.T) {
	a := newAttemptEnv(t)
	a.attempt(farLat, farLng, goodAcc) // location failure
	a.mock.set(0, mismatchBody)
	a.good() // face mismatch
	a.mock.set(0, spoofBody)
	a.good() // liveness failure
	a.mock.set(422, `{"code":"no_face"}`)
	a.good() // no face
	a.mock.reset()
	a.good() // success

	list := func(q string) map[string]any { return jsonMap(a.t, a.get("/api/admin/attempts"+q, a.admin, 200)) }
	total := func(q string) float64 { return list(q)["total"].(float64) }

	all := list("")
	if all["total"] != 5.0 || len(all["items"].([]any)) != 5 {
		t.Fatalf("all: %v", all["total"])
	}
	for q, want := range map[string]float64{
		"?success=true": 1, "?success=false": 4,
		"?kind=location": 1, "?kind=face": 1, "?kind=liveness": 1, "?kind=no_face": 1,
		"?kind=location&success=true": 0,
		"?search=alpha":               5, "?search=ALPHA1": 5, "?search=zzz": 0,
		"?group_id=" + a.groupA: 5, "?group_id=" + a.groupB: 0,
		"?student_id=" + a.studentA1: 5, "?student_id=" + a.studentA2: 0,
	} {
		if got := total(q); got != want {
			t.Errorf("%s: got %v want %v", q, got, want)
		}
	}

	// date range is inclusive and in the business timezone; attempts were created just now
	today := ds(a.tEnv.today())
	yesterday := ds(a.tEnv.today().AddDate(0, 0, -1))
	if total("?from="+today+"&to="+today) != 5 || total("?to="+yesterday) != 0 || total("?from="+ds(a.tEnv.today().AddDate(0, 0, 1))) != 0 {
		t.Fatal("date filters misbehave")
	}

	// a row carries everything the page shows
	var failed map[string]any
	for _, it := range list("?kind=face")["items"].([]any) {
		failed = it.(map[string]any)
	}
	if failed["student_name"] != "alpha1" || failed["group_name"] != "A-1" || failed["step"] != 1.0 || failed["shift"] != 1.0 ||
		failed["face_verified"] != false || failed["liveness_verified"] != true || failed["location_verified"] != true ||
		failed["face_distance"] != 0.8 || failed["face_threshold"] != 0.3 || failed["has_selfie"] != true || failed["success"] != false ||
		!strings.Contains(failed["error"].(string), "mos kelmadi") || failed["date"] != ds(a.day) {
		t.Fatalf("row: %v", failed)
	}
	noFace := list("?kind=no_face")["items"].([]any)[0].(map[string]any)
	if noFace["face_distance"] != nil || noFace["antispoof_score"] != nil {
		t.Fatalf("no-face row must have no scores: %v", noFace)
	}

	// pagination, newest first
	p3 := list("?page_size=2&page=3")
	if p3["total"] != 5.0 || len(p3["items"].([]any)) != 1 || p3["page"] != 3.0 || p3["page_size"] != 2.0 {
		t.Fatalf("page 3: %v", p3)
	}
	items := all["items"].([]any)
	for i := 1; i < len(items); i++ {
		prev, _ := time.Parse(time.RFC3339Nano, items[i-1].(map[string]any)["attempted_at"].(string))
		cur, _ := time.Parse(time.RFC3339Nano, items[i].(map[string]any)["attempted_at"].(string))
		if cur.After(prev) {
			t.Fatal("must be sorted newest first")
		}
	}

	// validation and permissions
	a.get("/api/admin/attempts?kind=bogus", a.admin, 422)
	a.get("/api/admin/attempts?from=bad", a.admin, 422)
	a.get("/api/admin/attempts?group_id=nope", a.admin, 422)
	a.get("/api/admin/attempts", a.cookieA, 403)
	a.get("/api/admin/attempts", a.stu, 403)
	a.get("/api/admin/attempts", "", 401)
}
