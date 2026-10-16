package hemis

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
)

const meBody = `{"success":true,"error":null,"code":0,"data":{
  "id": 3141592, "student_id_number":"320261100000",
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

func studentClient(t *testing.T, h http.HandlerFunc) *OAuth { return oauthClient(t, h) }

func TestStudentProfileParsesRealisticProfile(t *testing.T) {
	c := studentClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok123" {
			t.Errorf("bad auth %q", r.Header.Get("Authorization"))
		}
		w.Write([]byte(meBody))
	})
	p, err := c.StudentProfile(context.Background(), "tok123")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]string{
		"first": {p.FirstName, "ALI"}, "second": {p.SecondName, "VALIYEV"}, "third": {p.ThirdName, "KARIM O'G'LI"},
		"full": {p.FullName, "VALIYEV ALI KARIM O'G'LI"}, "short": {p.ShortName, "Valiyev A.K."}, "student id": {p.StudentIDNumber, "320261100000"},
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

func TestStudentProfileToleratesSparseAndBarePayloads(t *testing.T) {
	c := studentClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":9,"first_name":"Ali","second_name":"Valiyev","third_name":null,
			"birth_date":"06.05.2004","gender":"Erkak","group":null,"faculty":{"id":3,"name":"Iqtisod"}}`))
	})
	p, err := c.StudentProfile(context.Background(), "t")
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

func TestStudentProfileFailures(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"401": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) },
		"no id": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":true,"data":{"first_name":"A"}}`))
		},
		"garbage": func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`nope`)) },
	} {
		if _, err := studentClient(t, h).StudentProfile(context.Background(), "t"); !errors.Is(err, ErrOAuthProfile) {
			t.Errorf("%s: got %v", name, err)
		}
	}
	h500 := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }
	if _, err := studentClient(t, h500).StudentProfile(context.Background(), "t"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("500: %v", err)
	}
}

func TestStudentProfileFromTheRealShape(t *testing.T) {
	body, err := os.ReadFile("testdata/student_me_ok.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := studentClient(t, func(w http.ResponseWriter, r *http.Request) { w.Write(body) }).StudentProfile(context.Background(), "t")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{
		"student id": p.StudentIDNumber, "full": p.FullName, "first": p.FirstName, "short": p.ShortName,
		"birth": p.BirthDate, "pin": p.PassportPin, "passport": p.PassportNumber, "phone": p.Phone,
		"image": p.ImageURL, "gender": p.Gender, "level": p.Level, "semester": p.Semester, "gpa": p.GPA,
		"province": p.Province, "payment": p.PaymentForm,
	}
	want := map[string]string{
		"student id": "320230000001", "full": "VALIYEV ALI KARIM O'G'LI", "first": "ALI", "short": "VALIYEV A. K.",
		"birth": "2005-08-09", "pin": "00000000000000", "passport": "AA0000000", "phone": "+998900000000",
		"image": "https://hemis.example.uz/static/crop/320_320_90_x.jpg", "gender": "Ayol", "level": "4-kurs",
		"semester": "7-semestr", "gpa": "3.22", "province": "Samarqand viloyati", "payment": "To'lov-shartnoma",
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: got %q want %q", k, got[k], w)
		}
	}
	if p.HemisID != 8994 || p.Group == nil || p.Group.ID != 579 || p.Faculty == nil || p.Faculty.ID != 3 || p.Faculty.Name != "Pedagogika" {
		t.Errorf("ids/group/faculty: %+v %+v %+v", p.HemisID, p.Group, p.Faculty)
	}
}

func TestStudentProfileSummaryOnlyStillWorks(t *testing.T) {
	c := studentClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":5,"login":"3202","name":"A B","firstname":"B","surname":"A","picture_full":"http://x/p.jpg","birth_date":"09-08-2005"}`))
	})
	p, err := c.StudentProfile(context.Background(), "t")
	if err != nil || p.StudentIDNumber != "3202" || p.FullName != "A B" || p.ImageURL != "http://x/p.jpg" || p.BirthDate != "2005-08-09" {
		t.Fatalf("%+v %v", p, err)
	}
}
