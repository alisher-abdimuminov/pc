// Package hemis is the client of the university's HEMIS student API.
//
// It assumes the standard HEMIS student REST API:
//
//	POST {base}/rest/v1/auth/login   {"login": "...", "password": "..."}  -> {"success": true, "data": {"token": "..."}}
//	GET  {base}/rest/v1/account/me   Authorization: Bearer <token>        -> {"success": true, "data": {...profile...}}
//
// The HEMIS password is only ever forwarded; it is never stored or logged.
package hemis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidCredentials: HEMIS rejected the login or password.
	ErrInvalidCredentials = errors.New("hemis: invalid credentials")
	// ErrUnavailable: HEMIS is down, timed out, or answered with something unusable.
	ErrUnavailable = errors.New("hemis: service unavailable")
)

// UnavailableMessage is shown to the user when HEMIS cannot be reached.
const UnavailableMessage = "HEMIS tizimi vaqtincha ishlamayapti. Birozdan so'ng qayta urinib ko'ring"

// Client is what the app needs from HEMIS (mocked in tests).
type Client interface {
	Login(ctx context.Context, username, password string) (token string, err error)
	Me(ctx context.Context, token string) (Profile, error)
}

type Ref struct {
	ID   int64
	Name string
}

// Profile is the normalised student profile from /account/me.
type Profile struct {
	HemisID        int64
	FirstName      string
	SecondName     string
	ThirdName      string
	FullName       string
	ShortName      string
	ImageURL       string
	BirthDate      string // YYYY-MM-DD
	Phone          string
	PassportPin    string
	PassportNumber string
	Gender         string
	PaymentForm    string
	Level          string
	Semester       string
	GPA            string
	Address        string
	Country        string
	Province       string
	District       string
	Group          *Ref
	Faculty        *Ref
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
	for _, layout := range []string{"2006-01-02", "02.01.2006", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return s
}

type meJSON struct {
	ID             flexID  `json:"id"`
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

func (m meJSON) profile() Profile {
	p := Profile{
		HemisID: int64(m.ID), FirstName: string(m.FirstName), SecondName: string(m.SecondName), ThirdName: string(m.ThirdName),
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

// ---- HTTP client ----

type HTTP struct {
	base string
	http *http.Client
}

func New(baseURL string, timeout time.Duration) *HTTP {
	return &HTTP{base: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

func (c *HTTP) do(ctx context.Context, method, path string, body any, token string, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return 0, err
		}
		var ne net.Error
		reason := "network error"
		if errors.As(err, &ne) && ne.Timeout() {
			reason = "timeout"
		}
		return 0, fmt.Errorf("%w: %s", ErrUnavailable, reason) // never include err: its URL could carry data
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode == http.StatusOK && out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("%w: unreadable response", ErrUnavailable)
		}
	}
	return resp.StatusCode, nil
}

type envelope[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

func (c *HTTP) Login(ctx context.Context, username, password string) (string, error) {
	var out envelope[struct {
		Token string `json:"token"`
	}]
	status, err := c.do(ctx, http.MethodPost, "/rest/v1/auth/login", map[string]string{"login": username, "password": password}, "", &out)
	if err != nil {
		return "", err
	}
	switch {
	case status == http.StatusOK && out.Success && out.Data.Token != "":
		return out.Data.Token, nil
	case status == http.StatusOK: // 200 with success=false or no token: HEMIS says "no"
		return "", ErrInvalidCredentials
	case status == http.StatusBadRequest || status == http.StatusUnauthorized || status == http.StatusForbidden ||
		status == http.StatusNotFound || status == http.StatusUnprocessableEntity:
		return "", ErrInvalidCredentials
	default: // 5xx, 429, ...
		return "", fmt.Errorf("%w: status %d", ErrUnavailable, status)
	}
}

func (c *HTTP) Me(ctx context.Context, token string) (Profile, error) {
	var out envelope[meJSON]
	status, err := c.do(ctx, http.MethodGet, "/rest/v1/account/me", nil, token, &out)
	if err != nil {
		return Profile{}, err
	}
	if status != http.StatusOK || !out.Success {
		return Profile{}, fmt.Errorf("%w: status %d", ErrUnavailable, status)
	}
	p := out.Data.profile()
	if p.HemisID == 0 {
		return Profile{}, fmt.Errorf("%w: profile has no id", ErrUnavailable)
	}
	return p, nil
}
