package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/audit"
	"pc/internal/db"
)

type ScheduleService struct{ pool *pgxpool.Pool }

func NewScheduleService(pool *pgxpool.Pool) *ScheduleService { return &ScheduleService{pool: pool} }

type ScheduleInput struct {
	Name       string     `json:"name"`
	StartDate  string     `json:"start_date"`
	EndDate    string     `json:"end_date"`
	UserID     *uuid.UUID `json:"user_id"`
	GroupID    *uuid.UUID `json:"group_id"`
	LocationID *uuid.UUID `json:"location_id"`
	Monday     bool       `json:"monday"`
	Tuesday    bool       `json:"tuesday"`
	Wednesday  bool       `json:"wednesday"`
	Thursday   bool       `json:"thursday"`
	Friday     bool       `json:"friday"`
	Saturday   bool       `json:"saturday"`
}

func (in ScheduleInput) days() Weekdays {
	return Weekdays{in.Monday, in.Tuesday, in.Wednesday, in.Thursday, in.Friday, in.Saturday}
}

type ScheduleFilter struct {
	UserID  *uuid.UUID
	GroupID *uuid.UUID
}

func (s *ScheduleService) List(ctx context.Context, p ListParams, f ScheduleFilter) (Page[ScheduleDTO], error) {
	p = p.Norm()
	q := db.New(s.pool)
	rows, err := q.ListSchedules(ctx, db.ListSchedulesParams{
		UserID: nullUUID(f.UserID), GroupID: nullUUID(f.GroupID), Search: p.Search, Lim: p.Limit(), Off: p.Offset(),
	})
	if err != nil {
		return Page[ScheduleDTO]{}, err
	}
	total, err := q.CountSchedules(ctx, db.CountSchedulesParams{UserID: nullUUID(f.UserID), GroupID: nullUUID(f.GroupID), Search: p.Search})
	if err != nil {
		return Page[ScheduleDTO]{}, err
	}
	items := make([]ScheduleDTO, len(rows))
	for i, r := range rows {
		items[i] = scheduleDTO(db.Schedule{
			ID: r.ID, Name: r.Name, StartDate: r.StartDate, EndDate: r.EndDate, UserID: r.UserID, GroupID: r.GroupID,
			LocationID: r.LocationID, Monday: r.Monday, Tuesday: r.Tuesday, Wednesday: r.Wednesday,
			Thursday: r.Thursday, Friday: r.Friday, Saturday: r.Saturday,
		}, textPtr(r.GroupName), textPtr(r.UserName), textPtr(r.LocationName))
	}
	return newPage(items, total, p), nil
}

func (s *ScheduleService) dto(ctx context.Context, q *db.Queries, sc db.Schedule) (ScheduleDTO, error) {
	var group, user, loc *string
	if sc.GroupID.Valid {
		if g, err := q.GetGroup(ctx, sc.GroupID.UUID); err == nil {
			group = &g.Name
		}
	}
	if sc.UserID.Valid {
		if u, err := q.GetUser(ctx, sc.UserID.UUID); err == nil {
			user = &u.FullName
		}
	}
	if sc.LocationID.Valid {
		if l, err := q.GetLocation(ctx, sc.LocationID.UUID); err == nil {
			loc = &l.Name
		}
	}
	return scheduleDTO(sc, group, user, loc), nil
}

func (s *ScheduleService) Get(ctx context.Context, id uuid.UUID) (ScheduleDTO, error) {
	q := db.New(s.pool)
	sc, err := q.GetSchedule(ctx, id)
	if err != nil {
		return ScheduleDTO{}, notFoundOr(err)
	}
	return s.dto(ctx, q, sc)
}

// validate checks the input and the overlap rule. excludeID is uuid.Nil on create.
func (s *ScheduleService) validate(ctx context.Context, q *db.Queries, in ScheduleInput, excludeID uuid.UUID) (db.Schedule, error) {
	fields := map[string]string{}
	var out db.Schedule

	start, err1 := parseDate(in.StartDate)
	end, err2 := parseDate(in.EndDate)
	if err1 != nil {
		fields["start_date"] = "Sana noto'g'ri (YYYY-MM-DD)"
	}
	if err2 != nil {
		fields["end_date"] = "Sana noto'g'ri (YYYY-MM-DD)"
	}
	if err1 == nil && err2 == nil && end.Time.Before(start.Time) {
		fields["end_date"] = "Tugash sanasi boshlanish sanasidan oldin bo'lishi mumkin emas"
	}
	if (in.UserID == nil) == (in.GroupID == nil) {
		fields["target"] = "Guruh yoki talabadan birini tanlang"
	}
	if !in.days().Any() {
		fields["weekdays"] = "Kamida bitta hafta kunini tanlang"
	}
	if len(in.Name) > 200 {
		fields["name"] = "Nomi juda uzun"
	}
	if in.LocationID == nil {
		fields["location_id"] = "Joylashuvni tanlang"
	} else if _, err := q.GetLocation(ctx, *in.LocationID); err != nil {
		fields["location_id"] = "Joylashuv topilmadi"
	}
	if in.GroupID != nil {
		if _, err := q.GetGroup(ctx, *in.GroupID); err != nil {
			fields["group_id"] = "Guruh topilmadi"
		}
	}
	if in.UserID != nil {
		if u, err := q.GetUser(ctx, *in.UserID); err != nil || u.Role != "student" {
			fields["user_id"] = "Talaba topilmadi"
		}
	}
	if len(fields) > 0 {
		return out, apperr.Validation("Ma'lumotlar noto'g'ri", fields)
	}

	candidates, err := q.ListOverlapCandidates(ctx, db.ListOverlapCandidatesParams{
		ExcludeID: excludeID, UserID: nullUUID(in.UserID), GroupID: nullUUID(in.GroupID), StartDate: start, EndDate: end,
	})
	if err != nil {
		return out, err
	}
	mine := SchedulePeriod{Start: start.Time, End: end.Time, Days: in.days()}
	for _, c := range candidates {
		other := SchedulePeriod{Start: c.StartDate.Time, End: c.EndDate.Time,
			Days: Weekdays{c.Monday, c.Tuesday, c.Wednesday, c.Thursday, c.Friday, c.Saturday}}
		if Overlaps(mine, other) {
			return out, apperr.Validation(
				fmt.Sprintf("Jadval mavjud jadval bilan to'qnashadi (%s – %s)", dateStr(c.StartDate), dateStr(c.EndDate)),
				map[string]string{"start_date": "Mavjud jadval bilan to'qnashadi"})
		}
	}
	out = db.Schedule{
		Name: in.Name, StartDate: start, EndDate: end, UserID: nullUUID(in.UserID), GroupID: nullUUID(in.GroupID),
		LocationID: nullUUID(in.LocationID), Monday: in.Monday, Tuesday: in.Tuesday, Wednesday: in.Wednesday,
		Thursday: in.Thursday, Friday: in.Friday, Saturday: in.Saturday,
	}
	return out, nil
}

func (s *ScheduleService) Create(ctx context.Context, actor Actor, in ScheduleInput) (ScheduleDTO, error) {
	var out ScheduleDTO
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		v, err := s.validate(ctx, q, in, uuid.Nil)
		if err != nil {
			return err
		}
		sc, err := q.CreateSchedule(ctx, db.CreateScheduleParams{
			Name: v.Name, StartDate: v.StartDate, EndDate: v.EndDate, UserID: v.UserID, GroupID: v.GroupID, LocationID: v.LocationID,
			Monday: v.Monday, Tuesday: v.Tuesday, Wednesday: v.Wednesday, Thursday: v.Thursday, Friday: v.Friday, Saturday: v.Saturday,
		})
		if err != nil {
			return err
		}
		if out, err = s.dto(ctx, q, sc); err != nil {
			return err
		}
		return audit.Write(ctx, tx, actor.entry(audit.ActionCreate, "schedule", &sc.ID, nil, out))
	})
	return out, mapDBError(err)
}

func (s *ScheduleService) Update(ctx context.Context, actor Actor, id uuid.UUID, in ScheduleInput) (ScheduleDTO, error) {
	var out ScheduleDTO
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := q.GetSchedule(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		before, _ := s.dto(ctx, q, old)
		v, err := s.validate(ctx, q, in, id)
		if err != nil {
			return err
		}
		sc, err := q.UpdateSchedule(ctx, db.UpdateScheduleParams{
			ID: id, Name: v.Name, StartDate: v.StartDate, EndDate: v.EndDate, UserID: v.UserID, GroupID: v.GroupID, LocationID: v.LocationID,
			Monday: v.Monday, Tuesday: v.Tuesday, Wednesday: v.Wednesday, Thursday: v.Thursday, Friday: v.Friday, Saturday: v.Saturday,
		})
		if err != nil {
			return err
		}
		if out, err = s.dto(ctx, q, sc); err != nil {
			return err
		}
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "schedule", &id, before, out))
	})
	return out, mapDBError(err)
}

func (s *ScheduleService) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := q.GetSchedule(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		before, _ := s.dto(ctx, q, old)
		if err := q.DeleteSchedule(ctx, id); err != nil {
			return err
		}
		return audit.Write(ctx, tx, actor.entry(audit.ActionDelete, "schedule", &id, before, nil))
	})
	return mapDBError(err)
}
