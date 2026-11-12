package middleware

import "time"

// Engine holds configuration for the HTTP router.
type Engine struct {
	CORSAllowedOrigins []string
}

// RuleName returns the name of the rate limit rule.
func (e *Engine) RuleName() string {
	return "default"
}

// RuleWindow returns the duration of the rate limit window.
func (e *Engine) RuleWindow() time.Duration {
	return time.Minute
}