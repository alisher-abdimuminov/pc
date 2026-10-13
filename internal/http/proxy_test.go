package http_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"pc/internal/config"
)

// failedLoginIP sends one failed login carrying X-Forwarded-For and returns the IP the audit log recorded.
func failedLoginIP(t *testing.T, e *env, xff string) string {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"nobody","password":"wrong-wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", xff)
	resp, err := e.app.Test(req, -1)
	if err != nil || resp.StatusCode != 401 {
		t.Fatalf("login: %v %v", resp, err)
	}
	var ip string
	if err := e.pool.QueryRow(context.Background(),
		`SELECT host(ip) FROM audit_logs WHERE action='login_failed' ORDER BY created_at DESC LIMIT 1`).Scan(&ip); err != nil {
		t.Fatal(err)
	}
	return ip
}

func TestClientIPBehindTrustedProxy(t *testing.T) {
	trusted := newEnvDeps(t, nil, func(c *config.Config) { c.TrustedProxies = []string{"0.0.0.0/0"} })
	if got := failedLoginIP(t, trusted, "203.0.113.9"); got != "203.0.113.9" {
		t.Fatalf("the real client IP must be used behind the proxy, got %q", got)
	}
}

func TestForwardedForIsIgnoredWithoutTrustedProxy(t *testing.T) {
	e := newEnv(t)
	if got := failedLoginIP(t, e, "203.0.113.9"); got == "203.0.113.9" {
		t.Fatal("X-Forwarded-For must not be believed when no proxy is trusted (rate-limit bypass)")
	}
}
