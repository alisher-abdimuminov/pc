package http_test

import (
	"context"
	"encoding/json"
	"testing"
)

// newLittle has the admin create a little admin with the given permissions and logs them in.
// It returns the account id, the login cookie and the permissions the server stored.
func (a *adminEnv) newLittle(username string, perms ...string) (id, cookie string, stored []string) {
	a.t.Helper()
	out := a.call("POST", "/users", map[string]any{"username": username, "role": "little", "full_name": username, "permissions": perms}, 201)
	user := out["user"].(map[string]any)
	id = str(user, "id")
	for _, p := range user["permissions"].([]any) {
		stored = append(stored, p.(string))
	}
	if _, err := a.pool.Exec(context.Background(), `UPDATE users SET must_change_password=false WHERE id=$1`, id); err != nil {
		a.t.Fatal(err)
	}
	resp, body, c := a.login(username, str(out, "password"))
	if resp.StatusCode != 200 || c == "" {
		a.t.Fatalf("little login: %d %s", resp.StatusCode, body)
	}
	return id, c, stored
}

func (a *adminEnv) status(method, path, body, cookie string) int {
	a.t.Helper()
	resp, _ := a.do(method, "/api/admin"+path, body, cookie)
	return resp.StatusCode
}

func TestLittleAdminSeesOnlyWhatWasGranted(t *testing.T) {
	a := newAdminEnv(t)
	_, little, stored := a.newLittle("lit", "locations:create")

	if len(stored) != 2 || stored[0] != "locations:view" || stored[1] != "locations:create" {
		t.Fatalf("create must imply view and nothing more: %v", stored)
	}

	// granted
	if s := a.status("GET", "/locations", "", little); s != 200 {
		t.Fatalf("list locations: %d", s)
	}
	body := `{"name":"Obyekt","points":["39.1,66.1","39.1,66.2","39.2,66.2"],"tolerance_meters":30,"location_url":""}`
	if s := a.status("POST", "/locations", body, little); s != 201 {
		t.Fatalf("create location: %d", s)
	}

	// not granted: every other section and every other action is 403
	for _, c := range []struct{ method, path string }{
		{"GET", "/dashboard"}, {"GET", "/faculties"}, {"POST", "/faculties"}, {"GET", "/groups"},
		{"GET", "/users"}, {"GET", "/schedules"}, {"POST", "/schedules"}, {"GET", "/attempts"},
		{"GET", "/reports/attendance"}, {"GET", "/audit"},
		{"PUT", "/locations/00000000-0000-0000-0000-000000000000"}, {"DELETE", "/locations/00000000-0000-0000-0000-000000000000"},
	} {
		if s := a.status(c.method, c.path, "{}", little); s != 403 {
			t.Errorf("%s %s: got %d, want 403", c.method, c.path, s)
		}
	}

	// account security stays with admins
	for _, c := range []struct{ method, path string }{
		{"POST", "/users"}, {"GET", "/permissions"},
		{"DELETE", "/attempts/00000000-0000-0000-0000-000000000000"},
		{"POST", "/users/00000000-0000-0000-0000-000000000000/regenerate-password"},
		{"GET", "/users/00000000-0000-0000-0000-000000000000/sensitive"},
		{"PUT", "/users/00000000-0000-0000-0000-000000000000/permissions"},
	} {
		if s := a.status(c.method, c.path, "{}", little); s != 403 {
			t.Errorf("%s %s: got %d, want 403", c.method, c.path, s)
		}
	}

	// the other roles' areas are closed to them as before
	if resp, _ := a.do("GET", "/api/teacher/groups", "", little); resp.StatusCode != 403 {
		t.Fatalf("teacher area: %d", resp.StatusCode)
	}
	if resp, _ := a.do("GET", "/api/student/today", "", little); resp.StatusCode != 403 {
		t.Fatalf("student area: %d", resp.StatusCode)
	}
}

func TestMeReportsThePermissions(t *testing.T) {
	a := newAdminEnv(t)
	_, little, _ := a.newLittle("lit", "audit:view")
	resp, out := a.do("GET", "/api/auth/me", "", little)
	var me struct {
		User struct {
			Role        string   `json:"role"`
			Permissions []string `json:"permissions"`
		} `json:"user"`
	}
	if err := json.Unmarshal([]byte(out), &me); err != nil || resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, out)
	}
	if me.User.Role != "little" || len(me.User.Permissions) != 1 || me.User.Permissions[0] != "audit:view" {
		t.Fatalf("%+v", me.User)
	}
	_, adm := a.do("GET", "/api/auth/me", "", a.cookie)
	var am struct {
		User struct {
			Permissions []string `json:"permissions"`
		} `json:"user"`
	}
	json.Unmarshal([]byte(adm), &am)
	if am.User.Permissions == nil || len(am.User.Permissions) != 0 {
		t.Fatalf("admins carry no explicit list (they have everything): %s", adm)
	}
}

func TestLittleAdminManagesStudentsOnly(t *testing.T) {
	a := newAdminEnv(t)
	_, little, _ := a.newLittle("lit", "users:update")
	stu := a.user("stu", "student", false)
	tea := a.user("tea", "teacher", false)

	put := `{"full_name":"Yangi Ism","phone":"+998901112233","group_id":null,"faculty_id":null}`
	if s := a.status("PUT", "/users/"+stu.ID.String(), put, little); s != 200 {
		t.Fatalf("student update: %d", s)
	}
	if s := a.status("POST", "/users/"+stu.ID.String()+"/deactivate", "", little); s != 200 {
		t.Fatalf("student deactivate: %d", s)
	}
	if s := a.status("POST", "/users/"+stu.ID.String()+"/activate", "", little); s != 200 {
		t.Fatalf("student activate: %d", s)
	}

	var adminID string
	a.pool.QueryRow(context.Background(), `SELECT id::text FROM users WHERE username='adm'`).Scan(&adminID)
	for name, id := range map[string]string{"teacher": tea.ID.String(), "admin": adminID} {
		if s := a.status("PUT", "/users/"+id, put, little); s != 403 {
			t.Errorf("update %s: %d, want 403", name, s)
		}
		if s := a.status("POST", "/users/"+id+"/deactivate", "", little); s != 403 {
			t.Errorf("deactivate %s: %d, want 403", name, s)
		}
		if s := a.status("PUT", "/users/"+id+"/photo", "", little); s != 403 {
			t.Errorf("photo %s: %d, want 403", name, s)
		}
	}
	var active bool
	a.pool.QueryRow(context.Background(), `SELECT is_active FROM users WHERE id=$1`, adminID).Scan(&active)
	if !active {
		t.Fatal("the admin must still be active")
	}
}

func TestAdminChangesPermissionsAndItAppliesAtOnce(t *testing.T) {
	a := newAdminEnv(t)
	id, little, _ := a.newLittle("lit", "faculties:view")
	if s := a.status("GET", "/faculties", "", little); s != 200 {
		t.Fatalf("%d", s)
	}
	if s := a.status("GET", "/audit", "", little); s != 403 {
		t.Fatalf("%d", s)
	}

	out := a.call("PUT", "/users/"+id+"/permissions", map[string]any{"permissions": []string{"audit:view"}}, 200)
	if got := out["permissions"].([]any); len(got) != 1 || got[0] != "audit:view" {
		t.Fatalf("%v", got)
	}
	// the same session, no re-login
	if s := a.status("GET", "/audit", "", little); s != 200 {
		t.Fatalf("audit after grant: %d", s)
	}
	if s := a.status("GET", "/faculties", "", little); s != 403 {
		t.Fatalf("faculties after revoke: %d", s)
	}
	if a.audits("update", "user") == 0 {
		t.Fatal("the change must be audited")
	}

	// a deactivated little admin is out, whatever they were granted
	a.call("POST", "/users/"+id+"/deactivate", nil, 200)
	if s := a.status("GET", "/audit", "", little); s != 401 {
		t.Fatalf("deactivated: %d", s)
	}
}

func TestPermissionValidation(t *testing.T) {
	a := newAdminEnv(t)
	id, _, _ := a.newLittle("lit", "users:view")
	tea := a.user("tea", "teacher", false)

	a.call("POST", "/users", map[string]any{"username": "x1", "role": "little", "full_name": "X", "permissions": []string{"users:delete"}}, 422)
	a.call("POST", "/users", map[string]any{"username": "x2", "role": "teacher", "full_name": "X", "permissions": []string{"users:view"}}, 422)
	a.call("PUT", "/users/"+id+"/permissions", map[string]any{"permissions": []string{"root"}}, 422)
	a.call("PUT", "/users/"+tea.ID.String()+"/permissions", map[string]any{"permissions": []string{"users:view"}}, 422)

	// a little admin with nothing granted is allowed (it can only log in and see an empty panel)
	_, _, stored := a.newLittle("empty")
	if len(stored) != 0 {
		t.Fatalf("%v", stored)
	}
}
