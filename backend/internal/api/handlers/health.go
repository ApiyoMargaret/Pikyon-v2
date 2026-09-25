package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/db"
)

// HealthResponse defines the structure for health check output.
type HealthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	Database string `json:"database"`
}

// HealthCheck handles health status queries and performs real database ping checks.
// @Summary System Health Check
// @Description Returns server and live database connectivity status.
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health [get]
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	dbStatus := "disabled"
	statusCode := http.StatusOK

	pool := db.GetPool()
	if pool != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.PingContext(ctx); err != nil {
			dbStatus = "unreachable"
			statusCode = http.StatusServiceUnavailable
		} else {
			dbStatus = "connected"
		}
	}

	statusText := "ok"
	if statusCode != http.StatusOK {
		statusText = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:   statusText,
		Service:  "pikyon-backend",
		Database: dbStatus,
	})
}