// Package admin holds the /api/admin handlers.
package admin

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"pc/internal/export"

	"pc/internal/apperr"
	"pc/internal/http/httputil"
	"pc/internal/http/middleware"
	"pc/internal/permission"
	"pc/internal/service"
)

type Services struct {
	Dashboard   *service.DashboardService
	Faculties   *service.FacultyService
	Groups      *service.GroupService
	Users       *service.UserService
	Locations   *service.LocationService
	Options     *service.OptionService
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

// Register mounts all admin routes on g (already protected by auth and the admin/little roles). Every route
// is gated by a permission: admins hold all of them, a little admin only what it was granted. Account
// security (creating staff, passwords, passport data, permissions) is admin-only.
func (h *Handler) Register(g fiber.Router) {
	need := middleware.RequirePermission
	adminOnly := middleware.RequireRole("admin")

	g.Get("/dashboard", need("dashboard:view"), h.dashboard)

	// unpaginated id + name lists for select boxes; same permission as the list they mirror
	g.Get("/options/faculties", need("faculties:view"), h.options(h.s.Options.Faculties))
	g.Get("/options/groups", need("groups:view"), h.options(h.s.Options.Groups))
	g.Get("/options/locations", need("locations:view"), h.options(h.s.Options.Locations))
	g.Get("/options/schedules", need("schedules:view"), h.options(h.s.Options.Schedules))

	g.Get("/faculties", need("faculties:view"), h.listFaculties)
	g.Post("/faculties", need("faculties:create"), h.createFaculty)
	g.Post("/faculties/sync", need("faculties:create"), h.syncFaculties)
	g.Get("/faculties/:id", need("faculties:view"), h.getFaculty)
	g.Put("/faculties/:id", need("faculties:update"), h.updateFaculty)
	g.Delete("/faculties/:id", need("faculties:delete"), h.deleteFaculty)

	g.Get("/groups", need("groups:view"), h.listGroups)
	g.Post("/groups", need("groups:create"), h.createGroup)
	g.Post("/groups/sync", need("groups:create"), h.syncGroups)
	g.Get("/groups/:id", need("groups:view"), h.getGroup)
	g.Put("/groups/:id", need("groups:update"), h.updateGroup)
	g.Delete("/groups/:id", need("groups:delete"), h.deleteGroup)

	g.Get("/users", need("users:view"), h.listUsers)
	g.Post("/users", adminOnly, h.createUser)
	g.Get("/users/:id", need("users:view"), h.getUser)
	g.Put("/users/:id", need("users:update"), h.manageable, h.updateUser)
	g.Post("/users/:id/activate", need("users:update"), h.manageable, h.activateUser)
	g.Post("/users/:id/deactivate", need("users:update"), h.manageable, h.deactivateUser)
	g.Put("/users/:id/photo", need("users:update"), h.manageable, h.enrollPhoto)
	g.Post("/users/:id/regenerate-password", adminOnly, h.regeneratePassword)
	g.Get("/users/:id/sensitive", adminOnly, h.userSensitive)
	g.Put("/users/:id/permissions", adminOnly, h.setPermissions)
	g.Get("/permissions", adminOnly, h.permissionCatalog)

	g.Get("/locations", need("locations:view"), h.listLocations)
	g.Post("/locations", need("locations:create"), h.createLocation)
	g.Get("/locations/:id", need("locations:view"), h.getLocation)
	g.Put("/locations/:id", need("locations:update"), h.updateLocation)
	g.Delete("/locations/:id", need("locations:delete"), h.deleteLocation)

	g.Get("/schedules", need("schedules:view"), h.listSchedules)
	g.Post("/schedules", need("schedules:create"), h.createSchedule)
	g.Get("/schedules/:id", need("schedules:view"), h.getSchedule)
	g.Put("/schedules/:id", need("schedules:update"), h.updateSchedule)
	g.Delete("/schedules/:id", need("schedules:delete"), h.deleteSchedule)

	g.Get("/reports/attendance", need("reports:view"), h.report)
	g.Get("/reports/attendance.xlsx", need("reports:view"), h.reportXLSX)
	g.Get("/students/:id/attempts", need("attempts:view"), h.studentAttempts)
	g.Get("/attempts", need("attempts:view"), h.listAttempts)
	g.Put("/attempts/:id/review", adminOnly, h.reviewAttempt)
	g.Delete("/attempts/:id", adminOnly, h.deleteAttempt)

	g.Get("/audit", need("audit:view"), h.listAudit)
	g.Get("/audit/:id", need("audit:view"), h.getAudit)
}

func (h *Handler) options(load func(context.Context) ([]service.OptionDTO, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, err := load(c.Context())
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"items": items})
	}
}

// manageable keeps a little admin away from staff accounts: they may change students only.
func (h *Handler) manageable(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.s.Users.EnsureManageable(c.Context(), httputil.Actor(c), id); err != nil {
		return err
	}
	return c.Next()
}

type permissionInfo struct {
	Key      string   `json:"key"`
	Requires []string `json:"requires"`
}

// permissionCatalog lists every grantable permission with what it depends on (for the admin form).
func (h *Handler) permissionCatalog(c *fiber.Ctx) error {
	out := make([]permissionInfo, 0, len(permission.Catalog))
	for _, p := range permission.Catalog {
		out = append(out, permissionInfo{Key: p, Requires: permission.Requires(p)})
	}
	return c.JSON(fiber.Map{"permissions": out})
}

func (h *Handler) setPermissions(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var b struct {
		Permissions []string `json:"permissions"`
	}
	if err := httputil.Body(c, &b); err != nil {
		return err
	}
	u, err := h.s.Users.SetPermissions(c.Context(), httputil.Actor(c), id, b.Permissions)
	if err != nil {
		return err
	}
	return c.JSON(u)
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
	// Permissions (little admins only) are the sections the account may use.
	Permissions []string `json:"permissions"`
}

// createUser creates an admin, little admin or teacher; the generated password is returned once.
func (h *Handler) createUser(c *fiber.Ctx) error {
	var b createUserBody
	if err := httputil.Body(c, &b); err != nil {
		return err
	}
	if b.Role != "admin" && b.Role != "little" && b.Role != "teacher" {
		return apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"role": "Faqat administrator, kichik administrator yoki o'qituvchi yaratish mumkin"})
	}
	if b.FullName == "" {
		return apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"full_name": "Majburiy maydon"})
	}
	u, password, err := h.s.Users.CreateUser(c.Context(), httputil.Actor(c), service.CreateUserInput{
		Username: b.Username, Role: b.Role, FullName: b.FullName, MustChangePassword: true, HemisLogin: b.HemisLogin, Permissions: b.Permissions,
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

// reportGroup reads the mandatory group_id of an admin report.
func reportGroup(c *fiber.Ctx) (uuid.UUID, error) {
	gid, err := httputil.OptUUID(c, "group_id")
	if err != nil {
		return uuid.Nil, err
	}
	if gid == nil {
		return uuid.Nil, apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"group_id": "Guruhni tanlang"})
	}
	return *gid, nil
}

func (h *Handler) report(c *fiber.Ctx) error {
	gid, err := reportGroup(c)
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
	gid, err := reportGroup(c)
	if err != nil {
		return err
	}
	rep, err := h.s.Reports.AdminReport(c.Context(), gid, c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	data, err := export.Attendance(rep, false)
	if err != nil {
		return err
	}
	if err := h.s.Reports.AuditExport(c.Context(), httputil.Actor(c), &gid, rep); err != nil {
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
	out, err := h.s.Reports.StudentAttempts(c.Context(), nil, id, c.Query("from"), c.Query("to"))
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

func (h *Handler) deleteAttempt(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.s.Attempts.Delete(c.Context(), httputil.Actor(c), id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// reviewAttempt confirms or rejects an attempt accepted without a face check: {"approved": bool}.
func (h *Handler) reviewAttempt(c *fiber.Ctx) error {
	id, err := httputil.IDParam(c, "id")
	if err != nil {
		return err
	}
	var body struct {
		Approved bool `json:"approved"`
	}
	if err := c.BodyParser(&body); err != nil {
		return apperr.Validation("Ma'lumotlar noto'g'ri", nil)
	}
	if err := h.s.Attempts.Review(c.Context(), httputil.Actor(c), nil, id, body.Approved); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
