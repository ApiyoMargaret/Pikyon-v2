package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/db"
)

func TestConnect_MissingDatabaseURL(t *testing.T) {
	// Temporarily clear environment variable to test dynamic validation
	origURL := os.Getenv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL")
	defer func() {
		if origURL != "" {
			os.Setenv("DATABASE_URL", origURL)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := ConnectWrapper(ctx)
	if err == nil {
		if client != nil && client.DB != nil {
			client.Close()
		}
		t.Fatal("Expected error when DATABASE_URL is missing, got nil")
	}
}

func ConnectWrapper(ctx context.Context) (*db.Client, error) {
	return db.Connect(ctx)
}