// Package httputil holds small request helpers shared by the handlers.
package httputil

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"pc/internal/apperr"
	"pc/internal/db"
	"pc/internal/http/middleware"
	"pc/internal/service"
)

// Body parses a JSON request body.
func Body(c *fiber.Ctx, v any) error {
	if err := c.BodyParser(v); err != nil {
		return apperr.New(fiber.StatusBadRequest, "bad_request", "Noto'g'ri so'rov")
	}
	return nil
}

// IDParam parses a UUID path parameter; a malformed id is a 404.
func IDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, apperr.NotFound()
	}
	return id, nil
}

// OptUUID parses an optional UUID query parameter.
func OptUUID(c *fiber.Ctx, name string) (*uuid.UUID, error) {
	v := c.Query(name)
	if v == "" {
		return nil, nil
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return nil, apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{name: "Noto'g'ri identifikator"})
	}
	return &id, nil
}

func List(c *fiber.Ctx) service.ListParams {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	return service.ListParams{Page: page, PageSize: size, Search: strings.TrimSpace(c.Query("search"))}
}

// Actor builds the audit actor from the authenticated request.
func Actor(c *fiber.Ctx) service.Actor {
	a := service.Actor{IP: c.IP(), UserAgent: c.Get(fiber.HeaderUserAgent)}
	if u, ok := middleware.CurrentUser(c); ok {
		a.ID, a.Role = &u.ID, u.Role
	}
	return a
}

func CurrentUser(c *fiber.Ctx) db.User {
	u, _ := middleware.CurrentUser(c)
	return u
}
