package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/db"
)

// OptionService serves the unpaginated id + name lists that select boxes need.
type OptionService struct{ pool *pgxpool.Pool }

func NewOptionService(pool *pgxpool.Pool) *OptionService { return &OptionService{pool: pool} }

type OptionDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (s *OptionService) Faculties(ctx context.Context) ([]OptionDTO, error) {
	rows, err := db.New(s.pool).ListFacultyOptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OptionDTO, len(rows))
	for i, r := range rows {
		out[i] = OptionDTO{ID: r.ID, Name: r.Name}
	}
	return out, nil
}

func (s *OptionService) Groups(ctx context.Context) ([]OptionDTO, error) {
	rows, err := db.New(s.pool).ListGroupOptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OptionDTO, len(rows))
	for i, r := range rows {
		out[i] = OptionDTO{ID: r.ID, Name: r.Name}
	}
	return out, nil
}

func (s *OptionService) Locations(ctx context.Context) ([]OptionDTO, error) {
	rows, err := db.New(s.pool).ListLocationOptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OptionDTO, len(rows))
	for i, r := range rows {
		out[i] = OptionDTO{ID: r.ID, Name: r.Name}
	}
	return out, nil
}

func (s *OptionService) Schedules(ctx context.Context) ([]OptionDTO, error) {
	rows, err := db.New(s.pool).ListScheduleOptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OptionDTO, len(rows))
	for i, r := range rows {
		out[i] = OptionDTO{ID: r.ID, Name: r.Name}
	}
	return out, nil
}
