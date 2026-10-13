package hemis

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func oauthClient(t *testing.T, h http.HandlerFunc) *OAuth {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewOAuth(OAuthConfig{
		ClientID: "cid", ClientSecret: "csecret", RedirectURI: "http://app.test/auth/callback?x=1",
		AuthorizeURL: srv.URL + "/authorize?lang=uz", TokenURL: srv.URL + "/token", UserURL: srv.URL + "/user",
	}, 2*time.Second)
}

func TestAuthorizeURLEncodesEverything(t *testing.T) {
	c := NewOAuth(OAuthConfig{ClientID: "a b", RedirectURI: "http://app.test/auth/callback?x=1&y=2", AuthorizeURL: "https://h.test/oauth/authorize?lang=uz"}, 0)
	u, err := url.Parse(c.AuthorizeURL("st&te=1"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Host != "h.test" || u.Path != "/oauth/authorize" || q.Get("lang") != "uz" || q.Get("client_id") != "a b" ||
		q.Get("redirect_uri") != "http://app.test/auth/callback?x=1&y=2" || q.Get("response_type") != "code" || q.Get("state") != "st&te=1" {
		t.Fatalf("bad url %s", u)
	}
	if strings.Contains(u.String(), "secret") {
		t.Fatal("no secret in the authorize URL")
	}
}

func TestExchangeCodeSendsFormAndReadsToken(t *testing.T) {
	c := oauthClient(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Method != "POST" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") ||
			r.Form.Get("client_id") != "cid" || r.Form.Get("client_secret") != "csecret" || r.Form.Get("code") != "the code" ||
			r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("redirect_uri") != "http://app.test/auth/callback?x=1" {
			t.Errorf("bad token request: %v", r.Form)
		}
		w.Write([]byte(`{"access_token":"AT","token_type":"bearer"}`))
	})
	tok, err := c.ExchangeCode(context.Background(), "the code")
	if err != nil || tok != "AT" {
		t.Fatalf("%q %v", tok, err)
	}
}

func TestExchangeCodeErrors(t *testing.T) {
	reply := func(status int, body string) *OAuth {
		return oauthClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); w.Write([]byte(body)) })
	}
	for name, c := range map[string]*OAuth{
		"400":               reply(400, `{"error":"invalid_grant"}`),
		"401":               reply(401, ``),
		"200 no token":      reply(200, `{"token_type":"bearer"}`),
		"200 empty token":   reply(200, `{"access_token":"  "}`),
		"200 not json":      reply(200, `oops`),
		"200 array":         reply(200, `[]`),
		"200 token numeric": reply(200, `{"access_token":123}`),
	} {
		if _, err := c.ExchangeCode(context.Background(), "c"); !errors.Is(err, ErrOAuthToken) {
			t.Errorf("%s: got %v", name, err)
		}
	}
	for name, c := range map[string]*OAuth{"500": reply(500, `x`), "503": reply(503, ``), "429": reply(429, ``)} {
		if _, err := c.ExchangeCode(context.Background(), "c"); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: got %v", name, err)
		}
	}
	down := NewOAuth(OAuthConfig{TokenURL: "http://127.0.0.1:1/token"}, time.Second)
	if _, err := down.ExchangeCode(context.Background(), "c"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("connection refused: %v", err)
	}
}

func TestErrorsCarryNoSecrets(t *testing.T) {
	c := oauthClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"bad code SUPERSECRETCODE","access_token":"LEAKEDTOKEN"}`))
	})
	_, err := c.ExchangeCode(context.Background(), "SUPERSECRETCODE")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, s := range []string{"SUPERSECRETCODE", "LEAKEDTOKEN", "csecret", "127.0.0.1"} {
		if strings.Contains(err.Error(), s) {
			t.Fatalf("error text leaks %q: %v", s, err)
		}
	}
}

func TestTeacherProfileDecoding(t *testing.T) {
	cases := map[string]struct {
		body string
		want TeacherProfile
	}{
		"full": {`{"login":"t.k","employee_id":7,"id":99,"name":"A B","firstname":"B","surname":"A","patronymic":"C","phone":"+99890","passport_number":"AA1","passport_pin":"123","birth_date":"1985-03-12","picture_full":"http://x/p.jpg"}`,
			TeacherProfile{Login: "t.k", HemisID: 7, FullName: "A B", FirstName: "B", SecondName: "A", ThirdName: "C", Phone: "+99890", PassportNumber: "AA1", PassportPin: "123", BirthDate: "1985-03-12", PictureURL: "http://x/p.jpg"}},
		"id fallback and numeric birth date": {`{"login":"t.k","id":"42","birth_date":479001600}`,
			TeacherProfile{Login: "t.k", HemisID: 42, BirthDate: "1985-03-07"}},
		"millisecond birth date and null fields": {`{"login":"t.k","birth_date":479001600000,"phone":null,"name":null}`,
			TeacherProfile{Login: "t.k", BirthDate: "1985-03-07"}},
		"numeric login":   {`{"login":12345}`, TeacherProfile{Login: "12345"}},
		"wrapped in data": {`{"success":true,"data":{"login":"t.k","name":"W"}}`, TeacherProfile{Login: "t.k", FullName: "W"}},
	}
	for name, c := range cases {
		cl := oauthClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer AT" {
				w.WriteHeader(401)
				return
			}
			w.Write([]byte(c.body))
		})
		got, err := cl.TeacherProfile(context.Background(), "AT")
		if err != nil || *got != c.want {
			t.Errorf("%s: %+v %v (want %+v)", name, got, err, c.want)
		}
	}
}

func TestTeacherProfileErrors(t *testing.T) {
	reply := func(status int, body string) *OAuth {
		return oauthClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); w.Write([]byte(body)) })
	}
	for name, c := range map[string]*OAuth{
		"401": reply(401, ``), "403": reply(403, `{}`), "no login": reply(200, `{"name":"x"}`),
		"blank login": reply(200, `{"login":" "}`), "not json": reply(200, `<html>`),
	} {
		if _, err := c.TeacherProfile(context.Background(), "AT"); !errors.Is(err, ErrOAuthProfile) {
			t.Errorf("%s: got %v", name, err)
		}
	}
	if _, err := reply(500, ``).TeacherProfile(context.Background(), "AT"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("500: %v", err)
	}
}

func TestPhotoFetcherRules(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/ok.png":
			w.Write(png)
		case "/text.png":
			w.Write([]byte("<html>not an image</html>"))
		case "/huge.png":
			w.Write(append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, MaxPhotoSize)...))
		case "/hop":
			http.Redirect(w, r, "http://elsewhere.invalid/p.png", http.StatusFound)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")
	fetch := NewPhotoFetcher(srv.URL, nil, 2*time.Second)

	if data, name, err := fetch(context.Background(), srv.URL+"/ok.png"); err != nil || name != "ok.png" || len(data) != len(png) {
		t.Fatalf("ok: %v %q", err, name)
	}
	if _, _, err := fetch(context.Background(), "/ok.png"); err != nil { // relative to the base URL
		t.Fatalf("relative: %v", err)
	}
	for name, u := range map[string]string{
		"other host": "http://other.invalid/p.png", "ftp": "ftp://" + host + "/p.png", "file": "file:///etc/passwd",
		"not an image": srv.URL + "/text.png", "too large": srv.URL + "/huge.png", "404": srv.URL + "/missing.png", "redirect off-host": srv.URL + "/hop",
	} {
		if _, _, err := fetch(context.Background(), u); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
	// an extra allowlisted host is accepted
	allow := NewPhotoFetcher("http://base.invalid", []string{host}, 2*time.Second)
	if _, _, err := allow(context.Background(), srv.URL+"/ok.png"); err != nil {
		t.Fatalf("allowlisted host: %v", err)
	}
}
