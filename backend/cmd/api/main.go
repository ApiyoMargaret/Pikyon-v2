package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/router"
)

func main() {
	// Read deployment environment port or default to 8080 for local dev
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize configured Chi HTTP router
	r := router.NewRouter()

	fmt.Printf("Pikyon API Server starting on port %s...\n", port)
	
	// Start blocking HTTP server listener
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}