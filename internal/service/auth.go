package service

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"pc/internal/apperr"
	"pc/internal/audit"
	"pc/internal/auth"
	"pc/internal/config"
	"pc/internal/db"
	"pc/internal/hemis"
)

// AccountInactiveMessage is shown to a deactivated user (also right after a face-mismatch block).
const AccountInactiveMessage = "Hisobingiz faol emas. Tizim adminlariga murojaat qiling."

const (
	LoginAttempts = 5
	LoginWindow   = 5 * time.Minute
)

var (
	errBadCredentials  = apperr.New(http.StatusUnauthorized, "invalid_credentials", "Login yoki parol noto'g'ri")
	errRateLimited     = apperr.New(http.StatusTooManyRequests, "too_many_requests", "Juda ko'p urinish. Bir necha daqiqadan so'ng qayta urinib ko'ring")
	errStudentLogin    = apperr.New(http.StatusForbidden, "student_login_unavailable", "Talabalar HEMIS orqali kirishi kerak")
	errAccountInactive = apperr.New(http.StatusForbidden, "account_inactive", AccountInactiveMessage)
	errHemisDown       = apperr.New(http.StatusServiceUnavailable, "hemis_unavailable", hemis.UnavailableMessage)
)

type AuthService struct {
	pool    *pgxpool.Pool
	cfg     *config.Config
	limiter *auth.LoginLimiter
	// dummyHash keeps timing similar for unknown usernames.
	dummyHash string

	hemis  hemis.Client // nil when HEMIS_BASE_URL is not configured
	photos *HemisPhotoService
	runBG  func(func())
	log    *zap.Logger

	teacher *teacherOAuth // nil when HEMIS OAuth is not configured
}

func NewAuthService(pool *pgxpool.Pool, cfg *config.Config, limiter *auth.LoginLimiter) *AuthService {
	h, _ := auth.HashPassword("dummy-password-for-timing")
	return &AuthService{pool: pool, cfg: cfg, limiter: limiter, dummyHash: h, runBG: func(f func()) { go f() }, log: zap.NewNop()}
}

// WithHemis enables student login through HEMIS.
func (s *AuthService) WithHemis(c hemis.Client, photos *HemisPhotoService, log *zap.Logger) *AuthService {
	s.hemis, s.photos, s.log = c, photos, log
	return s
}

// WithBackground replaces how post-login work (photo sync) is run; tests run it inline.
func (s *AuthService) WithBackground(run func(func())) *AuthService {
	s.runBG = run
	return s
}

type LoginResult struct {
	Token   string
	User    db.User
	Expires time.Time
}

func limiterKey(ip, username string) string { return ip + "|" + username }

// Login verifies credentials, creates a session and audits the outcome.
//
// Which system checks the password is decided by who the user is:
//   - admins and teachers: the local password;
//   - students: HEMIS (their HEMIS password is forwarded, never stored). With
//     DEV_STUDENT_PASSWORD_LOGIN a student that has a local password may use it instead (dev only).
func (s *AuthService) Login(ctx context.Context, username, password string, meta Actor) (*LoginResult, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" || password == "" {
		return nil, apperr.Validation("Login va parolni kiriting", map[string]string{"username": "Majburiy maydon", "password": "Majburiy maydon"})
	}
	key := limiterKey(meta.IP, username)
	if !s.limiter.Allowed(key) {
		s.auditFailure(ctx, username, "rate_limited", meta)
		return nil, errRateLimited
	}

	user, err := db.New(s.pool).GetUserByUsername(ctx, pgtype.Text{String: username, Valid: true})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	found := err == nil
	hash := s.dummyHash
	if found && user.PasswordHash.Valid {
		hash = user.PasswordHash.String
	}
	localOK := auth.CheckPassword(hash, password) // always computed: keeps timing uniform

	switch {
	case found && user.Role != "student":
		return s.finishLocal(ctx, key, username, user, localOK, meta)
	case found && localOK && s.cfg.DevStudentPasswordLogin:
		return s.finishLocal(ctx, key, username, user, true, meta)
	case s.hemis != nil:
		return s.hemisLogin(ctx, key, username, password, meta)
	case found && localOK: // a student with the right password, but no HEMIS and no dev switch
		return s.fail(ctx, key, username, "student_login_disabled", errStudentLogin, meta)
	default:
		return s.fail(ctx, key, username, "invalid_credentials", errBadCredentials, meta)
	}
}

func (s *AuthService) fail(ctx context.Context, key, username, reason string, err error, meta Actor, detail ...string) (*LoginResult, error) {
	if reason != "hemis_unavailable" { // an outage is not the user's fault: do not count it against them
		s.limiter.RecordFailure(key)
	}
	s.auditFailure(ctx, username, reason, meta, detail...)
	return nil, err
}

func (s *AuthService) finishLocal(ctx context.Context, key, username string, user db.User, passOK bool, meta Actor) (*LoginResult, error) {
	switch {
	case !passOK:
		return s.fail(ctx, key, username, "invalid_credentials", errBadCredentials, meta)
	case !user.IsActive:
		return s.fail(ctx, key, username, "inactive", errAccountInactive, meta)
	}
	var raw string
	var expires time.Time
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) (err error) {
		raw, expires, err = s.createSession(ctx, tx, q, user, meta, "")
		return err
	})
	if err != nil {
		return nil, err
	}
	s.limiter.Reset(key)
	return &LoginResult{Token: raw, User: user, Expires: expires}, nil
}

// createSession stores a new session, stamps last_login_at and audits the login (inside tx).
func (s *AuthService) createSession(ctx context.Context, tx pgx.Tx, q *db.Queries, user db.User, meta Actor, method string) (string, time.Time, error) {
	raw, tokenHash, err := auth.NewToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(s.cfg.SessionTTL)
	p := db.CreateSessionParams{ID: tokenHash, UserID: user.ID, UserAgent: meta.UserAgent,
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true}}
	if addr, perr := netip.ParseAddr(meta.IP); perr == nil {
		p.Ip = &addr
	}
	if err := q.CreateSession(ctx, p); err != nil {
		return "", time.Time{}, err
	}
	if err := q.TouchUserLogin(ctx, user.ID); err != nil {
		return "", time.Time{}, err
	}
	e := audit.Entry{
		ActorID: &user.ID, ActorRole: user.Role, Action: audit.ActionLogin,
		EntityType: "user", EntityID: &user.ID, IP: meta.IP, UserAgent: meta.UserAgent,
	}
	if method != "" {
		e.After = map[string]any{"method": method}
	}
	return raw, expires, audit.Write(ctx, tx, e)
}

// auditFailure records a failed login. detail is an optional safe cause (e.g. "status 503").
func (s *AuthService) auditFailure(ctx context.Context, username, reason string, meta Actor, detail ...string) {
	after := map[string]any{"username": username, "reason": reason}
	if len(detail) > 0 && detail[0] != "" {
		after["detail"] = detail[0]
	}
	_ = audit.Write(ctx, s.pool, audit.Entry{
		Action: audit.ActionLoginFailed, EntityType: "user",
		After: after,
		IP:    meta.IP, UserAgent: meta.UserAgent,
	})
}

// Logout deletes the session and audits it.
func (s *AuthService) Logout(ctx context.Context, user db.User, sessionID string, meta Actor) error {
	return InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		if err := q.DeleteSession(ctx, sessionID); err != nil {
			return err
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID: &user.ID, ActorRole: user.Role, Action: audit.ActionLogout,
			EntityType: "user", EntityID: &user.ID, IP: meta.IP, UserAgent: meta.UserAgent,
		})
	})
}

// ChangePassword checks the current password, stores the new one and drops other sessions.
func (s *AuthService) ChangePassword(ctx context.Context, user db.User, sessionID, oldPass, newPass string, meta Actor) error {
	fields := map[string]string{}
	if !auth.CheckPassword(user.PasswordHash.String, oldPass) {
		fields["old_password"] = "Joriy parol noto'g'ri"
	}
	if err := auth.ValidateNewPassword(newPass); err != nil {
		fields["new_password"] = "Parol kamida 8 ta belgidan iborat bo'lishi kerak"
	} else if newPass == oldPass {
		fields["new_password"] = "Yangi parol joriy paroldan farq qilishi kerak"
	}
	if len(fields) > 0 {
		return apperr.Validation("Ma'lumotlar noto'g'ri", fields)
	}
	hash, err := auth.HashPassword(newPass)
	if err != nil {
		return err
	}
	return InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		if err := q.SetUserPassword(ctx, db.SetUserPasswordParams{
			ID: user.ID, PasswordHash: pgtype.Text{String: hash, Valid: true}, MustChangePassword: false,
		}); err != nil {
			return err
		}
		if err := q.DeleteUserSessionsExcept(ctx, db.DeleteUserSessionsExceptParams{UserID: user.ID, ID: sessionID}); err != nil {
			return err
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID: &user.ID, ActorRole: user.Role, Action: audit.ActionUpdate,
			EntityType: "user", EntityID: &user.ID,
			Before: map[string]any{"must_change_password": user.MustChangePassword},
			After:  map[string]any{"must_change_password": false, "password_changed": true},
			IP:     meta.IP, UserAgent: meta.UserAgent,
		})
	})
}
