// Package files serves stored files through permission checks.
package files

import (
	"io"
	"mime"

	"github.com/gofiber/fiber/v2"

	"pc/internal/http/httputil"
	"pc/internal/service"
)

type Handler struct{ files *service.FileService }

func New(f *service.FileService) *Handler { return &Handler{files: f} }

// Register mounts GET /:kind/:id on g (already behind RequireAuth).
func (h *Handler) Register(g fiber.Router) {
	g.Get("/:kind/:id", h.get)
}

func (h *Handler) get(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	res, err := h.files.Open(c.Context(), httputil.CurrentUser(c), c.Params("kind"), id)
	if err != nil {
		return err
	}
	defer res.Reader.Close()
	data, err := io.ReadAll(res.Reader)
	if err != nil {
		return err
	}
	disp := "attachment"
	if res.Inline {
		disp = "inline"
	}
	c.Set(fiber.HeaderContentType, res.ContentType)
	c.Set(fiber.HeaderContentDisposition, mime.FormatMediaType(disp, map[string]string{"filename": res.DownloadAs}))
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.Send(data)
}
