package service

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"pc/internal/audit"
	"pc/internal/db"
	"pc/internal/hemis"
	"pc/internal/storage"
)

// OAuthMethod is recorded on the audit entry of a login through HEMIS OAuth.
const OAuthMethod = "hemis_oauth"

// Error codes the callback hands to /login?error=<code>.
const (
	OAuthErrState         = "hemis_state"
	OAuthErrDenied        = "hemis_denied"
	OAuthErrToken         = "hemis_token"
	OAuthErrProfile       = "hemis_profile"
	OAuthErrNotTeacher    = "hemis_not_teacher"
	OAuthErrInactive      = "hemis_inactive"
	OAuthErrNotRegistered = "hemis_not_registered"
	OAuthErrUnavailable   = "hemis_unavailable"
	OAuthErrRateLimited   = "hemis_rate_limited"
	OAuthErrInternal      = "hemis_internal"
)

// OAuthError is a failed teacher sign-in; Code is safe to put in a redirect URL.
type OAuthError struct{ Code string }

func (e *OAuthError) Error() string { return "hemis oauth: " + e.Code }

func oauthErr(code string) error { return &OAuthError{Code: code} }

// TeacherOAuth holds what the teacher sign-in needs besides the AuthService basics.
type teacherOAuth struct {
	client     hemis.TeacherOAuth
	fetchPhoto hemis.PhotoFetcher
	store      storage.Storage
	autoCreate bool
}

// WithTeacherOAuth enables teacher sign-in through HEMIS OAuth.
func (s *AuthService) WithTeacherOAuth(c hemis.TeacherOAuth, fetch hemis.PhotoFetcher, store storage.Storage, autoCreate bool) *AuthService {
	s.teacher = &teacherOAuth{client: c, fetchPhoto: fetch, store: store, autoCreate: autoCreate}
	return s
}

// TeacherOAuthEnabled reports whether the HEMIS teacher button should work.
func (s *AuthService) TeacherOAuthEnabled() bool { return s.teacher != nil }

// TeacherAuthorizeURL is where the browser goes to sign in at HEMIS.
func (s *AuthService) TeacherAuthorizeURL(state string) string {
	return s.teacher.client.AuthorizeURL(state)
}

// OAuthFailed audits a failed attempt. The reason is a code, never a secret, code or token.
func (s *AuthService) OAuthFailed(ctx context.Context, reason string, meta Actor) {
	_ = audit.Write(ctx, s.pool, audit.Entry{
		Action: audit.ActionLoginFailed, EntityType: "user",
		After: map[string]any{"method": OAuthMethod, "reason": reason},
		IP:    meta.IP, UserAgent: meta.UserAgent,
	})
}

// OAuthAllowed rate limits the callback per IP.
func (s *AuthService) OAuthAllowed(ip string) bool { return s.limiter.Allowed("oauth|" + ip) }

// OAuthRecordFailure counts a failure against the caller's IP.
func (s *AuthService) OAuthRecordFailure(ip string) { s.limiter.RecordFailure("oauth|" + ip) }

// TeacherLoginWithCode finishes the authorization-code flow: exchange the code, read the profile,
// find or create the teacher, sync their fields and open our own session.
// Failures are *OAuthError (and are audited here).
func (s *AuthService) TeacherLoginWithCode(ctx context.Context, code string, meta Actor) (*LoginResult, error) {
	res, err := s.teacherLogin(ctx, code, meta)
	var oe *OAuthError
	switch {
	case err == nil:
		return res, nil
	case errors.As(err, &oe):
		s.OAuthFailed(ctx, oe.Code, meta)
		return nil, err
	default:
		s.log.Error("hemis oauth login failed", zap.Error(err))
		s.OAuthFailed(ctx, OAuthErrInternal, meta)
		return nil, oauthErr(OAuthErrInternal)
	}
}

func (s *AuthService) teacherLogin(ctx context.Context, code string, meta Actor) (*LoginResult, error) {
	t := s.teacher
	token, err := t.client.ExchangeCode(ctx, code)
	if err != nil {
		return nil, s.classifyOAuth(err, OAuthErrToken)
	}
	prof, err := t.client.TeacherProfile(ctx, token)
	if err != nil {
		return nil, s.classifyOAuth(err, OAuthErrProfile)
	}

	var (
		user    db.User
		raw     string
		expires time.Time
	)
	run := func() error {
		return InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
			u, err := s.syncTeacher(ctx, tx, q, prof, meta)
			if err != nil {
				return err
			}
			user = u
			raw, expires, err = s.createSession(ctx, tx, q, u, meta, OAuthMethod)
			return err
		})
	}
	err = run()
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // two first logins raced on the unique hemis_login
		err = run()
	}
	if err != nil {
		return nil, err
	}
	s.limiter.Reset("oauth|" + meta.IP)

	// The photo comes after the commit and never blocks the login.
	if prof.PictureURL != "" {
		if err := s.syncTeacherPhoto(ctx, &user, prof.PictureURL); err != nil {
			s.log.Warn("hemis teacher photo failed", zap.String("user_id", user.ID.String()), zap.Error(err))
		}
	}
	return &LoginResult{Token: raw, User: user, Expires: expires}, nil
}

func (s *AuthService) classifyOAuth(err error, code string) error {
	if errors.Is(err, hemis.ErrUnavailable) {
		s.log.Warn("hemis oauth unavailable", zap.Error(err)) // the error text holds no URL, body or token
		return oauthErr(OAuthErrUnavailable)
	}
	if errors.Is(err, hemis.ErrOAuthToken) || errors.Is(err, hemis.ErrOAuthProfile) {
		s.log.Warn("hemis oauth rejected", zap.Error(err))
		return oauthErr(code)
	}
	return err
}

// syncTeacher finds the teacher by hemis_login (creating one when allowed) and applies the profile.
// A value HEMIS leaves empty never overwrites what we already have. role, is_active, password_hash
// and groups.teacher_id are never touched.
func (s *AuthService) syncTeacher(ctx context.Context, tx pgx.Tx, q *db.Queries, p *hemis.TeacherProfile, meta Actor) (db.User, error) {
	login := pgtype.Text{String: p.Login, Valid: true}
	existing, err := q.GetUserByHemisLogin(ctx, login)
	found := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, err
	}
	if found {
		switch {
		case existing.Role != "teacher":
			return db.User{}, oauthErr(OAuthErrNotTeacher)
		case !existing.IsActive:
			return db.User{}, oauthErr(OAuthErrInactive)
		}
	} else if !s.teacher.autoCreate {
		return db.User{}, oauthErr(OAuthErrNotRegistered)
	}

	hid := pgtype.Int8{}
	if p.HemisID != 0 {
		hid = hemisID(p.HemisID)
	}
	fullName := p.FullName
	if fullName == "" {
		fullName = strings.TrimSpace(strings.Join([]string{p.SecondName, p.FirstName, p.ThirdName}, " "))
	}

	if !found {
		u, err := q.CreateHemisTeacher(ctx, db.CreateHemisTeacherParams{
			HemisLogin: login, HemisID: hid, FirstName: p.FirstName, SecondName: p.SecondName, ThirdName: p.ThirdName,
			FullName: fullName, BirthDate: p.BirthDate, Phone: p.Phone, PassportPin: p.PassportPin, PassportNumber: p.PassportNumber,
		})
		if err != nil {
			return db.User{}, err
		}
		return u, audit.Write(ctx, tx, audit.Entry{ActorID: &u.ID, ActorRole: "teacher", Action: audit.ActionCreate,
			EntityType: "user", EntityID: &u.ID, After: u, IP: meta.IP, UserAgent: meta.UserAgent})
	}

	u, err := q.UpdateTeacherFromHemis(ctx, db.UpdateTeacherFromHemisParams{
		ID: existing.ID, HemisID: hid, FirstName: p.FirstName, SecondName: p.SecondName, ThirdName: p.ThirdName,
		FullName: fullName, BirthDate: p.BirthDate, Phone: p.Phone, PassportPin: p.PassportPin, PassportNumber: p.PassportNumber,
	})
	if err != nil {
		return db.User{}, err
	}
	if audit.HasChanges(existing, u) {
		if err := audit.Write(ctx, tx, audit.Entry{ActorID: &u.ID, ActorRole: "teacher", Action: audit.ActionUpdate,
			EntityType: "user", EntityID: &u.ID, Before: existing, After: u, IP: meta.IP, UserAgent: meta.UserAgent}); err != nil {
			return db.User{}, err
		}
	}
	return u, nil
}

// syncTeacherPhoto downloads the HEMIS picture unless this very picture is already stored.
// Teachers need no face embedding, only the avatar.
func (s *AuthService) syncTeacherPhoto(ctx context.Context, u *db.User, pictureURL string) error {
	t := s.teacher
	pictureURL = strings.TrimSpace(pictureURL)
	if t.fetchPhoto == nil || t.store == nil || (u.Image != "" && u.HemisImgUrl == pictureURL) {
		return nil
	}
	data, name, err := t.fetchPhoto(ctx, pictureURL)
	if err != nil {
		return err
	}
	ext, err := storage.Validate(storage.KindAvatars, name, data[:min(len(data), 512)])
	if err != nil {
		return err
	}
	path, err := t.store.Save(ctx, storage.KindAvatars, ext, bytes.NewReader(data))
	if err != nil {
		return err
	}
	old := u.Image
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		if err := q.SetUserImage(ctx, db.SetUserImageParams{ID: u.ID, Image: path}); err != nil {
			return err
		}
		return q.SetUserHemisImageURL(ctx, db.SetUserHemisImageURLParams{ID: u.ID, HemisImgUrl: pictureURL})
	})
	if err != nil {
		_ = t.store.Delete(ctx, path)
		return err
	}
	u.Image, u.HemisImgUrl = path, pictureURL
	if old != "" && old != path {
		_ = t.store.Delete(ctx, old)
	}
	return nil
}
