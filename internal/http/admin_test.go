package http_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type adminEnv struct {
	*env
	cookie string
}

func newAdminEnv(t *testing.T) *adminEnv {
	e := newEnv(t)
	e.user("adm", "admin", false)
	return &adminEnv{env: e, cookie: e.mustLogin("adm")}
}

// call performs an admin request and decodes the JSON response.
func (a *adminEnv) call(method, path string, body any, wantStatus int) map[string]any {
	a.t.Helper()
	raw := ""
	if body != nil {
		b, _ := json.Marshal(body)
		raw = string(b)
	}
	resp, out := a.do(method, "/api/admin"+path, raw, a.cookie)
	if resp.StatusCode != wantStatus {
		a.t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, wantStatus, out)
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(out), &m)
	return m
}

func (a *adminEnv) audits(action, entity string) int {
	var n int
	err := a.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE action=$1 AND entity_type=$2`, action, entity).Scan(&n)
	if err != nil {
		a.t.Fatal(err)
	}
	return n
}

func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }

func TestAdminCRUDIsAudited(t *testing.T) {
	a := newAdminEnv(t)

	// faculty
	f := a.call("POST", "/faculties", map[string]any{"name": "Informatika"}, 201)
	fid := str(f, "id")
	a.call("PUT", "/faculties/"+fid, map[string]any{"name": "Informatika va AT"}, 200)
	a.call("POST", "/faculties", map[string]any{"name": "  "}, 422)

	// teacher is created with a one-time password
	tu := a.call("POST", "/users", map[string]any{"username": "Teach1", "role": "teacher", "full_name": "Ali Valiyev"}, 201)
	pw := str(tu, "password")
	teacher := tu["user"].(map[string]any)
	if len(pw) != 12 || teacher["must_change_password"] != true {
		t.Fatalf("bad teacher creation: %v", tu)
	}
	a.call("POST", "/users", map[string]any{"username": "teach1", "role": "teacher", "full_name": "Dup"}, 422)
	a.call("POST", "/users", map[string]any{"username": "stud", "role": "student", "full_name": "No"}, 422)
	if resp, _, cookie := a.login("teach1", pw); resp.StatusCode != 200 || cookie == "" {
		t.Fatal("generated password must work")
	}

	// group with faculty + teacher
	g := a.call("POST", "/groups", map[string]any{"name": "G-1", "faculty_id": fid, "teacher_id": teacher["id"]}, 201)
	gid := str(g, "id")
	a.call("POST", "/groups", map[string]any{"name": "G-2", "teacher_id": fid}, 422) // not a teacher
	a.call("PUT", "/groups/"+gid, map[string]any{"name": "G-1A", "faculty_id": fid, "teacher_id": teacher["id"]}, 200)

	// location
	loc := a.call("POST", "/locations", map[string]any{
		"name": "Maktab", "points": []string{"39.1,66.1", "39.2,66.1", "39.2,66.2"}, "tolerance_meters": 30,
	}, 201)
	lid := str(loc, "id")
	a.call("PUT", "/locations/"+lid, map[string]any{
		"name": "Maktab 1", "points": []string{"39.1,66.1", "39.2,66.1", "39.2,66.2"}, "tolerance_meters": 50,
	}, 200)

	// schedule
	sc := a.call("POST", "/schedules", map[string]any{
		"name": "Amaliyot", "start_date": "2026-09-01", "end_date": "2026-09-30",
		"group_id": gid, "location_id": lid, "monday": true, "wednesday": true,
	}, 201)
	a.call("PUT", "/schedules/"+str(sc, "id"), map[string]any{
		"name": "Amaliyot 2", "start_date": "2026-09-01", "end_date": "2026-10-30",
		"group_id": gid, "location_id": lid, "monday": true, "wednesday": true,
	}, 200)

	for _, c := range []struct{ action, entity string }{
		{"create", "faculty"}, {"update", "faculty"}, {"create", "group"}, {"update", "group"},
		{"create", "location"}, {"update", "location"}, {"create", "schedule"}, {"update", "schedule"}, {"create", "user"},
	} {
		if a.audits(c.action, c.entity) < 1 {
			t.Errorf("missing audit %s %s", c.action, c.entity)
		}
	}

	// deletes
	a.call("DELETE", "/schedules/"+str(sc, "id"), nil, 204)
	a.call("DELETE", "/locations/"+lid, nil, 204)
	a.call("DELETE", "/groups/"+gid, nil, 204)
	a.call("DELETE", "/faculties/"+fid, nil, 204)
	a.call("DELETE", "/faculties/"+fid, nil, 404)
	for _, e := range []string{"schedule", "location", "group", "faculty"} {
		if a.audits("delete", e) != 1 {
			t.Errorf("missing delete audit for %s", e)
		}
	}

	// the audit API lists them, with the update diff
	list := a.call("GET", "/audit?entity_type=faculty&action=update", nil, 200)
	items := list["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("audit list: %v", list)
	}
	ch, _ := json.Marshal(items[0].(map[string]any)["changes"])
	if !strings.Contains(string(ch), "Informatika va AT") || !strings.Contains(string(ch), `"before"`) {
		t.Fatalf("diff missing: %s", ch)
	}
	a.call("GET", "/audit/"+str(items[0].(map[string]any), "id"), nil, 200)
	a.call("GET", "/audit?from=bad", nil, 422)
}

func TestLocationPolygonValidation(t *testing.T) {
	a := newAdminEnv(t)
	cases := map[string][]string{
		"too few":     {"39.1,66.1", "39.2,66.1"},
		"not numbers": {"39.1,66.1", "abc", "39.2,66.2"},
		"range":       {"39.1,66.1", "99,66.1", "39.2,66.2"},
		"format":      {"39.1;66.1", "39.2,66.1", "39.2,66.2"},
	}
	for name, pts := range cases {
		out := a.call("POST", "/locations", map[string]any{"name": "X", "points": pts}, 422)
		fields := out["error"].(map[string]any)["fields"].(map[string]any)
		if _, ok := fields["points"]; !ok {
			t.Errorf("%s: no points error: %v", name, out)
		}
	}
	// stored as [[lat,lng],...]
	loc := a.call("POST", "/locations", map[string]any{"name": "X", "points": []string{"39.1,66.1", "39.2,66.1", "39.2,66.2"}}, 201)
	if poly := loc["polygon"].([]any); len(poly) != 3 || poly[0].([]any)[0].(float64) != 39.1 {
		t.Fatalf("polygon: %v", loc["polygon"])
	}
	if loc["tolerance_meters"].(float64) != 0 { // explicit zero is allowed when omitted
		t.Fatalf("tolerance: %v", loc["tolerance_meters"])
	}
}

func TestScheduleValidationAndOverlap(t *testing.T) {
	a := newAdminEnv(t)
	g := a.call("POST", "/groups", map[string]any{"name": "G"}, 201)
	gid := str(g, "id")
	loc := a.call("POST", "/locations", map[string]any{"name": "L", "points": []string{"1,1", "1,2", "2,2"}, "tolerance_meters": 30}, 201)
	lid := str(loc, "id")
	stu := a.user("stu1", "student", false)

	base := func(m map[string]any) map[string]any {
		b := map[string]any{"start_date": "2026-09-01", "end_date": "2026-09-30", "group_id": gid, "location_id": lid, "monday": true}
		for k, v := range m {
			b[k] = v
		}
		return b
	}
	// validation errors
	a.call("POST", "/schedules", base(map[string]any{"end_date": "2026-08-01"}), 422)   // end < start
	a.call("POST", "/schedules", base(map[string]any{"user_id": stu.ID.String()}), 422) // both targets
	a.call("POST", "/schedules", base(map[string]any{"group_id": nil}), 422)            // neither target
	a.call("POST", "/schedules", base(map[string]any{"monday": false}), 422)            // no weekday
	a.call("POST", "/schedules", base(map[string]any{"location_id": nil}), 422)         // no location
	a.call("POST", "/schedules", base(map[string]any{"start_date": "01.09.2026"}), 422) // bad date
	a.call("POST", "/schedules", base(map[string]any{"group_id": lid}), 422)            // unknown group

	first := a.call("POST", "/schedules", base(nil), 201)

	// overlapping: same group, intersecting dates, shared weekday
	out := a.call("POST", "/schedules", base(map[string]any{"start_date": "2026-09-15", "end_date": "2026-10-15", "monday": true, "friday": true}), 422)
	if out["error"].(map[string]any)["code"] != "validation_error" {
		t.Fatalf("want validation_error: %v", out)
	}
	// same dates but different weekdays: fine
	a.call("POST", "/schedules", base(map[string]any{"monday": false, "tuesday": true}), 201)
	// same weekday but later dates: fine
	a.call("POST", "/schedules", base(map[string]any{"start_date": "2026-10-01", "end_date": "2026-10-31"}), 201)
	// a personal schedule for a student in that group is a different target
	a.call("POST", "/schedules", base(map[string]any{"group_id": nil, "user_id": stu.ID.String()}), 201)
	// updating a schedule must not collide with itself
	a.call("PUT", "/schedules/"+str(first, "id"), base(map[string]any{"end_date": "2026-09-29"}), 200)
}

func TestScheduleDBConstraints(t *testing.T) {
	a := newAdminEnv(t)
	ctx := context.Background()
	var uid, gid string
	a.pool.QueryRow(ctx, `INSERT INTO groups (name) VALUES ('g') RETURNING id`).Scan(&gid)
	a.pool.QueryRow(ctx, `INSERT INTO users (username, role) VALUES ('s', 'student') RETURNING id`).Scan(&uid)

	bad := map[string]string{
		"both user and group": `INSERT INTO schedules (start_date, end_date, user_id, group_id) VALUES ('2026-01-01','2026-01-02','` + uid + `','` + gid + `')`,
		"neither":             `INSERT INTO schedules (start_date, end_date) VALUES ('2026-01-01','2026-01-02')`,
		"end before start":    `INSERT INTO schedules (start_date, end_date, group_id) VALUES ('2026-01-02','2026-01-01','` + gid + `')`,
	}
	for name, q := range bad {
		if _, err := a.pool.Exec(ctx, q); err == nil {
			t.Errorf("%s: CHECK constraint did not fire", name)
		}
	}
	if _, err := a.pool.Exec(ctx, `INSERT INTO schedules (start_date, end_date, group_id) VALUES ('2026-01-01','2026-01-01','`+gid+`')`); err != nil {
		t.Errorf("valid schedule rejected: %v", err)
	}
}

func TestUserAdminActions(t *testing.T) {
	a := newAdminEnv(t)
	stu := a.user("stu1", "student", false)
	a.pool.Exec(context.Background(), `UPDATE users SET passport_pin='12345678901234', passport_number='AB1234567' WHERE id=$1`, stu.ID)

	// the regular user payload never carries passport data or the hash
	for _, path := range []string{"/users", "/users/" + stu.ID.String()} {
		_, body := a.do("GET", "/api/admin"+path, "", a.cookie)
		if strings.Contains(body, "12345678901234") || strings.Contains(body, "AB1234567") || strings.Contains(body, "$2a$") {
			t.Fatalf("%s leaks sensitive data: %s", path, body)
		}
	}
	before := a.audits("view_sensitive", "user")
	s := a.call("GET", "/users/"+stu.ID.String()+"/sensitive", nil, 200)
	if s["passport_pin"] != "12345678901234" || s["passport_number"] != "AB1234567" {
		t.Fatalf("sensitive: %v", s)
	}
	if a.audits("view_sensitive", "user") != before+1 {
		t.Fatal("view_sensitive not audited")
	}
	var changes string
	a.pool.QueryRow(context.Background(), `SELECT changes::text FROM audit_logs WHERE action='view_sensitive'`).Scan(&changes)
	if strings.Contains(changes, "12345678901234") {
		t.Fatalf("audit leaks passport: %s", changes)
	}

	// filters
	out := a.call("GET", "/users?role=student&search=stu", nil, 200)
	if out["total"].(float64) != 1 {
		t.Fatalf("filter: %v", out)
	}
	out = a.call("GET", "/users?is_active=false", nil, 200)
	if out["total"].(float64) != 0 {
		t.Fatalf("inactive filter: %v", out)
	}

	// deactivate ends sessions and blocks login; self-deactivation is refused
	t1 := a.user("tea", "teacher", false)
	tcookie := a.mustLogin("tea")
	a.call("POST", "/users/"+t1.ID.String()+"/deactivate", nil, 200)
	if r, _ := a.do("GET", "/api/auth/me", "", tcookie); r.StatusCode != 401 {
		t.Fatal("deactivated user's session must end")
	}
	if resp, out, _ := a.login("tea", testPass); resp.StatusCode != 403 || errCode(t, out) != "account_inactive" {
		t.Fatalf("deactivated user must not log in: %d %s", resp.StatusCode, out)
	}
	var adminID string
	a.pool.QueryRow(context.Background(), `SELECT id FROM users WHERE username='adm'`).Scan(&adminID)
	a.call("POST", "/users/"+adminID+"/deactivate", nil, 409)
	a.call("POST", "/users/"+t1.ID.String()+"/activate", nil, 200)

	// regenerate password: old one stops working, new one forces a change
	pw := a.call("POST", "/users/"+t1.ID.String()+"/regenerate-password", nil, 200)["password"].(string)
	if resp, _, _ := a.login("tea", testPass); resp.StatusCode != 401 {
		t.Fatal("old password must stop working")
	}
	resp, out2, cookie := a.login("tea", pw)
	if resp.StatusCode != 200 || !strings.Contains(out2, `"must_change_password":true`) || cookie == "" {
		t.Fatalf("regenerated login: %d %s", resp.StatusCode, out2)
	}

	// update profile (students only get group/faculty)
	g := a.call("POST", "/groups", map[string]any{"name": "G"}, 201)
	u := a.call("PUT", "/users/"+stu.ID.String(), map[string]any{"full_name": "Yangi Ism", "phone": "+998901234567", "group_id": g["id"]}, 200)
	if u["full_name"] != "Yangi Ism" || u["group_id"] != g["id"] {
		t.Fatalf("update: %v", u)
	}
	a.call("PUT", "/users/"+stu.ID.String(), map[string]any{"full_name": ""}, 422)
}

func TestDashboardCounts(t *testing.T) {
	a := newAdminEnv(t)
	ctx := context.Background()
	var gid, lid string
	a.pool.QueryRow(ctx, `INSERT INTO groups (name) VALUES ('g') RETURNING id`).Scan(&gid)
	a.pool.QueryRow(ctx, `INSERT INTO locations (name, polygon) VALUES ('l','[[1,1],[1,2],[2,2]]') RETURNING id`).Scan(&lid)
	var sids [3]string
	for i := range sids {
		a.pool.QueryRow(ctx, `INSERT INTO users (username, role, group_id) VALUES ($1,'student',$2) RETURNING id`,
			"s"+string(rune('a'+i)), gid).Scan(&sids[i])
	}
	var sched string
	// every weekday, a wide date range, so "today" is always expected
	a.pool.QueryRow(ctx, `INSERT INTO schedules (start_date, end_date, group_id, location_id, monday, tuesday, wednesday, thursday, friday, saturday)
		VALUES (current_date - 30, current_date + 30, $1, $2, true, true, true, true, true, true) RETURNING id`, gid, lid).Scan(&sched)

	// student 0: 3/3 steps today (full); student 1: 1 step (partial); student 2: nothing (absent)
	day := "(now() AT TIME ZONE 'Asia/Tashkent')::date"
	mk := func(sid string, ok int) {
		var aid string
		a.pool.QueryRow(ctx, `INSERT INTO attendances (student_id, schedule_id, shift, date) VALUES ($1,$2,1,`+day+`) RETURNING id`, sid, sched).Scan(&aid)
		for step := 1; step <= 3; step++ {
			a.pool.Exec(ctx, `INSERT INTO attendance_steps (attendance_id, step, success) VALUES ($1,$2,$3)`, aid, step, step <= ok)
		}
	}
	mk(sids[0], 3)
	mk(sids[1], 1)

	d := a.call("GET", "/dashboard", nil, 200)
	today := d["today"].(map[string]any)
	if d["students"].(float64) != 3 || d["groups"].(float64) != 1 {
		t.Fatalf("counts: %v", d)
	}
	if today["expected"].(float64) != 3 || today["full"].(float64) != 1 || today["partial"].(float64) != 1 || today["absent"].(float64) != 1 {
		t.Fatalf("today stats: %v", today)
	}
	if len(d["last_7_days"].([]any)) != 7 {
		t.Fatal("need 7 days")
	}
}
