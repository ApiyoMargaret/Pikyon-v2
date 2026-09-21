package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/response"
)

func TestJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	payload := map[string]string{"message": "hello world"}

	response.JSON(rr, http.StatusOK, payload)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var resp response.JSONResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success to be true")
	}
}

func TestError(t *testing.T) {
	rr := httptest.NewRecorder()

	response.Error(rr, http.StatusBadRequest, "Invalid payload", nil)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", status)
	}

	var resp response.JSONResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if resp.Success {
		t.Errorf("Expected success to be false")
	}
}