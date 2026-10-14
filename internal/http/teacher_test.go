package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"pc/internal/config"
)

var pdfBytes = []byte("%PDF-1.4\n1 0 obj<<>>endobj\n%%EOF")

type tEnv struct {
	*env
	groupA, groupB       string // group ids
	teachA, teachB       string // user ids
	cookieA, cookieB     string
	admin                string
	studentA1, studentA2 string
	studentB1            string
	loc                  *time.Location
}

// newTeacherEnv builds: two teachers, each with one group and students; student logins enabled.
func newTeacherEnv(t *testing.T, mutate ...func(*config.Config)) *tEnv {
	e := newEnv(t, append([]func(*config.Config){func(c *config.Config) { c.DevStudentPasswordLogin = true }}, mutate...)...)
	te := &tEnv{env: e, loc: e.cfg.Location}
	ctx := context.Background()
	te.teachA = e.user("teachA", "teacher", false).ID.String()
	te.teachB = e.user("teachB", "teacher", false).ID.String()
	e.user("adm", "admin", false)
	e.pool.QueryRow(ctx, `INSERT INTO groups (name, teacher_id) VALUES ('A-1', $1) RETURNING id`, te.teachA).Scan(&te.groupA)
	e.pool.QueryRow(ctx, `INSERT INTO groups (name, teacher_id) VALUES ('B-1', $1) RETURNING id`, te.teachB).Scan(&te.groupB)
	mk := func(name, group string) string {
		u := e.user(name, "student", false)
		e.pool.Exec(ctx, `UPDATE users SET group_id=$2, full_name=$3 WHERE id=$1`, u.ID, group, name)
		return u.ID.String()
	}
	te.studentA1, te.studentA2, te.studentB1 = mk("alpha1", te.groupA), mk("alpha2", te.groupA), mk("beta1", te.groupB)
	te.cookieA, te.cookieB, te.admin = e.mustLogin("teachA"), e.mustLogin("teachB"), e.mustLogin("adm")
	return te
}

func (te *env) get(path, cookie string, want int) []byte {
	te.t.Helper()
	resp, body := te.do("GET", path, "", cookie)
	if resp.StatusCode != want {
		te.t.Fatalf("GET %s: got %d want %d: %.200s", path, resp.StatusCode, want, body)
	}
	return []byte(body)
}

func jsonMap(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("bad json %q", b)
	}
	return m
}

// multipart sends a multipart form; files maps field -> (filename, content).
func (te *tEnv) multipart(method, path, cookie string, fields map[string]string, file *[2]any, wantStatus int) map[string]any {
	te.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		w.WriteField(k, v)
	}
	if file != nil {
		fw, _ := w.CreateFormFile("file", file[0].(string))
		fw.Write(file[1].([]byte))
	}
	w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Cookie", "session="+cookie)
	resp, err := te.app.Test(req, -1)
	if err != nil {
		te.t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		te.t.Fatalf("%s %s: got %d want %d: %.300s", method, path, resp.StatusCode, wantStatus, b)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func (te *tEnv) today() time.Time {
	n := time.Now().In(te.loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

func ds(t time.Time) string { return t.Format("2006-01-02") }

// ---- 3.8: another teacher's group is a 404 everywhere ----

func TestTeacherCannotAccessOtherTeachersData(t *testing.T) {
	te := newTeacherEnv(t)
	a := te.multipart("POST", "/api/teacher/assignments", te.cookieA, map[string]string{
		"group_id": te.groupA, "title": "Hisobot", "deadline": "2099-01-01T10:00"}, nil, 201)
	aid := a["id"].(string)

	own := []string{
		"/api/teacher/groups/" + te.groupA,
		"/api/teacher/groups/" + te.groupA + "/report",
		"/api/teacher/groups/" + te.groupA + "/report.xlsx",
		"/api/teacher/assignments/" + aid,
		"/api/teacher/assignments/" + aid + "/submissions",
		"/api/teacher/assignments/" + aid + "/grades.xlsx",
	}
	for _, p := range own {
		te.get(p, te.cookieA, 200)
		if resp, _ := te.do("GET", p, "", te.cookieB); resp.StatusCode != 404 {
			t.Errorf("teacher B on %s: got %d want 404 (not 403)", p, resp.StatusCode)
		}
	}
	// unknown / malformed ids look the same
	te.get("/api/teacher/groups/00000000-0000-0000-0000-000000000000", te.cookieA, 404)
	te.get("/api/teacher/groups/not-a-uuid", te.cookieA, 404)

	// mutations on someone else's assignment are 404 too
	te.multipart("PUT", "/api/teacher/assignments/"+aid, te.cookieB, map[string]string{"title": "x", "deadline": "2099-01-01T10:00"}, nil, 404)
	if resp, _ := te.do("DELETE", "/api/teacher/assignments/"+aid, "", te.cookieB); resp.StatusCode != 404 {
		t.Errorf("delete by other teacher: %d", resp.StatusCode)
	}
	// creating an assignment for another teacher's group is refused
	te.multipart("POST", "/api/teacher/assignments", te.cookieB, map[string]string{"group_id": te.groupA, "title": "x", "deadline": "2099-01-01T10:00"}, nil, 422)

	// the group list only holds my groups
	list := jsonMap(t, te.get("/api/teacher/groups", te.cookieB, 200))["items"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["id"] != te.groupB {
		t.Fatalf("group list: %v", list)
	}
	// admins and students are not teachers
	te.get("/api/teacher/groups", te.admin, 403)
}

// ---- 3.3 / 3.8: expected-days computation in reports ----

func TestReportExpectedDaysAndStatuses(t *testing.T) {
	te := newTeacherEnv(t)
	ctx := context.Background()
	var loc string
	te.pool.QueryRow(ctx, `INSERT INTO locations (name, polygon) VALUES ('l','[[1,1],[1,2],[2,2]]') RETURNING id`).Scan(&loc)

	today := te.today()
	start := today.AddDate(0, 0, -20)
	end := today.AddDate(0, 0, -1) // schedule ends yesterday
	// Monday-Friday group schedule over [start, end].
	var sched string
	te.pool.QueryRow(ctx, `INSERT INTO schedules (start_date, end_date, group_id, location_id, monday, tuesday, wednesday, thursday, friday)
		VALUES ($1,$2,$3,$4,true,true,true,true,true) RETURNING id`, ds(start), ds(end), te.groupA, loc).Scan(&sched)

	// independent expected-day count: Mon-Fri inside [start, end]
	var workdays []time.Time
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			workdays = append(workdays, d)
		}
	}
	att := func(student string, day time.Time, ok int) {
		var aid string
		te.pool.QueryRow(ctx, `INSERT INTO attendances (student_id, schedule_id, shift, date) VALUES ($1,$2,2,$3) RETURNING id`, student, sched, ds(day)).Scan(&aid)
		for step := 1; step <= 3; step++ {
			te.pool.Exec(ctx, `INSERT INTO attendance_steps (attendance_id, step, success) VALUES ($1,$2,$3)`, aid, step, step <= ok)
		}
	}
	att(te.studentA1, workdays[0], 3) // full
	att(te.studentA1, workdays[1], 2) // partial
	// a weekend attendance still shows (they came), but is not "expected"
	var weekend time.Time
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Sunday {
			weekend = d
			break
		}
	}
	att(te.studentA1, weekend, 3)

	// query the range through tomorrow: future days must not count
	to := today.AddDate(0, 0, 1)
	rep := jsonMap(t, te.get("/api/teacher/groups/"+te.groupA+"/report?from="+ds(start)+"&to="+ds(to), te.cookieA, 200))
	students := rep["students"].([]any)
	if len(students) != 2 {
		t.Fatalf("students: %d", len(students))
	}
	byName := map[string]map[string]any{}
	for _, s := range students {
		m := s.(map[string]any)
		byName[m["full_name"].(string)] = m
	}
	a1 := byName["alpha1"]["totals"].(map[string]any)
	n := float64(len(workdays))
	// expected = workdays + the Sunday where he attended anyway
	if a1["expected"] != n+1 || a1["full"] != 2.0 || a1["partial"] != 1.0 || a1["absent"] != n-2 {
		t.Fatalf("alpha1 totals %v (workdays %v)", a1, n)
	}
	a2 := byName["alpha2"]["totals"].(map[string]any)
	if a2["expected"] != n || a2["absent"] != n || a2["full"] != 0.0 || a2["percent"] != 0.0 {
		t.Fatalf("alpha2 totals %v", a2)
	}
	cells := byName["alpha1"]["cells"].(map[string]any)
	first := cells[ds(workdays[0])].(map[string]any)
	if first["status"] != "full" || first["shift"] != 2.0 {
		t.Fatalf("first cell: %v", first)
	}
	if _, ok := cells[ds(to)]; ok {
		t.Fatal("future date must have no cell")
	}
	if _, ok := cells[ds(today)]; ok {
		t.Fatal("today is after the schedule end: not a working day")
	}
	// a Saturday inside the range is not scheduled
	for _, d := range rep["dates"].([]any) {
		if dt, _ := time.Parse("2006-01-02", d.(string)); dt.Weekday() == time.Saturday {
			if _, ok := cells[d.(string)]; ok {
				t.Fatalf("saturday %s must be empty", d)
			}
		}
	}

	// a personal schedule overrides the group's on the dates it covers: here it covers one
	// workday with no weekdays enabled, so that day is no longer expected for alpha2.
	day := workdays[len(workdays)-1]
	te.pool.Exec(ctx, `INSERT INTO schedules (start_date, end_date, user_id, location_id) VALUES ($1,$1,$2,$3)`, ds(day), te.studentA2, loc)
	rep = jsonMap(t, te.get("/api/teacher/groups/"+te.groupA+"/report?from="+ds(start)+"&to="+ds(end), te.cookieA, 200))
	for _, s := range rep["students"].([]any) {
		m := s.(map[string]any)
		if m["full_name"] == "alpha2" {
			if tot := m["totals"].(map[string]any); tot["expected"] != n-1 {
				t.Fatalf("personal override: alpha2 expected %v, want %v", tot["expected"], n-1)
			}
		}
	}
}

func TestReportRangeValidation(t *testing.T) {
	te := newTeacherEnv(t)
	base := "/api/teacher/groups/" + te.groupA + "/report"
	for _, q := range []string{"?from=bad", "?from=2026-09-10&to=2026-09-01", "?from=2026-01-01&to=2026-12-31"} {
		te.get(base+q, te.cookieA, 422)
	}
	// defaults to today
	rep := jsonMap(t, te.get(base, te.cookieA, 200))
	if rep["from"] != ds(te.today()) || rep["to"] != ds(te.today()) {
		t.Fatalf("default range: %v %v", rep["from"], rep["to"])
	}
	// the admin report is per group: group_id is required
	te.get("/api/admin/reports/attendance", te.admin, 422)
	te.get("/api/admin/reports/attendance.xlsx", te.admin, 422)
	te.get("/api/admin/reports/attendance?group_id=bad", te.admin, 422)
	te.get("/api/admin/reports/attendance?group_id="+te.groupA+"&from=2026-01-01&to=2026-12-31", te.admin, 422)
}

// ---- 3.5 Excel export ----

// headerRow finds the table header (the title block sits above it).
func headerRow(rows [][]string) int {
	for i, r := range rows {
		if len(r) > 1 && r[0] == "T/r" && r[1] == "F.I.Sh." {
			return i
		}
	}
	return -1
}

func TestReportExcelExportIsAudited(t *testing.T) {
	te := newTeacherEnv(t)
	today := ds(te.today())
	body := te.get("/api/teacher/groups/"+te.groupA+"/report.xlsx?from="+today+"&to="+today, te.cookieA, 200)
	f, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("not a valid xlsx: %v", err)
	}
	rows, _ := f.GetRows("Davomat")
	h := headerRow(rows)
	if h < 0 || rows[h][len(rows[h])-1] != "Foiz" || rows[0][0] != "Davomat hisoboti" {
		t.Fatalf("headers/rows: %v", rows)
	}
	if rows[h+1][1] != "alpha1" || rows[h+2][1] != "alpha2" {
		t.Fatalf("student rows: %v", rows)
	}
	if te.auditCount("export") != 1 {
		t.Fatal("export not audited")
	}
	// admin export of one group
	body = te.get("/api/admin/reports/attendance.xlsx?group_id="+te.groupB+"&from="+today+"&to="+today, te.admin, 200)
	f, _ = excelize.OpenReader(bytes.NewReader(body))
	rows, _ = f.GetRows("Davomat")
	h = headerRow(rows)
	if h < 0 || len(rows) < h+2 || rows[h+1][1] != "beta1" {
		t.Fatalf("admin export: %v", rows)
	}
	if te.auditCount("export") != 2 {
		t.Fatal("admin export not audited")
	}
}

// ---- attempt details are for admins only; teachers only see attendance ----

func TestAttemptDetailsAreAdminOnly(t *testing.T) {
	te := newTeacherEnv(t)
	ctx := context.Background()
	var loc, sched, att, step, attempt string
	te.pool.QueryRow(ctx, `INSERT INTO locations (name, polygon) VALUES ('l','[[1,1],[1,2],[2,2]]') RETURNING id`).Scan(&loc)
	te.pool.QueryRow(ctx, `INSERT INTO schedules (start_date, end_date, group_id, location_id, monday) VALUES (current_date-5, current_date+5, $1, $2, true) RETURNING id`, te.groupA, loc).Scan(&sched)
	day := ds(te.today())
	te.pool.QueryRow(ctx, `INSERT INTO attendances (student_id, schedule_id, shift, date) VALUES ($1,$2,1,$3) RETURNING id`, te.studentA1, sched, day).Scan(&att)
	te.pool.QueryRow(ctx, `INSERT INTO attendance_steps (attendance_id, step, success) VALUES ($1,1,true) RETURNING id`, att).Scan(&step)
	te.pool.QueryRow(ctx, `INSERT INTO attendance_attempts (step_id, image, face_verified, location_verified, liveness_verified, success, face_distance, face_threshold, antispoof_score, gps_accuracy)
		VALUES ($1,'selfies/2026/01/x.jpg',true,true,true,true,0.2,0.3,0.99,8) RETURNING id`, step).Scan(&attempt)

	// the admin sees steps and attempts
	out := jsonMap(t, te.get("/api/admin/students/"+te.studentA1+"/attempts?from="+day+"&to="+day, te.admin, 200))
	days := out["days"].([]any)
	steps := days[0].(map[string]any)["steps"].([]any)
	first := steps[0].(map[string]any)["attempts"].([]any)[0].(map[string]any)
	if len(days) != 1 || first["face_distance"] != 0.2 || first["antispoof_score"] != 0.99 || first["has_selfie"] != true {
		t.Fatalf("admin view: %v", out)
	}

	// a teacher gets nothing: not for their own student, not for anyone else's, not via any route
	for _, path := range []string{
		"/api/teacher/students/" + te.studentA1 + "/attempts",
		"/api/teacher/students/" + te.studentA1 + "/attempts?from=" + day + "&to=" + day,
		"/api/teacher/groups/" + te.groupA + "/attempts",
		"/api/admin/students/" + te.studentA1 + "/attempts",
		"/api/admin/attempts",
	} {
		for name, cookie := range map[string]string{"own teacher": te.cookieA, "other teacher": te.cookieB} {
			if r, _ := te.do("GET", path, "", cookie); r.StatusCode != 404 && r.StatusCode != 403 {
				t.Errorf("%s on %s: got %d, want 403/404", name, path, r.StatusCode)
			}
		}
	}
	// ...and not the selfie either
	te.get("/api/files/selfies/"+attempt, te.cookieA, 404)

	// attendance itself stays available to the teacher, without attempt data
	rep := te.get("/api/teacher/groups/"+te.groupA+"/report?from="+day+"&to="+day, te.cookieA, 200)
	if strings.Contains(string(rep), "face_distance") || strings.Contains(string(rep), "attempts") {
		t.Fatalf("the report must not carry attempt data: %.300s", rep)
	}
	grp := te.get("/api/teacher/groups/"+te.groupA, te.cookieA, 200)
	if strings.Contains(string(grp), "face_distance") || strings.Contains(string(grp), "attempts") {
		t.Fatalf("the group page must not carry attempt data: %.300s", grp)
	}
}

// ---- 3.6 / 3.7 assignments, files, grading ----

func TestAssignmentsFilesAndGrading(t *testing.T) {
	te := newTeacherEnv(t)
	ctx := context.Background()
	form := func(title string) map[string]string {
		return map[string]string{"group_id": te.groupA, "title": title, "description": "tavsif", "deadline": "2099-05-01T09:30"}
	}

	// file validation: content decides, not the name
	te.multipart("POST", "/api/teacher/assignments", te.cookieA, form("bad"), &[2]any{"x.pdf", []byte("MZ\x90\x00 not a pdf")}, 422)
	te.multipart("POST", "/api/teacher/assignments", te.cookieA, map[string]string{"group_id": te.groupA, "title": "", "deadline": "x"}, nil, 422)

	a := te.multipart("POST", "/api/teacher/assignments", te.cookieA, form("Amaliy ish 1"), &[2]any{"vazifa.pdf", pdfBytes}, 201)
	aid := a["id"].(string)
	// the deadline is entered in the business timezone (+05:00): 09:30 local = 04:30 UTC
	if dl, err := time.Parse(time.RFC3339, a["deadline"].(string)); a["has_file"] != true || err != nil || !dl.Equal(time.Date(2099, 5, 1, 4, 30, 0, 0, time.UTC)) {
		t.Fatalf("assignment: %v", a)
	}
	var path string
	te.pool.QueryRow(ctx, `SELECT file FROM assignments WHERE id=$1`, aid).Scan(&path)
	if !strings.HasPrefix(path, "assignments/") || !strings.HasSuffix(path, ".pdf") {
		t.Fatalf("stored path: %s", path)
	}
	if _, err := os.Stat(filepath.Join(te.cfg.UploadDir, path)); err != nil {
		t.Fatalf("file not on disk: %v", err)
	}

	// downloads go through the permission check
	fileURL := "/api/files/assignments/" + aid
	resp, body := te.do("GET", fileURL, "", te.cookieA)
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "application/pdf" || body != string(pdfBytes) ||
		resp.Header.Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(resp.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("owner download: %d %v", resp.StatusCode, resp.Header)
	}
	stuA, stuB := te.mustLogin("alpha1"), te.mustLogin("beta1")
	for name, c := range map[string]struct {
		cookie string
		want   int
	}{"admin": {te.admin, 200}, "student of group": {stuA, 200}, "student of other group": {stuB, 404}, "other teacher": {te.cookieB, 404}, "anonymous": {"", 401}} {
		if r, _ := te.do("GET", fileURL, "", c.cookie); r.StatusCode != c.want {
			t.Errorf("%s: got %d want %d", name, r.StatusCode, c.want)
		}
	}
	if r, _ := te.do("GET", "/api/files/unknown/"+aid, "", te.cookieA); r.StatusCode != 404 {
		t.Errorf("unknown kind: %d", r.StatusCode)
	}

	// replacing the file deletes the old one; removing clears it
	te.multipart("PUT", "/api/teacher/assignments/"+aid, te.cookieA, form("Amaliy ish 1 (yangi)"), &[2]any{"v2.pdf", pdfBytes}, 200)
	if _, err := os.Stat(filepath.Join(te.cfg.UploadDir, path)); !os.IsNotExist(err) {
		t.Fatal("old file must be deleted on replace")
	}
	f := form("Amaliy ish 1 (yangi)")
	f["remove_file"] = "true"
	if m := te.multipart("PUT", "/api/teacher/assignments/"+aid, te.cookieA, f, nil, 200); m["has_file"] != false {
		t.Fatalf("remove_file: %v", m)
	}

	// roster: nobody submitted yet
	view := jsonMap(t, te.get("/api/teacher/assignments/"+aid+"/submissions", te.cookieA, 200))
	if len(view["submitted"].([]any)) != 0 || len(view["missing"].([]any)) != 2 {
		t.Fatalf("roster: %v", view)
	}

	// alpha1 submits (done directly: student upload is phase 4)
	var sub string
	te.pool.QueryRow(ctx, `INSERT INTO submissions (assignment_id, student_id, file) VALUES ($1,$2,'') RETURNING id`, aid, te.studentA1).Scan(&sub)
	view = jsonMap(t, te.get("/api/teacher/assignments/"+aid+"/submissions", te.cookieA, 200))
	if len(view["submitted"].([]any)) != 1 || len(view["missing"].([]any)) != 1 {
		t.Fatalf("roster after submit: %v", view)
	}

	// grading rules: 1.0-5.0 in 0.5 steps
	gradeURL := "/api/teacher/submissions/" + sub + "/grade"
	put := func(cookie, body string, want int) map[string]any {
		resp, out := te.do("PUT", gradeURL, body, cookie)
		if resp.StatusCode != want {
			t.Fatalf("grade %s: got %d want %d: %s", body, resp.StatusCode, want, out)
		}
		return jsonMap(t, []byte(out))
	}
	for _, bad := range []string{`{"grade":0.5}`, `{"grade":5.5}`, `{"grade":4.3}`, `{"grade":0}`, `{"grade":"x"}`} {
		resp, _ := te.do("PUT", gradeURL, bad, te.cookieA)
		if resp.StatusCode != 422 && resp.StatusCode != 400 {
			t.Errorf("grade %s accepted: %d", bad, resp.StatusCode)
		}
	}
	put(te.cookieB, `{"grade":4}`, 404) // not their group
	got := put(te.cookieA, `{"grade":4.5,"feedback":"Yaxshi ish"}`, 200)
	if got["grade"] != 4.5 || got["feedback"] != "Yaxshi ish" || got["graded_at"] == nil {
		t.Fatalf("graded: %v", got)
	}
	var stored string
	te.pool.QueryRow(ctx, `SELECT grade::text FROM submissions WHERE id=$1`, sub).Scan(&stored)
	if stored != "4.5" || te.auditCount("grade") != 1 {
		t.Fatalf("stored grade %q, grade audits %d", stored, te.auditCount("grade"))
	}
	put(te.cookieA, `{"grade":5}`, 200) // regrading is allowed
	if te.auditCount("grade") != 2 {
		t.Fatal("regrade must be audited")
	}

	// grades export
	x := te.get("/api/teacher/assignments/"+aid+"/grades.xlsx", te.cookieA, 200)
	xf, err := excelize.OpenReader(bytes.NewReader(x))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := xf.GetRows("Baholar")
	if len(rows) != 6 || rows[4][1] != "alpha1" || rows[4][2] != "Topshirilgan" || rows[4][4] != "5" || rows[5][2] != "Topshirilmagan" {
		t.Fatalf("grades sheet: %v", rows)
	}
	if te.auditCount("export") != 1 {
		t.Fatal("grades export not audited")
	}

	// deleting an assignment that has submissions is refused
	resp, out := te.do("DELETE", "/api/teacher/assignments/"+aid, "", te.cookieA)
	if resp.StatusCode != 409 || errCode(t, out) != "has_submissions" {
		t.Fatalf("delete with submissions: %d %s", resp.StatusCode, out)
	}
	te.pool.Exec(ctx, `DELETE FROM submissions WHERE id=$1`, sub)
	// ...and allowed (with the file removed) once there are none
	a2 := te.multipart("POST", "/api/teacher/assignments", te.cookieA, form("Vaqtinchalik"), &[2]any{"t.pdf", pdfBytes}, 201)
	var p2 string
	te.pool.QueryRow(ctx, `SELECT file FROM assignments WHERE id=$1`, a2["id"]).Scan(&p2)
	if r, _ := te.do("DELETE", "/api/teacher/assignments/"+a2["id"].(string), "", te.cookieA); r.StatusCode != 204 {
		t.Fatalf("delete: %d", r.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(te.cfg.UploadDir, p2)); !os.IsNotExist(err) {
		t.Fatal("file must be deleted with the assignment")
	}
	if te.auditCount("delete") != 1 || te.auditCount("create") < 3 {
		t.Fatalf("audits: delete=%d create=%d", te.auditCount("delete"), te.auditCount("create"))
	}
}

func TestGroupOverviewShowsTodayStatus(t *testing.T) {
	te := newTeacherEnv(t)
	ctx := context.Background()
	var loc, sched, att string
	te.pool.QueryRow(ctx, `INSERT INTO locations (name, polygon) VALUES ('l','[[1,1],[1,2],[2,2]]') RETURNING id`).Scan(&loc)
	te.pool.QueryRow(ctx, `INSERT INTO schedules (start_date, end_date, group_id, location_id, monday, tuesday, wednesday, thursday, friday, saturday)
		VALUES (current_date-5, current_date+5, $1, $2, true,true,true,true,true,true) RETURNING id`, te.groupA, loc).Scan(&sched)
	te.pool.QueryRow(ctx, `INSERT INTO attendances (student_id, schedule_id, shift, date) VALUES ($1,$2,1,$3) RETURNING id`, te.studentA1, sched, ds(te.today())).Scan(&att)
	te.pool.Exec(ctx, `INSERT INTO attendance_steps (attendance_id, step, success) VALUES ($1,1,true), ($1,2,false)`, att)

	if te.today().Weekday() == time.Sunday {
		t.Skip("no working day on Sunday")
	}
	g := jsonMap(t, te.get("/api/teacher/groups/"+te.groupA, te.cookieA, 200))
	today := g["today"].(map[string]any)
	if today["expected"] != 2.0 || today["partial"] != 1.0 || today["absent"] != 1.0 || today["attended_percent"] != 50.0 {
		t.Fatalf("today: %v", today)
	}
	st := g["students"].([]any)
	if st[0].(map[string]any)["status"] != "partial" || st[0].(map[string]any)["shift"] != 1.0 {
		t.Fatalf("students: %v", st)
	}
	list := jsonMap(t, te.get("/api/teacher/groups", te.cookieA, 200))["items"].([]any)
	if list[0].(map[string]any)["student_count"] != 2.0 {
		t.Fatalf("list: %v", list)
	}
}

// ---- select-box options: the whole list, id + name only ----

func TestAdminOptionsAreUnpaginated(t *testing.T) {
	te := newTeacherEnv(t)
	ctx := context.Background()
	for i := 0; i < 25; i++ { // more than the default page size
		te.pool.Exec(ctx, `INSERT INTO groups (name) VALUES ($1)`, fmt.Sprintf("extra-%02d", i))
	}
	for _, kind := range []string{"faculties", "groups", "locations", "schedules"} {
		te.get("/api/admin/options/"+kind, te.admin, 200)
	}
	rep := jsonMap(t, te.get("/api/admin/options/groups", te.admin, 200))
	items := rep["items"].([]any)
	if len(items) < 27 {
		t.Fatalf("groups options: got %d, want all", len(items))
	}
	if first := items[0].(map[string]any); len(first) != 2 || first["id"] == nil || first["name"] == nil {
		t.Fatalf("option shape: %v", first)
	}
	te.get("/api/admin/options/groups", te.cookieA, 403) // teachers have no admin routes
}
