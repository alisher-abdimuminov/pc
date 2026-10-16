package hemis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrOAuthToken: HEMIS refused the authorization code or sent no access token.
	ErrOAuthToken = errors.New("hemis oauth: token exchange failed")
	// ErrOAuthProfile: the resource-owner call failed or returned no usable profile.
	ErrOAuthProfile = errors.New("hemis oauth: profile request failed")
)

// OAuthTimeout bounds every HEMIS OAuth call.
const OAuthTimeout = 15 * time.Second

// TeacherOAuth is the teacher sign-in side of HEMIS (mocked in tests).
type TeacherOAuth interface {
	AuthorizeURL(state string) string
	ExchangeCode(ctx context.Context, code string) (accessToken string, err error)
	TeacherProfile(ctx context.Context, accessToken string) (*TeacherProfile, error)
}

// StudentOAuth is the student sign-in side of HEMIS (mocked in tests). The resource-owner response is
// the student profile, the same shape HEMIS gives for /account/me.
type StudentOAuth interface {
	AuthorizeURL(state string) string
	ExchangeCode(ctx context.Context, code string) (accessToken string, err error)
	StudentProfile(ctx context.Context, accessToken string) (*Profile, error)
}

// TeacherProfile is the normalised resource-owner response. Everything except Login is optional.
type TeacherProfile struct {
	Login          string
	HemisID        int64
	FullName       string
	FirstName      string
	SecondName     string
	ThirdName      string
	Phone          string
	PassportNumber string
	PassportPin    string
	BirthDate      string // YYYY-MM-DD
	PictureURL     string
}

// OAuthConfig holds the endpoints and credentials. The secret is only ever sent to TokenURL.
type OAuthConfig struct {
	ClientID, ClientSecret, RedirectURI string
	AuthorizeURL, TokenURL, UserURL     string
}

// OAuth is the standard authorization-code client.
type OAuth struct {
	cfg  OAuthConfig
	http *http.Client
}

func NewOAuth(cfg OAuthConfig, timeout time.Duration) *OAuth {
	if timeout <= 0 {
		timeout = OAuthTimeout
	}
	return &OAuth{cfg: cfg, http: &http.Client{Timeout: timeout}}
}

// AuthorizeURL is where the browser is sent to sign in at HEMIS.
func (o *OAuth) AuthorizeURL(state string) string {
	u, err := url.Parse(o.cfg.AuthorizeURL)
	if err != nil {
		return o.cfg.AuthorizeURL
	}
	q := u.Query()
	q.Set("client_id", o.cfg.ClientID)
	q.Set("redirect_uri", o.cfg.RedirectURI)
	q.Set("response_type", "code")
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}

// call performs the request and returns the body of a 2xx answer. Errors never contain URLs,
// bodies, codes or tokens: only the class of failure.
func (o *OAuth) call(req *http.Request, bad error) ([]byte, error) {
	resp, err := o.http.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		reason := "network error"
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			reason = "timeout"
		}
		return nil, fmt.Errorf("%w: %s", ErrUnavailable, reason)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	switch {
	case resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return nil, fmt.Errorf("%w: status %d", bad, resp.StatusCode)
	}
	return raw, nil
}

// ExchangeCode trades the authorization code for an access token.
func (o *OAuth) ExchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{
		"client_id":     {o.cfg.ClientID},
		"client_secret": {o.cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {o.cfg.RedirectURI},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", ErrOAuthToken
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	raw, err := o.call(req, ErrOAuthToken)
	if err != nil {
		return "", err
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if json.Unmarshal(raw, &out) != nil || strings.TrimSpace(out.AccessToken) == "" {
		return "", fmt.Errorf("%w: no access token", ErrOAuthToken)
	}
	return out.AccessToken, nil
}

type teacherJSON struct {
	Login          flex   `json:"login"`
	ID             flexID `json:"id"`
	EmployeeID     flexID `json:"employee_id"`
	Name           flex   `json:"name"`
	FirstName      flex   `json:"firstname"`
	Surname        flex   `json:"surname"`
	Patronymic     flex   `json:"patronymic"`
	Phone          flex   `json:"phone"`
	PassportNumber flex   `json:"passport_number"`
	PassportPin    flex   `json:"passport_pin"`
	BirthDate      flex   `json:"birth_date"`
	PictureFull    flex   `json:"picture_full"`
}

func (t teacherJSON) profile() *TeacherProfile {
	id := t.EmployeeID
	if id == 0 {
		id = t.ID
	}
	return &TeacherProfile{
		Login: string(t.Login), HemisID: int64(id), FullName: string(t.Name),
		FirstName: string(t.FirstName), SecondName: string(t.Surname), ThirdName: string(t.Patronymic),
		Phone: string(t.Phone), PassportNumber: string(t.PassportNumber), PassportPin: string(t.PassportPin),
		BirthDate: parseBirthDate(t.BirthDate), PictureURL: string(t.PictureFull),
	}
}

// TeacherProfile fetches the signed-in teacher. A profile without a login is unusable.
func (o *OAuth) TeacherProfile(ctx context.Context, accessToken string) (*TeacherProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.cfg.UserURL, nil)
	if err != nil {
		return nil, ErrOAuthProfile
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	raw, err := o.call(req, ErrOAuthProfile)
	if err != nil {
		return nil, err
	}
	var top struct {
		teacherJSON
		Data *teacherJSON `json:"data"`
	}
	if json.Unmarshal(raw, &top) != nil {
		return nil, fmt.Errorf("%w: unreadable response", ErrOAuthProfile)
	}
	t := top.teacherJSON
	if t.Login == "" && top.Data != nil { // some deployments wrap the user in {"data": {...}}
		t = *top.Data
	}
	p := t.profile()
	if p.Login == "" {
		return nil, fmt.Errorf("%w: no login in profile", ErrOAuthProfile)
	}
	return p, nil
}

// StudentProfile fetches the signed-in student. The profile may be wrapped in {"success":..,"data":{..}}.
func (o *OAuth) StudentProfile(ctx context.Context, accessToken string) (*Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.cfg.UserURL, nil)
	if err != nil {
		return nil, ErrOAuthProfile
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	raw, err := o.call(req, ErrOAuthProfile)
	if err != nil {
		return nil, err
	}
	// HEMIS answers with a short summary at the top level (login, id, passport, picture_full...) and the
	// full student record under "data". Some deployments return only one of the two.
	var top struct {
		meJSON
		Login       flex    `json:"login"`
		PictureFull flex    `json:"picture_full"`
		Name        flex    `json:"name"`
		Firstname   flex    `json:"firstname"`
		Surname     flex    `json:"surname"`
		Patronymic  flex    `json:"patronymic"`
		Data        *meJSON `json:"data"`
	}
	if json.Unmarshal(raw, &top) != nil {
		return nil, fmt.Errorf("%w: unreadable response", ErrOAuthProfile)
	}
	m := top.meJSON
	if top.Data != nil {
		m = top.Data.withFallback(top.meJSON)
	}
	first := func(vals ...flex) flex {
		for _, v := range vals {
			if v != "" {
				return v
			}
		}
		return ""
	}
	m.FirstName = first(m.FirstName, top.Firstname)
	m.SecondName = first(m.SecondName, top.Surname)
	m.ThirdName = first(m.ThirdName, top.Patronymic)
	m.FullName = first(m.FullName, top.Name)
	m.Image = first(m.Image, top.PictureFull)
	m.StudentID = first(m.StudentID, top.Login)
	p := m.profile()
	if p.HemisID == 0 {
		return nil, fmt.Errorf("%w: profile has no id", ErrOAuthProfile)
	}
	return &p, nil
}
