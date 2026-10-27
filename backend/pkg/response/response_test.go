package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	apperr "github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/errors"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/response"
)

func init() { gin.SetMode(gin.TestMode) }

// run executes handler against a fresh request and returns the recorder and gin context.
func run(handler func(c *gin.Context)) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	handler(c)
	return w, c
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("response is not valid JSON: %v\n%s", err, w.Body.String())
	}
	return m
}

func TestSuccessResponses(t *testing.T) {
	tests := []struct {
		name       string
		call       func(c *gin.Context)
		wantStatus int
	}{
		{"OK", func(c *gin.Context) { response.OK(c, gin.H{"id": "1"}, "Fetched") }, http.StatusOK},
		{"Created", func(c *gin.Context) { response.Created(c, gin.H{"id": "1"}, "Fetched") }, http.StatusCreated},
		{"Accepted", func(c *gin.Context) { response.Accepted(c, gin.H{"id": "1"}, "Fetched") }, http.StatusAccepted},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := run(tc.call)
			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
			m := decode(t, w)
			if m["success"] != true {
				t.Errorf("success = %v", m["success"])
			}
			if m["message"] != "Fetched" {
				t.Errorf("message = %v", m["message"])
			}
			if data, _ := m["data"].(map[string]any); data["id"] != "1" {
				t.Errorf("data = %v", m["data"])
			}
			if _, has := m["error"]; has {
				t.Error("success responses must not contain an error")
			}
			if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("content type = %q", ct)
			}
		})
	}
}

func TestOK_OmitsEmptyMessage(t *testing.T) {
	w, _ := run(func(c *gin.Context) { response.OK(c, gin.H{"a": 1}, "") })
	if _, has := decode(t, w)["message"]; has {
		t.Error("an empty message must be omitted")
	}
}

func TestNoContent(t *testing.T) {
	w, c := run(func(c *gin.Context) { response.NoContent(c) })
	c.Writer.WriteHeaderNow() // gin defers writing the status until the handler finishes
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("body must be empty, got %q", w.Body.String())
	}
}

func TestList_EnvelopeAndMeta(t *testing.T) {
	cursor := "abc123"
	type item struct {
		ID string `json:"id"`
	}
	w, _ := run(func(c *gin.Context) {
		response.List(c, []item{{"1"}, {"2"}}, response.ListMeta{NextCursor: &cursor, HasMore: true})
	})
	m := decode(t, w)
	items := m["data"].(map[string]any)["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %v", items)
	}
	meta := m["meta"].(map[string]any)
	if meta["next_cursor"] != "abc123" || meta["has_more"] != true || meta["returned"] != float64(2) {
		t.Errorf("meta = %v", meta)
	}
	if _, has := meta["total"]; has {
		t.Error("list responses must never include a total (cursor-only pagination)")
	}
}

func TestList_NilSliceIsEmptyArrayNotNull(t *testing.T) {
	var none []string
	w, _ := run(func(c *gin.Context) { response.List(c, none, response.ListMeta{}) })
	body := w.Body.String()
	if !strings.Contains(body, `"items":[]`) {
		t.Errorf("nil slice must encode as [], got %s", body)
	}
	if !strings.Contains(body, `"next_cursor":null`) {
		t.Errorf("a missing cursor must encode as null, got %s", body)
	}
	if m := decode(t, w); m["meta"].(map[string]any)["returned"] != float64(0) {
		t.Errorf("returned = %v", m["meta"])
	}
}

func TestError_AppErrorEnvelope(t *testing.T) {
	w, c := run(func(c *gin.Context) {
		response.Error(c, apperr.Validation(apperr.FieldError{Field: "title", Message: "Title is required"}))
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
	m := decode(t, w)
	if m["success"] != false {
		t.Errorf("success = %v", m["success"])
	}
	e := m["error"].(map[string]any)
	if e["code"] != "VALIDATION_ERROR" || e["message"] != "Request validation failed" {
		t.Errorf("error = %v", e)
	}
	details := e["details"].([]any)[0].(map[string]any)
	if details["field"] != "title" || details["message"] != "Title is required" {
		t.Errorf("details = %v", details)
	}
	if _, has := m["data"]; has {
		t.Error("error responses must not contain data")
	}
	if !c.IsAborted() {
		t.Error("Error must abort the handler chain")
	}
}

func TestError_OmitsEmptyDetails(t *testing.T) {
	w, _ := run(func(c *gin.Context) { response.Error(c, apperr.NotFound("Memory")) })
	if _, has := decode(t, w)["error"].(map[string]any)["details"]; has {
		t.Error("details must be omitted when there are none")
	}
}

// The core safety property: unexpected errors never reach the client, but are kept for logging.
func TestError_UnknownErrorIsGeneric500AndNeverLeaks(t *testing.T) {
	secret := errors.New(`pq: password authentication failed for user "pikyon_app" host db.abc.supabase.co`)
	w, c := run(func(c *gin.Context) { response.Error(c, secret) })

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
	body := w.Body.String()
	for _, leak := range []string{"password", "pikyon_app", "supabase", "pq:"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks %q: %s", leak, body)
		}
	}
	if code := decode(t, w)["error"].(map[string]any)["code"]; code != "INTERNAL_ERROR" {
		t.Errorf("code = %v", code)
	}
	if len(c.Errors) != 1 || !errors.Is(c.Errors[0].Err, secret) {
		t.Errorf("the original error must be attached to the context for logging; got %v", c.Errors)
	}
}

func TestError_WrappedAppErrorKeepsItsStatus(t *testing.T) {
	w, _ := run(func(c *gin.Context) {
		response.Error(c, errors.Join(errors.New("context"), apperr.Forbidden()))
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestError_NilIsTreatedAsInternal(t *testing.T) {
	w, _ := run(func(c *gin.Context) { response.Error(c, nil) })
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestError_HeadersForAuthAndRateLimit(t *testing.T) {
	w, _ := run(func(c *gin.Context) { response.Error(c, apperr.Unauthorized()) })
	if h := w.Header().Get("WWW-Authenticate"); !strings.HasPrefix(h, "Bearer") {
		t.Errorf("401 must carry WWW-Authenticate: Bearer, got %q", h)
	}
	w, _ = run(func(c *gin.Context) { response.Error(c, apperr.TokenExpired()) })
	if h := w.Header().Get("WWW-Authenticate"); !strings.HasPrefix(h, "Bearer") {
		t.Errorf("TOKEN_EXPIRED must carry WWW-Authenticate, got %q", h)
	}
	w, _ = run(func(c *gin.Context) { response.Error(c, apperr.InvalidCredentials()) })
	if h := w.Header().Get("WWW-Authenticate"); h != "" {
		t.Errorf("a failed login is not a bearer challenge, got %q", h)
	}

	w, _ = run(func(c *gin.Context) { response.Error(c, apperr.RateLimited(90*time.Second)) })
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "90" {
		t.Errorf("429 status/Retry-After = %d / %q", w.Code, w.Header().Get("Retry-After"))
	}
	w, _ = run(func(c *gin.Context) { response.Error(c, apperr.RateLimited(1500*time.Millisecond)) })
	if got := w.Header().Get("Retry-After"); got != "2" {
		t.Errorf("Retry-After must round up to whole seconds, got %q", got)
	}
	w, _ = run(func(c *gin.Context) { response.Error(c, apperr.RateLimited(0)) })
	if got := w.Header().Get("Retry-After"); got != "" {
		t.Errorf("no Retry-After when unknown, got %q", got)
	}
}

func TestError_StopsLaterHandlers(t *testing.T) {
	r := gin.New()
	reached := false
	r.GET("/x", func(c *gin.Context) { response.Error(c, apperr.Forbidden()) }, func(c *gin.Context) { reached = true })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if reached {
		t.Error("handlers after response.Error must not run")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d", w.Code)
	}
}
