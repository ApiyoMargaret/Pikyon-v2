package middleware

import (
	"context"
	"time"
)

// Engine holds configuration for the HTTP router.
type Engine struct {
	CORSAllowedOrigins []string
	RateLimiter        // embedded for access to rate limit methods
}

// RateLimiter defines the interface for rate limit checking.
type RateLimiter interface {
	Hit(ctx context.Context, key string, window time.Duration) (count int, retryAfter time.Duration, err error)
}

// RuleName returns the name of the rate limit rule.
func (e *Engine) RuleName() string {
	return "default"
}

// RuleWindow returns the duration of the rate limit window.
func (e *Engine) RuleWindow() time.Duration {
	return time.Minute
}