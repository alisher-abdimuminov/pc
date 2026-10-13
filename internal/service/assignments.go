package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
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

type AssignmentService struct {
	pool  *pgxpool.Pool
	store storage.Storage
	loc   *time.Location
}

func NewAssignmentService(pool *pgxpool.Pool, store storage.Storage, loc *time.Location) *AssignmentService {
	return &AssignmentService{pool: pool, store: store, loc: loc}
}

// Upload is an incoming file (already size-limited by the transport).
type Upload struct {
	Filename string
	Size     int64
	Reader   io.Reader
}

type AssignmentInput struct {
	GroupID     *uuid.UUID
	Title       string
	Description string
	Deadline    string // RFC3339, or "2006-01-02T15:04" (interpreted in the business timezone)
	File        *Upload
	RemoveFile  bool
}

type AssignmentDTO struct {
	ID              uuid.UUID `json:"id"`
	GroupID         uuid.UUID `json:"group_id"`
	GroupName       string    `json:"group_name"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	HasFile         bool      `json:"has_file"`
	Deadline        time.Time `json:"deadline"`
	SubmissionCount int64     `json:"submission_count"`
	StudentCount    int64     `json:"student_count"`
	CreatedAt       time.Time `json:"created_at"`
}

func (s *AssignmentService) parseDeadline(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, v, s.loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("bad deadline")
}

func (s *AssignmentService) saveUpload(ctx context.Context, kind string, up *Upload) (string, error) {
	if up.Size > storage.MaxSize(kind) {
		return "", apperr.Validation("Fayl juda katta", map[string]string{"file": "Fayl hajmi 20 MB dan oshmasligi kerak"})
	}
	br := bufio.NewReader(up.Reader)
	head, _ := br.Peek(512)
	ext, err := storage.Validate(kind, up.Filename, head)
	if err != nil {
		return "", apperr.Validation("Fayl turi qo'llab-quvvatlanmaydi", map[string]string{"file": "Ruxsat etilgan: pdf, doc, docx, xls, xlsx, ppt, pptx, zip, rar, jpg, png"})
	}
	return s.store.Save(ctx, kind, ext, io.LimitReader(br, storage.MaxSize(kind)+1))
}

func (s *AssignmentService) toDTO(a db.Assignment, groupName string, subs, students int64) AssignmentDTO {
	return AssignmentDTO{ID: a.ID, GroupID: a.GroupID, GroupName: groupName, Title: a.Title, Description: a.Description,
		HasFile: a.File != "", Deadline: a.Deadline.Time, SubmissionCount: subs, StudentCount: students, CreatedAt: a.CreatedAt.Time}
}

func (s *AssignmentService) snapshot(ctx context.Context, q *db.Queries, a db.Assignment) AssignmentDTO {
	name := ""
	if g, err := q.GetGroup(ctx, a.GroupID); err == nil {
		name = g.Name
	}
	n, _ := q.CountAssignmentSubmissions(ctx, a.ID)
	return s.toDTO(a, name, n, 0)
}

func (s *AssignmentService) List(ctx context.Context, teacherID uuid.UUID, p ListParams, groupID *uuid.UUID) (Page[AssignmentDTO], error) {
	p = p.Norm()
	q := db.New(s.pool)
	tid := teacherID
	rows, err := q.ListTeacherAssignments(ctx, db.ListTeacherAssignmentsParams{TeacherID: tid, GroupID: nullUUID(groupID), Search: p.Search, Lim: p.Limit(), Off: p.Offset()})
	if err != nil {
		return Page[AssignmentDTO]{}, err
	}
	total, err := q.CountTeacherAssignments(ctx, db.CountTeacherAssignmentsParams{TeacherID: tid, GroupID: nullUUID(groupID), Search: p.Search})
	if err != nil {
		return Page[AssignmentDTO]{}, err
	}
	items := make([]AssignmentDTO, len(rows))
	for i, r := range rows {
		items[i] = s.toDTO(db.Assignment{ID: r.ID, TeacherID: r.TeacherID, GroupID: r.GroupID, Title: r.Title, Description: r.Description,
			File: r.File, Deadline: r.Deadline, CreatedAt: r.CreatedAt}, r.GroupName, r.SubmissionCount, r.StudentCount)
	}
	return newPage(items, total, p), nil
}

// owned loads an assignment of one of the teacher's groups, or 404.
func (s *AssignmentService) owned(ctx context.Context, q *db.Queries, teacherID, id uuid.UUID) (db.Assignment, error) {
	a, err := q.GetAssignmentForTeacher(ctx, db.GetAssignmentForTeacherParams{ID: id, TeacherID: uuid.NullUUID{UUID: teacherID, Valid: true}})
	return a, notFoundOr(err)
}

func (s *AssignmentService) Get(ctx context.Context, teacherID, id uuid.UUID) (AssignmentDTO, error) {
	q := db.New(s.pool)
	a, err := s.owned(ctx, q, teacherID, id)
	if err != nil {
		return AssignmentDTO{}, err
	}
	d := s.snapshot(ctx, q, a)
	d.StudentCount = int64(len(mustRoster(ctx, q, a)))
	return d, nil
}

func mustRoster(ctx context.Context, q *db.Queries, a db.Assignment) []db.ListAssignmentRosterRow {
	r, _ := q.ListAssignmentRoster(ctx, db.ListAssignmentRosterParams{AssignmentID: a.ID, GroupID: a.GroupID})
	return r
}

func (in AssignmentInput) validate(loc func(string) (time.Time, error)) (string, time.Time, error) {
	fields := map[string]string{}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		fields["title"] = "Majburiy maydon"
	} else if len([]rune(title)) > 200 {
		fields["title"] = "Sarlavha juda uzun"
	}
	if len([]rune(in.Description)) > 5000 {
		fields["description"] = "Tavsif juda uzun"
	}
	deadline, err := loc(in.Deadline)
	if err != nil {
		fields["deadline"] = "Muddat noto'g'ri"
	}
	if len(fields) > 0 {
		return "", time.Time{}, apperr.Validation("Ma'lumotlar noto'g'ri", fields)
	}
	return title, deadline, nil
}

func (s *AssignmentService) Create(ctx context.Context, actor Actor, teacherID uuid.UUID, in AssignmentInput) (AssignmentDTO, error) {
	title, deadline, err := in.validate(s.parseDeadline)
	if err != nil {
		return AssignmentDTO{}, err
	}
	if in.GroupID == nil {
		return AssignmentDTO{}, fieldErr("group_id", "Guruhni tanlang")
	}
	q := db.New(s.pool)
	if _, err := q.GetGroupForTeacher(ctx, db.GetGroupForTeacherParams{ID: *in.GroupID, TeacherID: uuid.NullUUID{UUID: teacherID, Valid: true}}); err != nil {
		return AssignmentDTO{}, fieldErr("group_id", "Guruh topilmadi")
	}
	path := ""
	if in.File != nil {
		if path, err = s.saveUpload(ctx, storage.KindAssignments, in.File); err != nil {
			return AssignmentDTO{}, err
		}
	}
	var out AssignmentDTO
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		a, err := q.CreateAssignment(ctx, db.CreateAssignmentParams{TeacherID: teacherID, GroupID: *in.GroupID, Title: title,
			Description: strings.TrimSpace(in.Description), File: path, Deadline: pgtype.Timestamptz{Time: deadline, Valid: true}})
		if err != nil {
			return err
		}
		out = s.snapshot(ctx, q, a)
		return audit.Write(ctx, tx, actor.entry(audit.ActionCreate, "assignment", &a.ID, nil, out))
	})
	if err != nil {
		if path != "" {
			_ = s.store.Delete(ctx, path)
		}
		return AssignmentDTO{}, mapDBError(err)
	}
	return out, nil
}

func (s *AssignmentService) Update(ctx context.Context, actor Actor, teacherID, id uuid.UUID, in AssignmentInput) (AssignmentDTO, error) {
	title, deadline, err := in.validate(s.parseDeadline)
	if err != nil {
		return AssignmentDTO{}, err
	}
	newPath := ""
	var oldPath string
	var out AssignmentDTO
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := s.owned(ctx, q, teacherID, id)
		if err != nil {
			return err
		}
		before := s.snapshot(ctx, q, old)
		path := old.File
		switch {
		case in.File != nil:
			if newPath, err = s.saveUpload(ctx, storage.KindAssignments, in.File); err != nil {
				return err
			}
			path, oldPath = newPath, old.File
		case in.RemoveFile:
			path, oldPath = "", old.File
		}
		a, err := q.UpdateAssignment(ctx, db.UpdateAssignmentParams{ID: id, Title: title, Description: strings.TrimSpace(in.Description),
			File: path, Deadline: pgtype.Timestamptz{Time: deadline, Valid: true}})
		if err != nil {
			return err
		}
		out = s.snapshot(ctx, q, a)
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "assignment", &id, before, out))
	})
	if err != nil {
		if newPath != "" {
			_ = s.store.Delete(ctx, newPath)
		}
		return AssignmentDTO{}, mapDBError(err)
	}
	if oldPath != "" {
		_ = s.store.Delete(ctx, oldPath)
	}
	return out, nil
}

var errHasSubmissions = apperr.New(http.StatusConflict, "has_submissions", "Topshirilgan ishlar mavjud, topshiriqni o'chirib bo'lmaydi")

// Delete is refused when any submission exists. The assignment row is locked so a submission
// cannot slip in (and be cascaded away) between the check and the delete.
func (s *AssignmentService) Delete(ctx context.Context, actor Actor, teacherID, id uuid.UUID) error {
	var path string
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		if _, err := s.owned(ctx, q, teacherID, id); err != nil {
			return err
		}
		a, err := q.LockAssignment(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		n, err := q.CountAssignmentSubmissions(ctx, id)
		if err != nil {
			return err
		}
		if n > 0 {
			return errHasSubmissions
		}
		before := s.snapshot(ctx, q, a)
		if err := q.DeleteAssignment(ctx, id); err != nil {
			return err
		}
		path = a.File
		return audit.Write(ctx, tx, actor.entry(audit.ActionDelete, "assignment", &id, before, nil))
	})
	if err != nil {
		return mapDBError(err)
	}
	if path != "" {
		_ = s.store.Delete(ctx, path)
	}
	return nil
}

// ---- submissions & grading ----

type RosterEntry struct {
	StudentID    uuid.UUID  `json:"student_id"`
	FullName     string     `json:"full_name"`
	SubmissionID *uuid.UUID `json:"submission_id"`
	SubmittedAt  *time.Time `json:"submitted_at"`
	HasFile      bool       `json:"has_file"`
	Grade        *float64   `json:"grade"`
	Feedback     string     `json:"feedback"`
	GradedAt     *time.Time `json:"graded_at"`
}

type SubmissionsView struct {
	Assignment AssignmentDTO `json:"assignment"`
	Submitted  []RosterEntry `json:"submitted"`
	Missing    []RosterEntry `json:"missing"`
}

func numericFloat(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	return &f.Float64
}

func rosterEntries(rows []db.ListAssignmentRosterRow) (submitted, missing []RosterEntry) {
	submitted, missing = []RosterEntry{}, []RosterEntry{}
	for _, r := range rows {
		e := RosterEntry{StudentID: r.StudentID, FullName: r.FullName}
		if r.SubmissionID.Valid {
			e.SubmissionID = uuidPtr(r.SubmissionID)
			e.SubmittedAt = tsPtr(r.SubmittedAt)
			e.HasFile = r.File.Valid && r.File.String != ""
			e.Grade = numericFloat(r.Grade)
			e.Feedback = r.Feedback.String
			e.GradedAt = tsPtr(r.GradedAt)
			submitted = append(submitted, e)
		} else {
			missing = append(missing, e)
		}
	}
	return
}

func (s *AssignmentService) Submissions(ctx context.Context, teacherID, id uuid.UUID) (SubmissionsView, error) {
	q := db.New(s.pool)
	a, err := s.owned(ctx, q, teacherID, id)
	if err != nil {
		return SubmissionsView{}, err
	}
	rows, err := q.ListAssignmentRoster(ctx, db.ListAssignmentRosterParams{AssignmentID: a.ID, GroupID: a.GroupID})
	if err != nil {
		return SubmissionsView{}, err
	}
	v := SubmissionsView{Assignment: s.snapshot(ctx, q, a)}
	v.Assignment.StudentCount = int64(len(rows))
	v.Submitted, v.Missing = rosterEntries(rows)
	return v, nil
}

// ValidGrade accepts 1.0-5.0 in steps of 0.5.
func ValidGrade(g float64) bool {
	if math.IsNaN(g) || g < 1 || g > 5 {
		return false
	}
	return math.Abs(g*2-math.Round(g*2)) < 1e-9
}

type GradeInput struct {
	Grade    float64 `json:"grade"`
	Feedback string  `json:"feedback"`
}

func (s *AssignmentService) Grade(ctx context.Context, actor Actor, teacherID, submissionID uuid.UUID, in GradeInput) (RosterEntry, error) {
	if !ValidGrade(in.Grade) {
		return RosterEntry{}, fieldErr("grade", "Baho 1 dan 5 gacha, 0.5 qadam bilan bo'lishi kerak")
	}
	if len([]rune(in.Feedback)) > 2000 {
		return RosterEntry{}, fieldErr("feedback", "Izoh juda uzun")
	}
	var out RosterEntry
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		own, err := q.GetSubmissionOwner(ctx, submissionID)
		if err != nil || own.TeacherID != (uuid.NullUUID{UUID: teacherID, Valid: true}) {
			return apperr.NotFound()
		}
		old, err := q.GetSubmission(ctx, submissionID)
		if err != nil {
			return notFoundOr(err)
		}
		var num pgtype.Numeric
		if err := num.Scan(fmt.Sprintf("%.1f", in.Grade)); err != nil {
			return err
		}
		sub, err := q.GradeSubmission(ctx, db.GradeSubmissionParams{ID: submissionID, Grade: num, Feedback: strings.TrimSpace(in.Feedback)})
		if err != nil {
			return err
		}
		out = RosterEntry{StudentID: sub.StudentID, SubmissionID: &sub.ID, SubmittedAt: tsPtr(sub.SubmittedAt), HasFile: sub.File != "",
			Grade: numericFloat(sub.Grade), Feedback: sub.Feedback, GradedAt: tsPtr(sub.GradedAt)}
		return audit.Write(ctx, tx, actor.entry(audit.ActionGrade, "submission", &sub.ID,
			map[string]any{"grade": numericFloat(old.Grade), "feedback": old.Feedback},
			map[string]any{"grade": out.Grade, "feedback": out.Feedback, "student_id": sub.StudentID, "assignment_id": sub.AssignmentID}))
	})
	return out, err
}

// GradesExport is the data for the grades spreadsheet; it also writes the export audit entry.
type GradesExport struct {
	Title   string
	Group   string
	Entries []RosterEntry // submitted first, then missing
}

func (s *AssignmentService) ExportGrades(ctx context.Context, actor Actor, teacherID, id uuid.UUID) (GradesExport, error) {
	v, err := s.Submissions(ctx, teacherID, id)
	if err != nil {
		return GradesExport{}, err
	}
	if err := audit.Write(ctx, s.pool, actor.entry(audit.ActionExport, "assignment", &id, nil,
		map[string]any{"export": "grades", "rows": len(v.Submitted) + len(v.Missing)})); err != nil {
		return GradesExport{}, err
	}
	return GradesExport{Title: v.Assignment.Title, Group: v.Assignment.GroupName, Entries: append(v.Submitted, v.Missing...)}, nil
}
