package config

import (
	"strings"
	"testing"
	"time"
)

const (
	testAdminURL = "postgresql://postgres.abcdef:AdminPass123@aws-0-eu-west-1.pooler.supabase.com:5432/postgres?sslmode=require"
	testAppURL   = "postgresql://pikyon_app.abcdef:AppPass456@aws-0-eu-west-1.pooler.supabase.com:5432/postgres?sslmode=require"
	testSecret   = "0123456789abcdef0123456789abcdef0123456789abcdef"
)

// env builds a getenv function from a base set of valid values plus overrides.
// An override of "-" removes the variable.
func env(base map[string]string, overrides map[string]string) func(string) string {
	m := map[string]string{}
	for k, v := range base {
		m[k] = v
	}
	for k, v := range overrides {
		if v == "-" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	return func(k string) string { return m[k] }
}

func devBase() map[string]string {
	return map[string]string{
		"DATABASE_URL":     testAdminURL,
		"APP_DATABASE_URL": testAppURL,
		"JWT_SECRET":       testSecret,
	}
}

func prodBase() map[string]string {
	return map[string]string{
		"APP_ENV":              "production",
		"DATABASE_URL":         testAdminURL,
		"APP_DATABASE_URL":     testAppURL,
		"JWT_SECRET":           testSecret,
		"FRONTEND_BASE_URL":    "https://pikyon.vercel.app",
		"CORS_ALLOWED_ORIGINS": "https://pikyon.vercel.app",
		"ACCESS_TOKEN_TTL":     "15m",
	}
}

func TestLoad_DevelopmentDefaults(t *testing.T) {
	cfg, err := load(env(devBase(), nil))
	if err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}
	if cfg.Environment != EnvDevelopment || cfg.IsProduction() {
		t.Errorf("environment = %q, want development", cfg.Environment)
	}
	if cfg.Port != "8080" || cfg.LogLevel != "info" {
		t.Errorf("port/log level defaults wrong: %q %q", cfg.Port, cfg.LogLevel)
	}
	if cfg.AccessTokenDuration != 15*time.Minute || cfg.RefreshTokenDuration != 168*time.Hour || cfg.RotationGracePeriod != 30*time.Second {
		t.Errorf("token duration defaults wrong: %v %v %v", cfg.AccessTokenDuration, cfg.RefreshTokenDuration, cfg.RotationGracePeriod)
	}
	if cfg.DBAppMaxConns != 8 || cfg.DBAdminMaxConns != 4 {
		t.Errorf("pool defaults wrong: %d %d", cfg.DBAppMaxConns, cfg.DBAdminMaxConns)
	}
	if cfg.FrontendBaseURL != "http://localhost:5173" || len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "http://localhost:5173" {
		t.Errorf("frontend/CORS defaults wrong: %q %v", cfg.FrontendBaseURL, cfg.CORSAllowedOrigins)
	}
}

func TestLoad_ValidProduction(t *testing.T) {
	cfg, err := load(env(prodBase(), nil))
	if err != nil {
		t.Fatalf("expected valid production config, got: %v", err)
	}
	if !cfg.IsProduction() {
		t.Error("IsProduction should be true")
	}
}

func TestLoad_LocalPostgresWithoutSSLAllowed(t *testing.T) {
	_, err := load(env(devBase(), map[string]string{
		"DATABASE_URL":     "postgresql://postgres:postgres@localhost:5432/pikyon_test",
		"APP_DATABASE_URL": "postgresql://pikyon_app:secret@localhost:5432/pikyon_test",
	}))
	if err != nil {
		t.Fatalf("local CI-style Postgres without sslmode should be allowed: %v", err)
	}
}

func TestLoad_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		base      map[string]string
		overrides map[string]string
		wantInErr string
	}{
		{"missing DATABASE_URL", devBase(), map[string]string{"DATABASE_URL": "-"}, "DATABASE_URL is required"},
		{"missing APP_DATABASE_URL", devBase(), map[string]string{"APP_DATABASE_URL": "-"}, "APP_DATABASE_URL is required"},
		{"missing JWT_SECRET", devBase(), map[string]string{"JWT_SECRET": "-"}, "JWT_SECRET is required"},
		{"short JWT_SECRET", devBase(), map[string]string{"JWT_SECRET": "tooshort"}, "at least 32 characters"},
		{"placeholder JWT_SECRET", devBase(), map[string]string{"JWT_SECRET": "<generate: openssl rand -hex 64>"}, "placeholder"},
		{"placeholder database URL", devBase(), map[string]string{"DATABASE_URL": "postgresql://postgres.<PROJECT_REF>:<PW>@<HOST>:5432/postgres"}, "placeholder"},
		{"wrong scheme", devBase(), map[string]string{"DATABASE_URL": "mysql://u:p@host:3306/db"}, "postgresql:// URL"},
		{"missing password", devBase(), map[string]string{"DATABASE_URL": "postgresql://postgres.abc@host.example.com:5432/postgres?sslmode=require"}, "must include a password"},
		{"missing database name", devBase(), map[string]string{"DATABASE_URL": "postgresql://postgres.abc:pw@host.example.com:5432/?sslmode=require"}, "database name"},
		{"no sslmode on remote host", devBase(), map[string]string{"DATABASE_URL": "postgresql://postgres.abc:pw@host.example.com:5432/postgres"}, "sslmode=require"},
		{"sslmode=disable on remote host", devBase(), map[string]string{"APP_DATABASE_URL": "postgresql://pikyon_app.abc:pw@host.example.com:5432/postgres?sslmode=disable"}, "sslmode=require"},
		{"app pool uses owner role", devBase(), map[string]string{"APP_DATABASE_URL": testAdminURL}, "restricted role"},
		{"admin pool uses app role", devBase(), map[string]string{"DATABASE_URL": testAppURL}, "privileged owner role"},
		{"bad port", devBase(), map[string]string{"PORT": "99999"}, "PORT must be"},
		{"non-numeric port", devBase(), map[string]string{"PORT": "eight"}, "PORT must be"},
		{"bad environment", devBase(), map[string]string{"APP_ENV": "staging"}, "APP_ENV must be"},
		{"bad log level", devBase(), map[string]string{"LOG_LEVEL": "verbose"}, "LOG_LEVEL must be"},
		{"bad access ttl", devBase(), map[string]string{"ACCESS_TOKEN_TTL": "soon"}, "ACCESS_TOKEN_TTL must be a positive duration"},
		{"zero refresh ttl", devBase(), map[string]string{"REFRESH_TOKEN_TTL": "0s"}, "REFRESH_TOKEN_TTL must be a positive duration"},
		{"refresh not longer than access", devBase(), map[string]string{"ACCESS_TOKEN_TTL": "24h", "REFRESH_TOKEN_TTL": "1h"}, "longer than ACCESS_TOKEN_TTL"},
		{"grace too large", devBase(), map[string]string{"ROTATION_GRACE_PERIOD": "1h"}, "ROTATION_GRACE_PERIOD must be 5m or less"},
		{"pool size not a number", devBase(), map[string]string{"DB_APP_MAX_CONNS": "lots"}, "DB_APP_MAX_CONNS must be"},
		{"pool size zero", devBase(), map[string]string{"DB_ADMIN_MAX_CONNS": "0"}, "DB_ADMIN_MAX_CONNS must be"},
		{"CORS wildcard", devBase(), map[string]string{"CORS_ALLOWED_ORIGINS": "*"}, "wildcard"},
		{"CORS with path", devBase(), map[string]string{"CORS_ALLOWED_ORIGINS": "https://app.example.com/dashboard"}, "must not contain a path"},
		{"CORS not an origin", devBase(), map[string]string{"CORS_ALLOWED_ORIGINS": "app.example.com"}, "must be an origin"},
		{"production access ttl too long", prodBase(), map[string]string{"ACCESS_TOKEN_TTL": "24h"}, "1h or less in production"},
		{"production http origin", prodBase(), map[string]string{"CORS_ALLOWED_ORIGINS": "http://pikyon.vercel.app"}, "https in production"},
		{"production http frontend", prodBase(), map[string]string{"FRONTEND_BASE_URL": "http://pikyon.vercel.app"}, "https in production"},
		{"production missing frontend", prodBase(), map[string]string{"FRONTEND_BASE_URL": "-"}, "FRONTEND_BASE_URL is required"},
		{"production missing CORS", prodBase(), map[string]string{"CORS_ALLOWED_ORIGINS": "-"}, "CORS_ALLOWED_ORIGINS is required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := load(env(tc.base, tc.overrides))
			if err == nil {
				t.Fatalf("expected an error containing %q, got a valid config: %+v", tc.wantInErr, cfg)
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantInErr)
			}
		})
	}
}

func TestLoad_ReportsAllProblemsAtOnce(t *testing.T) {
	_, err := load(env(map[string]string{}, nil))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"DATABASE_URL is required", "APP_DATABASE_URL is required", "JWT_SECRET is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("combined error missing %q:\n%v", want, err)
		}
	}
}

func TestLoad_CORSNormalisation(t *testing.T) {
	cfg, err := load(env(devBase(), map[string]string{
		"CORS_ALLOWED_ORIGINS": " https://a.example.com/ , http://localhost:5173,, ",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"https://a.example.com", "http://localhost:5173"}
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != want[0] || cfg.CORSAllowedOrigins[1] != want[1] {
		t.Errorf("origins = %v, want %v", cfg.CORSAllowedOrigins, want)
	}
}

func TestLoad_EnvironmentIsCaseInsensitive(t *testing.T) {
	cfg, err := load(env(prodBase(), map[string]string{"APP_ENV": "Production"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Environment != EnvProduction {
		t.Errorf("environment = %q", cfg.Environment)
	}
}

// Errors must never reveal secrets, even for malformed values.
func TestLoad_ErrorsNeverContainSecrets(t *testing.T) {
	const pw = "Sup3rSecretPassw0rd"
	_, err := load(env(devBase(), map[string]string{
		"DATABASE_URL":     "postgresql://postgres.abc:" + pw + "@ho st:5432/postgres",
		"APP_DATABASE_URL": "postgresql://postgres.abc:" + pw + "@host.example.com:5432/postgres?sslmode=require",
		"JWT_SECRET":       "shortsecret" + pw[:3],
	}))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, secret := range []string{pw, "shortsecret"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("error leaks a secret value (%q):\n%v", secret, err)
		}
	}
}
