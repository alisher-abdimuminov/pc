// Package student holds the /api/student handlers.
package student

import (
	"io"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"pc/internal/apperr"
	"pc/internal/http/middleware"
	"pc/internal/storage"

	"pc/internal/http/httputil"
	"pc/internal/service"
)

type Handler struct {
	svc      *service.StudentService
	attempts *service.AttemptService
}

func New(s *service.StudentService, a *service.AttemptService) *Handler {
	return &Handler{svc: s, attempts: a}
}

func (h *Handler) Register(g fiber.Router) {
	g.Get("/today", h.today)
	g.Get("/attendance", h.month)
	g.Post("/attendance/attempt", h.attempt)
	g.Get("/profile", h.profile)
	g.Get("/assignments", h.assignments)
	g.Get("/assignments/:id", h.assignment)
	g.Put("/assignments/:id/submission", h.submit)
}

func (h *Handler) today(c *fiber.Ctx) error {
	v, err := h.svc.Today(c.Context(), httputil.CurrentUser(c))
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(v)
}

func (h *Handler) month(c *fiber.Ctx) error {
	v, err := h.svc.Month(c.Context(), httputil.CurrentUser(c), c.Query("month"))
	if err != nil {
		return err
	}
	return c.JSON(v)
}

func (h *Handler) profile(c *fiber.Ctx) error {
	p, err := h.svc.Profile(c.Context(), httputil.CurrentUser(c))
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) assignments(c *fiber.Ctx) error {
	v, err := h.svc.Assignments(c.Context(), httputil.CurrentUser(c))
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(v)
}

func (h *Handler) assignment(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	a, err := h.svc.Assignment(c.Context(), httputil.CurrentUser(c), id)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(a)
}

func (h *Handler) submit(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var up *service.Upload
	if fh, err := c.FormFile("file"); err == nil && fh.Size > 0 {
		f, err := fh.Open()
		if err != nil {
			return err
		}
		defer f.Close()
		up = &service.Upload{Filename: fh.Filename, Size: fh.Size, Reader: f}
	}
	a, err := h.svc.Submit(c.Context(), httputil.Actor(c), httputil.CurrentUser(c), id, up)
	if err != nil {
		return err
	}
	return c.JSON(a)
}

// attempt: multipart with image, latitude, longitude, accuracy.
func (h *Handler) attempt(c *fiber.Ctx) error {
	fields := map[string]string{}
	num := func(name string) float64 {
		v, err := strconv.ParseFloat(c.FormValue(name), 64)
		if err != nil {
			fields[name] = "Son kiriting"
		}
		return v
	}
	in := service.AttemptInput{Latitude: num("latitude"), Longitude: num("longitude"), Accuracy: num("accuracy")}
	fh, err := c.FormFile("image")
	if err != nil || fh.Size == 0 {
		fields["image"] = "Rasmni yuboring"
	} else if fh.Size > storage.MaxSelfieSize {
		fields["image"] = "Rasm hajmi 5 MB dan oshmasligi kerak"
	}
	if len(fields) > 0 {
		return apperr.Validation("Ma'lumotlar noto'g'ri", fields)
	}
	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	if in.Image, err = io.ReadAll(io.LimitReader(f, storage.MaxSelfieSize+1)); err != nil {
		return err
	}
	in.Filename = fh.Filename
	res, err := h.attempts.Attempt(c.Context(), httputil.Actor(c), httputil.CurrentUser(c), in)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	if res.Blocked { // every session of this student was just ended: drop the cookie too
		c.Cookie(&fiber.Cookie{Name: middleware.CookieName, Value: "", Path: "/", Expires: time.Unix(0, 0), HTTPOnly: true, SameSite: fiber.CookieSameSiteLaxMode})
	}
	return c.JSON(res)
}
