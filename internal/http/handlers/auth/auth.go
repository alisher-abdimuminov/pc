package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"pc/internal/apperr"
	"pc/internal/config"
	"pc/internal/db"
	"pc/internal/http/middleware"
	"pc/internal/service"
)

type Handler struct {
	cfg  *config.Config
	auth *service.AuthService
}

func New(cfg *config.Config, a *service.AuthService) *Handler {
	return &Handler{cfg: cfg, auth: a}
}

// Register mounts the routes on the /api/auth group.
func (h *Handler) Register(g fiber.Router, requireAuth fiber.Handler) {
	g.Post("/login", h.Login)
	g.Post("/logout", requireAuth, h.Logout)
	g.Get("/me", requireAuth, h.Me)
	g.Post("/change-password", requireAuth, h.ChangePassword)
}

type UserResponse struct {
	ID                 uuid.UUID  `json:"id"`
	Username           string     `json:"username"`
	Role               string     `json:"role"`
	FullName           string     `json:"full_name"`
	MustChangePassword bool       `json:"must_change_password"`
	GroupID            *uuid.UUID `json:"group_id"`
	FacultyID          *uuid.UUID `json:"faculty_id"`
	Permissions        []string   `json:"permissions"`
}

func toResponse(u db.User) UserResponse {
	r := UserResponse{ID: u.ID, Username: u.Username.String, Role: u.Role, FullName: u.FullName, MustChangePassword: u.MustChangePassword, Permissions: []string{}}
	if u.Role == "little" && u.Permissions != nil {
		r.Permissions = u.Permissions
	}
	if u.GroupID.Valid {
		r.GroupID = &u.GroupID.UUID
	}
	if u.FacultyID.Valid {
		r.FacultyID = &u.FacultyID.UUID
	}
	return r
}

func meta(c *fiber.Ctx) service.Actor {
	return service.Actor{IP: c.IP(), UserAgent: c.Get(fiber.HeaderUserAgent)}
}

func actorOf(c *fiber.Ctx, u db.User) service.Actor {
	m := meta(c)
	m.ID, m.Role = &u.ID, u.Role
	return m
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.New(fiber.StatusBadRequest, "bad_request", "Noto'g'ri so'rov")
	}
	res, err := h.auth.Login(c.Context(), req.Username, req.Password, meta(c))
	if err != nil {
		return err
	}
	c.Cookie(&fiber.Cookie{
		Name: middleware.CookieName, Value: res.Token, Path: "/",
		Expires: res.Expires, HTTPOnly: true, SameSite: fiber.CookieSameSiteLaxMode, Secure: h.cfg.CookieSecure,
	})
	return c.JSON(fiber.Map{"user": toResponse(res.User)})
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	u, _ := middleware.CurrentUser(c)
	if err := h.auth.Logout(c.Context(), u, middleware.SessionID(c), actorOf(c, u)); err != nil {
		return err
	}
	c.Cookie(&fiber.Cookie{
		Name: middleware.CookieName, Value: "", Path: "/", Expires: time.Unix(0, 0),
		HTTPOnly: true, SameSite: fiber.CookieSameSiteLaxMode, Secure: h.cfg.CookieSecure,
	})
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Me(c *fiber.Ctx) error {
	u, _ := middleware.CurrentUser(c)
	return c.JSON(fiber.Map{"user": toResponse(u)})
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ChangePassword(c *fiber.Ctx) error {
	var req changePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.New(fiber.StatusBadRequest, "bad_request", "Noto'g'ri so'rov")
	}
	u, _ := middleware.CurrentUser(c)
	if err := h.auth.ChangePassword(c.Context(), u, middleware.SessionID(c), req.OldPassword, req.NewPassword, actorOf(c, u)); err != nil {
		return err
	}
	u.MustChangePassword = false
	return c.JSON(fiber.Map{"user": toResponse(u)})
}
