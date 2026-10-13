package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                  string
	HTTPAddr                string
	DatabaseURL             string
	TZName                  string
	Location                *time.Location
	UploadDir               string
	FaceServiceURL          string
	FaceServiceToken        string
	FaceTimeout             time.Duration
	SessionTTL              time.Duration
	CookieSecure            bool
	MaxAttemptsPerStep      int
	FaceMismatchLimit       int
	GPSMaxAccuracyM         float64
	SelfieRetentionDays     int
	DevStudentPasswordLogin bool
	HemisBaseURL            string
	HemisTimeout            time.Duration
	HemisGroupsURL          string
	HemisGroupsToken        string
	HemisFacultiesURL       string
	HemisImageHosts         []string
	// TrustedProxies are the CIDRs/IPs whose X-Forwarded-For is believed (the reverse proxy). Empty = none.
	TrustedProxies []string

	// Teacher sign-in through HEMIS OAuth. Enabled when any of the six values is set; then all are required.
	OAuthClientID           string
	OAuthClientSecret       string // never logged
	OAuthRedirectURI        string
	TeacherAuthorizeURL     string
	TeacherAccessTokenURL   string
	TeacherResourceOwnerURL string
	TeacherAutoCreate       bool
}

// OAuthEnabled reports whether teacher sign-in through HEMIS is configured.
func (c *Config) OAuthEnabled() bool {
	return c.OAuthClientID != "" || c.OAuthClientSecret != "" || c.OAuthRedirectURI != "" ||
		c.TeacherAuthorizeURL != "" || c.TeacherAccessTokenURL != "" || c.TeacherResourceOwnerURL != ""
}

func (c *Config) IsProd() bool { return c.AppEnv == "prod" }

// Load reads and validates the environment.
func Load() (*Config, error) {
	var errs []string
	str := func(key, def string) string {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
		return def
	}
	integer := func(key string, def int) int {
		v := str(key, "")
		if v == "" {
			return def
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid integer %q", key, v))
			return def
		}
		return n
	}
	boolean := func(key string, def bool) bool {
		v := str(key, "")
		if v == "" {
			return def
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid bool %q", key, v))
			return def
		}
		return b
	}

	c := &Config{
		AppEnv:                  str("APP_ENV", "dev"),
		HTTPAddr:                str("HTTP_ADDR", ":8080"),
		DatabaseURL:             str("DATABASE_URL", ""),
		TZName:                  str("TZ_NAME", "Asia/Tashkent"),
		UploadDir:               str("UPLOAD_DIR", "/data/uploads"),
		FaceServiceURL:          strings.TrimRight(str("FACE_SERVICE_URL", "http://face-service:8001"), "/"),
		FaceServiceToken:        str("FACE_SERVICE_TOKEN", ""),
		CookieSecure:            boolean("COOKIE_SECURE", true),
		MaxAttemptsPerStep:      integer("MAX_ATTEMPTS_PER_STEP", 5),
		FaceMismatchLimit:       integer("FACE_MISMATCH_LIMIT", 3),
		SelfieRetentionDays:     integer("SELFIE_RETENTION_DAYS", 30),
		DevStudentPasswordLogin: boolean("DEV_STUDENT_PASSWORD_LOGIN", false),
		HemisBaseURL:            strings.TrimRight(str("HEMIS_BASE_URL", ""), "/"),
		HemisGroupsToken:        str("HEMIS_GROUPS_TOKEN", ""),

		OAuthClientID:           str("HEMIS_OAUTH_CLIENT_ID", ""),
		OAuthClientSecret:       str("HEMIS_OAUTH_CLIENT_SECRET", ""),
		OAuthRedirectURI:        str("HEMIS_OAUTH_REDIRECT_URI", ""),
		TeacherAuthorizeURL:     str("HEMIS_TEACHER_AUTHORIZE_URL", ""),
		TeacherAccessTokenURL:   str("HEMIS_TEACHER_ACCESS_TOKEN_URL", ""),
		TeacherResourceOwnerURL: str("HEMIS_TEACHER_RESOURCE_OWNER_URL", ""),
		TeacherAutoCreate:       boolean("HEMIS_TEACHER_AUTO_CREATE", true),
	}
	for _, h := range strings.Split(str("HEMIS_IMAGE_HOSTS", ""), ",") {
		if h = strings.TrimSpace(h); h != "" {
			c.HemisImageHosts = append(c.HemisImageHosts, h)
		}
	}

	ttl, err := time.ParseDuration(str("SESSION_TTL", "168h"))
	if err != nil || ttl <= 0 {
		errs = append(errs, "SESSION_TTL: invalid duration")
	}
	c.SessionTTL = ttl

	ft, err := time.ParseDuration(str("FACE_TIMEOUT", "15s"))
	if err != nil || ft <= 0 {
		errs = append(errs, "FACE_TIMEOUT: invalid duration")
	}
	c.FaceTimeout = ft

	ht, err := time.ParseDuration(str("HEMIS_TIMEOUT", "10s"))
	if err != nil || ht <= 0 {
		errs = append(errs, "HEMIS_TIMEOUT: invalid duration")
	}
	c.HemisTimeout = ht
	// The group list lives next to the student API unless HEMIS_GROUPS_URL says otherwise.
	c.HemisGroupsURL = str("HEMIS_GROUPS_URL", "")
	if c.HemisGroupsURL == "" && c.HemisBaseURL != "" {
		c.HemisGroupsURL = c.HemisBaseURL + "/rest/v1/data/group-list"
	}
	// Faculties come from the department list (structureType "Fakultet"); it uses the groups token.
	c.HemisFacultiesURL = str("HEMIS_FACULTIES_URL", "")
	if c.HemisFacultiesURL == "" && c.HemisBaseURL != "" {
		c.HemisFacultiesURL = c.HemisBaseURL + "/rest/v1/data/department-list"
	}

	acc, err := strconv.ParseFloat(str("GPS_MAX_ACCURACY_M", "100"), 64)
	if err != nil || acc <= 0 {
		errs = append(errs, "GPS_MAX_ACCURACY_M: invalid number")
	}
	c.GPSMaxAccuracyM = acc

	loc, err := time.LoadLocation(c.TZName)
	if err != nil {
		errs = append(errs, fmt.Sprintf("TZ_NAME: %v", err))
	}
	c.Location = loc

	if c.AppEnv != "dev" && c.AppEnv != "prod" {
		errs = append(errs, "APP_ENV: must be dev or prod")
	}
	if c.DatabaseURL == "" {
		errs = append(errs, "DATABASE_URL: required")
	}
	if c.FaceServiceToken == "" {
		errs = append(errs, "FACE_SERVICE_TOKEN: required")
	}
	for _, p := range strings.Split(str("TRUSTED_PROXIES", ""), ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, _, cidrErr := net.ParseCIDR(p)
		if cidrErr != nil && net.ParseIP(p) == nil {
			errs = append(errs, "TRUSTED_PROXIES: entries must be IPs or CIDRs")
			break
		}
		c.TrustedProxies = append(c.TrustedProxies, p)
	}
	if c.IsProd() && !c.CookieSecure {
		errs = append(errs, "COOKIE_SECURE must be true in prod")
	}
	if c.IsProd() && c.FaceServiceToken == "change-me" {
		errs = append(errs, "FACE_SERVICE_TOKEN: set a real secret in prod")
	}
	if c.IsProd() && c.DevStudentPasswordLogin {
		errs = append(errs, "DEV_STUDENT_PASSWORD_LOGIN must be false in prod")
	}

	if c.OAuthEnabled() {
		for _, r := range []struct{ name, v string }{
			{"HEMIS_OAUTH_CLIENT_ID", c.OAuthClientID}, {"HEMIS_OAUTH_CLIENT_SECRET", c.OAuthClientSecret},
			{"HEMIS_OAUTH_REDIRECT_URI", c.OAuthRedirectURI}, {"HEMIS_TEACHER_AUTHORIZE_URL", c.TeacherAuthorizeURL},
			{"HEMIS_TEACHER_ACCESS_TOKEN_URL", c.TeacherAccessTokenURL}, {"HEMIS_TEACHER_RESOURCE_OWNER_URL", c.TeacherResourceOwnerURL},
		} {
			if r.v == "" {
				errs = append(errs, r.name+": required when HEMIS OAuth is enabled")
			}
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid configuration: %s", strings.Join(errs, "; "))
	}
	return c, nil
}
