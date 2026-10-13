package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/config"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/auth"
)

// @title Pikyon API
// @version 2.0
// @description Production REST API for Pikyon backend services.
// @host localhost:8080
// @BasePath /api/v1
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[Info] No .env file found, relying on system environment variables")
	}

	// 1. Load Application Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load application configuration: %v", err)
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 2. Initialize Database Connection Pool
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database driver: %v", err)
	}
	defer db.Close()

	// Configure DB Connection Pooling Defaults
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(15 * time.Minute)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer pingCancel()

	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("Failed to connect to PostgreSQL database: %v", err)
	}
	log.Println("[Database] Successfully connected to PostgreSQL")

	// 3. Initialize Core Domain Layers (Auth)
	authConfig := auth.Config{
		JWTSecret:            cfg.JWTSecret,
		AccessTokenDuration:  cfg.AccessTokenDuration,
		RefreshTokenDuration: cfg.RefreshTokenDuration,
		RotationGracePeriod:  cfg.RotationGracePeriod,
	}

	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, authConfig)
	authHandler := auth.NewHandler(authSvc)

	// 4. Start Background Session Worker
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	worker := auth.NewWorker(authSvc, 1*time.Hour)
	go worker.Start(workerCtx)

	// 5. Setup Router & Register HTTP Middleware
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Explicitly configure trusted proxies
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}

	// 6. Register Auth Routes
	authMiddleware := auth.AuthMiddleware(authSvc)
	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)

		// Protected endpoints
		authGroup.POST("/logout", authMiddleware, authHandler.Logout)
	}

	// 7. Setup HTTP Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run Server in a separate goroutine
	go func() {
		log.Printf("[Server] Starting Pikyon-v2 Backend on port %s (%s mode)", cfg.Port, cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[Server] HTTP listener error: %v", err)
		}
	}()

	// 8. Graceful Shutdown Signal Handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Server] Shutting down application gracefully...")

	// Explicitly stop worker goroutine
	cancelWorker()

	// Shutdown HTTP server with context timeout
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Server] HTTP server forced shutdown: %v", err)
	}

	log.Println("[Server] Application exited safely")
}