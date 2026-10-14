package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"pc/internal/apperr"
	"pc/internal/db"
)

const DateLayout = "2006-01-02"

// ListParams are the common list query parameters.
type ListParams struct {
	Page     int
	PageSize int
	Search   string
}

func (p ListParams) Norm() ListParams {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p
}

func (p ListParams) Limit() int32  { return int32(p.PageSize) }
func (p ListParams) Offset() int32 { return int32((p.Page - 1) * p.PageSize) }

type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func newPage[T any](items []T, total int64, p ListParams) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Total: total, Page: p.Page, PageSize: p.PageSize}
}

// ---- error helpers ----

func notFoundOr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.NotFound()
	}
	return err
}

var errInUse = apperr.New(http.StatusConflict, "in_use", "Bu yozuvga bog'liq ma'lumotlar mavjud, o'chirib bo'lmaydi")

// mapDBError converts constraint violations into API errors.
func mapDBError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign key
			return errInUse
		case "23505": // unique
			if pgErr.ConstraintName == "users_hemis_login_key" {
				return fieldErr("hemis_login", "Bu HEMIS login boshqa foydalanuvchiga ulangan")
			}
			if pgErr.ConstraintName == "locations_name_key" {
				return fieldErr("name", "Bu nomli joylashuv allaqachon mavjud")
			}
			return apperr.Validation("Bunday yozuv allaqachon mavjud", nil)
		}
	}
	return err
}

func fieldErr(field, msg string) error {
	return apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{field: msg})
}

// ---- value helpers ----

func uuidPtr(n uuid.NullUUID) *uuid.UUID {
	if !n.Valid {
		return nil
	}
	return &n.UUID
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func tsPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func dateStr(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format(DateLayout)
}

func parseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func nullUUIDValue() uuid.NullUUID { return uuid.NullUUID{} }

func uuidNull(id uuid.UUID) uuid.NullUUID { return uuid.NullUUID{UUID: id, Valid: true} }

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

// ---- DTOs (never expose db models directly: they carry password_hash etc.) ----

type FacultyDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	HemisID   *int64    `json:"hemis_id"`
	CreatedAt time.Time `json:"created_at"`
	// StudentCount is filled by List only.
	StudentCount int64 `json:"student_count"`
}

func facultyDTO(f db.Faculty) FacultyDTO {
	d := FacultyDTO{ID: f.ID, Name: f.Name, CreatedAt: f.CreatedAt.Time}
	if f.HemisID.Valid {
		d.HemisID = &f.HemisID.Int64
	}
	return d
}

type GroupDTO struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	FacultyID    *uuid.UUID `json:"faculty_id"`
	FacultyName  *string    `json:"faculty_name"`
	TeacherID    *uuid.UUID `json:"teacher_id"`
	TeacherName  *string    `json:"teacher_name"`
	StudentCount int64      `json:"student_count"`
	CreatedAt    time.Time  `json:"created_at"`
}

type UserDTO struct {
	ID                 uuid.UUID  `json:"id"`
	Username           string     `json:"username"`
	Role               string     `json:"role"`
	IsActive           bool       `json:"is_active"`
	MustChangePassword bool       `json:"must_change_password"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	HemisLogin         string     `json:"hemis_login"`
	HemisSyncedAt      *time.Time `json:"hemis_synced_at"`
	// Permissions are the sections a "little" admin may use (empty for every other role).
	Permissions  []string   `json:"permissions"`
	FirstName    string     `json:"first_name"`
	SecondName   string     `json:"second_name"`
	ThirdName    string     `json:"third_name"`
	FullName     string     `json:"full_name"`
	ShortName    string     `json:"short_name"`
	Phone        string     `json:"phone"`
	Gender       string     `json:"gender"`
	BirthDate    string     `json:"birth_date"`
	Level        string     `json:"level"`
	Semester     string     `json:"semester"`
	Address      string     `json:"address"`
	HasPhoto     bool       `json:"has_photo"`
	FaceEnrolled bool       `json:"face_enrolled"`
	GroupID      *uuid.UUID `json:"group_id"`
	GroupName    *string    `json:"group_name"`
	FacultyID    *uuid.UUID `json:"faculty_id"`
	FacultyName  *string    `json:"faculty_name"`
	CreatedAt    time.Time  `json:"created_at"`
}

func userDTO(u db.User, groupName, facultyName *string) UserDTO {
	return UserDTO{
		ID: u.ID, Username: u.Username.String, Role: u.Role, IsActive: u.IsActive,
		MustChangePassword: u.MustChangePassword, LastLoginAt: tsPtr(u.LastLoginAt),
		HemisLogin: u.HemisLogin.String, HemisSyncedAt: tsPtr(u.HemisSyncedAt), Permissions: permsOrEmpty(u.Permissions),
		FirstName: u.FirstName, SecondName: u.SecondName, ThirdName: u.ThirdName,
		FullName: u.FullName, ShortName: u.ShortName, Phone: u.Phone, Gender: u.Gender,
		BirthDate: u.BirthDate, Level: u.Level, Semester: u.Semester, Address: u.Address,
		HasPhoto: u.Image != "", FaceEnrolled: u.FaceEnrolledAt.Valid,
		GroupID: uuidPtr(u.GroupID), GroupName: groupName,
		FacultyID: uuidPtr(u.FacultyID), FacultyName: facultyName, CreatedAt: u.CreatedAt.Time,
	}
}

type LocationDTO struct {
	ID              uuid.UUID    `json:"id"`
	Name            string       `json:"name"`
	Polygon         [][2]float64 `json:"polygon"`
	ToleranceMeters int32        `json:"tolerance_meters"`
	LocationURL     string       `json:"location_url"`
	CreatedAt       time.Time    `json:"created_at"`
}

func locationDTO(l db.Location) LocationDTO {
	var poly [][2]float64
	_ = json.Unmarshal(l.Polygon, &poly)
	return LocationDTO{ID: l.ID, Name: l.Name, Polygon: poly, ToleranceMeters: l.ToleranceMeters,
		LocationURL: l.LocationUrl, CreatedAt: l.CreatedAt.Time}
}

type ScheduleDTO struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	StartDate    string     `json:"start_date"`
	EndDate      string     `json:"end_date"`
	UserID       *uuid.UUID `json:"user_id"`
	UserName     *string    `json:"user_name"`
	GroupID      *uuid.UUID `json:"group_id"`
	GroupName    *string    `json:"group_name"`
	LocationID   *uuid.UUID `json:"location_id"`
	LocationName *string    `json:"location_name"`
	Monday       bool       `json:"monday"`
	Tuesday      bool       `json:"tuesday"`
	Wednesday    bool       `json:"wednesday"`
	Thursday     bool       `json:"thursday"`
	Friday       bool       `json:"friday"`
	Saturday     bool       `json:"saturday"`
}

func scheduleDTO(s db.Schedule, group, user, loc *string) ScheduleDTO {
	return ScheduleDTO{
		ID: s.ID, Name: s.Name, StartDate: dateStr(s.StartDate), EndDate: dateStr(s.EndDate),
		UserID: uuidPtr(s.UserID), UserName: user, GroupID: uuidPtr(s.GroupID), GroupName: group,
		LocationID: uuidPtr(s.LocationID), LocationName: loc,
		Monday: s.Monday, Tuesday: s.Tuesday, Wednesday: s.Wednesday,
		Thursday: s.Thursday, Friday: s.Friday, Saturday: s.Saturday,
	}
}

type AuditDTO struct {
	ID            uuid.UUID       `json:"id"`
	ActorID       *uuid.UUID      `json:"actor_id"`
	ActorRole     string          `json:"actor_role"`
	ActorUsername *string         `json:"actor_username"`
	ActorName     *string         `json:"actor_name"`
	Action        string          `json:"action"`
	EntityType    string          `json:"entity_type"`
	EntityID      *uuid.UUID      `json:"entity_id"`
	Changes       json.RawMessage `json:"changes"`
	IP            string          `json:"ip"`
	UserAgent     string          `json:"user_agent"`
	CreatedAt     time.Time       `json:"created_at"`
}

func permsOrEmpty(p []string) []string {
	if p == nil {
		return []string{}
	}
	return p
}
