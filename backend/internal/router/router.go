package router

import (
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes and configures the HTTP router with middleware and core API routes.
func SetupRouter(eng *middleware.Engine) *gin.Engine {
	r := gin.New()
	r.Use(middleware.SecurityHeaders())
	// r.Use(middleware.CORS(eng.CORSAllowedOrigins))
	r.Use(middleware.RequestID())
	// r.Use(middleware.RequestLogger(eng))
	r.Use(middleware.BodyLimit(10 * 1024 * 1024)) // 10MB max
	// r.Use(middleware.RateLimit(eng.RateLimiter))

	return r
}