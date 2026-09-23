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

func main() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logger.Setup(env, logLevel)
	logger.Info("Starting Pikyon API service...", "env", env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if os.Getenv("DATABASE_URL") != "" {
		pool, err := db.Connect(ctx)
		if err != nil {
			logger.Error("Failed to initialize database connection", "error", err)
		} else {
			defer pool.Close()
			logger.Info("Database connection pool initialized successfully")
		}
	} else {
		logger.Warn("DATABASE_URL not set; skipping database initialization")
	}

	r := router.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("HTTP server listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed to start", "error", err)
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

	logger.Info("Server exited cleanly")
}