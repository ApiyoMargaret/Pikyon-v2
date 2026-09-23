package router

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/ApiyoMargaret/Pikyon-v2/backend/docs"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/api/handlers"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/api/middleware"
)

// SetupRouter initializes and configures the Chi router with middleware and core API routes.
func SetupRouter() *chi.Mux {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestLogger)

	// Swagger UI Route
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// API v1 Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handlers.HealthCheck)
	})

	return r
}