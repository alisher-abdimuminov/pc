package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"

	"pc/internal/http/middleware"
	"pc/internal/service"
)

const (
	stateCookie   = "hemis_oauth_state"
	callbackPath  = "/auth/callback"
	stateLifetime = 10 * time.Minute
)

// RegisterHemisOAuth mounts the teacher OAuth start route on /api/auth.
func (h *Handler) RegisterHemisOAuth(g fiber.Router) {
	g.Get("/hemis/teacher/start", h.OAuthStart)
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

// OAuthStart sets the CSRF state cookie and sends the browser to HEMIS.
func (h *Handler) OAuthStart(c *fiber.Ctx) error {
	if !h.auth.TeacherOAuthEnabled() {
		return loginError(c, service.OAuthErrUnavailable)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	state := base64.RawURLEncoding.EncodeToString(buf)
	c.Cookie(h.stateCookie(state, time.Now().Add(stateLifetime)))
	return c.Redirect(h.auth.TeacherAuthorizeURL(state), fiber.StatusFound)
}

// OAuthCallback finishes the sign-in. Every outcome is a redirect: /teacher or /login?error=<code>.
func (h *Handler) OAuthCallback(c *fiber.Ctx) error {
	m := meta(c)
	want := c.Cookies(stateCookie)
	c.Cookie(h.stateCookie("", time.Unix(0, 0))) // single use, whatever happens next

	fail := func(code string) error {
		h.auth.OAuthRecordFailure(m.IP)
		return loginError(c, code)
	}
	if !h.auth.TeacherOAuthEnabled() {
		return loginError(c, service.OAuthErrUnavailable)
	}
	if !h.auth.OAuthAllowed(m.IP) {
		h.auth.OAuthFailed(c.Context(), service.OAuthErrRateLimited, m)
		return loginError(c, service.OAuthErrRateLimited)
	}

	got := c.Query("state")
	if want == "" || got == "" || subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		h.auth.OAuthFailed(c.Context(), service.OAuthErrState, m)
		return fail(service.OAuthErrState)
	}
	code := c.Query("code")
	if c.Query("error") != "" || code == "" {
		h.auth.OAuthFailed(c.Context(), service.OAuthErrDenied, m)
		return fail(service.OAuthErrDenied)
	}

	res, err := h.auth.TeacherLoginWithCode(c.Context(), code, m)
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
	return c.Redirect("/teacher", fiber.StatusFound)
}
