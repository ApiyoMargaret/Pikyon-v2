// Package v1 wires the /api/v1 HTTP layer. This file holds the global middleware that every
// request passes through (06-api-design.md, 04-system-architecture.md §4.3, 07-security-plan.md):
// request IDs, panic recovery, structured request logging, security headers, CORS and a body
// size limit.
//
// Recommended order (outermost first):
//
//	RequestID, RequestLogger, Recovery, SecurityHeaders, CORS, BodyLimit
//
// Logging follows the "no PII, no tokens, ever" rule: only the route template (never the real
// URL, query string or headers) is logged, and error causes are logged only for server errors.
package v1

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperr "github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/errors"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/response"
)

// HeaderRequestID is the header used to propagate the request ID.
const HeaderRequestID = "X-Request-ID"

const ctxRequestID = "request_id"

// Incoming request IDs are accepted only if they are short and made of safe characters, so a
// client cannot inject newlines or huge values into logs.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// RequestID assigns every request an ID (reusing a valid X-Request-ID from the caller),
// stores it in the Gin context and returns it in the response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if !validRequestID.MatchString(id) {
			id = uuid.NewString()
		}
		c.Set(ctxRequestID, id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

// RequestIDFrom returns the request ID set by RequestID, or "" if there is none.
func RequestIDFrom(c *gin.Context) string {
	if v, ok := c.Get(ctxRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Recovery turns a panic in a handler into a logged INTERNAL_ERROR response in the standard
// envelope. The client never sees the panic value.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			if r == http.ErrAbortHandler {
				panic(r) // net/http uses this to abort a response on purpose
			}
			logger.LogAttrs(c.Request.Context(), slog.LevelError, "panic recovered",
				slog.String("request_id", RequestIDFrom(c)),
				slog.String("panic", fmt.Sprint(r)),
				slog.String("stack", string(debug.Stack())),
			)
			response.Error(c, apperr.Internal(fmt.Errorf("panic: %v", r)))
		}()
		c.Next()
	}
}

// RequestLogger writes one structured log line per request after it completes.
//
// It logs the route template (for example /api/v1/memories/:id), never the real path, query
// string, headers or client address. Level: 5xx is ERROR, 4xx is WARN, everything else INFO.
// For 5xx the underlying error is logged; for 4xx only the error code.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		attrs := []slog.Attr{
			slog.String("request_id", RequestIDFrom(c)),
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
			slog.Int("bytes_out", c.Writer.Size()),
		}
		if len(c.Errors) > 0 {
			e := apperr.From(c.Errors[0].Err)
			attrs = append(attrs, slog.String("error_code", string(e.Code)))
			if status >= http.StatusInternalServerError {
				attrs = append(attrs, slog.String("error", e.Error()))
			}
		}

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}
		logger.LogAttrs(c.Request.Context(), level, "request", attrs...)
	}
}

// BodyLimit caps the size of request bodies. Reads past the limit fail with *http.MaxBytesError,
// which the validator turns into a clean "Request body is too large" response. File uploads go
// straight to storage and are not affected.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// SecurityHeaders sets the headers required on every response (07-security-plan.md §7.5).
// Cache-Control: no-store is added so API responses are never cached by browsers or proxies.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https://*.supabase.co data:; "+
			"script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self' https://*.supabase.co")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(self), geolocation=()")
		h.Set("Cache-Control", "no-store")
		c.Next()
	}
}

// CORS allows cross-origin browser requests from exactly the configured origins. There is no
// wildcard, ever: the allowed origin is echoed back only when it matches the list.
//
// A request from an origin that is not listed is still served, but without CORS headers, so
// browsers refuse to expose the response. A preflight from an unlisted origin gets 403.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Add("Vary", "Origin")

		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next() // not a cross-origin browser request
			return
		}
		preflight := c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != ""

		if _, ok := allowed[origin]; !ok {
			if preflight {
				response.Error(c, apperr.Forbidden())
				return
			}
			c.Next()
			return
		}

		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Access-Control-Allow-Credentials", "true")
		h.Set("Access-Control-Expose-Headers", "X-Request-ID, Retry-After")
		if preflight {
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Pin-Token, X-Confirm-Delete, X-Refresh-Token")
			h.Set("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// NotFound is the handler for unknown routes, so they answer in the standard envelope.
// Register it with router.NoRoute(NotFound()).
func NotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Error(c, apperr.NotFound("Route"))
	}
}
