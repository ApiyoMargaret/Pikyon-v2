// Package response writes the API's standard JSON envelopes (06-api-design.md §6.2) from Gin
// handlers, so every endpoint answers in the same shape.
//
//	success: {"success": true, "data": ..., "message": "..."}
//	list:    {"success": true, "data": {"items": [...]}, "meta": {"next_cursor", "has_more", "returned"}}
//	error:   {"success": false, "error": {"code": "...", "message": "...", "details": [...]}}
//
// Handlers should call Error for every failure and never write err.Error() themselves.
package response

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/errors"
)

// Envelope is the top-level JSON body of every API response.
type Envelope struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Message string     `json:"message,omitempty"`
	Meta    *ListMeta  `json:"meta,omitempty"`
	Error   *ErrorBody `json:"error,omitempty"`
}

// ErrorBody is the "error" object of a failed response.
type ErrorBody struct {
	Code    apperr.Code         `json:"code"`
	Message string              `json:"message"`
	Details []apperr.FieldError `json:"details,omitempty"`
}

// ListMeta is the cursor-pagination metadata of list responses. There is deliberately no
// total count (06-api-design.md §6.2).
type ListMeta struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
	Returned   int     `json:"returned"`
}

// OK writes a 200 response. message may be empty.
func OK(c *gin.Context, data any, message string) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data, Message: message})
}

// Created writes a 201 response.
func Created(c *gin.Context, data any, message string) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data, Message: message})
}

// Accepted writes a 202 response for work that continues in the background (AI jobs).
func Accepted(c *gin.Context, data any, message string) {
	c.JSON(http.StatusAccepted, Envelope{Success: true, Data: data, Message: message})
}

// NoContent writes a 204 response with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// List writes a 200 cursor-paginated list. A nil slice is written as [] and never null.
// meta.Returned is set from len(items).
func List[T any](c *gin.Context, items []T, meta ListMeta) {
	if items == nil {
		items = []T{}
	}
	meta.Returned = len(items)
	c.JSON(http.StatusOK, Envelope{
		Success: true,
		Data:    map[string][]T{"items": items},
		Meta:    &meta,
	})
}

// Error writes the error envelope for err and aborts the handler chain.
//
// Errors that are not *apperr.Error become a generic 500: the client never sees the cause. The
// original error is attached to the Gin context (c.Errors) so logging middleware can record it.
func Error(c *gin.Context, err error) {
	if err == nil {
		err = apperr.Internal(nil)
	}
	e := apperr.From(err)
	_ = c.Error(err)

	switch e.Code {
	case apperr.CodeUnauthorized, apperr.CodeTokenExpired:
		c.Header("WWW-Authenticate", `Bearer realm="pikyon"`)
	case apperr.CodeRateLimited:
		if e.RetryAfter > 0 {
			secs := int(math.Ceil(e.RetryAfter.Seconds()))
			c.Header("Retry-After", strconv.Itoa(secs))
		}
	}

	c.AbortWithStatusJSON(e.Status, Envelope{
		Success: false,
		Error:   &ErrorBody{Code: e.Code, Message: e.Message, Details: e.Details},
	})
}
