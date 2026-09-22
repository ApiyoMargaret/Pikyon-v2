package logger_test

import (
	"context"
	"testing"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/logger"
)

func TestLoggerSetupAndExecution(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		logLevel string
	}{
		{
			name:     "Development Environment with Debug level",
			env:      "development",
			logLevel: "debug",
		},
		{
			name:     "Production Environment with Info level",
			env:      "production",
			logLevel: "info",
		},
		{
			name:     "Staging Environment with Warn level",
			env:      "staging",
			logLevel: "warn",
		},
		{
			name:     "Default fallback level",
			env:      "production",
			logLevel: "invalid_level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify logger setup executes without panicking
			logger.Setup(tt.env, tt.logLevel)

			// Exercise log invocation methods
			logger.Debug("Test debug message", "key", "value")
			logger.Info("Test info message", "key", "value")
			logger.Warn("Test warn message", "key", "value")
			logger.Error("Test error message", "key", "value")

			l := logger.WithContext(context.Background())
			if l == nil {
				t.Errorf("Expected non-nil logger instance from WithContext")
			}
		})
	}
}