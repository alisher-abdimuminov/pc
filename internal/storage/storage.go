// Package storage stores uploaded files on local disk behind a small interface.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Storage interface {
	// Save stores r under kind and returns the relative path "{kind}/{yyyy}/{mm}/{uuid}.{ext}".
	Save(ctx context.Context, kind, ext string, r io.Reader) (string, error)
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
}

var ErrInvalidPath = errors.New("invalid storage path")

type Local struct {
	root string
	now  func() time.Time
}

func NewLocal(root string) *Local { return &Local{root: root, now: time.Now} }

// resolve maps a relative storage path to an absolute one, refusing anything outside root.
func (l *Local) resolve(rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) {
		return "", ErrInvalidPath
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrInvalidPath
	}
	return filepath.Join(l.root, clean), nil
}

func (l *Local) Save(ctx context.Context, kind, ext string, r io.Reader) (string, error) {
	if kind == "" || strings.ContainsAny(kind, `/\.`) || strings.ContainsAny(ext, `/\.`) {
		return "", ErrInvalidPath
	}
	t := l.now()
	name := uuid.NewString()
	if ext != "" {
		name += "." + ext
	}
	rel := filepath.ToSlash(filepath.Join(kind, fmt.Sprintf("%04d", t.Year()), fmt.Sprintf("%02d", int(t.Month())), name))
	full, err := l.resolve(rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return "", err
	}
	f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(full)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(full)
		return "", err
	}
	return rel, nil
}

func (l *Local) Open(_ context.Context, path string) (io.ReadCloser, error) {
	full, err := l.resolve(path)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

// Delete removes a file; a missing file is not an error.
func (l *Local) Delete(_ context.Context, path string) error {
	full, err := l.resolve(path)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
