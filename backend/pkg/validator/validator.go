// Package validator reads and validates request data for Gin handlers and turns every failure
// into the API's VALIDATION_ERROR response with per-field details (06-api-design.md §6.2):
//
//	{"success": false, "error": {"code": "VALIDATION_ERROR", "message": "Request validation failed",
//	  "details": [{"field": "title", "message": "Title is required"}]}}
//
// Field names in the details are the JSON (or query-string) names the client sent, not Go
// struct field names. Messages never echo the submitted value.
//
// Typical handler use:
//
//	var req CreateMemoryRequest
//	if !validator.Bind(c, &req) {
//		return // the error response has already been written
//	}
package validator

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	playground "github.com/go-playground/validator/v10"

	apperr "github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/errors"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/response"
)

// Importing this package makes Gin's validator report json/form tag names instead of Go field
// names, so error details match what the client sent.
func init() {
	if v, ok := binding.Validator.Engine().(*playground.Validate); ok {
		v.RegisterTagNameFunc(tagName)
	}
}

func tagName(f reflect.StructField) string {
	for _, key := range []string{"json", "form"} {
		if name := strings.Split(f.Tag.Get(key), ",")[0]; name != "" && name != "-" {
			return name
		}
	}
	return ""
}

// Bind reads the JSON request body into obj and validates it using its `binding` tags.
// On failure it writes the VALIDATION_ERROR response and returns false; the caller must return.
func Bind(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		response.Error(c, Translate(err))
		return false
	}
	return true
}

// BindQuery reads the URL query string into obj (using `form` tags) and validates it.
// On failure it writes the VALIDATION_ERROR response and returns false; the caller must return.
func BindQuery(c *gin.Context, obj any) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		response.Error(c, Translate(err))
		return false
	}
	return true
}

// Translate converts a binding error into an application error. Validation problems become
// VALIDATION_ERROR with one detail per invalid field. Errors that indicate a programming
// mistake (for example binding into a non-pointer) become INTERNAL_ERROR.
func Translate(err error) *apperr.Error {
	var (
		ve     playground.ValidationErrors
		invVal *playground.InvalidValidationError
		syntax *json.SyntaxError
		typeEr *json.UnmarshalTypeError
		invUn  *json.InvalidUnmarshalError
		tooBig *http.MaxBytesError
	)
	switch {
	case errors.As(err, &ve):
		details := make([]apperr.FieldError, 0, len(ve))
		for _, fe := range ve {
			details = append(details, apperr.FieldError{Field: fieldPath(fe), Message: message(fe)})
		}
		return apperr.Validation(details...)
	case errors.As(err, &invVal), errors.As(err, &invUn):
		return apperr.Internal(err)
	case errors.As(err, &tooBig):
		return bodyProblem("Request body is too large")
	case errors.Is(err, io.EOF):
		return bodyProblem("Request body is required")
	case errors.As(err, &syntax), errors.Is(err, io.ErrUnexpectedEOF):
		return bodyProblem("Request body must be valid JSON")
	case errors.As(err, &typeEr):
		field := typeEr.Field
		if field == "" {
			field = "body"
		}
		return apperr.Validation(apperr.FieldError{
			Field:   field,
			Message: humanize(field) + " must be " + typeName(typeEr.Type),
		})
	default:
		// Not echoed: parse errors can contain the submitted value.
		return bodyProblem("Request could not be read")
	}
}

func bodyProblem(msg string) *apperr.Error {
	return apperr.Validation(apperr.FieldError{Field: "body", Message: msg})
}

// fieldPath returns the client-facing path of the invalid field, for example "title" or
// "items[0].name", without the Go struct type name that validator puts first.
func fieldPath(fe playground.FieldError) string {
	ns := fe.Namespace()
	if i := strings.Index(ns, "."); i >= 0 {
		ns = ns[i+1:]
	}
	return ns
}

func message(fe playground.FieldError) string {
	name := humanize(fe.Field())
	param := fe.Param()
	switch fe.Tag() {
	case "required":
		return name + " is required"
	case "email":
		return name + " must be a valid email address"
	case "uuid", "uuid4":
		return name + " must be a valid UUID"
	case "url", "http_url":
		return name + " must be a valid URL"
	case "alphanum":
		return name + " must contain only letters and numbers"
	case "oneof":
		return name + " must be one of: " + strings.Join(strings.Fields(param), ", ")
	case "datetime":
		if param == "2006-01-02" {
			return name + " must be a date in the format YYYY-MM-DD"
		}
		return name + " must match the format " + param
	case "min", "gte":
		return name + " must be at least " + sized(fe.Kind(), param)
	case "max", "lte":
		return name + " must be at most " + sized(fe.Kind(), param)
	case "len":
		return name + " must be exactly " + sized(fe.Kind(), param)
	case "gt":
		return name + " must be greater than " + sized(fe.Kind(), param)
	case "lt":
		return name + " must be less than " + sized(fe.Kind(), param)
	default:
		return name + " is invalid"
	}
}

// sized renders a limit with the right unit: characters for text, items for lists.
func sized(kind reflect.Kind, n string) string {
	switch kind {
	case reflect.String:
		if n == "1" {
			return "1 character"
		}
		return n + " characters"
	case reflect.Slice, reflect.Array, reflect.Map:
		if n == "1" {
			return "1 item"
		}
		return n + " items"
	default:
		return n
	}
}

// typeName describes the type a field must have, for JSON type mismatches.
func typeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "text"
	case reflect.Bool:
		return "true or false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "a whole number"
	case reflect.Float32, reflect.Float64:
		return "a number"
	case reflect.Slice, reflect.Array:
		return "a list"
	case reflect.Struct, reflect.Map:
		return "an object"
	default:
		return "a different type"
	}
}

// humanize turns a field path into a sentence-start label: "memory_date" becomes "Memory date".
func humanize(field string) string {
	if i := strings.LastIndex(field, "."); i >= 0 {
		field = field[i+1:]
	}
	if i := strings.Index(field, "["); i >= 0 {
		field = field[:i]
	}
	field = strings.ReplaceAll(field, "_", " ")
	if field == "" {
		return "Value"
	}
	r, size := utf8.DecodeRuneInString(field)
	return string(unicode.ToUpper(r)) + field[size:]
}
