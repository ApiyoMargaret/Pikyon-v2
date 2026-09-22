package errors

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound         = errors.New("resource not found")
	ErrUnauthorized     = errors.New("unauthorized access")
	ErrForbidden        = errors.New("forbidden operation")
	ErrBadRequest       = errors.New("bad request")
	ErrConflict         = errors.New("resource conflict")
	ErrInternalServer   = errors.New("internal server error")
	ErrValidationFailed = errors.New("validation failed")
)

// AppError represents a structured application error with an HTTP status code.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// NewAppError constructs a custom AppError with status code and message.
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// MapToHTTPStatus inspects standard errors and converts them to appropriate HTTP status codes.
func MapToHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}

	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrBadRequest), errors.Is(err, ErrValidationFailed):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}