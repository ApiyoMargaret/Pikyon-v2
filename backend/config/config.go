package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Environment          string        `json:"environment"`
	Port                 string        `json:"port"`
	DatabaseURL          string        `json:"-"`
	JWTSecret            string        `json:"-"`
	AccessTokenDuration  time.Duration `json:"access_token_duration"`
	RefreshTokenDuration time.Duration `json:"refresh_token_duration"`
	RotationGracePeriod  time.Duration `json:"rotation_grace_period"`
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("REQUIRED environment variable DATABASE_URL is missing")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("REQUIRED environment variable JWT_SECRET is missing")
	}
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters long for HMAC-SHA256 security")
	}

	accessTokenDur, err := time.ParseDuration(getEnv("ACCESS_TOKEN_DURATION", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid ACCESS_TOKEN_DURATION format: %w", err)
	}

	refreshTokenDur, err := time.ParseDuration(getEnv("REFRESH_TOKEN_DURATION", "168h")) // 7 days
	if err != nil {
		return nil, fmt.Errorf("invalid REFRESH_TOKEN_DURATION format: %w", err)
	}

	gracePeriodDur, err := time.ParseDuration(getEnv("ROTATION_GRACE_PERIOD", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid ROTATION_GRACE_PERIOD format: %w", err)
	}

	cfg := &Config{
		Environment:          getEnv("APP_ENV", "development"),
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          dbURL,
		JWTSecret:            jwtSecret,
		AccessTokenDuration:  accessTokenDur,
		RefreshTokenDuration: refreshTokenDur,
		RotationGracePeriod:  gracePeriodDur,
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}