package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLocalSaveOpenDelete(t *testing.T) {
	s := NewLocal(t.TempDir())
	ctx := context.Background()
	rel, err := s.Save(ctx, KindSubmissions, "pdf", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^submissions/\d{4}/\d{2}/[0-9a-f-]{36}\.pdf$`).MatchString(rel) {
		t.Fatalf("bad path layout: %s", rel)
	}
	r, err := s.Open(ctx, rel)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(r)
	r.Close()
	if string(b) != "hello" {
		t.Fatalf("content: %q", b)
	}
	if err := s.Delete(ctx, rel); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, rel); err != nil {
		t.Fatalf("deleting a missing file must not fail: %v", err)
	}
	if _, err := s.Open(ctx, rel); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want not-exist, got %v", err)
	}
}

func TestLocalRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	os.WriteFile(secret, []byte("x"), 0o600)
	s := NewLocal(dir)
	for _, p := range []string{"../secret.txt", "a/../../secret.txt", "/etc/passwd", "", ".", ".."} {
		if _, err := s.Open(context.Background(), p); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("Open(%q) = %v, want ErrInvalidPath", p, err)
		}
		if err := s.Delete(context.Background(), p); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("Delete(%q) = %v, want ErrInvalidPath", p, err)
		}
	}
	if _, err := s.Save(context.Background(), "../x", "pdf", strings.NewReader("")); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("Save with bad kind: %v", err)
	}
	if _, err := s.Save(context.Background(), "selfies", "p/df", strings.NewReader("")); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("Save with bad ext: %v", err)
	}
}

func TestValidateByContent(t *testing.T) {
	pdf := []byte("%PDF-1.7 ...")
	zip := []byte("PK\x03\x04rest")
	ole := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1, 0, 0}
	png := append([]byte("\x89PNG\r\n\x1a\n"), 0, 0)
	jpg := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	webp := append([]byte("RIFF\x00\x00\x00\x00WEBP"), 0)
	exe := []byte("MZ\x90\x00")

	ok := []struct {
		kind, name string
		data       []byte
		ext        string
	}{
		{KindSubmissions, "report.pdf", pdf, "pdf"},
		{KindSubmissions, "report.docx", zip, "docx"},
		{KindSubmissions, "REPORT.XLSX", zip, "xlsx"},
		{KindSubmissions, "data.zip", zip, "zip"},
		{KindSubmissions, "legacy.doc", ole, "doc"},
		{KindAssignments, "photo.jpeg", jpg, "jpg"},
		{KindSelfies, "x.jpg", jpg, "jpg"},
		{KindSelfies, "x.png", png, "png"},
		{KindSelfies, "x", webp, "webp"},
		// extension that disagrees with the content is ignored, content decides
		{KindSubmissions, "evil.docx", pdf, "pdf"},
		{KindSubmissions, "noext", zip, "zip"},
	}
	for _, c := range ok {
		ext, err := Validate(c.kind, c.name, c.data)
		if err != nil || ext != c.ext {
			t.Errorf("%s %s: got %q %v want %q", c.kind, c.name, ext, err, c.ext)
		}
	}

	bad := []struct {
		kind, name string
		data       []byte
	}{
		{KindSubmissions, "virus.pdf", exe}, // claims pdf, is an executable
		{KindSubmissions, "script.sh", []byte("#!/bin/sh")},
		{KindSelfies, "doc.pdf", pdf}, // documents are not selfies
		{KindSelfies, "a.zip", zip},
		{KindSubmissions, "empty.pdf", nil},
		{KindSubmissions, "html.pdf", []byte("<html><script>")},
	}
	for _, c := range bad {
		if _, err := Validate(c.kind, c.name, c.data); !errors.Is(err, ErrUnsupportedType) {
			t.Errorf("%s %s should be rejected, got %v", c.kind, c.name, err)
		}
	}
}

func TestMaxSizeAndMIME(t *testing.T) {
	if MaxSize(KindSelfies) != 5<<20 || MaxSize(KindSubmissions) != 20<<20 {
		t.Fatal("limits")
	}
	if MIMEForExt("a/b.PDF") != "application/pdf" || MIMEForExt("x.unknown") != "application/octet-stream" || !IsImage("s.jpg") || IsImage("s.pdf") {
		t.Fatal("mime")
	}
}
