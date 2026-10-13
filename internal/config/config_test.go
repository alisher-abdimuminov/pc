package config

import (
	"strings"
	"testing"
)

func setBase(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("FACE_SERVICE_TOKEN", "tok")
}

func TestLoadDefaults(t *testing.T) {
	setBase(t)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Location.String() != "Asia/Tashkent" || c.MaxAttemptsPerStep != 5 || c.SessionTTL.Hours() != 168 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestLoadValidation(t *testing.T) {
	cases := map[string]map[string]string{
		"bad tz":                {"TZ_NAME": "Nowhere/Land"},
		"bad env":               {"APP_ENV": "staging"},
		"bad ttl":               {"SESSION_TTL": "soon"},
		"dev login: prod":       {"APP_ENV": "prod", "DEV_STUDENT_PASSWORD_LOGIN": "true"},
		"insecure cookie: prod": {"APP_ENV": "prod", "COOKIE_SECURE": "false"},
		"default token: prod":   {"APP_ENV": "prod", "FACE_SERVICE_TOKEN": "change-me"},
		"bad proxy":             {"TRUSTED_PROXIES": "172.16.0.0/12,not-an-ip"},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			setBase(t)
			for k, v := range env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestOAuthConfig(t *testing.T) {
	setBase(t)
	c, err := Load()
	if err != nil || c.OAuthEnabled() || !c.TeacherAutoCreate {
		t.Fatalf("OAuth should be off by default (auto-create on): %v %+v", err, c)
	}

	t.Setenv("HEMIS_OAUTH_CLIENT_ID", "cid") // partly configured: refuse to start
	_, err = Load()
	if err == nil {
		t.Fatal("expected error for incomplete OAuth config")
	}
	if strings.Contains(err.Error(), "cid") {
		t.Fatal("error must not echo values")
	}

	t.Setenv("HEMIS_OAUTH_CLIENT_SECRET", "s3cret")
	t.Setenv("HEMIS_OAUTH_REDIRECT_URI", "http://localhost:8080/auth/callback")
	t.Setenv("HEMIS_TEACHER_AUTHORIZE_URL", "http://h/oauth/authorize")
	t.Setenv("HEMIS_TEACHER_ACCESS_TOKEN_URL", "http://h/oauth/access-token")
	t.Setenv("HEMIS_TEACHER_RESOURCE_OWNER_URL", "http://h/oauth/api/user")
	t.Setenv("HEMIS_TEACHER_AUTO_CREATE", "false")
	c, err = Load()
	if err != nil || !c.OAuthEnabled() || c.TeacherAutoCreate {
		t.Fatalf("complete OAuth config should load: %v", err)
	}
}

func TestTrustedProxies(t *testing.T) {
	setBase(t)
	if c, err := Load(); err != nil || len(c.TrustedProxies) != 0 {
		t.Fatalf("none by default: %v %v", err, c.TrustedProxies)
	}
	t.Setenv("TRUSTED_PROXIES", "172.16.0.0/12, 10.0.0.5 ,")
	c, err := Load()
	if err != nil || len(c.TrustedProxies) != 2 || c.TrustedProxies[1] != "10.0.0.5" {
		t.Fatalf("%v %v", err, c.TrustedProxies)
	}
}

func TestProdWithSaneSettingsLoads(t *testing.T) {
	setBase(t)
	t.Setenv("APP_ENV", "prod")
	t.Setenv("COOKIE_SECURE", "true")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}
