package http

import (
	"context"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"pc/internal/apperr"
	"pc/internal/auth"
	"pc/internal/config"
	"pc/internal/db"
	"pc/internal/face"
	"pc/internal/hemis"
	adminh "pc/internal/http/handlers/admin"
	authh "pc/internal/http/handlers/auth"
	filesh "pc/internal/http/handlers/files"
	studenth "pc/internal/http/handlers/student"
	teacherh "pc/internal/http/handlers/teacher"
	"pc/internal/http/middleware"
	"pc/internal/service"
	"pc/internal/storage"
)

type Deps struct {
	Cfg  *config.Config
	Log  *zap.Logger
	Pool *pgxpool.Pool
	Dist fs.FS
	// Optional overrides, used by tests.
	Face  face.Service
	Clock func() time.Time
	// HemisGroups replaces the HEMIS group list source (tests).
	HemisGroups hemis.GroupSource
	// HemisFaculties replaces the HEMIS faculty list source (tests).
	HemisFaculties hemis.FacultySource
	// RunBackground runs post-login work (HEMIS photo sync); nil = a goroutine.
	RunBackground func(func())
	// HemisPhotoFetch replaces the HEMIS photo downloader (tests).
	HemisPhotoFetch hemis.PhotoFetcher
	// TeacherOAuth replaces the HEMIS teacher OAuth client (tests).
	TeacherOAuth hemis.TeacherOAuth
	// StudentOAuth replaces the HEMIS student OAuth client (tests).
	StudentOAuth hemis.StudentOAuth
}

func NewServer(d Deps) *fiber.App {
	fcfg := fiber.Config{
		ErrorHandler:          NewErrorHandler(d.Log),
		DisableStartupMessage: true,
		BodyLimit:             25 * 1024 * 1024,
	}
	// Behind the reverse proxy c.IP() must be the real client, not the proxy: the login limiter and the
	// audit log depend on it. X-Forwarded-For is believed only when the peer is a trusted proxy.
	if len(d.Cfg.TrustedProxies) > 0 {
		fcfg.EnableTrustedProxyCheck = true
		fcfg.TrustedProxies = d.Cfg.TrustedProxies
		fcfg.ProxyHeader = fiber.HeaderXForwardedFor
	}
	app := fiber.New(fcfg)

	app.Use(requestid.New())
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(accessLog(d.Log))

	q := db.New(d.Pool)
	requireAuth := middleware.RequireAuth(q)
	limiter := auth.NewLoginLimiter(service.LoginAttempts, service.LoginWindow)

	store := storage.NewLocal(d.Cfg.UploadDir)
	var faceSvc face.Service = d.Face
	if faceSvc == nil {
		faceSvc = face.New(d.Cfg.FaceServiceURL, d.Cfg.FaceServiceToken, d.Cfg.FaceTimeout)
	}
	enrollSvc := service.NewEnrollmentService(d.Pool, store, faceSvc)

	authSvc := service.NewAuthService(d.Pool, d.Cfg, limiter)
	photoFetch := d.HemisPhotoFetch
	if photoFetch == nil {
		hosts := append([]string{}, d.Cfg.HemisImageHosts...)
		for _, raw := range []string{d.Cfg.TeacherResourceOwnerURL, d.Cfg.TeacherAuthorizeURL, d.Cfg.StudentResourceOwnerURL, d.Cfg.StudentAuthorizeURL} {
			if u, err := url.Parse(raw); err == nil && u.Host != "" {
				hosts = append(hosts, u.Host)
			}
		}
		photoFetch = hemis.NewPhotoFetcher(d.Cfg.HemisBaseURL, hosts, d.Cfg.HemisTimeout)
	}
	if d.RunBackground != nil {
		authSvc.WithBackground(d.RunBackground)
	}
	newOAuth := func(authorize, token, user string) *hemis.OAuth {
		return hemis.NewOAuth(hemis.OAuthConfig{
			ClientID: d.Cfg.OAuthClientID, ClientSecret: d.Cfg.OAuthClientSecret, RedirectURI: d.Cfg.OAuthRedirectURI,
			AuthorizeURL: authorize, TokenURL: token, UserURL: user,
		}, hemis.OAuthTimeout)
	}
	teacherOAuth := d.TeacherOAuth
	if teacherOAuth == nil && d.Cfg.TeacherOAuthEnabled() {
		teacherOAuth = newOAuth(d.Cfg.TeacherAuthorizeURL, d.Cfg.TeacherAccessTokenURL, d.Cfg.TeacherResourceOwnerURL)
	}
	if teacherOAuth != nil {
		authSvc.WithTeacherOAuth(teacherOAuth, photoFetch, store, d.Cfg.TeacherAutoCreate)
	} else {
		d.Log.Info("HEMIS teacher OAuth is not configured: teacher sign-in through HEMIS is disabled")
	}
	studentOAuth := d.StudentOAuth
	if studentOAuth == nil && d.Cfg.StudentOAuthEnabled() {
		studentOAuth = newOAuth(d.Cfg.StudentAuthorizeURL, d.Cfg.StudentAccessTokenURL, d.Cfg.StudentResourceOwnerURL)
	}
	if studentOAuth != nil {
		photos := service.NewHemisPhotoService(d.Pool, enrollSvc, photoFetch, d.Cfg.HemisTimeout, d.Log)
		authSvc.WithStudentOAuth(studentOAuth, photos, d.Log)
	} else {
		d.Log.Info("HEMIS student OAuth is not configured: student sign-in through HEMIS is disabled")
	}

	var groupSource hemis.GroupSource = d.HemisGroups
	if groupSource == nil && d.Cfg.HemisGroupsToken != "" && d.Cfg.HemisGroupsURL != "" {
		groupSource = hemis.NewGroups(d.Cfg.HemisGroupsURL, d.Cfg.HemisGroupsToken, 30*time.Second)
	}

	var facultySource hemis.FacultySource = d.HemisFaculties
	if facultySource == nil && d.Cfg.HemisGroupsToken != "" && d.Cfg.HemisFacultiesURL != "" {
		facultySource = hemis.NewFaculties(d.Cfg.HemisFacultiesURL, d.Cfg.HemisGroupsToken, 30*time.Second)
	}

	api := app.Group("/api")
	api.Get("/health", healthHandler(d))
	authHandler := authh.New(d.Cfg, authSvc)
	authHandler.Register(api.Group("/auth"), requireAuth)
	authHandler.RegisterHemisOAuth(api.Group("/auth"))
	authHandler.RegisterCallback(app) // before the SPA fallback

	// Role-scoped groups; their routes arrive in later phases.
	reports := service.NewReportService(d.Pool, d.Cfg.Location)

	clock := d.Clock
	if clock == nil {
		clock = time.Now
	}
	adminh.New(adminh.Services{
		Reports:     reports,
		Enroll:      enrollSvc,
		Attempts:    service.NewAttemptsAdminService(d.Pool, d.Cfg.Location).WithStorage(store),
		GroupSync:   service.NewGroupSyncService(d.Pool, groupSource),
		FacultySync: service.NewFacultySyncService(d.Pool, facultySource),
		Dashboard:   service.NewDashboardService(d.Pool, d.Cfg.Location),
		Faculties:   service.NewFacultyService(d.Pool),
		Groups:      service.NewGroupService(d.Pool),
		Users:       service.NewUserService(d.Pool),
		Locations:   service.NewLocationService(d.Pool),
		Options:     service.NewOptionService(d.Pool),
		Schedules:   service.NewScheduleService(d.Pool),
		Audit:       service.NewAuditService(d.Pool, d.Cfg.Location),
	}).Register(api.Group("/admin", requireAuth, middleware.RequireRole("admin", "little")))
	teacherh.New(reports, service.NewAssignmentService(d.Pool, store, d.Cfg.Location)).
		Register(api.Group("/teacher", requireAuth, middleware.RequireRole("teacher")))
	filesh.New(service.NewFileService(d.Pool, store)).Register(api.Group("/files", requireAuth))
	studenth.New(
		service.NewStudentService(d.Pool, store, d.Cfg.Location, reports, d.Cfg.MaxAttemptsPerStep).WithClock(clock).WithGPSLimit(d.Cfg.GPSMaxAccuracyM),
		service.NewAttemptService(d.Pool, store, faceSvc, d.Cfg.Location, d.Cfg.MaxAttemptsPerStep, d.Cfg.GPSMaxAccuracyM).WithClock(clock).WithMismatchLimit(d.Cfg.FaceMismatchLimit),
	).Register(api.Group("/student", requireAuth, middleware.RequireRole("student")))
	// Unknown API paths must not fall through to the SPA.
	api.All("/*", func(c *fiber.Ctx) error { return apperr.NotFound() })

	mountSPA(app, d.Dist)
	return app
}

func accessLog(log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		log.Info("request",
			zap.String("request_id", c.GetRespHeader(fiber.HeaderXRequestID)),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("took", time.Since(start)),
			zap.String("ip", c.IP()),
		)
		return err
	}
}

func healthHandler(d Deps) fiber.Handler {
	client := &http.Client{Timeout: 3 * time.Second}
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		status := fiber.Map{"db": "ok", "face_service": "ok"}
		healthy := true

		if err := d.Pool.Ping(ctx); err != nil {
			status["db"] = "down"
			healthy = false
		}

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, d.Cfg.FaceServiceURL+"/health", nil)
		req.Header.Set("X-Internal-Token", d.Cfg.FaceServiceToken)
		if resp, err := client.Do(req); err != nil || resp.StatusCode != http.StatusOK {
			status["face_service"] = "down"
			healthy = false
			if resp != nil {
				resp.Body.Close()
			}
		} else {
			resp.Body.Close()
		}

		status["status"] = "ok"
		code := fiber.StatusOK
		if !healthy {
			status["status"] = "degraded"
			code = fiber.StatusServiceUnavailable
		}
		return c.Status(code).JSON(status)
	}
}

// mountSPA serves the embedded Vue build; every other GET falls back to index.html.
func mountSPA(app *fiber.App, dist fs.FS) {
	app.Use("/assets", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		return c.Next()
	})
	app.Use(filesystem.New(filesystem.Config{
		Root:   http.FS(dist),
		Browse: false,
		// Only serve real files; everything else falls through to the SPA handler.
		Next: func(c *fiber.Ctx) bool {
			name := strings.TrimPrefix(c.Path(), "/")
			if name == "" {
				return true
			}
			info, err := fs.Stat(dist, name)
			return err != nil || info.IsDir()
		},
	}))
	app.Get("/*", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/") || strings.HasPrefix(c.Path(), "/assets/") {
			return apperr.NotFound()
		}
		buf, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "frontend not built")
		}
		c.Set(fiber.HeaderCacheControl, "no-cache")
		c.Type("html")
		return c.Send(buf)
	})
}
