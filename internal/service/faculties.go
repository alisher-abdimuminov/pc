package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/audit"
	"pc/internal/db"
)

type FacultyService struct{ pool *pgxpool.Pool }

func NewFacultyService(pool *pgxpool.Pool) *FacultyService { return &FacultyService{pool: pool} }

func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fieldErr("name", "Majburiy maydon")
	}
	if len([]rune(name)) > 200 {
		return "", fieldErr("name", "Nomi juda uzun")
	}
	return name, nil
}

func (s *FacultyService) List(ctx context.Context, p ListParams) (Page[FacultyDTO], error) {
	p = p.Norm()
	q := db.New(s.pool)
	rows, err := q.ListFacultiesAdmin(ctx, db.ListFacultiesAdminParams{Search: p.Search, Lim: p.Limit(), Off: p.Offset()})
	if err != nil {
		return Page[FacultyDTO]{}, err
	}
	total, err := q.CountFaculties(ctx, p.Search)
	if err != nil {
		return Page[FacultyDTO]{}, err
	}
	items := make([]FacultyDTO, len(rows))
	for i, r := range rows {
		items[i] = facultyDTO(db.Faculty{ID: r.ID, Name: r.Name, HemisID: r.HemisID, CreatedAt: r.CreatedAt})
		items[i].StudentCount = r.StudentCount
	}
	return newPage(items, total, p), nil
}

func (s *FacultyService) Get(ctx context.Context, id uuid.UUID) (FacultyDTO, error) {
	f, err := db.New(s.pool).GetFaculty(ctx, id)
	if err != nil {
		return FacultyDTO{}, notFoundOr(err)
	}
	return facultyDTO(f), nil
}

func (s *FacultyService) Create(ctx context.Context, actor Actor, name string) (FacultyDTO, error) {
	name, err := validateName(name)
	if err != nil {
		return FacultyDTO{}, err
	}
	var out FacultyDTO
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		f, err := q.CreateFaculty(ctx, db.CreateFacultyParams{Name: name})
		if err != nil {
			return err
		}
		out = facultyDTO(f)
		return audit.Write(ctx, tx, actor.entry(audit.ActionCreate, "faculty", &f.ID, nil, out))
	})
	return out, mapDBError(err)
}

func (s *FacultyService) Update(ctx context.Context, actor Actor, id uuid.UUID, name string) (FacultyDTO, error) {
	name, err := validateName(name)
	if err != nil {
		return FacultyDTO{}, err
	}
	var out FacultyDTO
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := q.GetFaculty(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		f, err := q.UpdateFaculty(ctx, db.UpdateFacultyParams{ID: id, Name: name})
		if err != nil {
			return err
		}
		out = facultyDTO(f)
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "faculty", &id, facultyDTO(old), out))
	})
	return out, mapDBError(err)
}

func (s *FacultyService) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := q.GetFaculty(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		if err := q.DeleteFaculty(ctx, id); err != nil {
			return err
		}
		return audit.Write(ctx, tx, actor.entry(audit.ActionDelete, "faculty", &id, facultyDTO(old), nil))
	})
	return mapDBError(err)
}
