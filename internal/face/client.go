// Package face is the HTTP client of the private face-service.
package face

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

// Typed errors. Callers map them to user-facing behaviour with errors.Is / errors.As.
var (
	// ErrUnavailable: the service is down, timed out or returned a server error.
	ErrUnavailable = errors.New("face service unavailable")
	// ErrNoFace / ErrMultipleFaces / ErrInvalidImage: a problem with the submitted photo.
	ErrNoFace        = errors.New("no face in image")
	ErrMultipleFaces = errors.New("more than one face in image")
	ErrInvalidImage  = errors.New("invalid image")
)

// UnavailableMessage is shown to users when the service cannot be reached.
const UnavailableMessage = "Yuzni tekshirish xizmati vaqtincha ishlamayapti"

// Service is what the rest of the app needs from the face-service (mocked in tests).
type Service interface {
	Embed(ctx context.Context, image []byte, antiSpoofing bool) (EmbedResult, error)
	Verify(ctx context.Context, image []byte, embedding []float32) (VerifyResult, error)
}

type EmbedResult struct {
	Embedding []float32 `json:"embedding"`
	Model     string    `json:"model"`
	FaceCount int       `json:"face_count"`
}

type VerifyResult struct {
	FaceFound      bool     `json:"face_found"`
	IsReal         bool     `json:"is_real"`
	AntispoofScore *float64 `json:"antispoof_score"`
	Distance       *float64 `json:"distance"`
	Threshold      float64  `json:"threshold"`
	Verified       bool     `json:"verified"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, http: &http.Client{Timeout: timeout}}
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *Client) post(ctx context.Context, path string, image []byte, fields map[string]string, out any) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="image"; filename="capture"`)
	h.Set("Content-Type", "application/octet-stream")
	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	if _, err := part.Write(image); err != nil {
		return err
	}
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Internal-Token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return fmt.Errorf("%w: %v", ErrUnavailable, unwrapNet(err))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	switch {
	case resp.StatusCode == http.StatusOK:
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("%w: bad response: %v", ErrUnavailable, err)
		}
		return nil
	case resp.StatusCode == http.StatusUnprocessableEntity:
		var e apiError
		_ = json.Unmarshal(body, &e)
		switch e.Code {
		case "no_face":
			return ErrNoFace
		case "multiple_faces":
			return ErrMultipleFaces
		default: // invalid_image, invalid_embedding, ...
			return fmt.Errorf("%w: %s", ErrInvalidImage, e.Code)
		}
	default: // 401 (wrong token), 5xx, 503 while loading: all mean "the service cannot help right now"
		return fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}
}

func unwrapNet(err error) error {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return errors.New("timeout")
	}
	return err
}

func (c *Client) Embed(ctx context.Context, image []byte, antiSpoofing bool) (EmbedResult, error) {
	var out EmbedResult
	err := c.post(ctx, "/embed", image, map[string]string{"anti_spoofing": fmt.Sprint(antiSpoofing)}, &out)
	return out, err
}

func (c *Client) Verify(ctx context.Context, image []byte, embedding []float32) (VerifyResult, error) {
	emb, err := json.Marshal(embedding)
	if err != nil {
		return VerifyResult{}, err
	}
	var out VerifyResult
	err = c.post(ctx, "/verify", image, map[string]string{"embedding": string(emb)}, &out)
	return out, err
}
