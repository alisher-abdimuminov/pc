package teacher

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"pc/internal/apperr"
	"pc/internal/export"
	"pc/internal/http/httputil"
	"pc/internal/service"
)

// parseAssignmentForm reads the multipart form: group_id, title, description, deadline, file, remove_file.
func parseAssignmentForm(c *fiber.Ctx) (service.AssignmentInput, func(), error) {
	in := service.AssignmentInput{
		Title: c.FormValue("title"), Description: c.FormValue("description"), Deadline: c.FormValue("deadline"),
		RemoveFile: c.FormValue("remove_file") == "true",
	}
	if v := c.FormValue("group_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return in, nil, apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"group_id": "Noto'g'ri identifikator"})
		}
		in.GroupID = &id
	}
	closer := func() {}
	if fh, err := c.FormFile("file"); err == nil && fh.Size > 0 {
		f, err := fh.Open()
		if err != nil {
			return in, nil, err
		}
		in.File = &service.Upload{Filename: fh.Filename, Size: fh.Size, Reader: f}
		closer = func() { f.Close() }
	}
	return in, closer, nil
}

func (h *Handler) listAssignments(c *fiber.Ctx) error {
	gid, err := httputil.OptUUID(c, "group_id")
	if err != nil {
		return err
	}
	p, err := h.assignments.List(c.Context(), me(c), httputil.List(c), gid)
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) getAssignment(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	a, err := h.assignments.Get(c.Context(), me(c), id)
	if err != nil {
		return err
	}
	return c.JSON(a)
}

func (h *Handler) createAssignment(c *fiber.Ctx) error {
	in, closeFn, err := parseAssignmentForm(c)
	if err != nil {
		return err
	}
	defer closeFn()
	a, err := h.assignments.Create(c.Context(), httputil.Actor(c), me(c), in)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(a)
}

func (h *Handler) updateAssignment(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	in, closeFn, err := parseAssignmentForm(c)
	if err != nil {
		return err
	}
	defer closeFn()
	a, err := h.assignments.Update(c.Context(), httputil.Actor(c), me(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(a)
}

func (h *Handler) deleteAssignment(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.assignments.Delete(c.Context(), httputil.Actor(c), me(c), id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) submissions(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	v, err := h.assignments.Submissions(c.Context(), me(c), id)
	if err != nil {
		return err
	}
	return c.JSON(v)
}

func (h *Handler) gradesXLSX(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	g, err := h.assignments.ExportGrades(c.Context(), httputil.Actor(c), me(c), id)
	if err != nil {
		return err
	}
	data, err := export.Grades(g)
	if err != nil {
		return err
	}
	return sendXLSX(c, data, "baholar.xlsx")
}

func (h *Handler) grade(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var in service.GradeInput
	if err := httputil.Body(c, &in); err != nil {
		return err
	}
	e, err := h.assignments.Grade(c.Context(), httputil.Actor(c), me(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(e)
}
