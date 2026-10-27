// Package errors defines the application's typed errors and how they map to the API's
// standard error envelope (06-api-design.md §6.2 and §6.3).
//
// Services and repositories return *Error values for expected failures (not found, conflict,
// validation). The response package turns them into the JSON envelope and HTTP status.
// Anything that is not an *Error is treated as an internal failure: the client only ever sees
// a generic message, while the real cause stays in the error chain for server-side logging.
//
// This package is named "errors", so import it with an alias to avoid clashing with the
// standard library:
//
//	apperr "github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/errors"
package errors

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Code is a stable, machine-readable error code from the API design (§6.3).
type Code string

const (
	CodeValidation         Code = "VALIDATION_ERROR"
	CodeObjectKeyMismatch  Code = "OBJECT_KEY_MISMATCH"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeTokenExpired       Code = "TOKEN_EXPIRED"
	CodeInvalidCredentials Code = "INVALID_CREDENTIALS"
	CodeForbidden          Code = "FORBIDDEN"
	CodePINRequired        Code = "PIN_REQUIRED"
	CodeEmailNotVerified   Code = "EMAIL_NOT_VERIFIED"
	CodeTwoFARequired      Code = "TWO_FA_REQUIRED"
	CodeNotFound           Code = "NOT_FOUND"
	CodeObjectNotFound     Code = "OBJECT_NOT_FOUND"
	CodeAlreadyExists      Code = "ALREADY_EXISTS"
	CodeGone               Code = "GONE"
	CodeFileTooLarge       Code = "FILE_TOO_LARGE"
	CodeUnsupportedMedia   Code = "UNSUPPORTED_MEDIA"
	CodeRateLimited        Code = "RATE_LIMITED"
	CodeInternal           Code = "INTERNAL_ERROR"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
)

// FieldError describes one invalid field in a VALIDATION_ERROR.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error is an application error with an HTTP status and an API error code.
// Message is always safe to show to the client. The underlying cause, if any, is never sent.
type Error struct {
	Status     int
	Code       Code
	Message    string
	Details    []FieldError
	RetryAfter time.Duration // only set for RATE_LIMITED
	cause      error
}

// Error implements the error interface. It includes the cause so server logs are useful;
// it is never written to API responses.
func (e *Error) Error() string {
	s := fmt.Sprintf("%s: %s", e.Code, e.Message)
	if e.cause != nil {
		s += ": " + e.cause.Error()
	}
	return s
}

// Unwrap exposes the underlying cause to errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.cause }

// WithCause returns a copy of e that records cause for logging.
func (e *Error) WithCause(cause error) *Error {
	c := *e
	c.cause = cause
	return &c
}

// As returns the *Error in err's chain, if there is one.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// HasCode reports whether err's chain contains an *Error with the given code.
func HasCode(err error, code Code) bool {
	e, ok := As(err)
	return ok && e.Code == code
}

// From converts any error to an *Error. Errors that are not already *Error become INTERNAL_ERROR
// with the original error kept as the cause (for logs only).
func From(err error) *Error {
	if e, ok := As(err); ok {
		return e
	}
	return Internal(err)
}

// --- 400 ---

// Validation reports invalid request fields.
func Validation(details ...FieldError) *Error {
	return &Error{Status: http.StatusBadRequest, Code: CodeValidation, Message: "Request validation failed", Details: details}
}

// ObjectKeyMismatch reports an upload confirmation whose key was not the one issued.
func ObjectKeyMismatch() *Error {
	return &Error{Status: http.StatusBadRequest, Code: CodeObjectKeyMismatch, Message: "Upload key does not match the issued key"}
}

// --- 401 ---

func Unauthorized() *Error {
	return &Error{Status: http.StatusUnauthorized, Code: CodeUnauthorized, Message: "Authentication required"}
}

func TokenExpired() *Error {
	return &Error{Status: http.StatusUnauthorized, Code: CodeTokenExpired, Message: "Access token expired"}
}

func InvalidCredentials() *Error {
	return &Error{Status: http.StatusUnauthorized, Code: CodeInvalidCredentials, Message: "Invalid email or password"}
}

// --- 403 ---

func Forbidden() *Error {
	return &Error{Status: http.StatusForbidden, Code: CodeForbidden, Message: "You do not have access to this resource"}
}

func PINRequired() *Error {
	return &Error{Status: http.StatusForbidden, Code: CodePINRequired, Message: "PIN required"}
}

func EmailNotVerified() *Error {
	return &Error{Status: http.StatusForbidden, Code: CodeEmailNotVerified, Message: "Email address not verified"}
}

func TwoFARequired() *Error {
	return &Error{Status: http.StatusForbidden, Code: CodeTwoFARequired, Message: "Two-factor authentication required"}
}

// --- 404 ---

// NotFound reports a missing resource. Use it for resources that belong to someone else too,
// so the API never reveals whether another user's resource exists. resource is a label such as
// "Memory".
func NotFound(resource string) *Error {
	if resource == "" {
		resource = "Resource"
	}
	return &Error{Status: http.StatusNotFound, Code: CodeNotFound, Message: resource + " not found"}
}

func ObjectNotFound() *Error {
	return &Error{Status: http.StatusNotFound, Code: CodeObjectNotFound, Message: "Uploaded file not found"}
}

// --- 409, 410, 413, 415, 429 ---

func AlreadyExists(message string) *Error {
	if message == "" {
		message = "Resource already exists"
	}
	return &Error{Status: http.StatusConflict, Code: CodeAlreadyExists, Message: message}
}

// Gone reports an expired or exhausted share.
func Gone(message string) *Error {
	if message == "" {
		message = "This link is no longer available"
	}
	return &Error{Status: http.StatusGone, Code: CodeGone, Message: message}
}

func FileTooLarge() *Error {
	return &Error{Status: http.StatusRequestEntityTooLarge, Code: CodeFileTooLarge, Message: "File exceeds the 100MB limit"}
}

func UnsupportedMedia() *Error {
	return &Error{Status: http.StatusUnsupportedMediaType, Code: CodeUnsupportedMedia, Message: "Unsupported file type"}
}

// RateLimited reports too many requests. retryAfter becomes the Retry-After header.
func RateLimited(retryAfter time.Duration) *Error {
	return &Error{Status: http.StatusTooManyRequests, Code: CodeRateLimited, Message: "Too many requests. Please try again later.", RetryAfter: retryAfter}
}

// --- 5xx ---

// Internal reports an unexpected failure. The client only sees a generic message; cause is kept
// for server-side logging.
func Internal(cause error) *Error {
	return &Error{Status: http.StatusInternalServerError, Code: CodeInternal, Message: "Something went wrong. Please try again later.", cause: cause}
}

// ServiceUnavailable reports that an external dependency (storage, AI, email) is down.
func ServiceUnavailable(cause error) *Error {
	return &Error{Status: http.StatusServiceUnavailable, Code: CodeServiceUnavailable, Message: "Service temporarily unavailable", cause: cause}
}
