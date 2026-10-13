package http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"pc/internal/config"
	"pc/internal/db"
	apphttp "pc/internal/http"
	"pc/internal/service"
	"pc/internal/testutil"
)

const testPass = "Passw0rd-test"

type env struct {
	t    *testing.T
	app  *fiber.App
	pool *pgxpool.Pool
	cfg  *config.Config

	clockMu sync.Mutex
	fixed   time.Time // zero = real time; set with setNow to drive the attempt/today flows
}

func (e *env) now() time.Time {
	e.clockMu.Lock()
	defer e.clockMu.Unlock()
	if e.fixed.IsZero() {
		return time.Now()
	}
	return e.fixed
}

func (e *env) setNow(t time.Time) {
	e.clockMu.Lock()
	e.fixed = t
	e.clockMu.Unlock()
}

func newEnv(t *testing.T, mutate ...func(*config.Config)) *env {
	t.Helper()
	return newEnvDeps(t, nil, mutate...)
}

// newEnvDeps is newEnv with a hook to customise the server dependencies (HEMIS fake, etc.).
func newEnvDeps(t *testing.T, deps func(*apphttp.Deps), mutate ...func(*config.Config)) *env {
	t.Helper()
	pool := testutil.Pool(t)
	cfg := &config.Config{
		AppEnv: "dev", SessionTTL: time.Hour, CookieSecure: false, Location: time.FixedZone("UTC+5", 5*3600),
		FaceServiceURL: "http://127.0.0.1:1", FaceServiceToken: "t", UploadDir: t.TempDir(),
	}
	for _, m := range mutate {
		m(cfg)
	}
	e := &env{t: t, pool: pool, cfg: cfg}
	d := apphttp.Deps{Cfg: cfg, Log: zap.NewNop(), Pool: pool, Clock: e.now,
		Dist: fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}}
	if deps != nil {
		deps(&d)
	}
	e.app = apphttp.NewServer(d)
	return e
}

func (e *env) user(username, role string, mustChange bool) db.User {
	e.t.Helper()
	u, _, err := service.NewUserService(e.pool).CreateUser(context.Background(), service.Actor{Role: "system"},
		service.CreateUserInput{Username: username, Role: role, FullName: username, Password: testPass, MustChangePassword: mustChange})
	if err != nil {
		e.t.Fatal(err)
	}
	return u
}

func (e *env) do(method, path, body, cookie string) (*http.Response, string) {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", "session="+cookie)
	}
	resp, err := e.app.Test(req, -1)
	if err != nil {
		e.t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func (e *env) login(username, password string) (*http.Response, string, string) {
	body := `{"username":"` + username + `","password":"` + password + `"}`
	resp, out := e.do("POST", "/api/auth/login", body, "")
	cookie := ""
	for _, c := range resp.Cookies() {
		if c.Name == "session" && c.Value != "" {
			cookie = c.Value
		}
	}
	return resp, out, cookie
}

func (e *env) mustLogin(username string) string {
	e.t.Helper()
	resp, out, cookie := e.login(username, testPass)
	if resp.StatusCode != 200 || cookie == "" {
		e.t.Fatalf("login failed: %d %s", resp.StatusCode, out)
	}
	return cookie
}

func (e *env) auditCount(action string) int {
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs WHERE action=$1`, action).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func errCode(t *testing.T, body string) string {
	t.Helper()
	var r struct {
		Error struct{ Code string } `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("bad error body %q", body)
	}
	return r.Error.Code
}

func TestLoginMeLogout(t *testing.T) {
	e := newEnv(t)
	e.user("admin1", "admin", false)

	resp, out, cookie := e.login("Admin1", testPass) // username is case-insensitive
	if resp.StatusCode != 200 || cookie == "" {
		t.Fatalf("login: %d %s", resp.StatusCode, out)
	}
	if c := resp.Cookies()[0]; !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie flags wrong: %+v", c)
	}
	if strings.Contains(out, "hash") || strings.Contains(out, "$2a$") {
		t.Fatalf("login response leaks password data: %s", out)
	}
	// DB holds the hash, never the raw token.
	var n int
	e.pool.QueryRow(context.Background(), `SELECT count(*) FROM sessions WHERE id=$1`, cookie).Scan(&n)
	if n != 0 {
		t.Fatal("raw token stored in DB")
	}
	if r, _ := e.do("GET", "/api/auth/me", "", cookie); r.StatusCode != 200 {
		t.Fatalf("me: %d", r.StatusCode)
	}
	if e.auditCount("login") != 1 {
		t.Fatal("login not audited")
	}

	if r, _ := e.do("POST", "/api/auth/logout", "", cookie); r.StatusCode != 204 {
		t.Fatalf("logout: %d", r.StatusCode)
	}
	if r, _ := e.do("GET", "/api/auth/me", "", cookie); r.StatusCode != 401 {
		t.Fatalf("session must be gone after logout, got %d", r.StatusCode)
	}
	var sessions int
	e.pool.QueryRow(context.Background(), `SELECT count(*) FROM sessions`).Scan(&sessions)
	if sessions != 0 || e.auditCount("logout") != 1 {
		t.Fatal("logout must delete the session and be audited")
	}
}

func TestLoginFailures(t *testing.T) {
	e := newEnv(t)
	e.user("admin1", "admin", false)
	inactive := e.user("gone", "teacher", false)
	e.pool.Exec(context.Background(), `UPDATE users SET is_active=false WHERE id=$1`, inactive.ID)

	for name, c := range map[string][2]string{
		"wrong password": {"admin1", "nope-nope"},
		"unknown user":   {"nobody", testPass},
	} {
		resp, out, cookie := e.login(c[0], c[1])
		if resp.StatusCode != 401 || cookie != "" || errCode(t, out) != "invalid_credentials" {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
	// a deactivated user with the right password is told why (the warning shown after a block)
	resp, out, cookie := e.login("gone", testPass)
	if resp.StatusCode != 403 || cookie != "" || errCode(t, out) != "account_inactive" || !strings.Contains(out, service.AccountInactiveMessage) {
		t.Fatalf("inactive: %d %s", resp.StatusCode, out)
	}
	// ...but a wrong password for it reveals nothing
	if resp, _, _ := e.login("gone", "wrong-wrong"); resp.StatusCode != 401 {
		t.Fatalf("inactive + wrong password: %d", resp.StatusCode)
	}
	if e.auditCount("login_failed") != 4 {
		t.Fatalf("login_failed audits = %d", e.auditCount("login_failed"))
	}
}

func TestStudentLoginOnlyInDevMode(t *testing.T) {
	e := newEnv(t)
	e.user("stud", "student", false)
	resp, out, _ := e.login("stud", testPass)
	if resp.StatusCode != 403 || errCode(t, out) != "student_login_unavailable" {
		t.Fatalf("student must be refused: %d %s", resp.StatusCode, out)
	}

	e2 := newEnv(t, func(c *config.Config) { c.DevStudentPasswordLogin = true })
	e2.user("stud", "student", false)
	if resp, out, _ := e2.login("stud", testPass); resp.StatusCode != 200 {
		t.Fatalf("dev student login: %d %s", resp.StatusCode, out)
	}
}

func TestLoginRateLimit(t *testing.T) {
	e := newEnv(t)
	e.user("admin1", "admin", false)
	for i := 0; i < service.LoginAttempts; i++ {
		if resp, _, _ := e.login("admin1", "wrong-wrong"); resp.StatusCode != 401 {
			t.Fatalf("attempt %d: %d", i, resp.StatusCode)
		}
	}
	// Even the right password is refused once the limit is hit.
	resp, out, _ := e.login("admin1", testPass)
	if resp.StatusCode != 429 || errCode(t, out) != "too_many_requests" {
		t.Fatalf("expected 429, got %d %s", resp.StatusCode, out)
	}
	// A different username from the same IP is a separate bucket.
	e.user("admin2", "admin", false)
	if resp, _, _ := e.login("admin2", testPass); resp.StatusCode != 200 {
		t.Fatalf("other username must not be blocked: %d", resp.StatusCode)
	}
}

func TestSessionMiddleware(t *testing.T) {
	e := newEnv(t)
	u := e.user("admin1", "admin", false)

	if r, _ := e.do("GET", "/api/auth/me", "", ""); r.StatusCode != 401 {
		t.Fatalf("no cookie: %d", r.StatusCode)
	}
	if r, _ := e.do("GET", "/api/auth/me", "", "garbage"); r.StatusCode != 401 {
		t.Fatalf("garbage cookie: %d", r.StatusCode)
	}

	cookie := e.mustLogin("admin1")
	e.pool.Exec(context.Background(), `UPDATE sessions SET expires_at = now() - interval '1 minute'`)
	if r, _ := e.do("GET", "/api/auth/me", "", cookie); r.StatusCode != 401 {
		t.Fatalf("expired session must be rejected: %d", r.StatusCode)
	}

	cookie = e.mustLogin("admin1")
	e.pool.Exec(context.Background(), `UPDATE users SET is_active=false WHERE id=$1`, u.ID)
	if r, _ := e.do("GET", "/api/auth/me", "", cookie); r.StatusCode != 401 {
		t.Fatalf("deactivated user must be rejected: %d", r.StatusCode)
	}

	n, err := db.New(e.pool).DeleteExpiredSessions(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("cleanup removed %d (%v), want 1", n, err)
	}
}

func TestRoleChecks(t *testing.T) {
	e := newEnv(t, func(c *config.Config) { c.DevStudentPasswordLogin = true })
	e.user("adm", "admin", false)
	e.user("tea", "teacher", false)
	e.user("stu", "student", false)
	adm, tea, stu := e.mustLogin("adm"), e.mustLogin("tea"), e.mustLogin("stu")

	cases := []struct {
		path, cookie string
		want         int
	}{
		{"/api/admin/x", "", 401},
		{"/api/admin/x", tea, 403},
		{"/api/admin/x", stu, 403},
		{"/api/admin/x", adm, 404}, // allowed through; route does not exist yet
		{"/api/teacher/x", adm, 403},
		{"/api/teacher/x", tea, 404},
		{"/api/student/x", tea, 403},
		{"/api/student/x", stu, 404},
	}
	for _, c := range cases {
		if r, _ := e.do("GET", c.path, "", c.cookie); r.StatusCode != c.want {
			t.Errorf("%s: got %d want %d", c.path, r.StatusCode, c.want)
		}
	}
}

func TestMustChangePassword(t *testing.T) {
	e := newEnv(t)
	e.user("adm", "admin", true)
	cookie := e.mustLogin("adm")

	if r, out := e.do("GET", "/api/admin/x", "", cookie); r.StatusCode != 403 || errCode(t, out) != "password_change_required" {
		t.Fatalf("restricted endpoint must be blocked: %d %s", r.StatusCode, out)
	}
	if r, _ := e.do("GET", "/api/auth/me", "", cookie); r.StatusCode != 200 {
		t.Fatalf("me must stay available: %d", r.StatusCode)
	}

	bad := `{"old_password":"wrong","new_password":"short"}`
	r, out := e.do("POST", "/api/auth/change-password", bad, cookie)
	if r.StatusCode != 422 || !strings.Contains(out, "old_password") || !strings.Contains(out, "new_password") {
		t.Fatalf("validation: %d %s", r.StatusCode, out)
	}

	good := `{"old_password":"` + testPass + `","new_password":"Brand-new-pass1"}`
	if r, out := e.do("POST", "/api/auth/change-password", good, cookie); r.StatusCode != 200 {
		t.Fatalf("change: %d %s", r.StatusCode, out)
	}
	if r, _ := e.do("GET", "/api/admin/x", "", cookie); r.StatusCode != 404 {
		t.Fatalf("after change the user is unrestricted: %d", r.StatusCode)
	}
	if resp, _, _ := e.login("adm", testPass); resp.StatusCode != 401 {
		t.Fatal("old password must stop working")
	}
	if resp, _, _ := e.login("adm", "Brand-new-pass1"); resp.StatusCode != 200 {
		t.Fatal("new password must work")
	}

	// The audit entry never contains the hash.
	var changes string
	e.pool.QueryRow(context.Background(), `SELECT changes::text FROM audit_logs WHERE action='update' AND entity_type='user'`).Scan(&changes)
	if strings.Contains(changes, "$2a$") || strings.Contains(changes, "password_hash") {
		t.Fatalf("audit leaks hash: %s", changes)
	}
}

func TestGeneratedPasswordCanSkipTheForcedChange(t *testing.T) {
	e := newEnv(t, func(c *config.Config) { c.DevStudentPasswordLogin = true })
	u := e.user("stud", "student", false)
	users := service.NewUserService(e.pool)
	system := service.Actor{Role: "system"}

	pw, err := users.SetGeneratedPassword(context.Background(), system, u.ID, false)
	if err != nil || len(pw) != 12 {
		t.Fatalf("%q %v", pw, err)
	}
	resp, body, cookie := e.login("stud", pw)
	if resp.StatusCode != 200 || cookie == "" || strings.Contains(body, `"must_change_password":true`) {
		t.Fatalf("login with the kept password: %d %s", resp.StatusCode, body)
	}
	if resp, _, _ := e.login("stud", testPass); resp.StatusCode != 401 {
		t.Fatal("the old password must stop working")
	}

	pw2, _ := users.SetGeneratedPassword(context.Background(), system, u.ID, true)
	if pw2 == pw {
		t.Fatal("a new password must differ")
	}
	if _, body, _ := e.login("stud", pw2); !strings.Contains(body, `"must_change_password":true`) {
		t.Fatalf("forced change expected: %s", body)
	}
}
