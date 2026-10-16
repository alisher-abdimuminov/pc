package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pc/internal/config"
	"pc/internal/hemis"
	apphttp "pc/internal/http"
)

const studentToken = "the-student-access-token-555"

func aliProfile() hemis.Profile {
	return hemis.Profile{
		HemisID: 3141592, StudentIDNumber: "320261100000", FirstName: "ALI", SecondName: "VALIYEV", ThirdName: "KARIM O'G'LI",
		FullName: "VALIYEV ALI KARIM O'G'LI", ShortName: "Valiyev A.K.", BirthDate: "2004-05-06", Phone: "+998901234567",
		PassportPin: "12345678901234", PassportNumber: "AB1234567", Gender: "Erkak", PaymentForm: "Davlat granti",
		Level: "1-kurs", Semester: "1-semestr", GPA: "3.75", Address: "Toshkent", Country: "O'zbekiston", Province: "Toshkent", District: "Chilonzor",
		Group: &hemis.Ref{ID: 778, Name: "101-guruh"}, Faculty: &hemis.Ref{ID: 55, Name: "Informatika fakulteti"},
	}
}

// profileJSON renders a profile the way the HEMIS resource-owner endpoint does.
func profileJSON(p hemis.Profile) string {
	m := map[string]any{
		"id": p.HemisID, "student_id_number": p.StudentIDNumber, "first_name": p.FirstName, "second_name": p.SecondName,
		"third_name": p.ThirdName, "full_name": p.FullName, "short_name": p.ShortName, "image": p.ImageURL,
		"birth_date": p.BirthDate, "phone": p.Phone, "passport_pin": p.PassportPin, "passport_number": p.PassportNumber,
		"gender": map[string]any{"name": p.Gender}, "paymentForm": map[string]any{"name": p.PaymentForm},
		"level": map[string]any{"name": p.Level}, "semester": map[string]any{"name": p.Semester}, "avg_gpa": p.GPA,
		"address": p.Address, "country": map[string]any{"name": p.Country}, "province": map[string]any{"name": p.Province},
		"district": map[string]any{"name": p.District},
	}
	if p.Group != nil {
		m["group"] = map[string]any{"id": p.Group.ID, "name": p.Group.Name}
	}
	if p.Faculty != nil {
		m["faculty"] = map[string]any{"id": p.Faculty.ID, "name": p.Faculty.Name}
	}
	// HEMIS puts a short summary at the top level (the passport data lives only there) and the record in "data"
	data := map[string]any{}
	for k, v := range m {
		if k != "passport_pin" && k != "passport_number" {
			data[k] = v
		}
	}
	b, _ := json.Marshal(map[string]any{
		"id": p.HemisID, "student_id_number": p.StudentIDNumber, "login": p.StudentIDNumber, "type": "student",
		"passport_pin": p.PassportPin, "passport_number": p.PassportNumber, "data": data,
	})
	return string(b)
}

type sEnv struct {
	*env
	srv     *httptest.Server
	mock    *mockFace
	fetches atomic.Int32
	fetchOK atomic.Bool

	mu          sync.Mutex
	profile     hemis.Profile
	tokenStatus int // 0 = ok
	userStatus  int // 0 = ok
}

func newStudentEnv(t *testing.T, mutate ...func(*config.Config)) *sEnv {
	s := &sEnv{mock: newMockFace(t), profile: aliProfile()}
	s.fetchOK.Store(true)
	mux := http.NewServeMux()
	mux.HandleFunc("/student/access-token", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		r.ParseForm()
		if s.tokenStatus != 0 {
			w.WriteHeader(s.tokenStatus)
			return
		}
		if r.Form.Get("code") != oauthCode || r.Form.Get("client_secret") != oauthSecret || r.Form.Get("grant_type") != "authorization_code" {
			w.WriteHeader(400)
			return
		}
		fmt.Fprintf(w, `{"access_token":%q}`, studentToken)
	})
	mux.HandleFunc("/student/user", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.userStatus != 0 {
			w.WriteHeader(s.userStatus)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+studentToken {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(profileJSON(s.profile)))
	})
	s.srv = httptest.NewServer(mux)
	t.Cleanup(s.srv.Close)

	s.env = newEnvDeps(t, func(d *apphttp.Deps) {
		d.RunBackground = func(f func()) { f() } // run photo sync inline so tests are deterministic
		d.HemisPhotoFetch = func(ctx context.Context, u string) ([]byte, string, error) {
			s.fetches.Add(1)
			if !s.fetchOK.Load() {
				return nil, "", os.ErrNotExist
			}
			return tinyPNG, "photo.png", nil
		}
	}, append([]func(*config.Config){func(c *config.Config) {
		c.FaceServiceURL, c.FaceServiceToken, c.FaceTimeout = s.mock.srv.URL, "tok", 5*time.Second
		c.OAuthClientID, c.OAuthClientSecret = "pc-app", oauthSecret
		c.OAuthRedirectURI = "http://localhost:8080/auth/callback"
		c.StudentAuthorizeURL = s.srv.URL + "/student/authorize"
		c.StudentAccessTokenURL = s.srv.URL + "/student/access-token"
		c.StudentResourceOwnerURL = s.srv.URL + "/student/user"
	}}, mutate...)...)
	return s
}

func (s *sEnv) set(f func()) { s.mu.Lock(); f(); s.mu.Unlock() }

func (s *sEnv) setProfile(p hemis.Profile) { s.set(func() { s.profile = p }) }

func (s *sEnv) hit(path, stateCookie string) *http.Response {
	s.t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	if stateCookie != "" {
		req.Header.Set("Cookie", "hemis_oauth_state="+stateCookie)
	}
	resp, err := s.app.Test(req, -1)
	if err != nil {
		s.t.Fatal(err)
	}
	return resp
}

func (s *sEnv) start() (state string, resp *http.Response) {
	resp = s.hit("/api/auth/hemis/student/start", "")
	for _, c := range resp.Cookies() {
		if c.Name == "hemis_oauth_state" {
			state = c.Value
		}
	}
	return state, resp
}

// signIn runs a whole round trip and returns where the browser lands and the session cookie.
func (s *sEnv) signIn() (location, session string) {
	s.t.Helper()
	state, _ := s.start()
	resp := s.hit("/auth/callback?code="+oauthCode+"&state="+state, state)
	if resp.StatusCode != 302 {
		s.t.Fatalf("callback status %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "session" && c.Value != "" {
			session = c.Value
		}
	}
	return resp.Header.Get("Location"), session
}

func (s *sEnv) failure(location string) string {
	u, _ := url.Parse(location)
	if u.Path != "/login" {
		return "ok:" + location
	}
	return u.Query().Get("error")
}

func (s *sEnv) scalar(sql string, args ...any) string {
	var v any
	if err := s.pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		s.t.Fatal(err)
	}
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func (s *sEnv) n(sql string, args ...any) int {
	var n int
	s.pool.QueryRow(context.Background(), sql, args...).Scan(&n)
	return n
}

func TestStudentOAuthStartUsesTheStudentEndpoint(t *testing.T) {
	s := newStudentEnv(t)
	state, resp := s.start()
	loc, _ := url.Parse(resp.Header.Get("Location"))
	if resp.StatusCode != 302 || !strings.HasPrefix(resp.Header.Get("Location"), s.srv.URL+"/student/authorize?") ||
		loc.Query().Get("state") != state || !strings.HasPrefix(state, "s.") || loc.Query().Get("client_id") != "pc-app" {
		t.Fatalf("bad start: %d %s state=%q", resp.StatusCode, loc, state)
	}
	// the teacher button is not configured here
	if r := s.hit("/api/auth/hemis/teacher/start", ""); !strings.Contains(r.Header.Get("Location"), "error=hemis_unavailable") {
		t.Fatalf("teacher start must be off: %s", r.Header.Get("Location"))
	}
}

func TestStudentOAuthFirstLoginCreatesStudentFacultyAndGroup(t *testing.T) {
	s := newStudentEnv(t)
	loc, session := s.signIn()
	if loc != "/student" || session == "" {
		t.Fatalf("location %q session %q", loc, session)
	}
	row := func(col string) string {
		return s.scalar(`SELECT ` + col + `::text FROM users WHERE username='320261100000'`)
	}
	for col, want := range map[string]string{
		"hemis_id": "3141592", "full_name": "VALIYEV ALI KARIM O'G'LI", "short_name": "Valiyev A.K.", "first_name": "ALI",
		"birth_date": "2004-05-06", "gender": "Erkak", "level": "1-kurs", "semester": "1-semestr", "gpa": "3.75",
		"passport_pin": "12345678901234", "passport_number": "AB1234567", "role": "student", "is_active": "true",
	} {
		if got := row(col); got != want {
			t.Errorf("%s = %q, want %q", col, got, want)
		}
	}
	if s.scalar(`SELECT (password_hash IS NULL)::text FROM users`) != "true" {
		t.Fatal("a HEMIS student must not get a local password hash")
	}
	if s.scalar(`SELECT f.name FROM users u JOIN faculties f ON f.id=u.faculty_id`) != "Informatika fakulteti" ||
		s.scalar(`SELECT g.name FROM users u JOIN groups g ON g.id=u.group_id`) != "101-guruh" ||
		s.scalar(`SELECT g.hemis_id FROM groups g`) != "778" || s.scalar(`SELECT g.faculty_id = f.id FROM groups g, faculties f`) != "true" {
		t.Fatal("faculty/group not upserted by hemis_id")
	}
	prof := s.get("/api/student/profile", session, 200)
	if strings.Contains(string(prof), "12345678901234") || strings.Contains(string(prof), "AB1234567") || !strings.Contains(string(prof), "101-guruh") {
		t.Fatalf("profile: %s", prof)
	}
	if s.auditCount("login") != 1 || s.audits("create", "user") != 1 {
		t.Fatal("login and user creation must be audited")
	}
	if s.n(`SELECT count(*) FROM audit_logs WHERE changes::text LIKE '%12345678901234%' OR changes::text LIKE '%AB1234567%'`) != 0 {
		t.Fatal("passport data leaked into the audit log")
	}
}

func TestStudentOAuthSecondLoginUpdatesInsteadOfDuplicating(t *testing.T) {
	s := newStudentEnv(t)
	s.signIn()
	id1 := s.scalar(`SELECT id::text FROM users`)

	p := aliProfile()
	p.FullName, p.Level = "VALIYEV ALI (YANGI)", "2-kurs"
	p.Group = &hemis.Ref{ID: 778, Name: "101-guruh (qayta nomlangan)"}
	s.setProfile(p)
	if loc, _ := s.signIn(); loc != "/student" {
		t.Fatalf("second login: %s", loc)
	}
	if s.n(`SELECT count(*) FROM users`) != 1 || s.n(`SELECT count(*) FROM groups`) != 1 || s.scalar(`SELECT id::text FROM users`) != id1 {
		t.Fatal("a repeat login must update the same rows")
	}
	if s.scalar(`SELECT full_name FROM users`) != "VALIYEV ALI (YANGI)" || s.scalar(`SELECT level FROM users`) != "2-kurs" ||
		s.scalar(`SELECT name FROM groups`) != "101-guruh (qayta nomlangan)" {
		t.Fatal("profile changes from HEMIS were not applied")
	}
	if s.audits("update", "user") != 1 {
		t.Fatalf("one audited update expected, got %d", s.audits("update", "user"))
	}
	s.signIn()
	if s.audits("update", "user") != 1 {
		t.Fatal("no-op profile sync must not be audited")
	}
}

func TestStudentOAuthErrorCodes(t *testing.T) {
	s := newStudentEnv(t)
	state, _ := s.start()
	check := func(name, want string, resp *http.Response) {
		t.Helper()
		if got := s.failure(resp.Header.Get("Location")); got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
	check("no state cookie", "hemis_state", s.hit("/auth/callback?code="+oauthCode+"&state="+state, ""))
	check("wrong state", "hemis_state", s.hit("/auth/callback?code="+oauthCode+"&state=zzz", state))
	check("denied", "hemis_denied", s.hit("/auth/callback?error=access_denied&state="+state, state))

	s.set(func() { s.tokenStatus = 400 })
	check("token refused", "hemis_token", s.hit("/auth/callback?code="+oauthCode+"&state="+state, state))
	s.set(func() { s.tokenStatus, s.userStatus = 0, 401 })
	check("profile refused", "hemis_profile", s.hit("/auth/callback?code="+oauthCode+"&state="+state, state))
}

func TestStudentOAuthOutageIsAuditedWithItsCause(t *testing.T) {
	s := newStudentEnv(t)
	s.set(func() { s.userStatus = 503 })
	loc, session := s.signIn()
	if s.failure(loc) != "hemis_unavailable" || session != "" {
		t.Fatalf("outage: %q", loc)
	}
	if s.n(`SELECT count(*) FROM audit_logs WHERE action='login_failed' AND changes->'after'->>'reason'='hemis_unavailable'
		AND changes->'after'->>'detail' LIKE '%status 503%'`) != 1 {
		t.Fatal("an outage must be audited with its cause")
	}
	if s.n(`SELECT count(*) FROM users`) != 0 {
		t.Fatal("a failed sign-in must not create a user")
	}
	s.set(func() { s.userStatus = 0 }) // HEMIS recovers: nothing was locked out
	if loc, _ := s.signIn(); loc != "/student" {
		t.Fatalf("after recovery: %s", loc)
	}
}

func TestStudentOAuthInactiveStudentCannotLogIn(t *testing.T) {
	s := newStudentEnv(t)
	s.signIn()
	s.pool.Exec(context.Background(), `UPDATE users SET is_active=false`)
	loc, session := s.signIn()
	if got := s.failure(loc); got != "hemis_inactive" || session != "" {
		t.Fatalf("inactive: %q session %q", got, session)
	}
	if s.scalar(`SELECT is_active::text FROM users`) != "false" {
		t.Fatal("logging in must not reactivate a user")
	}
}

func TestStudentOAuthAdoptsSeededStudentAndProtectsUsernames(t *testing.T) {
	s := newStudentEnv(t)
	seeded := s.user("320261100000", "student", false) // pre-existing local student, no hemis_id
	s.pool.Exec(context.Background(), `UPDATE users SET password_hash=NULL WHERE id=$1`, seeded.ID)
	if loc, _ := s.signIn(); loc != "/student" {
		t.Fatalf("adopt: %s", loc)
	}
	if s.n(`SELECT count(*) FROM users`) != 1 || s.scalar(`SELECT id::text FROM users`) != seeded.ID.String() || s.scalar(`SELECT hemis_id::text FROM users`) != "3141592" {
		t.Fatal("the seeded student must be linked, not duplicated")
	}
	// a different HEMIS account with the same student number must not take the linked username over
	other := aliProfile()
	other.HemisID = 999
	s.setProfile(other)
	if loc, session := s.signIn(); s.failure(loc) != "hemis_profile" || session != "" {
		t.Fatalf("username conflict: %q", loc)
	}
}

func TestStudentsHaveNoPasswordLoginForm(t *testing.T) {
	s := newStudentEnv(t)
	s.user("stud", "student", false) // has a local password (testPass), as a seeded student would
	if resp, body, cookie := s.login("stud", testPass); resp.StatusCode != 403 || cookie != "" || errCode(t, body) != "student_login_unavailable" {
		t.Fatalf("student password login must be refused: %d %s", resp.StatusCode, body)
	}
	if resp, _, _ := s.login("320261100000", "whatever"); resp.StatusCode != 401 {
		t.Fatalf("unknown login: %d", resp.StatusCode)
	}
	// the dev switch keeps seeded students usable locally
	d := newStudentEnv(t, func(c *config.Config) { c.DevStudentPasswordLogin = true })
	d.user("stud", "student", false)
	if resp, _, _ := d.login("stud", testPass); resp.StatusCode != 200 {
		t.Fatalf("dev local login: %d", resp.StatusCode)
	}
	// staff keep their local passwords
	s.user("adm", "admin", false)
	if resp, _, _ := s.login("adm", testPass); resp.StatusCode != 200 {
		t.Fatalf("admin login: %d", resp.StatusCode)
	}
}

func TestStudentOAuthDisabledWithoutConfig(t *testing.T) {
	s := newStudentEnv(t, func(c *config.Config) {
		c.StudentAuthorizeURL, c.StudentAccessTokenURL, c.StudentResourceOwnerURL = "", "", ""
		c.TeacherAuthorizeURL = "http://x/a"
		c.TeacherAccessTokenURL = "http://x/t"
		c.TeacherResourceOwnerURL = "http://x/u"
	})
	if r := s.hit("/api/auth/hemis/student/start", ""); !strings.Contains(r.Header.Get("Location"), "error=hemis_unavailable") {
		t.Fatalf("student start must be off: %s", r.Header.Get("Location"))
	}
}

func TestStudentOAuthPhotoIsEnrolledOncePerPhoto(t *testing.T) {
	s := newStudentEnv(t)
	p := aliProfile()
	p.ImageURL = "https://hemis.test/uploads/ali-v1.jpg"
	s.setProfile(p)

	s.signIn()
	if s.fetches.Load() != 1 || s.mock.calls.Load() != 1 {
		t.Fatalf("first login: %d downloads, %d embeds; want 1 and 1", s.fetches.Load(), s.mock.calls.Load())
	}
	if s.scalar(`SELECT array_length(face_embedding,1) FROM users`) != "3" || s.scalar(`SELECT hemis_img_url FROM users`) != p.ImageURL ||
		!strings.HasPrefix(s.scalar(`SELECT image FROM users`), "avatars/") {
		t.Fatal("photo not enrolled")
	}
	img := s.scalar(`SELECT image FROM users`)
	if _, err := os.Stat(filepath.Join(s.cfg.UploadDir, img)); err != nil {
		t.Fatal("avatar file missing")
	}
	s.signIn()
	if s.fetches.Load() != 1 || s.mock.calls.Load() != 1 {
		t.Fatalf("unchanged photo reprocessed: %d downloads, %d embeds", s.fetches.Load(), s.mock.calls.Load())
	}
	p.ImageURL = "https://hemis.test/uploads/ali-v2.jpg"
	s.setProfile(p)
	s.signIn()
	if s.fetches.Load() != 2 || s.mock.calls.Load() != 2 {
		t.Fatalf("changed photo: %d downloads, %d embeds", s.fetches.Load(), s.mock.calls.Load())
	}
	if _, err := os.Stat(filepath.Join(s.cfg.UploadDir, img)); !os.IsNotExist(err) {
		t.Fatal("old avatar must be deleted")
	}
}

func TestStudentOAuthPhotoProblemsNeverBreakLogin(t *testing.T) {
	s := newStudentEnv(t)
	p := aliProfile()
	p.ImageURL = "https://hemis.test/uploads/ali.jpg"
	s.setProfile(p)

	s.fetchOK.Store(false) // the download fails
	if loc, _ := s.signIn(); loc != "/student" {
		t.Fatalf("download failure broke login: %s", loc)
	}
	s.fetchOK.Store(true)
	s.mock.set(422, `{"code":"no_face"}`) // no face in the photo
	if loc, _ := s.signIn(); loc != "/student" {
		t.Fatalf("no-face broke login: %s", loc)
	}
	s.mock.set(503, ``) // face-service down
	if loc, _ := s.signIn(); loc != "/student" {
		t.Fatalf("face outage broke login: %s", loc)
	}
	if s.scalar(`SELECT hemis_img_url FROM users`) != "" {
		t.Fatal("a failed sync must leave the photo marked as not yet processed")
	}
	s.mock.reset()
	s.signIn()
	if s.scalar(`SELECT hemis_img_url FROM users`) != p.ImageURL || s.n(`SELECT count(*) FROM users WHERE face_embedding IS NOT NULL`) != 1 {
		t.Fatal("the sync must be retried on the next login")
	}
}

func TestStudentOAuthProfileWithoutPhotoOrGroup(t *testing.T) {
	s := newStudentEnv(t)
	p := aliProfile()
	p.Group, p.Faculty, p.ImageURL = nil, nil, ""
	s.setProfile(p)
	if loc, _ := s.signIn(); loc != "/student" {
		t.Fatalf("%s", loc)
	}
	if s.fetches.Load() != 0 || s.scalar(`SELECT (group_id IS NULL)::text FROM users`) != "true" {
		t.Fatal("no photo/group: nothing to sync")
	}
}
