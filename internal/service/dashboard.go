package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/db"
)

type DashboardService struct {
	pool *pgxpool.Pool
	loc  *time.Location
}

func NewDashboardService(pool *pgxpool.Pool, loc *time.Location) *DashboardService {
	return &DashboardService{pool: pool, loc: loc}
}

type DayStats struct {
	Date     string `json:"date"`
	Expected int64  `json:"expected"`
	Full     int64  `json:"full"`
	Partial  int64  `json:"partial"`
	Absent   int64  `json:"absent"`
}

type Dashboard struct {
	Students int64      `json:"students"`
	Teachers int64      `json:"teachers"`
	Groups   int64      `json:"groups"`
	Today    DayStats   `json:"today"`
	Last7    []DayStats `json:"last_7_days"` // oldest first, ends with today
}

func (s *DashboardService) dayStats(ctx context.Context, q *db.Queries, day time.Time) (DayStats, error) {
	r, err := q.DayAttendanceStats(ctx, pgtype.Date{Time: day, Valid: true})
	if err != nil {
		return DayStats{}, err
	}
	return DayStats{Date: day.Format(DateLayout), Expected: r.Expected, Full: r.FullCount, Partial: r.PartialCount,
		Absent: r.Expected - r.FullCount - r.PartialCount}, nil
}

func (s *DashboardService) Get(ctx context.Context) (Dashboard, error) {
	q := db.New(s.pool)
	var d Dashboard
	var err error
	if d.Students, err = q.CountUsersByRole(ctx, "student"); err != nil {
		return d, err
	}
	if d.Teachers, err = q.CountUsersByRole(ctx, "teacher"); err != nil {
		return d, err
	}
	if d.Groups, err = q.CountGroups(ctx, db.CountGroupsParams{}); err != nil {
		return d, err
	}
	now := time.Now().In(s.loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for i := 6; i >= 0; i-- {
		st, err := s.dayStats(ctx, q, today.AddDate(0, 0, -i))
		if err != nil {
			return d, err
		}
		d.Last7 = append(d.Last7, st)
	}
	d.Today = d.Last7[6]
	return d, nil
}
