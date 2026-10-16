package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"pc/internal/config"
	apphttp "pc/internal/http"
)

const (
	oauthSecret = "the-client-secret-777"
	oauthCode   = "the-auth-code-888"
	oauthToken  = "the-access-token-999"
)

const teacherJSON = `{"login":"t.karimov","employee_id":41007,"name":"KARIMOV TOSHPO'LAT","firstname":"TOSHPO'LAT","surname":"KARIMOV",
"patronymic":"ALI O'G'LI","phone":"+998901112233","passport_number":"AA7654321","birth_date":"1985-03-12","picture_full":"https://hemis.test/t.png"}`

type oEnv struct {
	*env
	srv     *httptest.Server
	logs    *observer.ObservedLogs
	fetches atomic.Int32

	mu          sync.Mutex
	profile     string // resource owner body
	tokenStatus int    // 0 = ok
	userStatus  int    // 0 = ok
}

func newOAuthEnv(t *testing.T, mutate ...func(*config.Config)) *oEnv {
	o := &oEnv{profile: teacherJSON}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/access-token", func(w http.ResponseWriter, r *http.Request) {
		o.mu.Lock()
		defer o.mu.Unlock()
		r.ParseForm()
		if o.tokenStatus != 0 {
			w.WriteHeader(o.tokenStatus)
			return
		}
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != oauthCode ||
			r.Form.Get("client_secret") != oauthSecret || r.Form.Get("client_id") != "pc-app" ||
			r.Form.Get("redirect_uri") != "http://localhost:8080/auth/callback" {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		fmt.Fprintf(w, `{"access_token":%q}`, oauthToken)
	})
	mux.HandleFunc("/oauth/api/user", func(w http.ResponseWriter, r *http.Request) {
		o.mu.Lock()
		defer o.mu.Unlock()
		if o.userStatus != 0 {
			w.WriteHeader(o.userStatus)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+oauthToken {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(o.profile))
	})
	o.srv = httptest.NewServer(mux)
	t.Cleanup(o.srv.Close)

	core, logs := observer.New(zap.DebugLevel)
	o.logs = logs
	o.env = newEnvDeps(t, func(d *apphttp.Deps) {
		d.Log = zap.New(core)
		d.HemisPhotoFetch = func(ctx context.Context, u string) ([]byte, string, error) {
			o.fetches.Add(1)
			return tinyPNG, "t.png", nil
		}
	}, append([]func(*config.Config){func(c *config.Config) {
		c.OAuthClientID, c.OAuthClientSecret = "pc-app", oauthSecret
		c.OAuthRedirectURI = "http://localhost:8080/auth/callback"
		c.TeacherAuthorizeURL = o.srv.URL + "/oauth/authorize"
		c.TeacherAccessTokenURL = o.srv.URL + "/oauth/access-token"
		c.TeacherResourceOwnerURL = o.srv.URL + "/oauth/api/user"
		c.TeacherAutoCreate = true
	}}, mutate...)...)
	return o
}

func (o *oEnv) set(f func()) { o.mu.Lock(); f(); o.mu.Unlock() }

func (o *oEnv) get(path, stateCookie string) *http.Response {
	o.t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	if stateCookie != "" {
		req.Header.Set("Cookie", "hemis_oauth_state="+stateCookie)
	}
	resp, err := o.app.Test(req, -1)
	if err != nil {
		o.t.Fatal(err)
	}
	return resp
}

// start performs GET /start and returns the state, as the browser would carry it to the callback.
func (o *oEnv) start() (state string, resp *http.Response) {
	resp = o.get("/api/auth/hemis/teacher/start", "")
	for _, c := range resp.Cookies() {
		if c.Name == "hemis_oauth_state" {
			state = c.Value
		}
	}
	return state, resp
}

// callback runs a full valid round trip unless overridden: it returns the redirect location and session cookie.
func (o *oEnv) callback(query string, state string) (location, session string) {
	o.t.Helper()
	resp := o.get("/auth/callback?"+query, state)
	if resp.StatusCode != 302 {
		o.t.Fatalf("callback status %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "session" && c.Value != "" {
			session = c.Value
		}
	}
	return resp.Header.Get("Location"), session
}

func (o *oEnv) goodLogin() (string, string) {
	o.t.Helper()
	state, _ := o.start()
	return o.callback("code="+oauthCode+"&state="+state, state)
}

func (o *oEnv) failure(location string) string {
	u, _ := url.Parse(location)
	if u.Path != "/login" {
		return "ok:" + location
	}
	return u.Query().Get("error")
}

func (o *oEnv) scalar(sql string, args ...any) string {
	var v any
	if err := o.pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		o.t.Fatal(err)
	}
	if v == nil {
		return "<null>"
	}
	return fmt.Sprint(v)
}

func TestOAuthStartRedirectsToHemisWithState(t *testing.T) {
	o := newOAuthEnv(t)
	state, resp := o.start()
	if resp.StatusCode != 302 || len(state) < 40 {
		t.Fatalf("status %d state %q", resp.StatusCode, state)
	}
	loc, _ := url.Parse(resp.Header.Get("Location"))
	q := loc.Query()
	if !strings.HasPrefix(resp.Header.Get("Location"), o.srv.URL+"/oauth/authorize?") ||
		q.Get("client_id") != "pc-app" || q.Get("response_type") != "code" || q.Get("state") != state ||
		q.Get("redirect_uri") != "http://localhost:8080/auth/callback" {
		t.Fatalf("bad authorize url %s", loc)
	}
	if strings.Contains(loc.String(), oauthSecret) {
		t.Fatal("client secret leaked into the authorize URL")
	}
	for _, c := range resp.Cookies() {
		if c.Name == "hemis_oauth_state" && (!c.HttpOnly || c.Path != "/auth/callback" || c.MaxAge == 0 && c.Expires.IsZero()) {
			t.Fatalf("bad state cookie %+v", c)
		}
	}
	if s2, _ := o.start(); s2 == state {
		t.Fatal("state must be random per request")
	}
}

func TestOAuthHappyPathCreatesTeacherAndSession(t *testing.T) {
	o := newOAuthEnv(t)
	loc, session := o.goodLogin()
	if loc != "/teacher" || session == "" {
		t.Fatalf("location %q session %q", loc, session)
	}
	row := func(col string) string {
		return o.scalar(`SELECT ` + col + ` FROM users WHERE hemis_login='t.karimov'`)
	}
	if row("role") != "teacher" || row("username") != "<null>" || row("password_hash") != "<null>" ||
		row("must_change_password") != "false" || row("is_active") != "true" {
		t.Fatal("auto-created teacher has the wrong shape")
	}
	if row("full_name") != "KARIMOV TOSHPO'LAT" || row("first_name") != "TOSHPO'LAT" || row("second_name") != "KARIMOV" ||
		row("third_name") != "ALI O'G'LI" || row("phone") != "+998901112233" || row("passport_number") != "AA7654321" ||
		row("birth_date") != "1985-03-12" || row("hemis_id") != "41007" {
		t.Fatal("profile was not mapped")
	}
	if row("last_login_at") == "<null>" || row("hemis_synced_at") == "<null>" || row("image") == "" || row("hemis_img_url") != "https://hemis.test/t.png" {
		t.Fatal("login time, sync time or photo missing")
	}
	if n := o.scalar(`SELECT count(*) FROM audit_logs WHERE action='login' AND changes->'after'->>'method'='hemis_oauth'`); n != "1" {
		t.Fatalf("login audit rows: %s", n)
	}

	resp, body := o.do("GET", "/api/auth/me", "", session)
	if resp.StatusCode != 200 || !strings.Contains(body, `"role":"teacher"`) {
		t.Fatalf("session does not work: %d %s", resp.StatusCode, body)
	}
	// the state cookie is cleared by the callback
	state, _ := o.start()
	r := o.get("/auth/callback?code="+oauthCode+"&state="+state, state)
	cleared := false
	for _, c := range r.Cookies() {
		if c.Name == "hemis_oauth_state" && c.Value == "" {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("state cookie must be cleared after use")
	}
}

func TestOAuthSecondLoginUpdatesFieldsOnly(t *testing.T) {
	o := newOAuthEnv(t)
	o.goodLogin()
	id := o.scalar(`SELECT id::text FROM users WHERE hemis_login='t.karimov'`)
	if _, err := o.pool.Exec(context.Background(), `UPDATE users SET is_active=true WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}

	// HEMIS now sends a new phone, drops the name, passport and picture
	o.set(func() { o.profile = `{"login":"t.karimov","employee_id":41007,"phone":"+998900000000"}` })
	o.goodLogin()

	if n := o.scalar(`SELECT count(*) FROM users`); n != "1" {
		t.Fatalf("a second login must not create a second user (%s)", n)
	}
	q := func(col string) string { return o.scalar(`SELECT `+col+` FROM users WHERE id=$1`, id) }
	if q("phone") != "+998900000000" {
		t.Fatal("phone should have been updated")
	}
	if q("full_name") != "KARIMOV TOSHPO'LAT" || q("passport_number") != "AA7654321" || q("birth_date") != "1985-03-12" {
		t.Fatal("empty HEMIS values must not erase stored data")
	}
	if q("role") != "teacher" || q("is_active") != "true" || q("password_hash") != "<null>" {
		t.Fatal("role, is_active and password_hash are not synced")
	}
	if o.fetches.Load() != 1 {
		t.Fatalf("an unchanged photo must not be downloaded again (fetches=%d)", o.fetches.Load())
	}
}

func TestOAuthStateIsRequired(t *testing.T) {
	o := newOAuthEnv(t)
	state, _ := o.start()
	cases := map[string]struct{ query, cookie string }{
		"no cookie":      {"code=" + oauthCode + "&state=" + state, ""},
		"no state param": {"code=" + oauthCode, state},
		"wrong state":    {"code=" + oauthCode + "&state=" + state + "x", state},
		"neither":        {"code=" + oauthCode, ""},
	}
	for name, c := range cases {
		loc, session := o.callback(c.query, c.cookie)
		if o.failure(loc) != "hemis_state" || session != "" {
			t.Errorf("%s: %q session=%q", name, loc, session)
		}
	}
	if o.scalar(`SELECT count(*) FROM users`) != "0" {
		t.Fatal("no user may be created without a valid state")
	}
	if n := o.scalar(`SELECT count(*) FROM audit_logs WHERE action='login_failed' AND changes->'after'->>'reason'='hemis_state'`); n != "4" {
		t.Fatalf("login_failed audit rows: %s", n)
	}
}

func TestOAuthErrorCodes(t *testing.T) {
	// HEMIS redirected back with ?error=, or without a code
	o := newOAuthEnv(t)
	state, _ := o.start()
	if loc, _ := o.callback("error=access_denied&state="+state, state); o.failure(loc) != "hemis_denied" {
		t.Errorf("denied: %q", loc)
	}
	if loc, _ := o.callback("state="+state, state); o.failure(loc) != "hemis_denied" { // neither code nor error
		t.Errorf("no code: %q", loc)
	}
	if n := o.scalar(`SELECT count(*) FROM audit_logs WHERE action='login_failed' AND changes->'after'->>'reason'='hemis_denied'`); n != "2" {
		t.Errorf("denied audit rows: %s", n)
	}

	cases := []struct {
		name  string
		setup func(o *oEnv)
		want  string
	}{
		{"token rejected", func(o *oEnv) { o.tokenStatus = 400 }, "hemis_token"},
		{"token 401", func(o *oEnv) { o.tokenStatus = 401 }, "hemis_token"},
		{"token server error", func(o *oEnv) { o.tokenStatus = 500 }, "hemis_unavailable"},
		{"profile rejected", func(o *oEnv) { o.userStatus = 403 }, "hemis_profile"},
		{"profile server error", func(o *oEnv) { o.userStatus = 502 }, "hemis_unavailable"},
		{"profile without login", func(o *oEnv) { o.profile = `{"employee_id":1,"name":"X"}` }, "hemis_profile"},
		{"profile not json", func(o *oEnv) { o.profile = `<html>` }, "hemis_profile"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := newOAuthEnv(t)
			o.set(func() { c.setup(o) })
			loc, session := o.goodLogin()
			if got := o.failure(loc); got != c.want || session != "" {
				t.Fatalf("got %q (session %q) want %q", got, session, c.want)
			}
			if o.scalar(`SELECT count(*) FROM users`) != "0" {
				t.Fatal("a failed sign-in must not create a user")
			}
			if n := o.scalar(`SELECT count(*) FROM audit_logs WHERE action='login_failed' AND changes->'after'->>'reason'=$1`, c.want); n != "1" {
				t.Fatalf("login_failed audit rows: %s", n)
			}
		})
	}
}

func TestOAuthLinkedUserRules(t *testing.T) {
	o := newOAuthEnv(t)
	link := func(username, role string) {
		o.user(username, role, false)
		if _, err := o.pool.Exec(context.Background(), `UPDATE users SET hemis_login='t.karimov' WHERE username=$1`, username); err != nil {
			t.Fatal(err)
		}
	}
	reset := func() {
		o.pool.Exec(context.Background(), `DELETE FROM sessions`)
		o.pool.Exec(context.Background(), `DELETE FROM audit_logs`)
		o.pool.Exec(context.Background(), `DELETE FROM users`)
	}

	link("some-student", "student")
	if loc, s := o.goodLogin(); o.failure(loc) != "hemis_not_teacher" || s != "" {
		t.Errorf("student: %q", loc)
	}
	reset()

	link("gone-teacher", "teacher")
	o.pool.Exec(context.Background(), `UPDATE users SET is_active=false WHERE username='gone-teacher'`)
	if loc, s := o.goodLogin(); o.failure(loc) != "hemis_inactive" || s != "" {
		t.Errorf("inactive: %q", loc)
	}
	reset()

	// an admin-linked local teacher can use both ways and keeps everything we own
	link("local-teacher", "teacher")
	var gid string
	o.pool.QueryRow(context.Background(), `INSERT INTO groups (name, teacher_id) SELECT 'G1', id FROM users WHERE username='local-teacher' RETURNING id`).Scan(&gid)
	hashBefore := o.scalar(`SELECT password_hash FROM users WHERE username='local-teacher'`)
	loc, session := o.goodLogin()
	if loc != "/teacher" || session == "" {
		t.Fatalf("linked teacher: %q", loc)
	}
	if o.scalar(`SELECT count(*) FROM users`) != "1" || o.scalar(`SELECT password_hash FROM users WHERE username='local-teacher'`) != hashBefore ||
		o.scalar(`SELECT username FROM users WHERE hemis_login='t.karimov'`) != "local-teacher" {
		t.Fatal("linking must update the existing account, not replace it")
	}
	if o.scalar(`SELECT teacher_id = (SELECT id FROM users WHERE username='local-teacher') FROM groups WHERE id=$1`, gid) != "true" {
		t.Fatal("groups.teacher_id must stay")
	}
	if resp, _, _ := o.login("local-teacher", testPass); resp.StatusCode != 200 {
		t.Fatal("the local password must keep working")
	}
}

func TestOAuthAutoCreateOff(t *testing.T) {
	o := newOAuthEnv(t, func(c *config.Config) { c.TeacherAutoCreate = false })
	if loc, s := o.goodLogin(); o.failure(loc) != "hemis_not_registered" || s != "" {
		t.Fatalf("%q", loc)
	}
	if o.scalar(`SELECT count(*) FROM users`) != "0" {
		t.Fatal("nothing may be created when auto-create is off")
	}
	// ... but a teacher an admin linked beforehand can sign in
	o.user("linked", "teacher", false)
	o.pool.Exec(context.Background(), `UPDATE users SET hemis_login='t.karimov' WHERE username='linked'`)
	if loc, _ := o.goodLogin(); loc != "/teacher" {
		t.Fatalf("%q", loc)
	}
}

func TestOAuthSameHemisIDForStudentAndTeacher(t *testing.T) {
	o := newOAuthEnv(t)
	o.user("stu", "student", false)
	if _, err := o.pool.Exec(context.Background(), `UPDATE users SET hemis_id=41007 WHERE username='stu'`); err != nil {
		t.Fatal(err)
	}
	if loc, _ := o.goodLogin(); loc != "/teacher" { // the teacher also has employee_id 41007
		t.Fatalf("%q", loc)
	}
	if o.scalar(`SELECT count(*) FROM users WHERE hemis_id=41007`) != "2" {
		t.Fatal("a student and a teacher with the same numeric hemis_id must coexist")
	}
}

func TestOAuthSecretsNeverLogged(t *testing.T) {
	o := newOAuthEnv(t)
	o.goodLogin()
	o.set(func() { o.tokenStatus = 500 })
	o.goodLogin() // an outage is logged as a warning
	o.set(func() { o.tokenStatus = 0; o.userStatus = 403 })
	o.goodLogin()
	state, _ := o.start()
	o.callback("error=access_denied&state="+state, state)

	for _, e := range o.logs.All() {
		b, _ := json.Marshal(append([]any{e.Message}, fieldsOf(e.Context)...))
		for _, secret := range []string{oauthSecret, oauthCode, oauthToken} {
			if strings.Contains(string(b), secret) {
				t.Fatalf("log line leaks a secret: %s", b)
			}
		}
	}
	var dump string
	o.pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(changes::text || ip::text || user_agent, ' '), '') FROM audit_logs`).Scan(&dump)
	for _, secret := range []string{oauthSecret, oauthCode, oauthToken} {
		if strings.Contains(dump, secret) {
			t.Fatal("audit rows leak a secret")
		}
	}
	if strings.Contains(dump, "AA7654321") {
		t.Fatal("passport number reached the audit log")
	}
}

func fieldsOf(fs []zap.Field) []any {
	out := make([]any, 0, len(fs))
	for _, f := range fs {
		out = append(out, fmt.Sprintf("%s=%v%v", f.Key, f.String, f.Interface))
	}
	return out
}

func TestOAuthDisabledWithoutConfig(t *testing.T) {
	e := newEnv(t)
	resp, _ := e.do("GET", "/api/auth/hemis/teacher/start", "", "")
	if resp.StatusCode != 302 || !strings.Contains(resp.Header.Get("Location"), "error=hemis_unavailable") {
		t.Fatalf("%d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestOAuthCallbackNotServedBySPA(t *testing.T) {
	o := newOAuthEnv(t)
	resp, body := o.do("GET", "/auth/callback", "", "")
	if resp.StatusCode != 302 || strings.Contains(body, "<html>") {
		t.Fatalf("callback must be handled by Go: %d %q", resp.StatusCode, body)
	}
}

func TestAdminLinksTeacherToHemis(t *testing.T) {
	e := newEnv(t)
	e.user("boss", "admin", false)
	admin := e.mustLogin("boss")

	resp, body := e.do("POST", "/api/admin/users", `{"username":"ustoz","role":"teacher","full_name":"Ustoz","hemis_login":" t.karimov "}`, admin)
	if resp.StatusCode != 201 || !strings.Contains(body, `"hemis_login":"t.karimov"`) {
		t.Fatalf("create: %d %s", resp.StatusCode, body)
	}
	var created struct{ User struct{ ID string } }
	json.Unmarshal([]byte(body), &created)

	// the same HEMIS login cannot be linked twice
	resp, body = e.do("POST", "/api/admin/users", `{"username":"ustoz2","role":"teacher","full_name":"Ustoz 2","hemis_login":"t.karimov"}`, admin)
	if resp.StatusCode != 422 || !strings.Contains(body, `"hemis_login"`) {
		t.Fatalf("duplicate: %d %s", resp.StatusCode, body)
	}

	url := "/api/admin/users/" + created.User.ID
	// absent = unchanged
	if _, body = e.do("PUT", url, `{"full_name":"Ustoz","phone":"1"}`, admin); !strings.Contains(body, `"hemis_login":"t.karimov"`) {
		t.Fatalf("an update without hemis_login must keep the link: %s", body)
	}
	// "" = unlink, then link another one
	if _, body = e.do("PUT", url, `{"full_name":"Ustoz","phone":"1","hemis_login":""}`, admin); !strings.Contains(body, `"hemis_login":""`) {
		t.Fatalf("unlink: %s", body)
	}
	if _, body = e.do("PUT", url, `{"full_name":"Ustoz","phone":"1","hemis_login":"other"}`, admin); !strings.Contains(body, `"hemis_login":"other"`) {
		t.Fatalf("relink: %s", body)
	}

	// students are never linked this way
	stu := e.user("talaba", "student", false)
	if _, body = e.do("PUT", "/api/admin/users/"+stu.ID.String(), `{"full_name":"T","phone":"","hemis_login":"x"}`, admin); !strings.Contains(body, `"hemis_login":""`) {
		t.Fatalf("student must not get a hemis_login: %s", body)
	}
}
