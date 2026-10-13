package http_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pc/internal/config"
	"pc/internal/hemis"
	apphttp "pc/internal/http"
	"pc/internal/service"
)

const hemisPW = "Hemis-Parol-777"

func aliProfile() hemis.Profile {
	return hemis.Profile{
		HemisID: 3141592, FirstName: "ALI", SecondName: "VALIYEV", ThirdName: "KARIM O'G'LI",
		FullName: "VALIYEV ALI KARIM O'G'LI", ShortName: "Valiyev A.K.", BirthDate: "2004-05-06", Phone: "+998901234567",
		PassportPin: "12345678901234", PassportNumber: "AB1234567", Gender: "Erkak", PaymentForm: "Davlat granti",
		Level: "1-kurs", Semester: "1-semestr", GPA: "3.75", Address: "Toshkent", Country: "O'zbekiston", Province: "Toshkent", District: "Chilonzor",
		Group: &hemis.Ref{ID: 778, Name: "101-guruh"}, Faculty: &hemis.Ref{ID: 55, Name: "Informatika fakulteti"},
	}
}

type hEnv struct {
	*env
	hemis   *hemis.Fake
	mock    *mockFace
	fetches atomic.Int32
	fetchOK atomic.Bool
}

func newHemisEnv(t *testing.T, mutate ...func(*config.Config)) *hEnv {
	h := &hEnv{hemis: hemis.NewFake(), mock: newMockFace(t)}
	h.fetchOK.Store(true)
	h.hemis.Add("ali", hemisPW, aliProfile())
	h.env = newEnvDeps(t, func(d *apphttp.Deps) {
		d.Hemis = h.hemis
		d.RunBackground = func(f func()) { f() } // run photo sync inline so tests are deterministic
		d.HemisPhotoFetch = func(ctx context.Context, url string) ([]byte, string, error) {
			h.fetches.Add(1)
			if !h.fetchOK.Load() {
				return nil, "", os.ErrNotExist
			}
			return tinyPNG, "photo.png", nil
		}
	}, append([]func(*config.Config){func(c *config.Config) {
		c.FaceServiceURL, c.FaceServiceToken, c.FaceTimeout = h.mock.srv.URL, "tok", 5*time.Second
		c.HemisBaseURL = "http://hemis.test"
	}}, mutate...)...)
	return h
}

func (h *hEnv) login(user, pw string) (int, string, string) {
	resp, body, cookie := h.env.login(user, pw)
	return resp.StatusCode, body, cookie
}

func (h *hEnv) scalar(sql string, args ...any) string {
	var v any
	if err := h.pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		h.t.Fatal(err)
	}
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func (h *hEnv) n(sql string, args ...any) int {
	var n int
	h.pool.QueryRow(context.Background(), sql, args...).Scan(&n)
	return n
}

func TestHemisFirstLoginCreatesStudentFacultyAndGroup(t *testing.T) {
	h := newHemisEnv(t)
	status, body, cookie := h.login("Ali", hemisPW) // login is case-insensitive
	if status != 200 || cookie == "" || !strings.Contains(body, `"role":"student"`) {
		t.Fatalf("login: %d %s", status, body)
	}
	row := func(col string) string {
		return h.scalar(`SELECT ` + col + `::text FROM users WHERE username='ali'`)
	}
	for col, want := range map[string]string{
		"hemis_id": "3141592", "full_name": "VALIYEV ALI KARIM O'G'LI", "short_name": "Valiyev A.K.", "first_name": "ALI",
		"second_name": "VALIYEV", "third_name": "KARIM O'G'LI", "birth_date": "2004-05-06", "gender": "Erkak",
		"level": "1-kurs", "semester": "1-semestr", "gpa": "3.75", "payment_form": "Davlat granti", "district": "Chilonzor",
		"passport_pin": "12345678901234", "passport_number": "AB1234567", "role": "student", "is_active": "true",
	} {
		if got := row(col); got != want {
			t.Errorf("%s = %q, want %q", col, got, want)
		}
	}
	if h.scalar(`SELECT (password_hash IS NULL)::text FROM users WHERE username='ali'`) != "true" {
		t.Fatal("a HEMIS student must not get a local password hash")
	}
	if h.scalar(`SELECT f.name FROM users u JOIN faculties f ON f.id=u.faculty_id WHERE u.username='ali'`) != "Informatika fakulteti" ||
		h.scalar(`SELECT g.name FROM users u JOIN groups g ON g.id=u.group_id WHERE u.username='ali'`) != "101-guruh" ||
		h.scalar(`SELECT g.hemis_id::text FROM groups g`) != "778" || h.scalar(`SELECT f.hemis_id::text FROM faculties f`) != "55" {
		t.Fatal("faculty/group not upserted by hemis_id")
	}
	if h.scalar(`SELECT g.faculty_id = f.id FROM groups g, faculties f`) != "true" {
		t.Fatal("group must point at the faculty")
	}

	// the session works and the student's own profile shows the data but never passport fields
	prof := h.get("/api/student/profile", cookie, 200)
	if strings.Contains(string(prof), "12345678901234") || strings.Contains(string(prof), "AB1234567") || !strings.Contains(string(prof), "101-guruh") {
		t.Fatalf("profile: %s", prof)
	}
	if h.auditCount("login") != 1 || h.audits("create", "user") != 1 {
		t.Fatal("login and user creation must be audited")
	}
}

func TestHemisPasswordIsNeverStoredOrLogged(t *testing.T) {
	h := newHemisEnv(t)
	h.login("ali", hemisPW)
	h.login("ali", "wrong-password-xyz")
	if h.hemis.LastPass != "wrong-password-xyz" {
		t.Fatalf("hemis must receive the password, got %q", h.hemis.LastPass)
	}
	for _, q := range []string{
		`SELECT count(*) FROM users WHERE users::text LIKE '%` + hemisPW + `%' OR users::text LIKE '%wrong-password-xyz%'`,
		`SELECT count(*) FROM audit_logs WHERE audit_logs::text LIKE '%` + hemisPW + `%' OR audit_logs::text LIKE '%wrong-password-xyz%'`,
		`SELECT count(*) FROM sessions WHERE sessions::text LIKE '%` + hemisPW + `%'`,
	} {
		if h.n(q) != 0 {
			t.Fatalf("password leaked into the database: %s", q)
		}
	}
	// the audit log carries no passport numbers either
	if h.n(`SELECT count(*) FROM audit_logs WHERE changes::text LIKE '%12345678901234%' OR changes::text LIKE '%AB1234567%'`) != 0 {
		t.Fatal("passport data leaked into the audit log")
	}
}

func TestHemisSecondLoginUpdatesInsteadOfDuplicating(t *testing.T) {
	h := newHemisEnv(t)
	h.login("ali", hemisPW)
	var id1 = h.scalar(`SELECT id::text FROM users WHERE username='ali'`)

	p := aliProfile()
	p.FullName, p.ShortName, p.Level = "VALIYEV ALI (YANGI)", "Valiyev A.", "2-kurs"
	p.Group = &hemis.Ref{ID: 778, Name: "101-guruh (qayta nomlangan)"}
	p.Faculty = &hemis.Ref{ID: 56, Name: "Boshqa fakultet"}
	h.hemis.SetProfile("ali", p)
	if status, _, _ := h.login("ali", hemisPW); status != 200 {
		t.Fatal("second login failed")
	}
	if h.n(`SELECT count(*) FROM users`) != 1 || h.n(`SELECT count(*) FROM groups`) != 1 || h.scalar(`SELECT id::text FROM users`) != id1 {
		t.Fatal("a repeat login must update the same rows")
	}
	if h.scalar(`SELECT full_name FROM users`) != "VALIYEV ALI (YANGI)" || h.scalar(`SELECT short_name FROM users`) != "Valiyev A." ||
		h.scalar(`SELECT level FROM users`) != "2-kurs" || h.scalar(`SELECT name FROM groups`) != "101-guruh (qayta nomlangan)" ||
		h.scalar(`SELECT f.name FROM users u JOIN faculties f ON f.id=u.faculty_id`) != "Boshqa fakultet" {
		t.Fatal("profile changes from HEMIS were not applied")
	}
	if h.audits("update", "user") != 1 {
		t.Fatalf("one audited update expected, got %d", h.audits("update", "user"))
	}
	// an unchanged third login writes no update audit
	h.login("ali", hemisPW)
	if h.audits("update", "user") != 1 {
		t.Fatal("no-op profile sync must not be audited")
	}
}

func TestHemisWrongPasswordIsRejectedAndRateLimited(t *testing.T) {
	h := newHemisEnv(t)
	for i := 0; i < service.LoginAttempts; i++ {
		status, body, cookie := h.login("ali", "nope")
		if status != 401 || cookie != "" || errCode(t, body) != "invalid_credentials" {
			t.Fatalf("attempt %d: %d %s", i, status, body)
		}
	}
	if h.n(`SELECT count(*) FROM users`) != 0 {
		t.Fatal("a failed login must not create a user")
	}
	callsBefore := h.hemis.Calls
	status, body, _ := h.login("ali", hemisPW) // right password, but the limit is hit: HEMIS is not even asked
	if status != 429 || errCode(t, body) != "too_many_requests" || h.hemis.Calls != callsBefore {
		t.Fatalf("rate limit: %d %s (calls %d -> %d)", status, body, callsBefore, h.hemis.Calls)
	}
	if h.n(`SELECT count(*) FROM audit_logs WHERE action='login_failed' AND changes->'after'->>'reason'='hemis_invalid_credentials'`) != 5 {
		t.Fatal("failures must be audited with the reason")
	}
	if status, _, _ := h.login("unknown-login", "x"); status != 401 {
		t.Fatalf("unknown login: %d", status)
	}
}

func TestHemisOutageIsReportedClearlyAndNotCountedAgainstTheUser(t *testing.T) {
	h := newHemisEnv(t)
	h.hemis.SetDown(true)
	for i := 0; i < service.LoginAttempts+3; i++ { // more than the rate limit allows
		status, body, cookie := h.login("ali", hemisPW)
		if status != 503 || cookie != "" || errCode(t, body) != "hemis_unavailable" || !strings.Contains(body, "HEMIS") {
			t.Fatalf("outage attempt %d: %d %s", i, status, body)
		}
	}
	if h.n(`SELECT count(*) FROM audit_logs WHERE action='login_failed' AND changes->'after'->>'reason'='hemis_unavailable'`) != service.LoginAttempts+3 {
		t.Fatal("outage failures must be audited with their reason")
	}
	// HEMIS recovers: the very same user can log in immediately (no lockout from the outage)
	h.hemis.SetDown(false)
	if status, body, _ := h.login("ali", hemisPW); status != 200 {
		t.Fatalf("after recovery: %d %s", status, body)
	}
}

func TestStaffLoginStaysLocalAndNeverCallsHemis(t *testing.T) {
	h := newHemisEnv(t)
	h.user("adm", "admin", false)
	h.user("tea", "teacher", false)
	for _, u := range []string{"adm", "tea"} {
		if status, body, _ := h.login(u, testPass); status != 200 {
			t.Fatalf("%s: %d %s", u, status, body)
		}
	}
	if h.hemis.Calls != 0 {
		t.Fatalf("staff logins must not reach HEMIS, got %d calls", h.hemis.Calls)
	}
	if status, _, _ := h.login("adm", "wrong"); status != 401 || h.hemis.Calls != 0 {
		t.Fatalf("wrong staff password: %d, hemis calls %d", status, h.hemis.Calls)
	}
	// even if HEMIS happens to know a login with the same name, a staff account is never taken over
	h.hemis.Add("adm", "hemis-pw-for-adm", aliProfile())
	if status, _, _ := h.login("adm", "hemis-pw-for-adm"); status != 401 {
		t.Fatalf("HEMIS password must not unlock a staff account: %d", status)
	}
}

func TestHemisInactiveStudentCannotLogIn(t *testing.T) {
	h := newHemisEnv(t)
	h.login("ali", hemisPW)
	h.pool.Exec(context.Background(), `UPDATE users SET is_active=false WHERE username='ali'`)
	status, body, cookie := h.login("ali", hemisPW)
	if status != 403 || cookie != "" || errCode(t, body) != "account_inactive" || !strings.Contains(body, service.AccountInactiveMessage) {
		t.Fatalf("inactive: %d %s", status, body)
	}
	if h.scalar(`SELECT is_active::text FROM users WHERE username='ali'`) != "false" {
		t.Fatal("logging in must not reactivate a user")
	}
	if h.n(`SELECT count(*) FROM audit_logs WHERE action='login_failed' AND changes->'after'->>'reason'='inactive'`) != 1 {
		t.Fatal("inactive login must be audited")
	}
}

func TestHemisAdoptsSeededStudentAndProtectsUsernames(t *testing.T) {
	h := newHemisEnv(t)
	seeded := h.user("ali", "student", false) // pre-existing local student, no hemis_id
	h.pool.Exec(context.Background(), `UPDATE users SET password_hash=NULL WHERE id=$1`, seeded.ID)
	if status, body, _ := h.login("ali", hemisPW); status != 200 {
		t.Fatalf("adopt: %d %s", status, body)
	}
	if h.n(`SELECT count(*) FROM users`) != 1 || h.scalar(`SELECT id::text FROM users`) != seeded.ID.String() || h.scalar(`SELECT hemis_id::text FROM users`) != "3141592" {
		t.Fatal("the seeded student must be linked, not duplicated")
	}

	// a different HEMIS account must not take over an already linked username
	other := aliProfile()
	other.HemisID = 999
	h.hemis.Add("ali2", "pw2", other)
	h.pool.Exec(context.Background(), `UPDATE users SET username='ali2' WHERE id=$1`, seeded.ID) // linked to hemis_id 3141592
	if status, _, _ := h.login("ali2", "pw2"); status != 401 {
		t.Fatalf("username conflict must be rejected, got %d", status)
	}
	if h.n(`SELECT count(*) FROM audit_logs WHERE changes->'after'->>'reason'='username_conflict'`) != 1 {
		t.Fatal("conflict must be audited")
	}
}

func TestDevStudentPasswordLoginWorksAlongsideHemis(t *testing.T) {
	h := newHemisEnv(t, func(c *config.Config) { c.DevStudentPasswordLogin = true })
	h.user("stud", "student", false) // local password testPass
	if status, _, _ := h.login("stud", testPass); status != 200 || h.hemis.Calls != 0 {
		t.Fatalf("dev local login: %d, hemis calls %d", status, h.hemis.Calls)
	}
	// a wrong local password falls through to HEMIS, which does not know the student
	if status, _, _ := h.login("stud", "wrong"); status != 401 || h.hemis.Calls != 1 {
		t.Fatalf("fallback: %d, hemis calls %d", status, h.hemis.Calls)
	}

	// with the dev switch off (the production default) the local student password does nothing
	p := newHemisEnv(t)
	p.user("stud", "student", false)
	if status, _, _ := p.login("stud", testPass); status != 401 {
		t.Fatalf("prod-like: local student password must not work: %d", status)
	}
}

// ---- 6.3 photo sync ----

func TestHemisPhotoIsEnrolledOncePerPhoto(t *testing.T) {
	h := newHemisEnv(t)
	p := aliProfile()
	p.ImageURL = "https://hemis.test/uploads/ali-v1.jpg"
	h.hemis.SetProfile("ali", p)

	h.login("ali", hemisPW)
	if h.fetches.Load() != 1 || h.mock.calls.Load() != 1 {
		t.Fatalf("first login: %d downloads, %d embeds; want 1 and 1", h.fetches.Load(), h.mock.calls.Load())
	}
	if h.scalar(`SELECT array_length(face_embedding,1)::text FROM users`) != "3" || h.scalar(`SELECT hemis_img_url FROM users`) != p.ImageURL ||
		!strings.HasPrefix(h.scalar(`SELECT image FROM users`), "avatars/") || h.scalar(`SELECT (face_enrolled_at IS NOT NULL)::text FROM users`) != "true" {
		t.Fatal("photo not enrolled")
	}
	img := h.scalar(`SELECT image FROM users`)
	if _, err := os.Stat(filepath.Join(h.cfg.UploadDir, img)); err != nil {
		t.Fatal("avatar file missing")
	}

	// same photo: nothing recomputed
	h.login("ali", hemisPW)
	if h.fetches.Load() != 1 || h.mock.calls.Load() != 1 {
		t.Fatalf("unchanged photo reprocessed: %d downloads, %d embeds", h.fetches.Load(), h.mock.calls.Load())
	}

	// the photo changed at HEMIS: recomputed, and the old avatar file goes away
	p.ImageURL = "https://hemis.test/uploads/ali-v2.jpg"
	h.hemis.SetProfile("ali", p)
	h.login("ali", hemisPW)
	if h.fetches.Load() != 2 || h.mock.calls.Load() != 2 || h.scalar(`SELECT hemis_img_url FROM users`) != p.ImageURL {
		t.Fatalf("changed photo: %d downloads, %d embeds", h.fetches.Load(), h.mock.calls.Load())
	}
	if _, err := os.Stat(filepath.Join(h.cfg.UploadDir, img)); !os.IsNotExist(err) {
		t.Fatal("old avatar must be deleted")
	}
}

func TestHemisPhotoProblemsNeverBreakLogin(t *testing.T) {
	h := newHemisEnv(t)
	p := aliProfile()
	p.ImageURL = "https://hemis.test/uploads/ali.jpg"
	h.hemis.SetProfile("ali", p)

	// 1) the download fails
	h.fetchOK.Store(false)
	if status, _, _ := h.login("ali", hemisPW); status != 200 {
		t.Fatalf("download failure broke login: %d", status)
	}
	if h.scalar(`SELECT hemis_img_url FROM users`) != "" || h.n(`SELECT count(*) FROM users WHERE face_embedding IS NOT NULL`) != 0 {
		t.Fatal("nothing must be recorded for a failed sync")
	}
	// 2) face-service has no face in the photo
	h.fetchOK.Store(true)
	h.mock.set(422, `{"code":"no_face"}`)
	if status, _, _ := h.login("ali", hemisPW); status != 200 {
		t.Fatalf("no-face broke login: %d", status)
	}
	// 3) face-service is down
	h.mock.set(503, ``)
	if status, _, _ := h.login("ali", hemisPW); status != 200 {
		t.Fatalf("face outage broke login: %d", status)
	}
	if h.scalar(`SELECT hemis_img_url FROM users`) != "" {
		t.Fatal("a failed sync must leave the photo marked as not yet processed")
	}
	// 4) everything healthy again: the next login retries and succeeds
	h.mock.reset()
	h.login("ali", hemisPW)
	if h.scalar(`SELECT hemis_img_url FROM users`) != p.ImageURL || h.n(`SELECT count(*) FROM users WHERE face_embedding IS NOT NULL`) != 1 {
		t.Fatal("the sync must be retried on the next login")
	}
}

func TestHemisProfileWithoutPhotoOrGroup(t *testing.T) {
	h := newHemisEnv(t)
	p := aliProfile()
	p.Group, p.Faculty, p.ImageURL = nil, nil, ""
	h.hemis.SetProfile("ali", p)
	if status, body, _ := h.login("ali", hemisPW); status != 200 {
		t.Fatalf("%d %s", status, body)
	}
	if h.fetches.Load() != 0 || h.scalar(`SELECT (group_id IS NULL)::text FROM users`) != "true" {
		t.Fatal("no photo/group: nothing to sync")
	}
}
