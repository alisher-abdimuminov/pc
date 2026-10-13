package storage

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
)

const (
	KindSelfies     = "selfies"
	KindAvatars     = "avatars"
	KindAssignments = "assignments"
	KindSubmissions = "submissions"

	MaxSelfieSize   = 5 << 20
	MaxDocumentSize = 20 << 20
)

var (
	ErrUnsupportedType = errors.New("unsupported file type")
	ErrTooLarge        = errors.New("file too large")
)

// content classes detected from the file's leading bytes.
const (
	classPDF  = "pdf"
	classZip  = "zip" // zip, docx, xlsx, pptx
	classOLE  = "ole" // legacy doc, xls, ppt
	classRar  = "rar"
	classJPEG = "jpeg"
	classPNG  = "png"
	classWebP = "webp"
)

// classExts lists the extensions each detected class may be stored as; the first is the default.
var classExts = map[string][]string{
	classPDF:  {"pdf"},
	classZip:  {"zip", "docx", "xlsx", "pptx"},
	classOLE:  {"doc", "xls", "ppt"},
	classRar:  {"rar"},
	classJPEG: {"jpg"},
	classPNG:  {"png"},
	classWebP: {"webp"},
}

var classMIME = map[string]string{
	classPDF: "application/pdf", classZip: "application/zip", classOLE: "application/octet-stream",
	classRar: "application/vnd.rar", classJPEG: "image/jpeg", classPNG: "image/png", classWebP: "image/webp",
}

var extMIME = map[string]string{
	"pdf": "application/pdf", "zip": "application/zip", "rar": "application/vnd.rar",
	"doc": "application/msword", "xls": "application/vnd.ms-excel", "ppt": "application/vnd.ms-powerpoint",
	"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"jpg":  "image/jpeg", "png": "image/png", "webp": "image/webp",
}

// kindClasses lists which content classes each kind accepts.
var kindClasses = map[string][]string{
	KindSelfies:     {classJPEG, classPNG, classWebP},
	KindAvatars:     {classJPEG, classPNG, classWebP},
	KindAssignments: {classPDF, classZip, classOLE, classRar, classJPEG, classPNG},
	KindSubmissions: {classPDF, classZip, classOLE, classRar, classJPEG, classPNG},
}

// MaxSize is the upload limit for a kind.
func MaxSize(kind string) int64 {
	if kind == KindSelfies || kind == KindAvatars {
		return MaxSelfieSize
	}
	return MaxDocumentSize
}

// detectClass inspects the content, never the filename.
func detectClass(h []byte) string {
	switch {
	case bytes.HasPrefix(h, []byte("%PDF-")):
		return classPDF
	case bytes.HasPrefix(h, []byte("PK\x03\x04")):
		return classZip
	case bytes.HasPrefix(h, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}):
		return classOLE
	case bytes.HasPrefix(h, []byte("Rar!\x1a\x07")):
		return classRar
	case bytes.HasPrefix(h, []byte{0xFF, 0xD8, 0xFF}):
		return classJPEG
	case bytes.HasPrefix(h, []byte("\x89PNG\r\n\x1a\n")):
		return classPNG
	case len(h) >= 12 && bytes.HasPrefix(h, []byte("RIFF")) && bytes.Equal(h[8:12], []byte("WEBP")):
		return classWebP
	}
	return ""
}

// Validate checks that the content (first bytes) is allowed for kind and returns the extension to
// store it under. The filename only picks between extensions the content already allows
// (e.g. docx vs zip); it never widens what is accepted.
func Validate(kind, filename string, header []byte) (ext string, err error) {
	class := detectClass(header)
	allowed := false
	for _, c := range kindClasses[kind] {
		if c == class {
			allowed = true
		}
	}
	if !allowed {
		return "", ErrUnsupportedType
	}
	want := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	if want == "jpeg" {
		want = "jpg"
	}
	for _, e := range classExts[class] {
		if e == want {
			return e, nil
		}
	}
	return classExts[class][0], nil
}

// MIMEForExt returns the download Content-Type for a stored file's extension.
func MIMEForExt(path string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if m, ok := extMIME[ext]; ok {
		return m
	}
	return "application/octet-stream"
}

// IsImage reports whether a stored path is displayed inline.
func IsImage(path string) bool { return strings.HasPrefix(MIMEForExt(path), "image/") }
