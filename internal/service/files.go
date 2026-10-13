package service

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/db"
	"pc/internal/storage"
)

type FileService struct {
	pool  *pgxpool.Pool
	store storage.Storage
}

func NewFileService(pool *pgxpool.Pool, store storage.Storage) *FileService {
	return &FileService{pool: pool, store: store}
}

type FileResult struct {
	Reader      io.ReadCloser
	DownloadAs  string
	ContentType string
	Inline      bool
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._\- ]+`)

func downloadName(base, path string) string {
	base = strings.Trim(unsafeName.ReplaceAllString(base, "_"), " _.")
	if base == "" {
		base = "fayl"
	}
	return base + filepath.Ext(path)
}

// Open resolves a file by (kind, id) after checking the user may see it.
// Any denial or missing file is a 404 so existence is not leaked.
func (s *FileService) Open(ctx context.Context, user db.User, kind string, id uuid.UUID) (*FileResult, error) {
	q := db.New(s.pool)
	var path, name string

	switch kind {
	case "assignments":
		a, err := q.GetAssignment(ctx, id)
		if err != nil || a.File == "" {
			return nil, apperr.NotFound()
		}
		allowed := user.Role == "admin"
		if user.Role == "teacher" {
			_, err := q.GetGroupForTeacher(ctx, db.GetGroupForTeacherParams{ID: a.GroupID, TeacherID: uuid.NullUUID{UUID: user.ID, Valid: true}})
			allowed = err == nil
		}
		if user.Role == "student" {
			allowed = user.GroupID.Valid && user.GroupID.UUID == a.GroupID
		}
		if !allowed {
			return nil, apperr.NotFound()
		}
		path, name = a.File, a.Title

	case "submissions":
		o, err := q.GetSubmissionOwner(ctx, id)
		if err != nil || o.File == "" {
			return nil, apperr.NotFound()
		}
		allowed := user.Role == "admin" ||
			(user.Role == "teacher" && o.TeacherID == uuid.NullUUID{UUID: user.ID, Valid: true}) ||
			(user.Role == "student" && o.StudentID == user.ID)
		if !allowed {
			return nil, apperr.NotFound()
		}
		path = o.File
		name = "ish"
		if stu, err := q.GetUser(ctx, o.StudentID); err == nil {
			name = stu.FullName
		}

	case "selfies":
		o, err := q.GetAttemptOwner(ctx, id)
		if err != nil || o.Image == "" {
			return nil, apperr.NotFound()
		}
		allowed := user.Role == "admin" // attempt details and selfies are for admins only
		if !allowed {
			return nil, apperr.NotFound()
		}
		path, name = o.Image, "selfi"

	case "avatars": // id is the user id
		owner, err := q.GetUser(ctx, id)
		if err != nil || owner.Image == "" {
			return nil, apperr.NotFound()
		}
		allowed := user.Role == "admin" || user.ID == owner.ID
		if user.Role == "teacher" && owner.GroupID.Valid {
			_, err := q.GetGroupForTeacher(ctx, db.GetGroupForTeacherParams{ID: owner.GroupID.UUID, TeacherID: uuid.NullUUID{UUID: user.ID, Valid: true}})
			allowed = err == nil
		}
		if !allowed {
			return nil, apperr.NotFound()
		}
		path, name = owner.Image, "rasm"

	default:
		return nil, apperr.NotFound()
	}

	r, err := s.store.Open(ctx, path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.NotFound()
		}
		return nil, err
	}
	return &FileResult{Reader: r, DownloadAs: downloadName(name, path), ContentType: storage.MIMEForExt(path), Inline: storage.IsImage(path)}, nil
}
