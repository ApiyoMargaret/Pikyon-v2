package errors_test

import (
	"fmt"
	"net/http"
	"testing"

	appErrors "github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/errors"
)

func TestMapToHTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "NotFound Error",
			err:            appErrors.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Unauthorized Error",
			err:            appErrors.ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Validation Error",
			err:            appErrors.ErrValidationFailed,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Custom AppError",
			err:            appErrors.NewAppError(http.StatusTeapot, "I'm a teapot", nil),
			expectedStatus: http.StatusTeapot,
		},
		{
			name:           "Unknown Internal Error",
			err:            fmt.Errorf("random database failure"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := appErrors.MapToHTTPStatus(tt.err)
			if status != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, status)
			}
		})
	}
}