package face

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func server(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(srv.URL, "secret", 2*time.Second)
}

func TestEmbedSendsTokenImageAndFlag(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embed" || r.Header.Get("X-Internal-Token") != "secret" {
			t.Errorf("path=%s token=%q", r.URL.Path, r.Header.Get("X-Internal-Token"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		f, _, err := r.FormFile("image")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if r.FormValue("anti_spoofing") != "true" {
			t.Errorf("anti_spoofing = %q", r.FormValue("anti_spoofing"))
		}
		json.NewEncoder(w).Encode(map[string]any{"embedding": []float32{0.5, -1}, "model": "Facenet512", "face_count": 1})
	})
	res, err := c.Embed(context.Background(), []byte("img"), true)
	if err != nil || len(res.Embedding) != 2 || res.Embedding[1] != -1 || res.Model != "Facenet512" {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestVerifyDecodesResultAndSendsEmbeddingJSON(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(1 << 20)
		var emb []float32
		if err := json.Unmarshal([]byte(r.FormValue("embedding")), &emb); err != nil || len(emb) != 3 {
			t.Errorf("embedding field: %q", r.FormValue("embedding"))
		}
		w.Write([]byte(`{"face_found":true,"is_real":true,"antispoof_score":0.97,"distance":0.21,"threshold":0.3,"verified":true}`))
	})
	res, err := c.Verify(context.Background(), []byte("img"), []float32{1, 2, 3})
	if err != nil || !res.Verified || !res.IsReal || *res.Distance != 0.21 || res.Threshold != 0.3 || *res.AntispoofScore != 0.97 {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestVerifySpoofHasNoDistance(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"face_found":true,"is_real":false,"antispoof_score":0.01,"distance":null,"threshold":0.3,"verified":false}`))
	})
	res, err := c.Verify(context.Background(), []byte("img"), []float32{1})
	if err != nil || res.IsReal || res.Distance != nil || res.Verified {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestTypedErrors(t *testing.T) {
	reply := func(status int, body string) *Client {
		return server(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(body))
		})
	}
	cases := []struct {
		name   string
		c      *Client
		target error
	}{
		{"no face", reply(422, `{"code":"no_face"}`), ErrNoFace},
		{"multiple faces", reply(422, `{"code":"multiple_faces"}`), ErrMultipleFaces},
		{"invalid image", reply(422, `{"code":"invalid_image"}`), ErrInvalidImage},
		{"server error", reply(500, `oops`), ErrUnavailable},
		{"still loading", reply(503, `{"code":"loading"}`), ErrUnavailable},
		{"wrong token", reply(401, `{"code":"unauthorized"}`), ErrUnavailable},
		{"garbage 200", reply(200, `not json`), ErrUnavailable},
	}
	for _, c := range cases {
		if _, err := c.c.Embed(context.Background(), []byte("x"), false); !errors.Is(err, c.target) {
			t.Errorf("%s: got %v want %v", c.name, err, c.target)
		}
	}
}

func TestServiceDownIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens any more
	c := New(url, "t", time.Second)
	_, err := c.Verify(context.Background(), []byte("x"), []float32{1})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
	if UnavailableMessage != "Yuzni tekshirish xizmati vaqtincha ishlamayapti" {
		t.Fatal("user message changed")
	}
}

func TestTimeoutIsUnavailable(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request) { time.Sleep(300 * time.Millisecond) })
	c.http.Timeout = 50 * time.Millisecond
	_, err := c.Embed(context.Background(), []byte("x"), false)
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("got %v", err)
	}
}

func TestCancelledContextIsNotReportedAsOutage(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request) { time.Sleep(200 * time.Millisecond) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Embed(ctx, []byte("x"), false)
	if errors.Is(err, ErrUnavailable) || !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
