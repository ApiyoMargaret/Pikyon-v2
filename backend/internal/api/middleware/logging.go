package middleware

import (
	"net/http"
	"time"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/logger"
)

// responseWriterDelegator wraps http.ResponseWriter to capture the HTTP status code.
type responseWriterDelegator struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterDelegator) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestLogger is a middleware that logs detailed HTTP request and response execution metrics.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrappedWriter := &responseWriterDelegator{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // default status code unless overridden by WriteHeader
		}

		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(start)

		logger.Info("HTTP Request Processed",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_ip", r.RemoteAddr,
			"status", wrappedWriter.statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	})
}