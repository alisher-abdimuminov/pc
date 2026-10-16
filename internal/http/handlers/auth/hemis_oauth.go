package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"pc/internal/http/middleware"
	"pc/internal/service"
)

const (
	stateCookie   = "hemis_oauth_state"
	callbackPath  = "/auth/callback"
	stateLifetime = 10 * time.Minute

	// The state starts with the role the flow was started for, so one callback serves both.
	teacherPrefix = "t."
	studentPrefix = "s."
)

// RegisterHemisOAuth mounts the teacher and student OAuth start routes on /api/auth.
func (h *Handler) RegisterHemisOAuth(g fiber.Router) {
	g.Get("/hemis/teacher/start", h.OAuthStart)
	g.Get("/hemis/student/start", h.StudentOAuthStart)
}

// RegisterCallback mounts GET /auth/callback. It is a Go route (not the SPA) and must be
// registered before the SPA fallback.
func (h *Handler) RegisterCallback(app fiber.Router) {
	app.Get(callbackPath, h.OAuthCallback)
}

func loginError(c *fiber.Ctx, code string) error {
	return c.Redirect("/login?error="+url.QueryEscape(code), fiber.StatusFound)
}

func (h *Handler) stateCookie(value string, expires time.Time) *fiber.Cookie {
	return &fiber.Cookie{
		Name: stateCookie, Value: value, Path: callbackPath, Expires: expires,
		HTTPOnly: true, SameSite: fiber.CookieSameSiteLaxMode, Secure: h.cfg.CookieSecure,
	}
}

// OAuthStart sets the CSRF state cookie and sends the browser to HEMIS as a teacher.
func (h *Handler) OAuthStart(c *fiber.Ctx) error {
	return h.start(c, h.auth.TeacherOAuthEnabled, teacherPrefix, func(s string) string { return h.auth.TeacherAuthorizeURL(s) })
}

// StudentOAuthStart does the same for a student.
func (h *Handler) StudentOAuthStart(c *fiber.Ctx) error {
	return h.start(c, h.auth.StudentOAuthEnabled, studentPrefix, func(s string) string { return h.auth.StudentAuthorizeURL(s) })
}

func (h *Handler) start(c *fiber.Ctx, enabled func() bool, prefix string, authorizeURL func(state string) string) error {
	if !enabled() {
		return loginError(c, service.OAuthErrUnavailable)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	state := prefix + base64.RawURLEncoding.EncodeToString(buf)
	c.Cookie(h.stateCookie(state, time.Now().Add(stateLifetime)))
	return c.Redirect(authorizeURL(state), fiber.StatusFound)
}

// OAuthCallback finishes the sign-in. Every outcome is a redirect: /teacher, /student or /login?error=<code>.
func (h *Handler) OAuthCallback(c *fiber.Ctx) error {
	m := meta(c)
	want := c.Cookies(stateCookie)
	c.Cookie(h.stateCookie("", time.Unix(0, 0))) // single use, whatever happens next

	fail := func(code string) error { return loginError(c, code) }

	got := c.Query("state")
	if want == "" || got == "" || subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		h.auth.OAuthFailed(c.Context(), service.OAuthErrState, m)
		return fail(service.OAuthErrState)
	}
	// the state is ours (httpOnly cookie, just compared), so its prefix says which flow this is
	student := strings.HasPrefix(want, studentPrefix)
	if (student && !h.auth.StudentOAuthEnabled()) || (!student && !h.auth.TeacherOAuthEnabled()) {
		h.auth.OAuthFailed(c.Context(), service.OAuthErrUnavailable, m)
		return loginError(c, service.OAuthErrUnavailable)
	}
	code := c.Query("code")
	if c.Query("error") != "" || code == "" {
		h.auth.OAuthFailed(c.Context(), service.OAuthErrDenied, m)
		return fail(service.OAuthErrDenied)
	}

	login, home := h.auth.TeacherLoginWithCode, "/teacher"
	if student {
		login, home = h.auth.StudentLoginWithCode, "/student"
	}
	res, err := login(c.Context(), code, m)
	if err != nil {
		var oe *service.OAuthError
		if errors.As(err, &oe) {
			return fail(oe.Code)
		}
		return fail(service.OAuthErrInternal)
	}
	c.Cookie(&fiber.Cookie{
		Name: middleware.CookieName, Value: res.Token, Path: "/",
		Expires: res.Expires, HTTPOnly: true, SameSite: fiber.CookieSameSiteLaxMode, Secure: h.cfg.CookieSecure,
	})
	return c.Redirect(home, fiber.StatusFound)
}
