package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log/slog"
)

// SecurityHeaders returns a Gin middleware that sets essential security headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https://*.supabase.co data:; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self' https://*.supabase.co")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(self), geolocation=()")
		c.Writer.Header().Set("Cache-Control", "no-store")
		c.Next()
	}
}

// CORS returns a Gin middleware that configures CORS headers.
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
			c.Next()
			return
		}
		preflight := c.Request.Method == "OPTIONS" && c.GetHeader("Access-Control-Request-Method") != ""

		if _, ok := allowed[origin]; !ok {
			if preflight {
				c.AbortWithStatus(403)
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

// RequestID returns a Gin middleware that assigns and propagates a request ID.
func RequestID() gin.HandlerFunc {
	const HeaderRequestID = "X-Request-ID"

	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

// RequestLogger writes one structured log line per request after it completes.
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
			slog.String("request_id", "unknown"),
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
			slog.Int("bytes_out", c.Writer.Size()),
		}
		if len(c.Errors) > 0 {
			if status >= 500 {
				attrs = append(attrs, slog.String("error", c.Errors[0].Error()))
			}
		}

		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}
		logger.LogAttrs(c.Request.Context(), level, "request", attrs...)
		c.Next()
	}
}

// BodyLimit caps the size of request bodies.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}