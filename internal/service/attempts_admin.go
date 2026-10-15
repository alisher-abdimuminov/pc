package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/audit"
	"pc/internal/db"
	"pc/internal/storage"
)

// AttemptsAdminService lists attendance attempts across all students (admin "Urinishlar" page).
type AttemptsAdminService struct {
	pool  *pgxpool.Pool
	loc   *time.Location
	store storage.Storage
}

// WithStorage lets Delete remove the attempt's selfie file too.
func (s *AttemptsAdminService) WithStorage(st storage.Storage) *AttemptsAdminService {
	s.store = st
	return s
}

func NewAttemptsAdminService(pool *pgxpool.Pool, loc *time.Location) *AttemptsAdminService {
	return &AttemptsAdminService{pool: pool, loc: loc}
}

type AttemptFilter struct {
	GroupID   *uuid.UUID
	StudentID *uuid.UUID
	Success   *bool
	// Kind narrows to one failure type: location | face | liveness | no_face.
	Kind     string
	From, To string // YYYY-MM-DD inclusive, business timezone
}

type AdminAttemptDTO struct {
	ID               uuid.UUID  `json:"id"`
	AttemptedAt      time.Time  `json:"attempted_at"`
	Date             string     `json:"date"`
	Shift            int        `json:"shift"`
	Step             int        `json:"step"`
	StudentID        uuid.UUID  `json:"student_id"`
	StudentName      string     `json:"student_name"`
	StudentUsername  string     `json:"student_username"`
	StudentActive    bool       `json:"student_active"`
	GroupID          *uuid.UUID `json:"group_id"`
	GroupName        *string    `json:"group_name"`
	FaceVerified     bool       `json:"face_verified"`
	LocationVerified bool       `json:"location_verified"`
	LivenessVerified bool       `json:"liveness_verified"`
	Success          bool       `json:"success"`
	Error            string     `json:"error"`
	FaceDistance     *float64   `json:"face_distance"`
	FaceThreshold    *float64   `json:"face_threshold"`
	AntispoofScore   *float64   `json:"antispoof_score"`
	GPSAccuracy      float64    `json:"gps_accuracy"`
	Latitude         float64    `json:"latitude"`
	Longitude        float64    `json:"longitude"`
	IP               string     `json:"ip"`
	HasSelfie        bool       `json:"has_selfie"`
}

var validKinds = map[string]bool{"": true, "location": true, "face": true, "liveness": true, "no_face": true}

func (s *AttemptsAdminService) bound(v string, plusDays int, field string) (pgtype.Timestamptz, error) {
	if v == "" {
		return pgtype.Timestamptz{}, nil
	}
	t, err := time.ParseInLocation(DateLayout, v, s.loc)
	if err != nil {
		return pgtype.Timestamptz{}, fieldErr(field, "Sana noto'g'ri (YYYY-MM-DD)")
	}
	return pgtype.Timestamptz{Time: t.AddDate(0, 0, plusDays), Valid: true}, nil
}

func (s *AttemptsAdminService) List(ctx context.Context, p ListParams, f AttemptFilter) (Page[AdminAttemptDTO], error) {
	p = p.Norm()
	if !validKinds[f.Kind] {
		return Page[AdminAttemptDTO]{}, apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"kind": "Noto'g'ri tur"})
	}
	from, err := s.bound(f.From, 0, "from")
	if err != nil {
		return Page[AdminAttemptDTO]{}, err
	}
	to, err := s.bound(f.To, 1, "to")
	if err != nil {
		return Page[AdminAttemptDTO]{}, err
	}
	success := pgtype.Bool{}
	if f.Success != nil {
		success = pgtype.Bool{Bool: *f.Success, Valid: true}
	}
	q := db.New(s.pool)
	rows, err := q.ListAttemptsAdmin(ctx, db.ListAttemptsAdminParams{
		Search: p.Search, GroupID: nullUUID(f.GroupID), StudentID: nullUUID(f.StudentID), Success: success,
		FromTs: from, ToTs: to, Kind: f.Kind, Lim: p.Limit(), Off: p.Offset(),
	})
	if err != nil {
		return Page[AdminAttemptDTO]{}, err
	}
	total, err := q.CountAttemptsAdmin(ctx, db.CountAttemptsAdminParams{
		Search: p.Search, GroupID: nullUUID(f.GroupID), StudentID: nullUUID(f.StudentID), Success: success,
		FromTs: from, ToTs: to, Kind: f.Kind,
	})
	if err != nil {
		return Page[AdminAttemptDTO]{}, err
	}
	items := make([]AdminAttemptDTO, len(rows))
	for i, r := range rows {
		d := AdminAttemptDTO{
			ID: r.ID, AttemptedAt: r.AttemptedAt.Time, Date: dateStr(r.Date), Shift: int(r.Shift), Step: int(r.Step),
			StudentID: r.StudentID, StudentName: r.StudentName, StudentUsername: r.StudentUsername.String, StudentActive: r.StudentActive,
			GroupID: uuidPtr(r.GroupID), GroupName: textPtr(r.GroupName),
			FaceVerified: r.FaceVerified, LocationVerified: r.LocationVerified, LivenessVerified: r.LivenessVerified,
			Success: r.Success, Error: r.Error, FaceDistance: floatPtr(r.FaceDistance), FaceThreshold: floatPtr(r.FaceThreshold),
			AntispoofScore: floatPtr(r.AntispoofScore), GPSAccuracy: r.GpsAccuracy, Latitude: r.Latitude, Longitude: r.Longitude,
			HasSelfie: r.Image != "",
		}
		if r.IpAddress != nil {
			d.IP = r.IpAddress.String()
		}
		items[i] = d
	}
	return newPage(items, total, p), nil
}

// Delete removes one attempt and its selfie. If it was the step's only successful attempt, the step
// goes back to unsuccessful so the daily status stays truthful. The audit entry is written in the
// same transaction; the file is removed after the commit (a leftover file is harmless).
func (s *AttemptsAdminService) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	var image string
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		a, err := q.GetAttemptForDelete(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		image = a.Image
		if err := q.DeleteAttempt(ctx, id); err != nil {
			return err
		}
		if err := q.RefreshStepSuccess(ctx, a.StepID); err != nil {
			return err
		}
		before := map[string]any{"student_id": a.StudentID, "date": dateStr(a.Date), "step": a.Step, "success": a.Success, "attempted_at": a.AttemptedAt.Time}
		return audit.Write(ctx, tx, actor.entry(audit.ActionDelete, "attendance_attempt", &id, before, nil))
	})
	if err != nil {
		return mapDBError(err)
	}
	if image != "" && s.store != nil {
		_ = s.store.Delete(ctx, image)
	}
	return nil
}
