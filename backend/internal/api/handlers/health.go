package handlers

import (
	"encoding/json"
	"net/http"
)

// HealthResponse defines the structure for health check output.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// HealthCheck handles health status queries.
// @Summary System Health Check
// @Description Returns server health status.
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:  "ok",
		Service: "pikyon-backend",
	})
}