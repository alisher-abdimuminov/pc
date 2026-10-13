package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"pc/internal/db"
	"pc/internal/hemis"
)

// HemisPhotoService downloads a student's HEMIS photo and enrolls it (photo + face embedding).
type HemisPhotoService struct {
	pool    *pgxpool.Pool
	enroll  *EnrollmentService
	fetch   hemis.PhotoFetcher
	log     *zap.Logger
	timeout time.Duration
}

// NewHemisPhotoService enrolls photos fetched by the shared HEMIS downloader.
func NewHemisPhotoService(pool *pgxpool.Pool, enroll *EnrollmentService, fetch hemis.PhotoFetcher, timeout time.Duration, log *zap.Logger) *HemisPhotoService {
	return &HemisPhotoService{pool: pool, enroll: enroll, fetch: fetch, log: log, timeout: timeout}
}

// WithFetcher replaces the downloader (tests).
func (s *HemisPhotoService) WithFetcher(f hemis.PhotoFetcher) *HemisPhotoService {
	s.fetch = f
	return s
}

// Sync enrolls the HEMIS photo unless the student already has an embedding made from this very
// photo. It never fails a login: callers log the error.
func (s *HemisPhotoService) Sync(ctx context.Context, userID uuid.UUID, imageURL string) error {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return nil
	}
	u, err := db.New(s.pool).GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if len(u.FaceEmbedding) > 0 && u.HemisImgUrl == imageURL {
		return nil // unchanged: keep the existing embedding
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout+2*time.Minute)
	defer cancel()
	data, name, err := s.fetch(ctx, imageURL)
	if err != nil {
		return fmt.Errorf("hemis photo: %w", err)
	}
	_, err = s.enroll.EnrollBytes(ctx, Actor{Role: "system"}, userID, data, name, imageURL)
	return err
}
