// Package admin holds the /api/admin handlers.
package admin

import (
	"github.com/gofiber/fiber/v2"

	"pc/internal/export"

	"pc/internal/apperr"
	"pc/internal/http/httputil"
	"pc/internal/service"
)

type Services struct {
	Dashboard   *service.DashboardService
	Faculties   *service.FacultyService
	Groups      *service.GroupService
	Users       *service.UserService
	Locations   *service.LocationService
	Schedules   *service.ScheduleService
	Audit       *service.AuditService
	Reports     *service.ReportService
	Enroll      *service.EnrollmentService
	Attempts    *service.AttemptsAdminService
	GroupSync   *service.GroupSyncService
	FacultySync *service.FacultySyncService
}

type Handler struct{ s Services }

func New(s Services) *Handler { return &Handler{s: s} }

// Register mounts all admin routes on g (already protected by auth + admin role).
func (h *Handler) Register(g fiber.Router) {
	g.Get("/dashboard", h.dashboard)

	g.Get("/faculties", h.listFaculties)
	g.Post("/faculties", h.createFaculty)
	g.Post("/faculties/sync", h.syncFaculties)
	g.Get("/faculties/:id", h.getFaculty)
	g.Put("/faculties/:id", h.updateFaculty)
	g.Delete("/faculties/:id", h.deleteFaculty)

	g.Get("/groups", h.listGroups)
	g.Post("/groups", h.createGroup)
	g.Post("/groups/sync", h.syncGroups)
	g.Get("/groups/:id", h.getGroup)
	g.Put("/groups/:id", h.updateGroup)
	g.Delete("/groups/:id", h.deleteGroup)

	g.Get("/users", h.listUsers)
	g.Post("/users", h.createUser)
	g.Get("/users/:id", h.getUser)
	g.Put("/users/:id", h.updateUser)
	g.Post("/users/:id/activate", h.activateUser)
	g.Post("/users/:id/deactivate", h.deactivateUser)
	g.Post("/users/:id/regenerate-password", h.regeneratePassword)
	g.Get("/users/:id/sensitive", h.userSensitive)
	g.Put("/users/:id/photo", h.enrollPhoto)

	g.Get("/locations", h.listLocations)
	g.Post("/locations", h.createLocation)
	g.Get("/locations/:id", h.getLocation)
	g.Put("/locations/:id", h.updateLocation)
	g.Delete("/locations/:id", h.deleteLocation)

	g.Get("/schedules", h.listSchedules)
	g.Post("/schedules", h.createSchedule)
	g.Get("/schedules/:id", h.getSchedule)
	g.Put("/schedules/:id", h.updateSchedule)
	g.Delete("/schedules/:id", h.deleteSchedule)

	g.Get("/reports/attendance", h.report)
	g.Get("/reports/attendance.xlsx", h.reportXLSX)
	g.Get("/students/:id/attempts", h.studentAttempts)
	g.Get("/attempts", h.listAttempts)

	g.Get("/audit", h.listAudit)
	g.Get("/audit/:id", h.getAudit)
}

func (h *Handler) dashboard(c *fiber.Ctx) error {
	d, err := h.s.Dashboard.Get(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(d)
}

// ---- faculties ----

type nameBody struct {
	Name string `json:"name"`
}

func (h *Handler) listFaculties(c *fiber.Ctx) error {
	p, err := h.s.Faculties.List(c.Context(), httputil.List(c))
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) getFaculty(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	f, err := h.s.Faculties.Get(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(f)
}

func (h *Handler) createFaculty(c *fiber.Ctx) error {
	var b nameBody
	if err := httputil.Body(c, &b); err != nil {
		return err
	}
	f, err := h.s.Faculties.Create(c.Context(), httputil.Actor(c), b.Name)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(f)
}

func (h *Handler) updateFaculty(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var b nameBody
	if err := httputil.Body(c, &b); err != nil {
		return err
	}
	f, err := h.s.Faculties.Update(c.Context(), httputil.Actor(c), id, b.Name)
	if err != nil {
		return err
	}
	return c.JSON(f)
}

func (h *Handler) deleteFaculty(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.s.Faculties.Delete(c.Context(), httputil.Actor(c), id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ---- groups ----

func (h *Handler) listGroups(c *fiber.Ctx) error {
	fid, err := httputil.OptUUID(c, "faculty_id")
	if err != nil {
		return err
	}
	p, err := h.s.Groups.List(c.Context(), httputil.List(c), fid)
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) getGroup(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	g, err := h.s.Groups.Get(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(g)
}

func (h *Handler) createGroup(c *fiber.Ctx) error {
	var in service.GroupInput
	if err := httputil.Body(c, &in); err != nil {
		return err
	}
	g, err := h.s.Groups.Create(c.Context(), httputil.Actor(c), in)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(g)
}

func (h *Handler) updateGroup(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var in service.GroupInput
	if err := httputil.Body(c, &in); err != nil {
		return err
	}
	g, err := h.s.Groups.Update(c.Context(), httputil.Actor(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(g)
}

func (h *Handler) deleteGroup(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.s.Groups.Delete(c.Context(), httputil.Actor(c), id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ---- users ----

func (h *Handler) listUsers(c *fiber.Ctx) error {
	f := service.UserFilter{Role: c.Query("role")}
	var err error
	if f.FacultyID, err = httputil.OptUUID(c, "faculty_id"); err != nil {
		return err
	}
	if f.GroupID, err = httputil.OptUUID(c, "group_id"); err != nil {
		return err
	}
	switch c.Query("is_active") {
	case "true":
		t := true
		f.IsActive = &t
	case "false":
		t := false
		f.IsActive = &t
	}
	p, err := h.s.Users.List(c.Context(), httputil.List(c), f)
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) getUser(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	u, err := h.s.Users.Get(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(u)
}

type createUserBody struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	FullName string `json:"full_name"`
	// HemisLogin (teachers only) links the account to the teacher's HEMIS account.
	HemisLogin string `json:"hemis_login"`
}

// createUser creates an admin or teacher; the generated password is returned once.
func (h *Handler) createUser(c *fiber.Ctx) error {
	var b createUserBody
	if err := httputil.Body(c, &b); err != nil {
		return err
	}
	if b.Role != "admin" && b.Role != "teacher" {
		return apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"role": "Faqat administrator yoki o'qituvchi yaratish mumkin"})
	}
	if b.FullName == "" {
		return apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"full_name": "Majburiy maydon"})
	}
	u, password, err := h.s.Users.CreateUser(c.Context(), httputil.Actor(c), service.CreateUserInput{
		Username: b.Username, Role: b.Role, FullName: b.FullName, MustChangePassword: true, HemisLogin: b.HemisLogin,
	})
	if err != nil {
		return err
	}
	dto, err := h.s.Users.Get(c.Context(), u.ID)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": dto, "password": password})
}

func (h *Handler) updateUser(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var in service.UserUpdateInput
	if err := httputil.Body(c, &in); err != nil {
		return err
	}
	u, err := h.s.Users.Update(c.Context(), httputil.Actor(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(u)
}

func (h *Handler) setActive(c *fiber.Ctx, active bool) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	u, err := h.s.Users.SetActive(c.Context(), httputil.Actor(c), id, active)
	if err != nil {
		return err
	}
	return c.JSON(u)
}

func (h *Handler) activateUser(c *fiber.Ctx) error   { return h.setActive(c, true) }
func (h *Handler) deactivateUser(c *fiber.Ctx) error { return h.setActive(c, false) }

func (h *Handler) regeneratePassword(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	pw, err := h.s.Users.RegeneratePassword(c.Context(), httputil.Actor(c), id)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"password": pw})
}

func (h *Handler) userSensitive(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	s, err := h.s.Users.Sensitive(c.Context(), httputil.Actor(c), id)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(s)
}

// ---- locations ----

func (h *Handler) listLocations(c *fiber.Ctx) error {
	p, err := h.s.Locations.List(c.Context(), httputil.List(c))
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) getLocation(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	l, err := h.s.Locations.Get(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(l)
}

func (h *Handler) createLocation(c *fiber.Ctx) error {
	var in service.LocationInput
	if err := httputil.Body(c, &in); err != nil {
		return err
	}
	l, err := h.s.Locations.Create(c.Context(), httputil.Actor(c), in)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(l)
}

func (h *Handler) updateLocation(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var in service.LocationInput
	if err := httputil.Body(c, &in); err != nil {
		return err
	}
	l, err := h.s.Locations.Update(c.Context(), httputil.Actor(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(l)
}

func (h *Handler) deleteLocation(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.s.Locations.Delete(c.Context(), httputil.Actor(c), id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ---- schedules ----

func (h *Handler) listSchedules(c *fiber.Ctx) error {
	var f service.ScheduleFilter
	var err error
	if f.UserID, err = httputil.OptUUID(c, "user_id"); err != nil {
		return err
	}
	if f.GroupID, err = httputil.OptUUID(c, "group_id"); err != nil {
		return err
	}
	p, err := h.s.Schedules.List(c.Context(), httputil.List(c), f)
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) getSchedule(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	s, err := h.s.Schedules.Get(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(s)
}

func (h *Handler) createSchedule(c *fiber.Ctx) error {
	var in service.ScheduleInput
	if err := httputil.Body(c, &in); err != nil {
		return err
	}
	s, err := h.s.Schedules.Create(c.Context(), httputil.Actor(c), in)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(s)
}

func (h *Handler) updateSchedule(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var in service.ScheduleInput
	if err := httputil.Body(c, &in); err != nil {
		return err
	}
	s, err := h.s.Schedules.Update(c.Context(), httputil.Actor(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(s)
}

func (h *Handler) deleteSchedule(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.s.Schedules.Delete(c.Context(), httputil.Actor(c), id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ---- audit ----

func (h *Handler) listAudit(c *fiber.Ctx) error {
	actor, err := httputil.OptUUID(c, "actor_id")
	if err != nil {
		return err
	}
	p, err := h.s.Audit.List(c.Context(), httputil.List(c), service.AuditFilter{
		ActorID: actor, Action: c.Query("action"), EntityType: c.Query("entity_type"),
		From: c.Query("from"), To: c.Query("to"),
	})
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) getAudit(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	a, err := h.s.Audit.Get(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(a)
}

// ---- attendance reports ----

func (h *Handler) report(c *fiber.Ctx) error {
	gid, err := httputil.OptUUID(c, "group_id")
	if err != nil {
		return err
	}
	rep, err := h.s.Reports.AdminReport(c.Context(), gid, c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	return c.JSON(rep)
}

func (h *Handler) reportXLSX(c *fiber.Ctx) error {
	gid, err := httputil.OptUUID(c, "group_id")
	if err != nil {
		return err
	}
	rep, err := h.s.Reports.AdminReport(c.Context(), gid, c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	data, err := export.Attendance(rep, gid == nil)
	if err != nil {
		return err
	}
	if err := h.s.Reports.AuditExport(c.Context(), httputil.Actor(c), gid, rep); err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, export.XLSXContentType)
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="davomat_`+rep.From+`_`+rep.To+`.xlsx"`)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Send(data)
}

func (h *Handler) studentAttempts(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	out, err := h.s.Reports.StudentAttempts(c.Context(), id, c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	return c.JSON(out)
}

// enrollPhoto stores the student's reference photo and computes its face embedding.
func (h *Handler) enrollPhoto(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var up *service.Upload
	if fh, err := c.FormFile("image"); err == nil && fh.Size > 0 {
		f, err := fh.Open()
		if err != nil {
			return err
		}
		defer f.Close()
		up = &service.Upload{Filename: fh.Filename, Size: fh.Size, Reader: f}
	}
	u, err := h.s.Enroll.Enroll(c.Context(), httputil.Actor(c), id, up)
	if err != nil {
		return err
	}
	return c.JSON(u)
}

// listAttempts lists every attendance attempt (filters: search, group_id, student_id, success, kind, from, to).
func (h *Handler) listAttempts(c *fiber.Ctx) error {
	var f service.AttemptFilter
	var err error
	if f.GroupID, err = httputil.OptUUID(c, "group_id"); err != nil {
		return err
	}
	if f.StudentID, err = httputil.OptUUID(c, "student_id"); err != nil {
		return err
	}
	switch c.Query("success") {
	case "true":
		t := true
		f.Success = &t
	case "false":
		t := false
		f.Success = &t
	}
	f.Kind, f.From, f.To = c.Query("kind"), c.Query("from"), c.Query("to")
	p, err := h.s.Attempts.List(c.Context(), httputil.List(c), f)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(p)
}

// syncGroups pulls the group list from HEMIS and returns {total, created, updated, unchanged}.
func (h *Handler) syncGroups(c *fiber.Ctx) error {
	res, err := h.s.GroupSync.Sync(c.Context(), httputil.Actor(c))
	if err != nil {
		return err
	}
	return c.JSON(res)
}

// syncFaculties pulls the faculty list from HEMIS and returns {total, created, updated, unchanged}.
func (h *Handler) syncFaculties(c *fiber.Ctx) error {
	res, err := h.s.FacultySync.Sync(c.Context(), httputil.Actor(c))
	if err != nil {
		return err
	}
	return c.JSON(res)
}
