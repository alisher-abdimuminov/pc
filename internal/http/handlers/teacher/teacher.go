// Package teacher holds the /api/teacher handlers.
package teacher

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"pc/internal/export"
	"pc/internal/http/httputil"
	"pc/internal/service"
)

type Handler struct {
	reports     *service.ReportService
	assignments *service.AssignmentService
}

func New(r *service.ReportService, a *service.AssignmentService) *Handler {
	return &Handler{reports: r, assignments: a}
}

func (h *Handler) Register(g fiber.Router) {
	g.Get("/groups", h.listGroups)
	g.Get("/groups/:id", h.getGroup)
	g.Get("/groups/:id/report", h.report)
	g.Get("/groups/:id/report.xlsx", h.reportXLSX)

	g.Get("/assignments", h.listAssignments)
	g.Post("/assignments", h.createAssignment)
	g.Get("/assignments/:id", h.getAssignment)
	g.Put("/assignments/:id", h.updateAssignment)
	g.Delete("/assignments/:id", h.deleteAssignment)
	g.Get("/assignments/:id/submissions", h.submissions)
	g.Get("/assignments/:id/grades.xlsx", h.gradesXLSX)
	g.Put("/submissions/:id/grade", h.grade)
}

func me(c *fiber.Ctx) uuid.UUID { return httputil.CurrentUser(c).ID }

func (h *Handler) listGroups(c *fiber.Ctx) error {
	gs, err := h.reports.TeacherGroups(c.Context(), me(c))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": gs})
}

func (h *Handler) getGroup(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	g, err := h.reports.TeacherGroup(c.Context(), me(c), id)
	if err != nil {
		return err
	}
	return c.JSON(g)
}

func (h *Handler) report(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	tid := me(c)
	rep, err := h.reports.GroupReport(c.Context(), &tid, id, c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	return c.JSON(rep)
}

func (h *Handler) reportXLSX(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	tid := me(c)
	rep, err := h.reports.GroupReport(c.Context(), &tid, id, c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	data, err := export.Attendance(rep, false)
	if err != nil {
		return err
	}
	if err := h.reports.AuditExport(c.Context(), httputil.Actor(c), &id, rep); err != nil {
		return err
	}
	return sendXLSX(c, data, "davomat_"+rep.From+"_"+rep.To+".xlsx")
}

func sendXLSX(c *fiber.Ctx, data []byte, name string) error {
	c.Set(fiber.HeaderContentType, export.XLSXContentType)
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+name+`"`)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Send(data)
}
