// Command api is the Pikyon backend entry point.
//
// TEMPORARY: while the auth, memory and media modules are rewritten on top of the new
// foundation (ADR-018), this entry point only verifies that configuration is valid and that both
// database pools connect (and that the app pool really is the restricted role), then exits. The
// full server wiring replaces this file once those modules are ready.
package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/config"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/db"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[Info] No .env file found, relying on system environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pools, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pools.Close()

	if err := pools.Ping(ctx); err != nil {
		log.Fatalf("database: %v", err)
	}
	log.Printf("[OK] configuration valid (%s); app and admin database pools connected; server wiring pending", cfg.Environment)
}
