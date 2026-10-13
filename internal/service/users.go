package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/audit"
	"pc/internal/auth"
	"pc/internal/db"
	"pc/internal/permission"
)

type UserService struct {
	pool *pgxpool.Pool
}

func NewUserService(pool *pgxpool.Pool) *UserService { return &UserService{pool: pool} }

type CreateUserInput struct {
	Username           string
	Role               string
	FullName           string
	Password           string // generated when empty
	MustChangePassword bool
	GroupID            *uuid.UUID
	FacultyID          *uuid.UUID
	HemisLogin         string   // optional: links the account to a HEMIS teacher
	Permissions        []string // "little" admins only: what they may use
}

// CreateUser creates a user and returns it together with the plain password (shown once).
func (s *UserService) CreateUser(ctx context.Context, actor Actor, in CreateUserInput) (db.User, string, error) {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.FullName = strings.TrimSpace(in.FullName)
	fields := map[string]string{}
	if in.Username == "" {
		fields["username"] = "Majburiy maydon"
	}
	switch in.Role {
	case "admin", "little", "teacher", "student":
	default:
		fields["role"] = "Noto'g'ri rol"
	}
	var perms []string
	if in.Role == permission.RoleLittle {
		var err error
		if perms, err = permission.Normalize(in.Permissions); err != nil {
			fields["permissions"] = "Noto'g'ri ruxsat"
		}
	} else if len(in.Permissions) > 0 {
		fields["permissions"] = "Ruxsatlar faqat kichik administratorga beriladi"
	}
	if len(fields) > 0 {
		return db.User{}, "", apperr.Validation("Ma'lumotlar noto'g'ri", fields)
	}

	plain := in.Password
	if plain == "" {
		var err error
		if plain, err = auth.GeneratePassword(); err != nil {
			return db.User{}, "", err
		}
	}
	hash, err := auth.HashPassword(plain)
	if err != nil {
		return db.User{}, "", err
	}

	var created db.User
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		u, err := q.CreateUser(ctx, db.CreateUserParams{
			Username:           pgtype.Text{String: in.Username, Valid: true},
			PasswordHash:       pgtype.Text{String: hash, Valid: true},
			Role:               in.Role,
			IsActive:           true,
			MustChangePassword: in.MustChangePassword,
			FullName:           in.FullName,
			GroupID:            nullUUID(in.GroupID),
			FacultyID:          nullUUID(in.FacultyID),
			Permissions:        perms,
		})
		if err != nil {
			return err
		}
		if login := strings.TrimSpace(in.HemisLogin); login != "" && in.Role == "teacher" {
			if err := q.SetUserHemisLogin(ctx, db.SetUserHemisLoginParams{ID: u.ID, HemisLogin: pgtype.Text{String: login, Valid: true}}); err != nil {
				return err
			}
			u.HemisLogin = pgtype.Text{String: login, Valid: true}
		}
		created = u
		return audit.Write(ctx, tx, audit.Entry{
			ActorID: actor.ID, ActorRole: actor.Role, Action: audit.ActionCreate,
			EntityType: "user", EntityID: &u.ID, After: u, IP: actor.IP, UserAgent: actor.UserAgent,
		})
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_hemis_login_key" {
		return db.User{}, "", mapDBError(err)
	}
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return db.User{}, "", apperr.Validation("Bu login band", map[string]string{"username": "Bu login band"})
	}
	if err != nil {
		return db.User{}, "", err
	}
	return created, plain, nil
}

// ---- admin operations ----

type UserFilter struct {
	Role      string
	FacultyID *uuid.UUID
	GroupID   *uuid.UUID
	IsActive  *bool
}

type UserUpdateInput struct {
	FullName  string     `json:"full_name"`
	Phone     string     `json:"phone"`
	GroupID   *uuid.UUID `json:"group_id"`
	FacultyID *uuid.UUID `json:"faculty_id"`
	// HemisLogin links a teacher to their HEMIS account ("" = unlink, absent = unchanged). Ignored for other roles.
	HemisLogin *string `json:"hemis_login"`
}

type SensitiveDTO struct {
	PassportPin    string `json:"passport_pin"`
	PassportNumber string `json:"passport_number"`
}

func (s *UserService) List(ctx context.Context, p ListParams, f UserFilter) (Page[UserDTO], error) {
	p = p.Norm()
	q := db.New(s.pool)
	role := pgtype.Text{String: f.Role, Valid: f.Role != ""}
	active := pgtype.Bool{}
	if f.IsActive != nil {
		active = pgtype.Bool{Bool: *f.IsActive, Valid: true}
	}
	rows, err := q.ListUsers(ctx, db.ListUsersParams{
		Role: role, FacultyID: nullUUID(f.FacultyID), GroupID: nullUUID(f.GroupID), IsActive: active,
		Search: p.Search, Lim: p.Limit(), Off: p.Offset(),
	})
	if err != nil {
		return Page[UserDTO]{}, err
	}
	total, err := q.CountUsers(ctx, db.CountUsersParams{
		Role: role, FacultyID: nullUUID(f.FacultyID), GroupID: nullUUID(f.GroupID), IsActive: active, Search: p.Search,
	})
	if err != nil {
		return Page[UserDTO]{}, err
	}
	items := make([]UserDTO, len(rows))
	for i, r := range rows {
		items[i] = userDTO(listRowUser(r), textPtr(r.GroupName), textPtr(r.FacultyName))
	}
	return newPage(items, total, p), nil
}

func (s *UserService) dto(ctx context.Context, q *db.Queries, u db.User) UserDTO {
	var gn, fn *string
	if u.GroupID.Valid {
		if g, err := q.GetGroup(ctx, u.GroupID.UUID); err == nil {
			gn = &g.Name
		}
	}
	if u.FacultyID.Valid {
		if f, err := q.GetFaculty(ctx, u.FacultyID.UUID); err == nil {
			fn = &f.Name
		}
	}
	return userDTO(u, gn, fn)
}

func (s *UserService) Get(ctx context.Context, id uuid.UUID) (UserDTO, error) {
	q := db.New(s.pool)
	u, err := q.GetUser(ctx, id)
	if err != nil {
		return UserDTO{}, notFoundOr(err)
	}
	return s.dto(ctx, q, u), nil
}

// Sensitive returns passport data and writes a view_sensitive audit entry.
func (s *UserService) Sensitive(ctx context.Context, actor Actor, id uuid.UUID) (SensitiveDTO, error) {
	var out SensitiveDTO
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		u, err := q.GetUser(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		out = SensitiveDTO{PassportPin: u.PassportPin, PassportNumber: u.PassportNumber}
		return audit.Write(ctx, tx, actor.entry(audit.ActionViewSensitive, "user", &id, nil,
			map[string]any{"fields": []string{"passport_pin", "passport_number"}}))
	})
	return out, err
}

func (s *UserService) Update(ctx context.Context, actor Actor, id uuid.UUID, in UserUpdateInput) (UserDTO, error) {
	var out UserDTO
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := q.GetUser(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		name := strings.TrimSpace(in.FullName)
		if name == "" {
			return fieldErr("full_name", "Majburiy maydon")
		}
		groupID, facultyID := in.GroupID, in.FacultyID
		if old.Role != "student" { // staff do not belong to groups/faculties
			groupID, facultyID = nil, nil
		}
		if groupID != nil {
			if _, err := q.GetGroup(ctx, *groupID); err != nil {
				return fieldErr("group_id", "Guruh topilmadi")
			}
		}
		if facultyID != nil {
			if _, err := q.GetFaculty(ctx, *facultyID); err != nil {
				return fieldErr("faculty_id", "Fakultet topilmadi")
			}
		}
		u, err := q.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
			ID: id, FirstName: old.FirstName, SecondName: old.SecondName, ThirdName: old.ThirdName,
			FullName: name, ShortName: old.ShortName, Phone: strings.TrimSpace(in.Phone),
			GroupID: nullUUID(groupID), FacultyID: nullUUID(facultyID),
		})
		if err != nil {
			return err
		}
		if old.Role == "teacher" && in.HemisLogin != nil {
			login := strings.TrimSpace(*in.HemisLogin)
			if login != old.HemisLogin.String {
				if err := q.SetUserHemisLogin(ctx, db.SetUserHemisLoginParams{ID: id, HemisLogin: pgtype.Text{String: login, Valid: login != ""}}); err != nil {
					return err
				}
				u.HemisLogin = pgtype.Text{String: login, Valid: login != ""}
			}
		}
		out = s.dto(ctx, q, u)
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "user", &id, s.dto(ctx, q, old), out))
	})
	return out, mapDBError(err)
}

// SetActive activates/deactivates a user. Deactivation ends all their sessions.
func (s *UserService) SetActive(ctx context.Context, actor Actor, id uuid.UUID, active bool) (UserDTO, error) {
	if actor.ID != nil && *actor.ID == id && !active {
		return UserDTO{}, apperr.New(http.StatusConflict, "self_deactivation", "O'zingizni o'chira olmaysiz")
	}
	var out UserDTO
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := q.GetUser(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		u, err := q.SetUserActive(ctx, db.SetUserActiveParams{ID: id, IsActive: active})
		if err != nil {
			return err
		}
		if !active {
			if err := q.DeleteUserSessions(ctx, id); err != nil {
				return err
			}
		}
		out = s.dto(ctx, q, u)
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "user", &id,
			map[string]any{"is_active": old.IsActive}, map[string]any{"is_active": active}))
	})
	return out, err
}

// RegeneratePassword sets a new random password (shown once), forces a change and ends sessions.
func (s *UserService) RegeneratePassword(ctx context.Context, actor Actor, id uuid.UUID) (string, error) {
	return s.SetGeneratedPassword(ctx, actor, id, true)
}

// SetGeneratedPassword sets a new random password (returned once) and ends the user's sessions.
// mustChange decides whether the user has to pick their own password at the next login.
func (s *UserService) SetGeneratedPassword(ctx context.Context, actor Actor, id uuid.UUID, mustChange bool) (string, error) {
	plain, err := auth.GeneratePassword()
	if err != nil {
		return "", err
	}
	hash, err := auth.HashPassword(plain)
	if err != nil {
		return "", err
	}
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		if _, err := q.GetUser(ctx, id); err != nil {
			return notFoundOr(err)
		}
		if err := q.SetUserPassword(ctx, db.SetUserPasswordParams{
			ID: id, PasswordHash: pgtype.Text{String: hash, Valid: true}, MustChangePassword: mustChange,
		}); err != nil {
			return err
		}
		if err := q.DeleteUserSessions(ctx, id); err != nil {
			return err
		}
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "user", &id, nil,
			map[string]any{"password_regenerated": true, "must_change_password": mustChange}))
	})
	return plain, err
}

// SetPermissions replaces what a little admin may use. Only little admins carry permissions.
func (s *UserService) SetPermissions(ctx context.Context, actor Actor, id uuid.UUID, requested []string) (UserDTO, error) {
	perms, err := permission.Normalize(requested)
	if err != nil {
		return UserDTO{}, apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"permissions": "Noto'g'ri ruxsat"})
	}
	var out UserDTO
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := q.GetUser(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		if old.Role != permission.RoleLittle {
			return apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"permissions": "Ruxsatlar faqat kichik administratorga beriladi"})
		}
		u, err := q.SetUserPermissions(ctx, db.SetUserPermissionsParams{ID: id, Permissions: perms})
		if err != nil {
			return err
		}
		out = s.dto(ctx, q, u)
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "user", &id,
			map[string]any{"permissions": permsOrEmpty(old.Permissions)}, map[string]any{"permissions": perms}))
	})
	return out, err
}

// EnsureManageable stops a little admin from touching staff accounts: they may manage students only.
// Admins may manage anyone.
func (s *UserService) EnsureManageable(ctx context.Context, actor Actor, id uuid.UUID) error {
	if actor.Role != permission.RoleLittle {
		return nil
	}
	u, err := db.New(s.pool).GetUser(ctx, id)
	if err != nil {
		return notFoundOr(err)
	}
	if u.Role != "student" {
		return apperr.Forbidden()
	}
	return nil
}
