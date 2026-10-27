package errors_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	apperr "github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/errors"
)

// Every constructor must match the status and code in the API design (06-api-design.md §6.3).
func TestConstructors_MatchAPIDesign(t *testing.T) {
	tests := []struct {
		name       string
		err        *apperr.Error
		wantStatus int
		wantCode   apperr.Code
	}{
		{"Validation", apperr.Validation(), http.StatusBadRequest, "VALIDATION_ERROR"},
		{"ObjectKeyMismatch", apperr.ObjectKeyMismatch(), http.StatusBadRequest, "OBJECT_KEY_MISMATCH"},
		{"Unauthorized", apperr.Unauthorized(), http.StatusUnauthorized, "UNAUTHORIZED"},
		{"TokenExpired", apperr.TokenExpired(), http.StatusUnauthorized, "TOKEN_EXPIRED"},
		{"InvalidCredentials", apperr.InvalidCredentials(), http.StatusUnauthorized, "INVALID_CREDENTIALS"},
		{"Forbidden", apperr.Forbidden(), http.StatusForbidden, "FORBIDDEN"},
		{"PINRequired", apperr.PINRequired(), http.StatusForbidden, "PIN_REQUIRED"},
		{"EmailNotVerified", apperr.EmailNotVerified(), http.StatusForbidden, "EMAIL_NOT_VERIFIED"},
		{"TwoFARequired", apperr.TwoFARequired(), http.StatusForbidden, "TWO_FA_REQUIRED"},
		{"NotFound", apperr.NotFound("Memory"), http.StatusNotFound, "NOT_FOUND"},
		{"ObjectNotFound", apperr.ObjectNotFound(), http.StatusNotFound, "OBJECT_NOT_FOUND"},
		{"AlreadyExists", apperr.AlreadyExists("x"), http.StatusConflict, "ALREADY_EXISTS"},
		{"Gone", apperr.Gone("x"), http.StatusGone, "GONE"},
		{"FileTooLarge", apperr.FileTooLarge(), http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE"},
		{"UnsupportedMedia", apperr.UnsupportedMedia(), http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA"},
		{"RateLimited", apperr.RateLimited(time.Minute), http.StatusTooManyRequests, "RATE_LIMITED"},
		{"Internal", apperr.Internal(errors.New("x")), http.StatusInternalServerError, "INTERNAL_ERROR"},
		{"ServiceUnavailable", apperr.ServiceUnavailable(errors.New("x")), http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Status != tc.wantStatus {
				t.Errorf("status = %d, want %d", tc.err.Status, tc.wantStatus)
			}
			if tc.err.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", tc.err.Code, tc.wantCode)
			}
			if tc.err.Message == "" {
				t.Error("message must not be empty")
			}
		})
	}
}

func TestValidation_CarriesFieldDetails(t *testing.T) {
	e := apperr.Validation(
		apperr.FieldError{Field: "title", Message: "Title is required"},
		apperr.FieldError{Field: "email", Message: "Invalid email"},
	)
	if len(e.Details) != 2 || e.Details[0].Field != "title" || e.Details[1].Message != "Invalid email" {
		t.Errorf("details = %+v", e.Details)
	}
}

func TestNotFound_Message(t *testing.T) {
	if got := apperr.NotFound("Memory").Message; got != "Memory not found" {
		t.Errorf("message = %q", got)
	}
	if got := apperr.NotFound("").Message; got != "Resource not found" {
		t.Errorf("empty label message = %q", got)
	}
}

func TestRateLimited_RetryAfter(t *testing.T) {
	if got := apperr.RateLimited(90 * time.Second).RetryAfter; got != 90*time.Second {
		t.Errorf("RetryAfter = %v", got)
	}
}

// The cause must be available for logs but never appear in the client-facing message.
func TestInternal_NeverExposesCauseToClients(t *testing.T) {
	cause := errors.New(`pq: password authentication failed for user "postgres"`)
	e := apperr.Internal(cause)
	if strings.Contains(e.Message, "password") || strings.Contains(e.Message, "postgres") {
		t.Errorf("client message leaks the cause: %q", e.Message)
	}
	if !strings.Contains(e.Error(), "password authentication failed") {
		t.Errorf("Error() should include the cause for logging, got %q", e.Error())
	}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should find the cause through Unwrap")
	}
}

func TestWithCause_DoesNotMutateOriginal(t *testing.T) {
	base := apperr.NotFound("Memory")
	cause := errors.New("no rows")
	withCause := base.WithCause(cause)
	if errors.Is(base, cause) {
		t.Error("original must not gain the cause")
	}
	if !errors.Is(withCause, cause) {
		t.Error("copy must carry the cause")
	}
	if withCause.Code != base.Code || withCause.Status != base.Status {
		t.Error("copy must keep code and status")
	}
}

func TestAs_FindsWrappedErrors(t *testing.T) {
	wrapped := fmt.Errorf("load memory: %w", apperr.NotFound("Memory"))
	e, ok := apperr.As(wrapped)
	if !ok || e.Code != apperr.CodeNotFound {
		t.Errorf("As = %v, %v", e, ok)
	}
	if _, ok := apperr.As(errors.New("plain")); ok {
		t.Error("As must not match a plain error")
	}
	if _, ok := apperr.As(nil); ok {
		t.Error("As(nil) must be false")
	}
}

func TestHasCode(t *testing.T) {
	err := fmt.Errorf("outer: %w", apperr.AlreadyExists("Email is already registered"))
	if !apperr.HasCode(err, apperr.CodeAlreadyExists) {
		t.Error("expected ALREADY_EXISTS")
	}
	if apperr.HasCode(err, apperr.CodeNotFound) {
		t.Error("must not match a different code")
	}
	if apperr.HasCode(errors.New("plain"), apperr.CodeNotFound) {
		t.Error("plain errors have no code")
	}
}

func TestFrom(t *testing.T) {
	known := apperr.Forbidden()
	if got := apperr.From(fmt.Errorf("wrap: %w", known)); got != known {
		t.Error("From must return the wrapped *Error unchanged")
	}

	for name, err := range map[string]error{
		"plain error":      errors.New("db exploded"),
		"context canceled": context.Canceled,
	} {
		got := apperr.From(err)
		if got.Status != http.StatusInternalServerError || got.Code != apperr.CodeInternal {
			t.Errorf("%s: From = %d %s, want 500 INTERNAL_ERROR", name, got.Status, got.Code)
		}
		if !errors.Is(got, err) {
			t.Errorf("%s: original error must stay in the chain for logging", name)
		}
	}
}
