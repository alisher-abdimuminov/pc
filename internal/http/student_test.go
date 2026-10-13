package http_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"pc/internal/db"
	"pc/internal/service"
)

// studentUpload sends PUT /api/student/assignments/:id/submission as a multipart form.
func (te *tEnv) studentUpload(cookie, assignmentID, filename string, content []byte, wantStatus int) map[string]any {
	te.t.Helper()
	var file *[2]any
	if content != nil {
		file = &[2]any{filename, content}
	}
	return te.multipart("PUT", "/api/student/assignments/"+assignmentID+"/submission", cookie, nil, file, wantStatus)
}

func (te *tEnv) newAssignment(title, deadline string) string {
	te.t.Helper()
	a := te.multipart("POST", "/api/teacher/assignments", te.cookieA, map[string]string{
		"group_id": te.groupA, "title": title, "deadline": deadline}, &[2]any{"vazifa.pdf", pdfBytes}, 201)
	return a["id"].(string)
}

func (te *tEnv) submissionFile(assignmentID, studentID string) string {
	var p string
	te.pool.QueryRow(context.Background(), `SELECT file FROM submissions WHERE assignment_id=$1 AND student_id=$2`, assignmentID, studentID).Scan(&p)
	return p
}

// ---- 4.5: submissions ----

func TestStudentSubmissionLifecycle(t *testing.T) {
	te := newTeacherEnv(t)
	stu := te.mustLogin("alpha1")
	aid := te.newAssignment("Amaliy ish", "2099-01-01T10:00")

	// the list shows it with a live submit window and no submission yet
	list := jsonMap(t, te.get("/api/student/assignments", stu, 200))
	items := list["items"].([]any)
	if len(items) != 1 || list["server_time"] == nil {
		t.Fatalf("list: %v", list)
	}
	it := items[0].(map[string]any)
	if it["can_submit"] != true || it["overdue"] != false || it["submission"] != nil || it["has_file"] != true {
		t.Fatalf("item: %v", it)
	}

	// validation: missing file, wrong content
	te.studentUpload(stu, aid, "x.pdf", nil, 422)
	te.studentUpload(stu, aid, "virus.pdf", []byte("MZ\x90\x00 not a pdf"), 422)
	if te.submissionFile(aid, te.studentA1) != "" {
		t.Fatal("rejected upload must not create a submission")
	}

	// first upload
	got := te.studentUpload(stu, aid, "ish.pdf", pdfBytes, 200)
	sub := got["submission"].(map[string]any)
	if sub["has_file"] != true || sub["grade"] != nil || got["can_submit"] != true {
		t.Fatalf("after upload: %v", got)
	}
	first := te.submissionFile(aid, te.studentA1)
	if !strings.HasPrefix(first, "submissions/") {
		t.Fatalf("path: %s", first)
	}
	if _, err := os.Stat(filepath.Join(te.cfg.UploadDir, first)); err != nil {
		t.Fatal("file missing on disk")
	}

	// re-upload while ungraded replaces the file and deletes the old one
	te.studentUpload(stu, aid, "ish2.pdf", pdfBytes, 200)
	second := te.submissionFile(aid, te.studentA1)
	if second == first {
		t.Fatal("path must change on re-upload")
	}
	if _, err := os.Stat(filepath.Join(te.cfg.UploadDir, first)); !os.IsNotExist(err) {
		t.Fatal("old file must be deleted on re-upload")
	}
	var n int
	te.pool.QueryRow(context.Background(), `SELECT count(*) FROM submissions WHERE assignment_id=$1`, aid).Scan(&n)
	if n != 1 {
		t.Fatalf("one submission per student, got %d", n)
	}
	if te.audits("create", "submission") != 1 || te.audits("update", "submission") != 1 {
		t.Fatalf("audit: create=%d update=%d", te.audits("create", "submission"), te.audits("update", "submission"))
	}

	// the teacher grades it; the student then sees grade + feedback and can no longer upload
	var subID string
	te.pool.QueryRow(context.Background(), `SELECT id FROM submissions WHERE assignment_id=$1`, aid).Scan(&subID)
	if r, _ := te.do("PUT", "/api/teacher/submissions/"+subID+"/grade", `{"grade":4.5,"feedback":"Zo'r"}`, te.cookieA); r.StatusCode != 200 {
		t.Fatalf("grade: %d", r.StatusCode)
	}
	graded := jsonMap(t, te.get("/api/student/assignments/"+aid, stu, 200))
	gs := graded["submission"].(map[string]any)
	if gs["grade"] != 4.5 || gs["feedback"] != "Zo'r" || gs["graded_at"] == nil || graded["can_submit"] != false {
		t.Fatalf("graded view: %v", graded)
	}
	r := te.uploadAnyStatus(stu, aid, pdfBytes)
	if r.status != 409 || r.code != "already_graded" {
		t.Fatalf("re-upload after grading: %d %s", r.status, r.code)
	}
	if te.submissionFile(aid, te.studentA1) != second {
		t.Fatal("graded submission must not change")
	}
}

type rawResult struct {
	status int
	code   string
}

// uploadAnyStatus uploads a PDF and reports whatever status/error code comes back.
func (te *tEnv) uploadAnyStatus(cookie, aid string, content []byte) rawResult {
	te.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "x.pdf")
	fw.Write(content)
	w.Close()
	req := httptest.NewRequest("PUT", "/api/student/assignments/"+aid+"/submission", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Cookie", "session="+cookie)
	resp, err := te.app.Test(req, -1)
	if err != nil {
		te.t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	code := ""
	if resp.StatusCode >= 400 {
		code = errCode(te.t, string(b))
	}
	return rawResult{resp.StatusCode, code}
}

func TestStudentDeadlineEnforcedByServer(t *testing.T) {
	te := newTeacherEnv(t)
	stu := te.mustLogin("alpha1")
	aid := te.newAssignment("Muddati o'tgan", "2099-01-01T10:00")
	ctx := context.Background()

	// an early submission exists, then the deadline passes
	te.studentUpload(stu, aid, "a.pdf", pdfBytes, 200)
	before := te.submissionFile(aid, te.studentA1)
	te.pool.Exec(ctx, `UPDATE assignments SET deadline = now() - interval '1 minute' WHERE id=$1`, aid)

	r := te.uploadAnyStatus(stu, aid, pdfBytes)
	if r.status != 409 || r.code != "deadline_passed" {
		t.Fatalf("re-upload after deadline: %d %s", r.status, r.code)
	}
	if te.submissionFile(aid, te.studentA1) != before {
		t.Fatal("submission must be unchanged after the deadline")
	}

	// a student who never submitted cannot create one after the deadline either
	other := te.mustLogin("alpha2")
	r = te.uploadAnyStatus(other, aid, pdfBytes)
	if r.status != 409 || r.code != "deadline_passed" {
		t.Fatalf("create after deadline: %d %s", r.status, r.code)
	}
	var n int
	te.pool.QueryRow(ctx, `SELECT count(*) FROM submissions WHERE student_id=$1`, te.studentA2).Scan(&n)
	if n != 0 {
		t.Fatal("no submission may be created after the deadline")
	}

	// the API tells the UI the window is closed
	it := jsonMap(t, te.get("/api/student/assignments/"+aid, other, 200))
	if it["overdue"] != true || it["can_submit"] != false {
		t.Fatalf("overdue flags: %v", it)
	}
	// the boundary itself: a deadline equal to "now" is closed (now >= deadline)
	te.pool.Exec(ctx, `UPDATE assignments SET deadline = now() WHERE id=$1`, aid)
	if r := te.uploadAnyStatus(other, aid, pdfBytes); r.status != 409 {
		t.Fatalf("deadline == now must be closed, got %d", r.status)
	}
}

func TestStudentCannotSeeOtherGroupsAssignment(t *testing.T) {
	te := newTeacherEnv(t)
	aid := te.newAssignment("Faqat A guruhi", "2099-01-01T10:00")
	alpha, beta := te.mustLogin("alpha1"), te.mustLogin("beta1")
	te.studentUpload(alpha, aid, "a.pdf", pdfBytes, 200)

	// the other group's student: 404 on everything, nothing listed
	te.get("/api/student/assignments/"+aid, beta, 404)
	te.get("/api/student/assignments/00000000-0000-0000-0000-000000000000", alpha, 404)
	te.get("/api/student/assignments/not-a-uuid", alpha, 404)
	te.studentUpload(beta, aid, "a.pdf", pdfBytes, 404)
	if items := jsonMap(t, te.get("/api/student/assignments", beta, 200))["items"].([]any); len(items) != 0 {
		t.Fatalf("beta sees %d assignments", len(items))
	}
	te.get("/api/files/assignments/"+aid, beta, 404)

	// submissions are private: not even a classmate may download them
	var subID string
	te.pool.QueryRow(context.Background(), `SELECT id FROM submissions WHERE assignment_id=$1`, aid).Scan(&subID)
	classmate := te.mustLogin("alpha2")
	for name, c := range map[string]struct {
		cookie string
		want   int
	}{"owner": {alpha, 200}, "classmate": {classmate, 404}, "other group": {beta, 404}, "teacher": {te.cookieA, 200}, "other teacher": {te.cookieB, 404}, "admin": {te.admin, 200}} {
		if r, _ := te.do("GET", "/api/files/submissions/"+subID, "", c.cookie); r.StatusCode != c.want {
			t.Errorf("%s: got %d want %d", name, r.StatusCode, c.want)
		}
	}
	// staff roles are not students
	te.get("/api/student/assignments", te.cookieA, 403)
	te.get("/api/student/today", te.admin, 403)
}

// ---- 4.1: today view with injected time ----

func (te *tEnv) lastWorkday() time.Time {
	d := te.today()
	for d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

func (te *tEnv) at(day time.Time, h, m int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, te.loc)
}

func (te *tEnv) studentUser(id string) db.User {
	u, err := db.New(te.pool).GetUser(context.Background(), uuid.MustParse(id))
	if err != nil {
		te.t.Fatal(err)
	}
	return u
}

func TestStudentTodayView(t *testing.T) {
	te := newTeacherEnv(t)
	ctx := context.Background()
	svc := service.NewStudentService(te.pool, nil, te.loc, service.NewReportService(te.pool, te.loc), 5)
	day := te.lastWorkday()

	var loc, sched string
	te.pool.QueryRow(ctx, `INSERT INTO locations (name, polygon, location_url) VALUES ('Shifoxona','[[1,1],[1,2],[2,2]]','https://maps.example/x') RETURNING id`).Scan(&loc)
	te.pool.QueryRow(ctx, `INSERT INTO schedules (start_date, end_date, group_id, location_id, monday, tuesday, wednesday, thursday, friday, saturday)
		VALUES ($1,$1,$2,$3,true,true,true,true,true,true) RETURNING id`, ds(day), te.groupA, loc).Scan(&sched)
	a1 := te.studentUser(te.studentA1)

	view := func(u db.User, h, m int) service.TodayView {
		v, err := svc.TodayAt(ctx, u, te.at(day, h, m))
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	states := func(v service.TodayView) string {
		var parts []string
		for _, s := range v.Steps {
			p := s.State
			if s.Current {
				p += "*"
			}
			parts = append(parts, p)
		}
		return strings.Join(parts, ",")
	}

	// no attendance yet
	v := view(a1, 7, 30)
	if v.State != "before_start" || v.Shift != 1 || states(v) != "pending,pending,pending" || v.Schedule == nil ||
		v.Schedule.LocationName != "Shifoxona" || v.Schedule.LocationURL != "https://maps.example/x" {
		t.Fatalf("07:30: %+v", v)
	}
	if v := view(a1, 8, 0); v.State != "active" || states(v) != "pending*,pending,pending" || v.Steps[0].Start != "08:00" || v.Steps[0].End != "10:00" || v.Steps[0].MaxAttempts != 5 {
		t.Fatalf("08:00: %+v", v)
	}
	if v := view(a1, 12, 30); v.Shift != 1 || states(v) != "missed,missed,pending*" {
		t.Fatalf("12:30: %+v", v) // nothing confirmed: earlier steps of the shift are missed
	}
	if v := view(a1, 13, 0); v.Shift != 2 || states(v) != "pending*,pending,pending" || v.Steps[2].Start != "16:00" {
		t.Fatalf("13:00 starts shift 2: %+v", v)
	}
	if v := view(a1, 18, 0); v.State != "finished" || len(v.Steps) != 0 {
		t.Fatalf("18:00 without attendance: %+v", v)
	}

	// attendance in shift 1 with step 1 confirmed after two attempts
	var att, step string
	te.pool.QueryRow(ctx, `INSERT INTO attendances (student_id, schedule_id, shift, date) VALUES ($1,$2,1,$3) RETURNING id`, te.studentA1, sched, ds(day)).Scan(&att)
	te.pool.QueryRow(ctx, `INSERT INTO attendance_steps (attendance_id, step, success) VALUES ($1,1,true) RETURNING id`, att).Scan(&step)
	te.pool.Exec(ctx, `INSERT INTO attendance_attempts (step_id) VALUES ($1), ($1)`, step)

	v = view(a1, 11, 0)
	if v.Shift != 1 || v.ShiftLocked || states(v) != "done,pending*,pending" || v.Steps[0].Attempts != 2 {
		t.Fatalf("11:00 with attendance: %+v", v)
	}
	// during shift-2 hours the day is locked to shift 1: nothing is "current"
	v = view(a1, 14, 30)
	if v.Shift != 1 || !v.ShiftLocked || states(v) != "done,missed,missed" {
		t.Fatalf("14:30 locked: %+v", v)
	}
	if v := view(a1, 18, 30); v.State != "finished" || states(v) != "done,missed,missed" {
		t.Fatalf("18:30 with attendance: %+v", v)
	}

	// students with no schedule (other group) or a day off
	if v := view(te.studentUser(te.studentB1), 9, 0); v.State != "no_schedule" || v.Schedule != nil || len(v.Steps) != 0 {
		t.Fatalf("no schedule: %+v", v)
	}
	// a personal schedule covering the day overrides the group's, even if it has the weekday off
	a2 := te.studentUser(te.studentA2)
	te.pool.Exec(ctx, `INSERT INTO schedules (start_date, end_date, user_id, location_id) VALUES ($1,$1,$2,$3)`, ds(day), te.studentA2, loc)
	if v := view(a2, 9, 0); v.State != "no_schedule" {
		t.Fatalf("personal override: %+v", v)
	}
}

func TestStudentTodayEndpointAndCalendar(t *testing.T) {
	te := newTeacherEnv(t)
	ctx := context.Background()
	stu := te.mustLogin("alpha1")
	day := te.lastWorkday()
	var loc, sched, att string
	te.pool.QueryRow(ctx, `INSERT INTO locations (name, polygon) VALUES ('l','[[1,1],[1,2],[2,2]]') RETURNING id`).Scan(&loc)
	te.pool.QueryRow(ctx, `INSERT INTO schedules (start_date, end_date, group_id, location_id, monday, tuesday, wednesday, thursday, friday, saturday)
		VALUES ($1,$2,$3,$4,true,true,true,true,true,true) RETURNING id`, ds(day.AddDate(0, 0, -10)), ds(day), te.groupA, loc).Scan(&sched)

	today := jsonMap(t, te.get("/api/student/today", stu, 200))
	if today["date"] == nil || today["server_time"] == nil || today["steps"] == nil {
		t.Fatalf("today: %v", today)
	}

	// calendar for the month of `day`: one full day, one partial, the rest absent up to the schedule end
	te.pool.QueryRow(ctx, `INSERT INTO attendances (student_id, schedule_id, shift, date) VALUES ($1,$2,2,$3) RETURNING id`, te.studentA1, sched, ds(day)).Scan(&att)
	for s := 1; s <= 3; s++ {
		te.pool.Exec(ctx, `INSERT INTO attendance_steps (attendance_id, step, success) VALUES ($1,$2,true)`, att, s)
	}
	month := day.Format("2006-01")
	cal := jsonMap(t, te.get("/api/student/attendance?month="+month, stu, 200))
	cells := cal["cells"].(map[string]any)
	c := cells[ds(day)].(map[string]any)
	if c["status"] != "full" || c["shift"] != 2.0 || cal["month"] != month {
		t.Fatalf("calendar cell: %v", c)
	}
	tot := cal["totals"].(map[string]any)
	if tot["full"] != 1.0 || tot["expected"].(float64) < 1 {
		t.Fatalf("totals: %v", tot)
	}
	for _, d := range cal["dates"].([]any) {
		if dt, _ := time.Parse("2006-01-02", d.(string)); dt.Weekday() == time.Sunday {
			if _, ok := cells[d.(string)]; ok {
				t.Fatalf("sunday %s must be empty", d)
			}
		}
	}
	// only the student's own data: beta1 sees nothing of alpha1's attendance
	other := jsonMap(t, te.get("/api/student/attendance?month="+month, te.mustLogin("beta1"), 200))
	if len(other["cells"].(map[string]any)) != 0 {
		t.Fatalf("beta cells: %v", other["cells"])
	}
	te.get("/api/student/attendance?month=2026-13", stu, 422)
	te.get("/api/student/attendance?month=oops", stu, 422)
}

// ---- 4.4 profile ----

func TestStudentProfileHidesPassportAndServesPhoto(t *testing.T) {
	te := newTeacherEnv(t)
	ctx := context.Background()
	stu := te.mustLogin("alpha1")
	te.pool.Exec(ctx, `UPDATE users SET passport_pin='12345678901234', passport_number='AB1234567', phone='+998901112233', birth_date='2004-05-06' WHERE id=$1`, te.studentA1)

	resp, body := te.do("GET", "/api/student/profile", "", stu)
	if resp.StatusCode != 200 || strings.Contains(body, "12345678901234") || strings.Contains(body, "AB1234567") || strings.Contains(body, "passport") || strings.Contains(body, "$2a$") {
		t.Fatalf("profile leaks sensitive data: %d %s", resp.StatusCode, body)
	}
	p := jsonMap(t, []byte(body))
	if p["full_name"] != "alpha1" || p["group_name"] != "A-1" || p["phone"] != "+998901112233" || p["photo_url"] != "" {
		t.Fatalf("profile: %v", p)
	}

	// a stored photo is served only to the student, an admin and the group's teacher
	dir := filepath.Join(te.cfg.UploadDir, "avatars", "2026", "01")
	os.MkdirAll(dir, 0o750)
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 16)...)
	os.WriteFile(filepath.Join(dir, "a.png"), png, 0o640)
	te.pool.Exec(ctx, `UPDATE users SET image='avatars/2026/01/a.png' WHERE id=$1`, te.studentA1)
	p = jsonMap(t, te.get("/api/student/profile", stu, 200))
	url, _ := p["photo_url"].(string)
	if url != "/api/files/avatars/"+te.studentA1 {
		t.Fatalf("photo_url: %q", url)
	}
	for name, c := range map[string]struct {
		cookie string
		want   int
	}{"self": {stu, 200}, "admin": {te.admin, 200}, "teacher": {te.cookieA, 200}, "other teacher": {te.cookieB, 404}, "classmate": {te.mustLogin("alpha2"), 404}} {
		r, _ := te.do("GET", url, "", c.cookie)
		if r.StatusCode != c.want {
			t.Errorf("%s: got %d want %d", name, r.StatusCode, c.want)
		}
		if c.want == 200 && (r.Header.Get("Content-Type") != "image/png" || !strings.HasPrefix(r.Header.Get("Content-Disposition"), "inline")) {
			t.Errorf("%s: headers %v", name, r.Header)
		}
	}
}

func (e *env) audits(action, entity string) int {
	var n int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE action=$1 AND entity_type=$2`, action, entity).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}
