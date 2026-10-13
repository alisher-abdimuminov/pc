package hemis

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const meBody = `{"success":true,"error":null,"code":0,"data":{
  "id": 3141592,
  "first_name":"ALI","second_name":"VALIYEV","third_name":"KARIM O'G'LI","full_name":"VALIYEV ALI KARIM O'G'LI","short_name":"Valiyev A.K.",
  "image":"https://student.example.uz/uploads/3141592.jpg",
  "birth_date": 1083801600,
  "phone":"+998901234567","passport_pin":"12345678901234","passport_number":"AB1234567",
  "gender":{"code":"11","name":"Erkak"},
  "paymentForm":{"code":"12","name":"Davlat granti"},
  "level":{"code":"12","name":"1-kurs"},
  "semester":{"code":"11","name":"1-semestr"},
  "avg_gpa": 3.75,
  "address":"Toshkent, Chilonzor",
  "country":{"code":"UZ","name":"O'zbekiston"},"province":{"code":"1726","name":"Toshkent shahri"},"district":{"name":"Chilonzor"},
  "group":{"id":"778","name":"101-guruh"},
  "department":{"id":55,"name":"Informatika fakulteti"}
}}`

func server(t *testing.T, h http.HandlerFunc) *HTTP {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(srv.URL, 2*time.Second)
}

func TestLoginSendsCredentialsAndReturnsToken(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request) {
		var in map[string]string
		json.NewDecoder(r.Body).Decode(&in)
		if r.Method != "POST" || r.URL.Path != "/rest/v1/auth/login" || in["login"] != "ali" || in["password"] != "s3cret" {
			t.Errorf("bad request: %s %s %v", r.Method, r.URL.Path, in)
		}
		w.Write([]byte(`{"success":true,"data":{"token":"tok123"}}`))
	})
	tok, err := c.Login(context.Background(), "ali", "s3cret")
	if err != nil || tok != "tok123" {
		t.Fatalf("%q %v", tok, err)
	}
}

func TestLoginRejections(t *testing.T) {
	reply := func(status int, body string) *HTTP {
		return server(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); w.Write([]byte(body)) })
	}
	invalid := map[string]*HTTP{
		"401":               reply(401, `{"success":false,"error":"Login yoki parol xato"}`),
		"400":               reply(400, `{}`),
		"200 success=false": reply(200, `{"success":false,"data":null}`),
		"200 without token": reply(200, `{"success":true,"data":{}}`),
		"422":               reply(422, `{}`),
	}
	for name, c := range invalid {
		if _, err := c.Login(context.Background(), "u", "p"); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("%s: got %v want ErrInvalidCredentials", name, err)
		}
	}
	down := map[string]*HTTP{
		"500":         reply(500, `boom`),
		"502":         reply(502, ``),
		"503":         reply(503, ``),
		"429":         reply(429, ``),
		"garbage 200": reply(200, `<html>maintenance</html>`),
	}
	for name, c := range down {
		if _, err := c.Login(context.Background(), "u", "p"); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: got %v want ErrUnavailable", name, err)
		}
	}
}

func TestOutageErrorsNeverLeakCredentials(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	_, err := New(url, time.Second).Login(context.Background(), "ali-login", "very-secret-pw")
	if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "very-secret-pw") || strings.Contains(err.Error(), "ali-login") {
		t.Fatalf("unsafe or wrong error: %v", err)
	}
}

func TestTimeoutIsUnavailable(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request) { time.Sleep(300 * time.Millisecond) })
	c.http.Timeout = 50 * time.Millisecond
	_, err := c.Login(context.Background(), "u", "p")
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("got %v", err)
	}
}

func TestMeParsesRealisticProfile(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/v1/account/me" || r.Header.Get("Authorization") != "Bearer tok123" {
			t.Errorf("bad request: %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		w.Write([]byte(meBody))
	})
	p, err := c.Me(context.Background(), "tok123")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]string{
		"first": {p.FirstName, "ALI"}, "second": {p.SecondName, "VALIYEV"}, "third": {p.ThirdName, "KARIM O'G'LI"},
		"full": {p.FullName, "VALIYEV ALI KARIM O'G'LI"}, "short": {p.ShortName, "Valiyev A.K."},
		"image": {p.ImageURL, "https://student.example.uz/uploads/3141592.jpg"}, "birth": {p.BirthDate, "2004-05-06"},
		"phone": {p.Phone, "+998901234567"}, "pin": {p.PassportPin, "12345678901234"}, "passport": {p.PassportNumber, "AB1234567"},
		"gender": {p.Gender, "Erkak"}, "payment": {p.PaymentForm, "Davlat granti"}, "level": {p.Level, "1-kurs"},
		"semester": {p.Semester, "1-semestr"}, "gpa": {p.GPA, "3.75"}, "address": {p.Address, "Toshkent, Chilonzor"},
		"country": {p.Country, "O'zbekiston"}, "province": {p.Province, "Toshkent shahri"}, "district": {p.District, "Chilonzor"},
	}
	for k, v := range want {
		if v[0] != v[1] {
			t.Errorf("%s: got %q want %q", k, v[0], v[1])
		}
	}
	if p.HemisID != 3141592 {
		t.Errorf("id: %d", p.HemisID)
	}
	if p.Group == nil || p.Group.ID != 778 || p.Group.Name != "101-guruh" {
		t.Errorf("group (string id): %+v", p.Group)
	}
	if p.Faculty == nil || p.Faculty.ID != 55 || p.Faculty.Name != "Informatika fakulteti" {
		t.Errorf("faculty (from department): %+v", p.Faculty)
	}
}

func TestMeToleratesSparseAndVariantPayloads(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"data":{"id":9,"first_name":"Ali","second_name":"Valiyev","third_name":null,
			"birth_date":"06.05.2004","gender":"Erkak","group":null,"faculty":{"id":3,"name":"Iqtisod"}}}`))
	})
	p, err := c.Me(context.Background(), "t")
	if err != nil {
		t.Fatal(err)
	}
	if p.FullName != "Valiyev Ali" || p.BirthDate != "2004-05-06" || p.Gender != "Erkak" || p.Group != nil || p.Faculty == nil || p.Faculty.Name != "Iqtisod" {
		t.Fatalf("%+v", p)
	}
	if got := parseBirthDate("1083801600000"); got != "2004-05-06" {
		t.Fatalf("millisecond timestamp: %s", got)
	}
}

func TestMeFailures(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"401": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) },
		"500": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) },
		"no id": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":true,"data":{"first_name":"A"}}`))
		},
		"success=false": func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"success":false}`)) },
		"garbage":       func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`nope`)) },
	} {
		if _, err := server(t, h).Me(context.Background(), "t"); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

func TestFakeBehavesLikeTheClient(t *testing.T) {
	f := NewFake()
	f.Add("ali", "pw", Profile{HemisID: 1, FullName: "A"})
	var c Client = f
	tok, err := c.Login(context.Background(), "ali", "pw")
	if err != nil {
		t.Fatal(err)
	}
	if p, err := c.Me(context.Background(), tok); err != nil || p.HemisID != 1 {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := c.Login(context.Background(), "ali", "bad"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("bad password")
	}
	f.SetDown(true)
	if _, err := c.Login(context.Background(), "ali", "pw"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("down")
	}
}
