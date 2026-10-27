// Package config loads and validates application configuration from environment variables.
//
// Validation is strict on purpose: a misconfiguration (for example, pointing the app pool at the
// owner role, which would silently bypass row level security) should stop the server at startup,
// not surface later as a data leak. Error messages name the variable and the problem but never
// print a value, so secrets cannot end up in logs.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"

	// appDBUserPrefix is the username prefix of the restricted application role (ADR-018).
	// On Supabase's pooler the role appears as "pikyon_app.<project-ref>".
	appDBUserPrefix = "pikyon_app"

	minJWTSecretLength = 32
)

// Config holds all validated application settings.
type Config struct {
	Environment        string
	Port               string
	LogLevel           string
	FrontendBaseURL    string
	CORSAllowedOrigins []string

	// DatabaseURL is the privileged (owner) connection: migrations, login/register/refresh,
	// cron jobs, public share-link resolution. It bypasses RLS and must be used sparingly.
	DatabaseURL string `json:"-"`
	// AppDatabaseURL is the restricted pikyon_app connection used for user-scoped requests.
	AppDatabaseURL string `json:"-"`
	// Pool sizes. Together they must fit within the Supabase session-pooler limit.
	DBAppMaxConns   int32
	DBAdminMaxConns int32

	JWTSecret            string        `json:"-"`
	AccessTokenDuration  time.Duration `json:"access_token_duration"`
	RefreshTokenDuration time.Duration `json:"refresh_token_duration"`
	RotationGracePeriod  time.Duration `json:"rotation_grace_period"`
}

// IsProduction reports whether the app is running in production mode.
func (c *Config) IsProduction() bool { return c.Environment == EnvProduction }

// Load reads configuration from the process environment and validates it.
// All problems are reported together so they can be fixed in one pass.
func Load() (*Config, error) {
	return load(os.Getenv)
}

func load(getenv func(string) string) (*Config, error) {
	var problems []error
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Errorf("  - "+format, args...))
	}
	get := func(key, fallback string) string {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			return v
		}
		return fallback
	}

	// --- Application ---
	env := strings.ToLower(get("APP_ENV", EnvDevelopment))
	switch env {
	case EnvDevelopment, EnvTest, EnvProduction:
	default:
		add("APP_ENV must be one of: development, test, production")
		env = EnvDevelopment
	}
	isProd := env == EnvProduction

	port := get("PORT", "8080")
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		add("PORT must be a number between 1 and 65535")
	}

	logLevel := strings.ToLower(get("LOG_LEVEL", "info"))
	switch logLevel {
	case "debug", "info", "warn", "error":
	default:
		add("LOG_LEVEL must be one of: debug, info, warn, error")
		logLevel = "info"
	}

	frontend := get("FRONTEND_BASE_URL", "")
	switch {
	case frontend == "" && isProd:
		add("FRONTEND_BASE_URL is required in production")
	case frontend == "":
		frontend = "http://localhost:5173"
	default:
		if o, err := normalizeOrigin(frontend, isProd); err != nil {
			add("FRONTEND_BASE_URL %v", err)
		} else {
			frontend = o
		}
	}

	origins := splitList(getenv("CORS_ALLOWED_ORIGINS"))
	if len(origins) == 0 {
		if isProd {
			add("CORS_ALLOWED_ORIGINS is required in production")
		} else {
			origins = []string{frontend}
		}
	}
	for i, o := range origins {
		n, err := normalizeOrigin(o, isProd)
		if err != nil {
			add("CORS_ALLOWED_ORIGINS entry %d %v", i+1, err)
			continue
		}
		origins[i] = n
	}

	// --- Database ---
	adminURL := get("DATABASE_URL", "")
	appURL := get("APP_DATABASE_URL", "")
	adminUser, appUser := "", ""
	if adminURL == "" {
		add("DATABASE_URL is required (privileged owner connection)")
	} else if u, err := checkDBURL(adminURL); err != nil {
		add("DATABASE_URL %v", err)
	} else {
		adminUser = u
	}
	if appURL == "" {
		add("APP_DATABASE_URL is required (restricted pikyon_app connection)")
	} else if u, err := checkDBURL(appURL); err != nil {
		add("APP_DATABASE_URL %v", err)
	} else {
		appUser = u
	}
	// Guard the most dangerous misconfiguration: using owner credentials for user requests
	// would bypass row level security without any error.
	if appUser != "" && !strings.HasPrefix(appUser, appDBUserPrefix) {
		add("APP_DATABASE_URL must connect as the restricted role (username starting with %q), not %q", appDBUserPrefix, appUser)
	}
	if adminUser != "" && strings.HasPrefix(adminUser, appDBUserPrefix) {
		add("DATABASE_URL must be the privileged owner role, not the restricted %q role", appDBUserPrefix)
	}

	appConns := getPositiveInt(getenv, "DB_APP_MAX_CONNS", 8, add)
	adminConns := getPositiveInt(getenv, "DB_ADMIN_MAX_CONNS", 4, add)

	// --- Auth ---
	secret := get("JWT_SECRET", "")
	switch {
	case secret == "":
		add("JWT_SECRET is required")
	case strings.HasPrefix(secret, "<"):
		add("JWT_SECRET is still the placeholder from .env.example")
	case len(secret) < minJWTSecretLength:
		add("JWT_SECRET must be at least %d characters (generate one with: openssl rand -hex 64)", minJWTSecretLength)
	}

	access := getDuration(getenv, "ACCESS_TOKEN_TTL", "15m", add)
	refresh := getDuration(getenv, "REFRESH_TOKEN_TTL", "168h", add)
	grace := getDuration(getenv, "ROTATION_GRACE_PERIOD", "30s", add)
	if access > 0 && refresh > 0 && refresh <= access {
		add("REFRESH_TOKEN_TTL must be longer than ACCESS_TOKEN_TTL")
	}
	if isProd && access > time.Hour {
		add("ACCESS_TOKEN_TTL must be 1h or less in production (the handoff specifies 15m)")
	}
	if grace > 5*time.Minute {
		add("ROTATION_GRACE_PERIOD must be 5m or less")
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid configuration:\n%w", errors.Join(problems...))
	}

	return &Config{
		Environment:          env,
		Port:                 port,
		LogLevel:             logLevel,
		FrontendBaseURL:      frontend,
		CORSAllowedOrigins:   origins,
		DatabaseURL:          adminURL,
		AppDatabaseURL:       appURL,
		DBAppMaxConns:        appConns,
		DBAdminMaxConns:      adminConns,
		JWTSecret:            secret,
		AccessTokenDuration:  access,
		RefreshTokenDuration: refresh,
		RotationGracePeriod:  grace,
	}, nil
}

// checkDBURL validates a Postgres connection URL and returns its username.
// It never includes the URL (and therefore the password) in its error.
func checkDBURL(raw string) (string, error) {
	if strings.Contains(raw, "<") {
		return "", errors.New("still contains a <placeholder> from .env.example")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return "", errors.New("must be a postgresql:// URL (URL-encode special characters in the password, for example @ as %40)")
	}
	if u.Hostname() == "" || u.User == nil || u.User.Username() == "" {
		return "", errors.New("must include a username and host")
	}
	if pw, ok := u.User.Password(); !ok || pw == "" {
		return "", errors.New("must include a password")
	}
	if strings.TrimPrefix(u.Path, "/") == "" {
		return "", errors.New("must include a database name")
	}
	if !isLocalHost(u.Hostname()) {
		switch strings.ToLower(u.Query().Get("sslmode")) {
		case "require", "verify-ca", "verify-full":
		default:
			return "", errors.New("must set sslmode=require (or verify-full) for non-local hosts")
		}
	}
	return u.User.Username(), nil
}

func isLocalHost(h string) bool {
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// normalizeOrigin validates a browser origin (scheme://host[:port]) and returns it without a
// trailing slash, which is the form browsers send in the Origin header.
func normalizeOrigin(raw string, requireHTTPS bool) (string, error) {
	if raw == "*" {
		return "", errors.New("must not be a wildcard; list explicit origins")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("must be an origin like https://app.example.com")
	}
	if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", errors.New("must not contain a path, query, fragment or credentials")
	}
	if requireHTTPS && u.Scheme != "https" {
		return "", errors.New("must use https in production")
	}
	return u.Scheme + "://" + u.Host, nil
}

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getPositiveInt(getenv func(string) string, key string, fallback int32, add func(string, ...any)) int32 {
	v := strings.TrimSpace(getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 100 {
		add("%s must be a whole number between 1 and 100", key)
		return fallback
	}
	return int32(n)
}

func getDuration(getenv func(string) string, key, fallback string, add func(string, ...any)) time.Duration {
	v := strings.TrimSpace(getenv(key))
	if v == "" {
		v = fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		add("%s must be a positive duration like 15m, 24h or 30s", key)
		return 0
	}
	return d
}
