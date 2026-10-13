package hemis

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// MaxPhotoSize is the largest photo accepted from HEMIS.
const MaxPhotoSize = 5 << 20

// ErrPhotoHost: the photo URL points at a host that is not allowed.
var ErrPhotoHost = errors.New("photo host not allowed")

// PhotoFetcher downloads a photo: (bytes, file name).
type PhotoFetcher func(ctx context.Context, imageURL string) ([]byte, string, error)

// NewPhotoFetcher returns the one downloader shared by students and teachers. Only http(s) URLs on
// an allowed host are fetched (baseURL's host is always allowed; relative URLs resolve against it),
// redirects never leave the allowlist, the body is limited to 5 MB and must be jpeg, png or webp.
func NewPhotoFetcher(baseURL string, allowedHosts []string, timeout time.Duration) PhotoFetcher {
	base, _ := url.Parse(baseURL)
	allowed := map[string]bool{}
	if base != nil && base.Host != "" {
		allowed[strings.ToLower(base.Host)] = true
	}
	for _, h := range allowedHosts {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			allowed[h] = true
		}
	}
	ok := func(u *url.URL) bool {
		return (u.Scheme == "http" || u.Scheme == "https") && allowed[strings.ToLower(u.Host)]
	}
	client := &http.Client{Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 || !ok(req.URL) {
			return ErrPhotoHost
		}
		return nil
	}}
	return func(ctx context.Context, raw string) ([]byte, string, error) {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil {
			return nil, "", err
		}
		if !u.IsAbs() && base != nil { // "/uploads/x.jpg" is relative to HEMIS
			u = base.ResolveReference(u)
		}
		if !ok(u) {
			return nil, "", ErrPhotoHost
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, "", err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("download failed")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("download status %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, MaxPhotoSize+1))
		if err != nil {
			return nil, "", err
		}
		if len(data) > MaxPhotoSize {
			return nil, "", errors.New("photo too large")
		}
		switch http.DetectContentType(data) {
		case "image/jpeg", "image/png", "image/webp":
		default:
			return nil, "", errors.New("photo is not jpeg, png or webp")
		}
		return data, path.Base(u.Path), nil
	}
}
