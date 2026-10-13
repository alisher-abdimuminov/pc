package middleware

import (
	"errors"
	"slices"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"pc/internal/apperr"
	"pc/internal/auth"
	"pc/internal/db"
	"pc/internal/permission"
)

const (
	CookieName = "session"

	localUser    = "user"
	localSession = "session_id"
)

// mustChangeAllowed are the only endpoints usable while must_change_password is set.
var mustChangeAllowed = []string{"/api/auth/change-password", "/api/auth/logout", "/api/auth/me"}

var errPasswordChange = apperr.New(fiber.StatusForbidden, "password_change_required", "Davom etish uchun parolni almashtiring")

// RequireAuth resolves the session cookie to an active user and stores both in the request context.
func RequireAuth(q *db.Queries) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := c.Cookies(CookieName)
		if raw == "" {
			return apperr.Unauthorized()
		}
		id := auth.HashToken(raw)
		sess, err := q.GetSession(c.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.Unauthorized()
			}
			return err
		}
		user, err := q.GetUser(c.Context(), sess.UserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.Unauthorized()
			}
			return err
		}
		if !user.IsActive {
			return apperr.Unauthorized()
		}
		if user.MustChangePassword && !slices.Contains(mustChangeAllowed, c.Path()) {
			return errPasswordChange
		}
		c.Locals(localUser, user)
		c.Locals(localSession, sess.ID)
		return c.Next()
	}
}

// RequireRole allows only the listed roles. Must run after RequireAuth.
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		u, ok := CurrentUser(c)
		if !ok {
			return apperr.Unauthorized()
		}
		if !slices.Contains(roles, u.Role) {
			return apperr.Forbidden()
		}
		return c.Next()
	}
}

// RequirePermission allows admins and little admins who were granted perm. Must run after RequireAuth.
func RequirePermission(perm string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		u, ok := CurrentUser(c)
		if !ok {
			return apperr.Unauthorized()
		}
		if !permission.Has(u, perm) {
			return apperr.Forbidden()
		}
		return c.Next()
	}
}

func CurrentUser(c *fiber.Ctx) (db.User, bool) {
	u, ok := c.Locals(localUser).(db.User)
	return u, ok
}

func SessionID(c *fiber.Ctx) string {
	s, _ := c.Locals(localSession).(string)
	return s
}
