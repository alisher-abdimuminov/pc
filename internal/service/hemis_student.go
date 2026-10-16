package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"pc/internal/audit"
	"pc/internal/db"
	"pc/internal/hemis"
)

var (
	errInactiveStudent = errors.New("inactive student")
	errUsernameTaken   = errors.New("username belongs to another account")
)

// studentOAuth holds what the student sign-in needs besides the AuthService basics.
type studentOAuth struct {
	client hemis.StudentOAuth
	photos *HemisPhotoService
}

// WithStudentOAuth enables student sign-in through HEMIS OAuth. photos may be nil.
func (s *AuthService) WithStudentOAuth(c hemis.StudentOAuth, photos *HemisPhotoService, log *zap.Logger) *AuthService {
	s.student = &studentOAuth{client: c, photos: photos}
	if log != nil {
		s.log = log
	}
	return s
}

// StudentOAuthEnabled reports whether the HEMIS student button should work.
func (s *AuthService) StudentOAuthEnabled() bool { return s.student != nil }

// StudentAuthorizeURL is where the browser goes to sign in at HEMIS as a student.
func (s *AuthService) StudentAuthorizeURL(state string) string {
	return s.student.client.AuthorizeURL(state)
}

// StudentLoginWithCode finishes the authorization-code flow for a student: exchange the code, read the
// profile, sync the student with their faculty and group and open our own session.
// Failures are *OAuthError (and are audited here).
func (s *AuthService) StudentLoginWithCode(ctx context.Context, code string, meta Actor) (*LoginResult, error) {
	res, err := s.studentLogin(ctx, code, meta)
	return s.finishOAuth(ctx, res, err, meta)
}

func (s *AuthService) studentLogin(ctx context.Context, code string, meta Actor) (*LoginResult, error) {
	st := s.student
	token, err := st.client.ExchangeCode(ctx, code)
	if err != nil {
		return nil, s.classifyOAuth(err, OAuthErrToken)
	}
	prof, err := st.client.StudentProfile(ctx, token)
	if err != nil {
		return nil, s.classifyOAuth(err, OAuthErrProfile)
	}
	username := strings.ToLower(strings.TrimSpace(prof.StudentIDNumber))
	if username == "" {
		username = strconv.FormatInt(prof.HemisID, 10)
	}

	var (
		user    db.User
		raw     string
		expires time.Time
	)
	run := func() error {
		return InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
			u, err := s.upsertStudent(ctx, tx, q, username, *prof, meta)
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
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // two first logins raced on the unique hemis_id/username
		err = run()
	}
	switch {
	case errors.Is(err, errInactiveStudent):
		return nil, oauthErr(OAuthErrInactive)
	case errors.Is(err, errUsernameTaken):
		return nil, oauthErr(OAuthErrProfile)
	case err != nil:
		return nil, err
	}

	if st.photos != nil && prof.ImageURL != "" {
		uid, url := user.ID, prof.ImageURL
		s.runBG(func() {
			// detached from the request: the login must not wait for (or fail on) the photo
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			if err := st.photos.Sync(ctx, uid, url); err != nil {
				s.log.Warn("hemis photo sync failed", zap.String("user_id", uid.String()), zap.Error(err))
			}
		})
	}
	return &LoginResult{Token: raw, User: user, Expires: expires}, nil
}

func hemisID(id int64) pgtype.Int8 { return pgtype.Int8{Int64: id, Valid: true} }

// upsertStudent creates or updates the student, their faculty and their group from a HEMIS profile.
// Users are matched by hemis_id; a seeded student without one is adopted by username.
func (s *AuthService) upsertStudent(ctx context.Context, tx pgx.Tx, q *db.Queries, username string, p hemis.Profile, meta Actor) (db.User, error) {
	var facultyID, groupID = nullUUIDValue(), nullUUIDValue()
	if p.Faculty != nil && p.Faculty.ID != 0 {
		f, err := q.UpsertFacultyByHemis(ctx, db.UpsertFacultyByHemisParams{Name: p.Faculty.Name, HemisID: hemisID(p.Faculty.ID)})
		if err != nil {
			return db.User{}, err
		}
		facultyID = uuidNull(f.ID)
	}
	if p.Group != nil && p.Group.ID != 0 {
		g, err := q.UpsertGroupByHemis(ctx, db.UpsertGroupByHemisParams{Name: p.Group.Name, HemisID: hemisID(p.Group.ID), FacultyID: facultyID})
		if err != nil {
			return db.User{}, err
		}
		groupID = uuidNull(g.ID)
	}

	existing, err := q.GetUserByRoleHemisID(ctx, db.GetUserByRoleHemisIDParams{Role: "student", HemisID: hemisID(p.HemisID)})
	if errors.Is(err, pgx.ErrNoRows) {
		existing, err = q.GetUserByUsername(ctx, pgtype.Text{String: username, Valid: true})
		if err == nil && (existing.Role != "student" || existing.HemisID.Valid) {
			// the username belongs to someone else (another HEMIS account or a staff member)
			return db.User{}, errUsernameTaken
		}
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, err
	}

	var user db.User
	if errors.Is(err, pgx.ErrNoRows) {
		user, err = q.CreateHemisStudent(ctx, db.CreateHemisStudentParams{
			Username: pgtype.Text{String: username, Valid: true}, HemisID: hemisID(p.HemisID),
			FirstName: p.FirstName, SecondName: p.SecondName, ThirdName: p.ThirdName, FullName: p.FullName, ShortName: p.ShortName,
			BirthDate: p.BirthDate, Phone: p.Phone, PassportPin: p.PassportPin, PassportNumber: p.PassportNumber, Gender: p.Gender,
			PaymentForm: p.PaymentForm, Level: p.Level, Semester: p.Semester, Gpa: p.GPA, Address: p.Address,
			Country: p.Country, Province: p.Province, District: p.District, GroupID: groupID, FacultyID: facultyID,
		})
		if err != nil {
			return db.User{}, err
		}
		return user, audit.Write(ctx, tx, audit.Entry{ActorID: &user.ID, ActorRole: "student", Action: audit.ActionCreate,
			EntityType: "user", EntityID: &user.ID, After: user, IP: meta.IP, UserAgent: meta.UserAgent})
	}

	if !existing.IsActive {
		return db.User{}, errInactiveStudent
	}
	user, err = q.UpdateUserFromHemis(ctx, db.UpdateUserFromHemisParams{
		ID: existing.ID, HemisID: hemisID(p.HemisID),
		FirstName: p.FirstName, SecondName: p.SecondName, ThirdName: p.ThirdName, FullName: p.FullName, ShortName: p.ShortName,
		BirthDate: p.BirthDate, Phone: p.Phone, PassportPin: p.PassportPin, PassportNumber: p.PassportNumber, Gender: p.Gender,
		PaymentForm: p.PaymentForm, Level: p.Level, Semester: p.Semester, Gpa: p.GPA, Address: p.Address,
		Country: p.Country, Province: p.Province, District: p.District, GroupID: groupID, FacultyID: facultyID,
	})
	if err != nil {
		return db.User{}, err
	}
	// only changed, non-sensitive fields reach the audit log (passport data is stripped by audit)
	if audit.HasChanges(existing, user) {
		if err := audit.Write(ctx, tx, audit.Entry{ActorID: &user.ID, ActorRole: "student", Action: audit.ActionUpdate,
			EntityType: "user", EntityID: &user.ID, Before: existing, After: user, IP: meta.IP, UserAgent: meta.UserAgent}); err != nil {
			return db.User{}, err
		}
	}
	return user, nil
}
