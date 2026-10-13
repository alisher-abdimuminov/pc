package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/audit"
	"pc/internal/db"
	"pc/internal/face"
	"pc/internal/storage"
)

// EnrollmentService stores a student's reference photo and its face embedding.
type EnrollmentService struct {
	pool  *pgxpool.Pool
	store storage.Storage
	face  face.Service
}

func NewEnrollmentService(pool *pgxpool.Pool, store storage.Storage, f face.Service) *EnrollmentService {
	return &EnrollmentService{pool: pool, store: store, face: f}
}

// Enroll replaces the student's photo and embedding. The face-service decides whether the photo
// holds exactly one face; nothing is stored if it does not.
func (s *EnrollmentService) Enroll(ctx context.Context, actor Actor, studentID uuid.UUID, up *Upload) (UserDTO, error) {
	if up == nil {
		return UserDTO{}, fieldErr("image", "Rasmni tanlang")
	}
	q := db.New(s.pool)
	stu, err := q.GetUser(ctx, studentID)
	if err != nil {
		return UserDTO{}, notFoundOr(err)
	}
	if stu.Role != "student" {
		return UserDTO{}, fieldErr("image", "Yuz namunasi faqat talabalar uchun")
	}
	if up.Size > storage.MaxSelfieSize {
		return UserDTO{}, fieldErr("image", "Rasm hajmi 5 MB dan oshmasligi kerak")
	}
	data, err := io.ReadAll(io.LimitReader(up.Reader, storage.MaxSelfieSize+1))
	if err != nil {
		return UserDTO{}, err
	}
	return s.EnrollBytes(ctx, actor, studentID, data, up.Filename, "")
}

// EnrollBytes stores photo bytes and their embedding. hemisURL, when set, is remembered as the HEMIS
// photo this enrollment came from (so an unchanged HEMIS photo is not processed again).
func (s *EnrollmentService) EnrollBytes(ctx context.Context, actor Actor, studentID uuid.UUID, data []byte, filename, hemisURL string) (UserDTO, error) {
	head, _ := bufio.NewReader(bytes.NewReader(data)).Peek(512)
	ext, err := storage.Validate(storage.KindAvatars, filename, head)
	if err != nil || int64(len(data)) > storage.MaxSelfieSize {
		return UserDTO{}, fieldErr("image", "Rasm turi noto'g'ri (jpeg, png yoki webp)")
	}

	res, err := s.face.Embed(ctx, data, false) // enrollment: anti-spoofing off
	switch {
	case err == nil:
	case errors.Is(err, face.ErrNoFace):
		return UserDTO{}, fieldErr("image", "Rasmda yuz topilmadi")
	case errors.Is(err, face.ErrMultipleFaces):
		return UserDTO{}, fieldErr("image", "Rasmda bir nechta yuz bor")
	case errors.Is(err, face.ErrInvalidImage):
		return UserDTO{}, fieldErr("image", "Rasmni o'qib bo'lmadi")
	case errors.Is(err, face.ErrUnavailable):
		return UserDTO{}, errFaceDown
	default:
		return UserDTO{}, err
	}
	if len(res.Embedding) == 0 {
		return UserDTO{}, apperr.New(502, "bad_embedding", "Yuz namunasini olib bo'lmadi")
	}

	path, err := s.store.Save(ctx, storage.KindAvatars, ext, bytes.NewReader(data))
	if err != nil {
		return UserDTO{}, err
	}
	var out UserDTO
	var oldPath string
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		cur, err := q.GetUser(ctx, studentID)
		if err != nil {
			return notFoundOr(err)
		}
		oldPath = cur.Image
		if err := q.SetUserImage(ctx, db.SetUserImageParams{ID: studentID, Image: path}); err != nil {
			return err
		}
		if err := q.SetUserFace(ctx, db.SetUserFaceParams{ID: studentID, FaceEmbedding: res.Embedding, FaceModel: res.Model}); err != nil {
			return err
		}
		if hemisURL != "" {
			if err := q.SetUserHemisImageURL(ctx, db.SetUserHemisImageURLParams{ID: studentID, HemisImgUrl: hemisURL}); err != nil {
				return err
			}
		}
		u, err := q.GetUser(ctx, studentID)
		if err != nil {
			return err
		}
		out = NewUserService(s.pool).dto(ctx, q, u)
		// the embedding itself is never audited (stripped), only the fact and the model
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "user", &studentID,
			map[string]any{"face_enrolled": cur.FaceEnrolledAt.Valid, "has_photo": cur.Image != ""},
			map[string]any{"face_enrolled": true, "has_photo": true, "face_model": res.Model}))
	})
	if err != nil {
		_ = s.store.Delete(ctx, path)
		return UserDTO{}, mapDBError(err)
	}
	if oldPath != "" && oldPath != path {
		_ = s.store.Delete(ctx, oldPath)
	}
	return out, nil
}
