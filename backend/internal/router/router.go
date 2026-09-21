package router

import (
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/api/handlers"
)

// NewRouter constructs and configures the core Chi HTTP router instance.
func NewRouter() *chi.Mux {
	r := chi.NewRouter()

	// 1. Core Middleware Pipeline
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// 2. Dynamic CORS Configuration (Reads environment variable or falls back to standard dev origin)
	allowedOrigins := getOrigins()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 3. API Version 1 Route Group
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handlers.HealthCheck)
	})

	return r
}

// getOrigins parses the ALLOWED_ORIGINS environment variable dynamically.
func getOrigins() []string {
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	if originsEnv == "" {
		return []string{"http://localhost:3000"}
	}

	origins := strings.Split(originsEnv, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return origins
}