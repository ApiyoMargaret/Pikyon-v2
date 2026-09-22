package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/api/middleware"
)

func TestRequestLoggerMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})

	loggedHandler := middleware.RequestLogger(testHandler)

	req := httptest.NewRequest("POST", "/api/v1/resource", nil)
	rr := httptest.NewRecorder()

	loggedHandler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, status)
	}

	if body := rr.Body.String(); body != "created" {
		t.Errorf("Expected body 'created', got '%s'", body)
	}
}