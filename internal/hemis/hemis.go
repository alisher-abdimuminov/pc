// Package hemis holds the clients of the university's HEMIS system: the OAuth sign-in for students
// and teachers, the group and faculty lists and the photo downloader. Passwords never reach us:
// users sign in on HEMIS itself.
package hemis

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrUnavailable: HEMIS is down, timed out, or answered with something unusable.
	ErrUnavailable = errors.New("hemis: service unavailable")
)

// UnavailableMessage is shown to the user when HEMIS cannot be reached.
const UnavailableMessage = "HEMIS tizimi vaqtincha ishlamayapti. Birozdan so'ng qayta urinib ko'ring"

type Ref struct {
	ID   int64
	Name string
}

// Profile is the normalised student profile (the resource-owner response of the student OAuth).
type Profile struct {
	HemisID         int64
	StudentIDNumber string
	FirstName       string
	SecondName      string
	ThirdName       string
	FullName        string
	ShortName       string
	ImageURL        string
	BirthDate       string // YYYY-MM-DD
	Phone           string
	PassportPin     string
	PassportNumber  string
	Gender          string
	PaymentForm     string
	Level           string
	Semester        string
	GPA             string
	Address         string
	Country         string
	Province        string
	District        string
	Group           *Ref
	Faculty         *Ref
}

// ---- tolerant JSON decoding: HEMIS returns scalars or {code,name} objects for the same field ----

type flex string

func (f *flex) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	switch {
	case s == "null" || s == "":
		*f = ""
	case s[0] == '"':
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*f = flex(strings.TrimSpace(v))
	case s[0] == '{':
		var o struct {
			Name  string `json:"name"`
			Label string `json:"label"`
			Code  any    `json:"code"`
		}
		if err := json.Unmarshal(b, &o); err != nil {
			return err
		}
		switch {
		case o.Name != "":
			*f = flex(strings.TrimSpace(o.Name))
		case o.Label != "":
			*f = flex(strings.TrimSpace(o.Label))
		case o.Code != nil:
			*f = flex(fmt.Sprint(o.Code))
		}
	default: // number or bool
		*f = flex(s)
	}
	return nil
}

type flexID int64

func (f *flexID) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("bad id %q", s)
	}
	*f = flexID(n)
	return nil
}

type refJSON struct {
	ID   flexID `json:"id"`
	Name flex   `json:"name"`
}

func (r refJSON) ref() *Ref {
	if r.ID == 0 && r.Name == "" {
		return nil
	}
	return &Ref{ID: int64(r.ID), Name: string(r.Name)}
}

// birthDate accepts "2004-05-06", "06.05.2004" or a unix timestamp (seconds or milliseconds).
func parseBirthDate(raw flex) string {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return ""
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		if n > 1e11 {
			n /= 1000
		}
		return time.Unix(n, 0).UTC().Format("2006-01-02")
	}
	for _, layout := range []string{"2006-01-02", "02.01.2006", "02-01-2006", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return s
}

type meJSON struct {
	ID             flexID  `json:"id"`
	StudentID      flex    `json:"student_id_number"`
	FirstName      flex    `json:"first_name"`
	SecondName     flex    `json:"second_name"`
	ThirdName      flex    `json:"third_name"`
	FullName       flex    `json:"full_name"`
	ShortName      flex    `json:"short_name"`
	Image          flex    `json:"image"`
	BirthDate      flex    `json:"birth_date"`
	Phone          flex    `json:"phone"`
	PassportPin    flex    `json:"passport_pin"`
	PassportNumber flex    `json:"passport_number"`
	Gender         flex    `json:"gender"`
	PaymentForm    flex    `json:"paymentForm"`
	Level          flex    `json:"level"`
	Semester       flex    `json:"semester"`
	AvgGPA         flex    `json:"avg_gpa"`
	Address        flex    `json:"address"`
	Country        flex    `json:"country"`
	Province       flex    `json:"province"`
	District       flex    `json:"district"`
	Group          refJSON `json:"group"`
	Department     refJSON `json:"department"`
	Faculty        refJSON `json:"faculty"`
}

// withFallback fills the fields that the full record lacks from the summary (the summary alone carries
// the passport data in the student OAuth answer).
func (m meJSON) withFallback(o meJSON) meJSON {
	if m.ID == 0 {
		m.ID = o.ID
	}
	for _, f := range []struct{ dst, src *flex }{
		{&m.StudentID, &o.StudentID}, {&m.Phone, &o.Phone}, {&m.PassportPin, &o.PassportPin},
		{&m.PassportNumber, &o.PassportNumber}, {&m.BirthDate, &o.BirthDate}, {&m.Image, &o.Image},
	} {
		if *f.dst == "" {
			*f.dst = *f.src
		}
	}
	return m
}

func (m meJSON) profile() Profile {
	p := Profile{
		HemisID: int64(m.ID), StudentIDNumber: string(m.StudentID), FirstName: string(m.FirstName), SecondName: string(m.SecondName), ThirdName: string(m.ThirdName),
		FullName: string(m.FullName), ShortName: string(m.ShortName), ImageURL: string(m.Image), BirthDate: parseBirthDate(m.BirthDate),
		Phone: string(m.Phone), PassportPin: string(m.PassportPin), PassportNumber: string(m.PassportNumber), Gender: string(m.Gender),
		PaymentForm: string(m.PaymentForm), Level: string(m.Level), Semester: string(m.Semester), GPA: string(m.AvgGPA),
		Address: string(m.Address), Country: string(m.Country), Province: string(m.Province), District: string(m.District),
		Group: m.Group.ref(),
	}
	// In HEMIS the faculty is the student's "department".
	if p.Faculty = m.Department.ref(); p.Faculty == nil {
		p.Faculty = m.Faculty.ref()
	}
	if p.FullName == "" {
		p.FullName = strings.TrimSpace(strings.Join([]string{p.SecondName, p.FirstName, p.ThirdName}, " "))
	}
	return p
}
