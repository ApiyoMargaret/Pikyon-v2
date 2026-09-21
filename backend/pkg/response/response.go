package response

import (
	"encoding/json"
	"net/http"
)

// JSONResponse defines the standardized envelope for API responses.
type JSONResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// JSON sends a JSON response with the specified HTTP status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := JSONResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// Error sends a standardized JSON error response.
func Error(w http.ResponseWriter, status int, errMessage string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errPayload := map[string]interface{}{
		"message": errMessage,
	}
	if details != nil {
		errPayload["details"] = details
	}

	resp := JSONResponse{
		Success: false,
		Error:   errPayload,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// WithMeta sends a JSON response containing pagination or response metadata.
func WithMeta(w http.ResponseWriter, status int, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := JSONResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
		Meta:    meta,
	}

	_ = json.NewEncoder(w).Encode(resp)
}