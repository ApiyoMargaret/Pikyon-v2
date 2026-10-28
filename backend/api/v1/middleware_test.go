package v1_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	v1 "github.com/ApiyoMargaret/Pikyon-v2/backend/api/v1"
	apperr "github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/errors"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/response"
)

func init() { gin.SetMode(gin.TestMode) }

func newLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})), &buf
}

// logLines parses the JSON log output into one map per line.
func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		out = append(out, m)
	}
	return out
}

func do(r http.Handler, method, target string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	if mutate != nil {
		mutate(req)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func okHandler(c *gin.Context) { response.OK(c, gin.H{"ok": true}, "") }

// --- RequestID ---

func TestRequestID_GeneratesAndExposes(t *testing.T) {
	r := gin.New()
	r.Use(v1.RequestID())
	var seen string
	r.GET("/x", func(c *gin.Context) { seen = v1.RequestIDFrom(c); okHandler(c) })

	w := do(r, http.MethodGet, "/x", nil)
	got := w.Header().Get(v1.HeaderRequestID)
	if _, err := uuid.Parse(got); err != nil {
		t.Errorf("generated ID %q is not a UUID", got)
	}
	if seen != got {
		t.Errorf("context ID %q != header ID %q", seen, got)
	}
}

func TestRequestID_ReusesSafeIncomingID(t *testing.T) {
	r := gin.New()
	r.Use(v1.RequestID())
	r.GET("/x", okHandler)
	w := do(r, http.MethodGet, "/x", func(req *http.Request) { req.Header.Set(v1.HeaderRequestID, "trace-abc_123.xyz") })
	if got := w.Header().Get(v1.HeaderRequestID); got != "trace-abc_123.xyz" {
		t.Errorf("ID = %q, want the caller's", got)
	}
}

func TestRequestID_ReplacesUnsafeIncomingID(t *testing.T) {
	r := gin.New()
	r.Use(v1.RequestID())
	r.GET("/x", okHandler)
	for _, bad := range []string{"short", strings.Repeat("a", 65), "has space in it", "inject\"quote12345", "<script>alert1</script>"} {
		w := do(r, http.MethodGet, "/x", func(req *http.Request) { req.Header.Set(v1.HeaderRequestID, bad) })
		got := w.Header().Get(v1.HeaderRequestID)
		if got == bad {
			t.Errorf("unsafe ID %q must be replaced", bad)
		}
		if _, err := uuid.Parse(got); err != nil {
			t.Errorf("replacement for %q is not a UUID: %q", bad, got)
		}
	}
}

func TestRequestIDFrom_EmptyWhenMiddlewareMissing(t *testing.T) {
	r := gin.New()
	var seen string
	r.GET("/x", func(c *gin.Context) { seen = v1.RequestIDFrom(c); okHandler(c) })
	do(r, http.MethodGet, "/x", nil)
	if seen != "" {
		t.Errorf("RequestIDFrom = %q, want empty", seen)
	}
}

// --- SecurityHeaders ---

func TestSecurityHeaders_MatchSecurityPlan(t *testing.T) {
	r := gin.New()
	r.Use(v1.SecurityHeaders())
	r.GET("/x", okHandler)
	w := do(r, http.MethodGet, "/x", nil)

	want := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
		"Content-Security-Policy": "default-src 'self'; img-src 'self' https://*.supabase.co data:; " +
			"script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self' https://*.supabase.co",
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(self), geolocation=()",
		"Cache-Control":          "no-store",
	}
	for k, v := range want {
		if got := w.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestSecurityHeaders_PresentOnErrorResponsesToo(t *testing.T) {
	r := gin.New()
	r.Use(v1.SecurityHeaders())
	r.GET("/x", func(c *gin.Context) { response.Error(c, apperr.Forbidden()) })
	if w := do(r, http.MethodGet, "/x", nil); w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("security headers must be set on error responses")
	}
}

// --- CORS ---

func corsRouter(origins ...string) *gin.Engine {
	r := gin.New()
	r.Use(v1.CORS(origins))
	r.GET("/x", okHandler)
	r.POST("/x", okHandler)
	return r
}

func TestCORS_AllowedOriginGetsHeaders(t *testing.T) {
	r := corsRouter("https://pikyon.vercel.app")
	w := do(r, http.MethodGet, "/x", func(req *http.Request) { req.Header.Set("Origin", "https://pikyon.vercel.app") })
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://pikyon.vercel.app" {
		t.Errorf("Allow-Origin = %q", got)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("credentials must be allowed")
	}
	if !strings.Contains(w.Header().Get("Access-Control-Expose-Headers"), "X-Request-ID") {
		t.Error("X-Request-ID must be exposed to the browser")
	}
	if !strings.Contains(w.Header().Get("Vary"), "Origin") {
		t.Error("Vary: Origin is required")
	}
}

func TestCORS_NeverWildcard(t *testing.T) {
	r := corsRouter("https://pikyon.vercel.app")
	for _, origin := range []string{"https://pikyon.vercel.app", "https://evil.example", "null"} {
		w := do(r, http.MethodGet, "/x", func(req *http.Request) { req.Header.Set("Origin", origin) })
		if w.Header().Get("Access-Control-Allow-Origin") == "*" {
			t.Errorf("origin %q got a wildcard", origin)
		}
	}
}

func TestCORS_UnlistedOriginGetsNoCORSHeaders(t *testing.T) {
	r := corsRouter("https://pikyon.vercel.app")
	w := do(r, http.MethodGet, "/x", func(req *http.Request) { req.Header.Set("Origin", "https://evil.example") })
	if w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Errorf("unlisted origin must get no CORS headers: %v", w.Header())
	}
}

func TestCORS_SimilarLookingOriginsAreNotAllowed(t *testing.T) {
	r := corsRouter("https://pikyon.vercel.app")
	for _, origin := range []string{
		"https://pikyon.vercel.app.evil.example", "http://pikyon.vercel.app", "https://pikyon.vercel.app:8443",
		"https://PIKYON.vercel.app", "https://pikyon.vercel.app/",
	} {
		w := do(r, http.MethodGet, "/x", func(req *http.Request) { req.Header.Set("Origin", origin) })
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("origin %q must not be allowed", origin)
		}
	}
}

func TestCORS_NoOriginHeaderIsUntouched(t *testing.T) {
	r := corsRouter("https://pikyon.vercel.app")
	w := do(r, http.MethodGet, "/x", nil)
	if w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("status=%d headers=%v", w.Code, w.Header())
	}
}

func TestCORS_PreflightAllowed(t *testing.T) {
	r := corsRouter("http://localhost:5173")
	w := do(r, http.MethodOptions, "/x", func(req *http.Request) {
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Access-Control-Request-Method", "POST")
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	for k, want := range map[string]string{
		"Access-Control-Allow-Origin":  "http://localhost:5173",
		"Access-Control-Allow-Methods": "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Authorization, Content-Type, X-Pin-Token, X-Confirm-Delete, X-Refresh-Token",
		"Access-Control-Max-Age":       "600",
	} {
		if got := w.Header().Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestCORS_PreflightFromUnlistedOriginIs403(t *testing.T) {
	r := corsRouter("http://localhost:5173")
	w := do(r, http.MethodOptions, "/x", func(req *http.Request) {
		req.Header.Set("Origin", "https://evil.example")
		req.Header.Set("Access-Control-Request-Method", "POST")
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("no CORS headers for a rejected preflight")
	}
}

// --- Recovery ---

func TestRecovery_PanicBecomesGenericEnvelope(t *testing.T) {
	logger, buf := newLogger()
	r := gin.New()
	r.Use(v1.RequestID(), v1.Recovery(logger))
	r.GET("/boom", func(c *gin.Context) { panic("secret token abc123 leaked from handler") })

	w := do(r, http.MethodGet, "/boom", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "secret token") || strings.Contains(w.Body.String(), "abc123") {
		t.Errorf("panic value leaked to the client: %s", w.Body.String())
	}
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Success || body.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("body = %s", w.Body.String())
	}

	lines := logLines(t, buf)
	if len(lines) != 1 || lines[0]["level"] != "ERROR" || lines[0]["msg"] != "panic recovered" {
		t.Fatalf("log = %v", lines)
	}
	if lines[0]["request_id"] != w.Header().Get(v1.HeaderRequestID) {
		t.Error("the panic log must carry the request ID")
	}
	if s, _ := lines[0]["stack"].(string); !strings.Contains(s, "goroutine") {
		t.Error("the panic log must include a stack trace")
	}
}

func TestRecovery_ServerKeepsServingAfterPanic(t *testing.T) {
	logger, _ := newLogger()
	r := gin.New()
	r.Use(v1.Recovery(logger))
	r.GET("/boom", func(c *gin.Context) { panic("x") })
	r.GET("/ok", okHandler)
	do(r, http.MethodGet, "/boom", nil)
	if w := do(r, http.MethodGet, "/ok", nil); w.Code != http.StatusOK {
		t.Errorf("after a panic the next request got %d", w.Code)
	}
}

func TestRecovery_RepanicsAbortHandler(t *testing.T) {
	logger, _ := newLogger()
	r := gin.New()
	r.Use(v1.Recovery(logger))
	r.GET("/abort", func(c *gin.Context) { panic(http.ErrAbortHandler) })
	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler {
			t.Errorf("recovered %v, want http.ErrAbortHandler to propagate", rec)
		}
	}()
	do(r, http.MethodGet, "/abort", nil)
}

// --- RequestLogger ---

func loggedRouter(logger *slog.Logger) *gin.Engine {
	r := gin.New()
	r.Use(v1.RequestID(), v1.RequestLogger(logger), v1.Recovery(logger))
	r.GET("/api/v1/memories/:id", func(c *gin.Context) { response.OK(c, nil, "") })
	r.GET("/missing/:id", func(c *gin.Context) {
		response.Error(c, apperr.NotFound("Memory").WithCause(errors.New("row for jane@example.com")))
	})
	r.GET("/fail", func(c *gin.Context) {
		response.Error(c, errors.New(`pq: connection to db.abc.supabase.co refused`))
	})
	return r
}

func TestRequestLogger_LogsRouteTemplateNotRealPath(t *testing.T) {
	logger, buf := newLogger()
	r := loggedRouter(logger)
	id := uuid.NewString()
	w := do(r, http.MethodGet, "/api/v1/memories/"+id+"?token=SECRETVALUE&email=jane@example.com", func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer SECRET-JWT")
		req.RemoteAddr = "203.0.113.9:5555"
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	raw := buf.String()
	for _, leak := range []string{id, "SECRETVALUE", "jane@example.com", "SECRET-JWT", "Bearer", "203.0.113.9"} {
		if strings.Contains(raw, leak) {
			t.Errorf("log leaks %q: %s", leak, raw)
		}
	}
	lines := logLines(t, buf)
	if len(lines) != 1 {
		t.Fatalf("want exactly one log line, got %d", len(lines))
	}
	l := lines[0]
	if l["route"] != "/api/v1/memories/:id" || l["method"] != "GET" || l["status"] != float64(200) || l["level"] != "INFO" {
		t.Errorf("log = %v", l)
	}
	if l["request_id"] != w.Header().Get(v1.HeaderRequestID) || l["request_id"] == "" {
		t.Errorf("request_id = %v", l["request_id"])
	}
	if _, ok := l["latency_ms"].(float64); !ok {
		t.Errorf("latency_ms missing: %v", l)
	}
}

func TestRequestLogger_ClientErrorIsWarnWithCodeButNoCause(t *testing.T) {
	logger, buf := newLogger()
	w := do(loggedRouter(logger), http.MethodGet, "/missing/1", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(buf.String(), "jane@example.com") {
		t.Errorf("a 4xx log must not include the error cause: %s", buf.String())
	}
	l := logLines(t, buf)[0]
	if l["level"] != "WARN" || l["error_code"] != "NOT_FOUND" || l["route"] != "/missing/:id" {
		t.Errorf("log = %v", l)
	}
	if _, has := l["error"]; has {
		t.Error("4xx logs must not carry the error text")
	}
}

func TestRequestLogger_ServerErrorIsErrorWithCause(t *testing.T) {
	logger, buf := newLogger()
	w := do(loggedRouter(logger), http.MethodGet, "/fail", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "supabase") {
		t.Error("the cause must not reach the client")
	}
	l := logLines(t, buf)[0]
	if l["level"] != "ERROR" || l["error_code"] != "INTERNAL_ERROR" {
		t.Errorf("log = %v", l)
	}
	if s, _ := l["error"].(string); !strings.Contains(s, "connection to db.abc.supabase.co refused") {
		t.Errorf("a 5xx log must include the underlying cause for debugging: %v", l)
	}
}

func TestRequestLogger_UnmatchedRoute(t *testing.T) {
	logger, buf := newLogger()
	r := gin.New()
	r.Use(v1.RequestLogger(logger))
	r.NoRoute(v1.NotFound())
	w := do(r, http.MethodGet, "/some/secret/path/"+uuid.NewString(), nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
	l := logLines(t, buf)[0]
	if l["route"] != "unmatched" {
		t.Errorf("an unmatched request must log route=unmatched, got %v", l["route"])
	}
	if strings.Contains(buf.String(), "secret/path") {
		t.Errorf("the raw path of an unmatched request must not be logged: %s", buf.String())
	}
}

// --- BodyLimit ---

func TestBodyLimit(t *testing.T) {
	r := gin.New()
	r.Use(v1.BodyLimit(100))
	var readErr error
	r.POST("/x", func(c *gin.Context) {
		_, readErr = io.ReadAll(c.Request.Body)
		okHandler(c)
	})

	post := func(body string) {
		readErr = nil
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	post(strings.Repeat("a", 100))
	if readErr != nil {
		t.Errorf("a body at the limit must be readable: %v", readErr)
	}
	post(strings.Repeat("a", 101))
	var tooBig *http.MaxBytesError
	if !errors.As(readErr, &tooBig) {
		t.Errorf("a body over the limit must fail with *http.MaxBytesError, got %v", readErr)
	}
}

// --- NotFound ---

func TestNotFound_UsesStandardEnvelope(t *testing.T) {
	r := gin.New()
	r.NoRoute(v1.NotFound())
	w := do(r, http.MethodGet, "/nope", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Success || body.Error.Code != "NOT_FOUND" {
		t.Errorf("body = %s", w.Body.String())
	}
}
