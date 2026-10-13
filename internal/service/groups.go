package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/audit"
	"pc/internal/db"
)

type GroupService struct{ pool *pgxpool.Pool }

func NewGroupService(pool *pgxpool.Pool) *GroupService { return &GroupService{pool: pool} }

type GroupInput struct {
	Name      string     `json:"name"`
	FacultyID *uuid.UUID `json:"faculty_id"`
	TeacherID *uuid.UUID `json:"teacher_id"`
}

type GroupDetail struct {
	GroupDTO
	Students []UserDTO `json:"students"`
}

func (s *GroupService) List(ctx context.Context, p ListParams, facultyID *uuid.UUID) (Page[GroupDTO], error) {
	p = p.Norm()
	q := db.New(s.pool)
	rows, err := q.ListGroupsAdmin(ctx, db.ListGroupsAdminParams{
		Search: p.Search, FacultyID: nullUUID(facultyID), Lim: p.Limit(), Off: p.Offset(),
	})
	if err != nil {
		return Page[GroupDTO]{}, err
	}
	total, err := q.CountGroups(ctx, db.CountGroupsParams{Search: p.Search, FacultyID: nullUUID(facultyID)})
	if err != nil {
		return Page[GroupDTO]{}, err
	}
	items := make([]GroupDTO, len(rows))
	for i, r := range rows {
		items[i] = GroupDTO{ID: r.ID, Name: r.Name, FacultyID: uuidPtr(r.FacultyID), FacultyName: textPtr(r.FacultyName),
			TeacherID: uuidPtr(r.TeacherID), TeacherName: textPtr(r.TeacherName), StudentCount: r.StudentCount, CreatedAt: r.CreatedAt.Time}
	}
	return newPage(items, total, p), nil
}

func groupAdminDTO(r db.GetGroupAdminRow) GroupDTO {
	return GroupDTO{ID: r.ID, Name: r.Name, FacultyID: uuidPtr(r.FacultyID), FacultyName: textPtr(r.FacultyName),
		TeacherID: uuidPtr(r.TeacherID), TeacherName: textPtr(r.TeacherName), StudentCount: r.StudentCount, CreatedAt: r.CreatedAt.Time}
}

func (s *GroupService) Get(ctx context.Context, id uuid.UUID) (GroupDetail, error) {
	q := db.New(s.pool)
	g, err := q.GetGroupAdmin(ctx, id)
	if err != nil {
		return GroupDetail{}, notFoundOr(err)
	}
	students, err := q.ListGroupStudents(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return GroupDetail{}, err
	}
	d := GroupDetail{GroupDTO: groupAdminDTO(g), Students: make([]UserDTO, len(students))}
	for i, u := range students {
		d.Students[i] = userDTO(u, &g.Name, textPtr(g.FacultyName))
	}
	return d, nil
}

// validate checks the name and that the referenced faculty/teacher exist.
func (s *GroupService) validate(ctx context.Context, q *db.Queries, in GroupInput) (string, error) {
	name, err := validateName(in.Name)
	if err != nil {
		return "", err
	}
	if in.FacultyID != nil {
		if _, err := q.GetFaculty(ctx, *in.FacultyID); err != nil {
			return "", fieldErr("faculty_id", "Fakultet topilmadi")
		}
	}
	if in.TeacherID != nil {
		t, err := q.GetUser(ctx, *in.TeacherID)
		if err != nil || t.Role != "teacher" || !t.IsActive {
			return "", fieldErr("teacher_id", "O'qituvchi topilmadi")
		}
	}
	return name, nil
}

func (s *GroupService) snapshot(ctx context.Context, q *db.Queries, id uuid.UUID) (GroupDTO, error) {
	g, err := q.GetGroupAdmin(ctx, id)
	if err != nil {
		return GroupDTO{}, notFoundOr(err)
	}
	return groupAdminDTO(g), nil
}

func (s *GroupService) Create(ctx context.Context, actor Actor, in GroupInput) (GroupDTO, error) {
	var out GroupDTO
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		name, err := s.validate(ctx, q, in)
		if err != nil {
			return err
		}
		g, err := q.CreateGroup(ctx, db.CreateGroupParams{Name: name, FacultyID: nullUUID(in.FacultyID), TeacherID: nullUUID(in.TeacherID)})
		if err != nil {
			return err
		}
		if out, err = s.snapshot(ctx, q, g.ID); err != nil {
			return err
		}
		return audit.Write(ctx, tx, actor.entry(audit.ActionCreate, "group", &g.ID, nil, out))
	})
	return out, mapDBError(err)
}

func (s *GroupService) Update(ctx context.Context, actor Actor, id uuid.UUID, in GroupInput) (GroupDTO, error) {
	var out GroupDTO
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		before, err := s.snapshot(ctx, q, id)
		if err != nil {
			return err
		}
		name, err := s.validate(ctx, q, in)
		if err != nil {
			return err
		}
		if _, err := q.UpdateGroup(ctx, db.UpdateGroupParams{ID: id, Name: name, FacultyID: nullUUID(in.FacultyID), TeacherID: nullUUID(in.TeacherID)}); err != nil {
			return err
		}
		if out, err = s.snapshot(ctx, q, id); err != nil {
			return err
		}
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "group", &id, before, out))
	})
	return out, mapDBError(err)
}

func (s *GroupService) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		before, err := s.snapshot(ctx, q, id)
		if err != nil {
			return err
		}
		if err := q.DeleteGroup(ctx, id); err != nil {
			return err
		}
		return audit.Write(ctx, tx, actor.entry(audit.ActionDelete, "group", &id, before, nil))
	})
	return mapDBError(err)
}
