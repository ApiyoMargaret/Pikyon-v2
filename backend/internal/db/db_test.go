package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/db"
)

func TestConnect(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping test: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if pool == nil {
		t.Fatal("Expected active database connection pool, got nil")
	}

	if err := pool.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	if db.GetPool() == nil {
		t.Fatal("Expected GetPool() to return active connection instance")
	}
}