package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

var globalLogger *slog.Logger

func init() {
	// Initialize default logger with INFO level
	globalLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Setup initializes the global logger based on environment and log level parameters.
func Setup(env, levelStr string) {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(env) == "development" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)
}

// Info logs an informational message with key-value attributes.
func Info(msg string, args ...any) {
	globalLogger.Info(msg, args...)
}

// Error logs an error message with key-value attributes.
func Error(msg string, args ...any) {
	globalLogger.Error(msg, args...)
}

// Debug logs a debug message with key-value attributes.
func Debug(msg string, args ...any) {
	globalLogger.Debug(msg, args...)
}

// Warn logs a warning message with key-value attributes.
func Warn(msg string, args ...any) {
	globalLogger.Warn(msg, args...)
}

// WithContext returns a logger instance populated with context values if needed.
func WithContext(ctx context.Context) *slog.Logger {
	return globalLogger
}