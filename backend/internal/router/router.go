package router

import (
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/api/handlers"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/middleware"
)

// SetupRouter initializes and configures the HTTP router with middleware and core API routes.
func SetupRouter(eng *middleware.Engine) *gin.Engine {
	r := gin.New()
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(eng.Config.CORSAllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger(eng))
	r.Use(middleware.BodyLimit(10 * 1024 * 1024)) // 10MB max
	r.Use(middleware.RateLimit(eng.RateLimiter))

	// /api/v1 routes
	v1 := r.Group("/api/v1")
	{
		// Auth endpoints (no auth required)
		v1.POST("/auth/register", handlers.Register)
		v1.POST("/auth/login", handlers.Login)
		v1.POST("/auth/logout", handlers.Logout, middleware.AuthRequired())
		v1.POST("/auth/refresh", handlers.Refresh, middleware.AuthRequired())
		v1.POST("/auth/verify-email", handlers.VerifyEmail)
		v1.POST("/auth/resend-verification", handlers.ResendVerification)
		v1.POST("/auth/forgot-password", handlers.ForgotPassword)
		v1.POST("/auth/reset-password", handlers.ResetPassword)

		// Google OAuth
		v1.GET("/auth/google", handlers.GoogleOAuthStart)
		v1.GET("/auth/google/callback", handlers.GoogleOAuthCallback)

		// PIN endpoints
		v1.POST("/auth/pin/set", handlers.PinSet, middleware.AuthRequired())
		v1.POST("/auth/pin/verify", handlers.PinVerify, middleware.AuthRequired())
		v1.POST("/auth/pin/remove", handlers.PinRemove, middleware.AuthRequired())

		// User endpoints
		v1.GET("/users/me", handlers.GetMe, middleware.AuthRequired())
		v1.PATCH("/users/me", handlers.UpdateMe, middleware.AuthRequired())
		v1.DELETE("/users/me", handlers.DeleteMe, middleware.AuthRequired())
		v1.PATCH("/users/me/password", handlers.ChangePassword, middleware.AuthRequired())
		v1.POST("/users/me/avatar", handlers.UploadAvatar, middleware.AuthRequired())
		v1.DELETE("/users/me/avatar", handlers.DeleteAvatar, middleware.AuthRequired())
		v1.GET("/users/me/sessions", handlers.ListSessions, middleware.AuthRequired())
		v1.DELETE("/users/me/sessions/:session_id", handlers.RevokeSession, middleware.AuthRequired())
		v1.DELETE("/users/me/sessions", handlers.RevokeAllSessions, middleware.AuthRequired())
		v1.GET("/users/me/storage", handlers.StorageUsage, middleware.AuthRequired())

		// Memory endpoints
		v1.GET("/memories", handlers.ListMemories, middleware.AuthRequired())
		v1.POST("/memories", handlers.CreateMemory, middleware.AuthRequired())
		v1.GET("/memories/:id", handlers.GetMemory, middleware.AuthRequired())
		v1.PATCH("/memories/:id", handlers.UpdateMemory, middleware.AuthRequired())
		v1.DELETE("/memories/:id", handlers.SoftDeleteMemory, middleware.AuthRequired())
		v1.POST("/memories/:id/restore", handlers.RestoreMemory, middleware.AuthRequired())
		v1.DELETE("/memories/:id/permanent", handlers.PermanentDeleteMemory, middleware.AuthRequired())
		v1.GET("/memories/trash", handlers.ListTrash, middleware.AuthRequired())
		v1.POST("/memories/trash/empty", handlers.EmptyTrash, middleware.AuthRequired())
		v1.POST("/memories/bulk-trash", handlers.BulkTrash, middleware.AuthRequired())

		// Media endpoints
		v1.POST("/media/upload-url", handlers.UploadURL, middleware.AuthRequired())
		v1.POST("/media/confirm", handlers.ConfirmUpload, middleware.AuthRequired())
		v1.GET("/media/:memory_id/stream", handlers.Stream, middleware.AuthRequired())
		v1.GET("/media/:memory_id/thumbnail", handlers.Thumbnail, middleware.AuthRequired())
		v1.DELETE("/media/:memory_id", handlers.DeleteMedia, middleware.AuthRequired())

		// Sharing endpoints
		v1.POST("/sharing", handlers.CreateShare, middleware.AuthRequired())
		v1.GET("/sharing", handlers.ListShares, middleware.AuthRequired())
		v1.DELETE("/sharing/:share_id", handlers.RevokeShare, middleware.AuthRequired())
		v1.GET("/memories/shared/:share_token", handlers.GetSharedMemory, middleware.AuthRequired())
		v1.POST("/memories/shared/:share_token/react", handlers.ReactToShared, middleware.AuthRequired())

		// AI endpoints
		v1.POST("/ai/analyze", handlers.AIAnalyze, middleware.AuthRequired())
		v1.GET("/ai/jobs/:job_id", handlers.AIJobStatus, middleware.AuthRequired())
		v1.GET("/ai/jobs/:job_id/stream", handlers.AIJobStream, middleware.AuthRequired())

		// Search endpoints
		v1.GET("/search", handlers.Search, middleware.AuthRequired())
		v1.GET("/search/suggestions", handlers.Suggestions, middleware.AuthRequired())

		// Notification endpoints
		v1.GET("/notifications", handlers.ListNotifications, middleware.AuthRequired())
		v1.PATCH("/notifications/:id/read", handlers.MarkRead, middleware.AuthRequired())
		v1.PATCH("/notifications/read-all", handlers.MarkAllRead, middleware.AuthRequired())
		v1.DELETE("/notifications/:id", handlers.DeleteNotification, middleware.AuthRequired())
		v1.GET("/notifications/unread-count", handlers.UnreadCount, middleware.AuthRequired())
	}

	// Public endpoints (no auth)
	r.GET("/health", handlers.HealthCheck)

	return r
}