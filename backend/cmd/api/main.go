package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/db"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/router"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/logger"
)

// @title Pikyon API
// @version 2.0
// @description Production REST API for Pikyon backend services.
// @host localhost:8080
// @BasePath /api/v1
func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logger.Setup(env, logLevel)
	logger.Info("Starting Pikyon API service...", "env", env)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.Connect(ctx); err != nil {
		logger.Error("Fatal database connection error", "error", err)
		os.Exit(1)
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}

	if err := db.RunMigrations(migrationsPath); err != nil {
		logger.Error("Fatal database migration error", "error", err)
		os.Exit(1)
	}
	logger.Info("Database migrations applied successfully")

	r := router.SetupRouter()

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("HTTP server listening", "port", 8080)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	if pool := db.GetPool(); pool != nil {
		_ = pool.Close()
	}

	logger.Info("Server stopped cleanly")
}