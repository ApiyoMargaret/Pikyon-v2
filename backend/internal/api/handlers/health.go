package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthResponse defines the standard JSON payload returned by health probes.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
}

// HealthCheck responds with a 200 OK and basic system metadata.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Service:   "pikyon-backend",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
