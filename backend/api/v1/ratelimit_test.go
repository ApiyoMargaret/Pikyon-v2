package v1_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	v1 "github.com/ApiyoMargaret/Pikyon-v2/backend/api/v1"
)

// fakeStore is an in-memory RateLimitStore that records the keys it was asked about.
type fakeStore struct {
	mu     sync.Mutex
	counts map[string]int
	keys   []string
	err    error
	retry  time.Duration
}

func newFakeStore() *fakeStore { return &fakeStore{counts: map[string]int{}, retry: 90 * time.Second} }

func (f *fakeStore) Hit(_ context.Context, key string, _ time.Duration) (int, time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys = append(f.keys, key)
	if f.err != nil {
		return 0, 0, f.err
	}
	f.counts[key]++
	return f.counts[key], f.retry, nil
}

func clientIPRouter(header string) (*gin.Engine, *string) {
	r := gin.New()
	var got string
	r.Use(v1.ClientIP(header))
	r.GET("/x", func(c *gin.Context) { got = v1.ClientIPFrom(c); okHandler(c) })
	return r, &got
}

func fromAddr(addr string, headers map[string]string) func(*http.Request) {
	return func(req *http.Request) {
		req.RemoteAddr = addr
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
}

// --- ClientIP ---

func TestClientIP_UsesSocketAddressByDefault(t *testing.T) {
	r, got := clientIPRouter("")
	do(r, http.MethodGet, "/x", fromAddr("203.0.113.9:5555", nil))
	if *got != "203.0.113.9" {
		t.Errorf("ip = %q", *got)
	}
}

// A client must never be able to choose its own address by sending headers.
func TestClientIP_IgnoresForgeableHeadersWhenNoTrustedHeaderIsSet(t *testing.T) {
	r, got := clientIPRouter("")
	do(r, http.MethodGet, "/x", fromAddr("203.0.113.9:5555", map[string]string{
		"X-Forwarded-For": "1.1.1.1", "X-Real-IP": "2.2.2.2", "CF-Connecting-IP": "3.3.3.3", "True-Client-IP": "4.4.4.4",
	}))
	if *got != "203.0.113.9" {
		t.Errorf("ip = %q, headers must be ignored", *got)
	}
}

func TestClientIP_UsesTrustedHeaderWhenConfigured(t *testing.T) {
	r, got := clientIPRouter("CF-Connecting-IP")
	do(r, http.MethodGet, "/x", fromAddr("10.0.0.5:1234", map[string]string{"CF-Connecting-IP": "198.51.100.7", "X-Forwarded-For": "9.9.9.9"}))
	if *got != "198.51.100.7" {
		t.Errorf("ip = %q, want the trusted header's value and not X-Forwarded-For", *got)
	}
}

func TestClientIP_FallsBackToSocketWhenHeaderIsBad(t *testing.T) {
	r, got := clientIPRouter("CF-Connecting-IP")
	for _, bad := range []string{"", "not-an-ip", "1.2.3.4, 5.6.7.8", "1.2.3.4:80", "1.2.3.4 ", "999.1.1.1", "<script>"} {
		*got = ""
		do(r, http.MethodGet, "/x", fromAddr("203.0.113.9:5555", map[string]string{"CF-Connecting-IP": bad}))
		want := "203.0.113.9"
		if strings.TrimSpace(bad) == "1.2.3.4" { // surrounding spaces are trimmed, so this one is valid
			want = "1.2.3.4"
		}
		if *got != want {
			t.Errorf("header %q: ip = %q, want %q", bad, *got, want)
		}
	}
}

func TestClientIP_NormalisesAddresses(t *testing.T) {
	r, got := clientIPRouter("CF-Connecting-IP")
	tests := []struct{ header, want string }{
		{"::ffff:198.51.100.7", "198.51.100.7"},
		{"2001:db8::1", "2001:db8::1"},
		{"fe80::1%eth0", "fe80::1"},
	}
	for _, tc := range tests {
		do(r, http.MethodGet, "/x", fromAddr("10.0.0.5:1", map[string]string{"CF-Connecting-IP": tc.header}))
		if *got != tc.want {
			t.Errorf("%q -> %q, want %q", tc.header, *got, tc.want)
		}
	}
}

func TestClientIPFrom_EmptyWithoutMiddleware(t *testing.T) {
	r := gin.New()
	var got string
	r.GET("/x", func(c *gin.Context) { got = v1.ClientIPFrom(c); okHandler(c) })
	do(r, http.MethodGet, "/x", nil)
	if got != "" {
		t.Errorf("ip = %q", got)
	}
}

// --- Rules ---

func TestRules_MatchAPIDesignTable(t *testing.T) {
	tests := []struct {
		rule   v1.RateRule
		name   string
		limit  int
		window time.Duration
	}{
		{v1.RuleRegister, "register", 5, time.Hour},
		{v1.RuleLogin, "login", 10, 15 * time.Minute},
		{v1.RuleForgotPassword, "forgot", 3, time.Hour},
		{v1.RulePINVerify, "pin", 5, 15 * time.Minute},
	}
	for _, tc := range tests {
		if tc.rule.Name != tc.name || tc.rule.Limit != tc.limit || tc.rule.Window != tc.window {
			t.Errorf("%s rule = %+v", tc.name, tc.rule)
		}
	}
}

// --- RateLimit ---

func limitedRouter(store v1.RateLimitStore, rule v1.RateRule, key v1.KeyFunc, reached *int) *gin.Engine {
	r := gin.New()
	r.Use(v1.ClientIP(""))
	r.POST("/login", v1.RateLimit(store, rule, key), func(c *gin.Context) { *reached++; okHandler(c) })
	return r
}

func TestRateLimit_AllowsUpToTheLimitThenBlocks(t *testing.T) {
	store, reached := newFakeStore(), 0
	r := limitedRouter(store, v1.RateRule{Name: "login", Limit: 3, Window: time.Minute}, v1.ByIP, &reached)
	from := fromAddr("203.0.113.9:1", nil)

	for i := 1; i <= 3; i++ {
		w := do(r, http.MethodPost, "/login", from)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d", i, w.Code)
		}
		if got := w.Header().Get("X-RateLimit-Remaining"); got != string(rune('0'+3-i)) {
			t.Errorf("request %d: remaining = %q", i, got)
		}
		if w.Header().Get("X-RateLimit-Limit") != "3" {
			t.Errorf("limit header = %q", w.Header().Get("X-RateLimit-Limit"))
		}
	}
	w := do(r, http.MethodPost, "/login", from)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request: status = %d, want 429", w.Code)
	}
	if w.Header().Get("Retry-After") != "90" {
		t.Errorf("Retry-After = %q", w.Header().Get("Retry-After"))
	}
	if !strings.Contains(w.Body.String(), `"code":"RATE_LIMITED"`) {
		t.Errorf("body = %s", w.Body.String())
	}
	if w.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("remaining = %q", w.Header().Get("X-RateLimit-Remaining"))
	}
	if reached != 3 {
		t.Errorf("handler ran %d times, want 3: blocked requests must not reach it", reached)
	}
}

func TestRateLimit_IPsAreCountedSeparately(t *testing.T) {
	store, reached := newFakeStore(), 0
	r := limitedRouter(store, v1.RateRule{Name: "login", Limit: 1, Window: time.Minute}, v1.ByIP, &reached)
	if w := do(r, http.MethodPost, "/login", fromAddr("203.0.113.1:1", nil)); w.Code != http.StatusOK {
		t.Fatal("first IP's first request must pass")
	}
	if w := do(r, http.MethodPost, "/login", fromAddr("203.0.113.1:1", nil)); w.Code != http.StatusTooManyRequests {
		t.Fatal("first IP's second request must be blocked")
	}
	if w := do(r, http.MethodPost, "/login", fromAddr("203.0.113.2:1", nil)); w.Code != http.StatusOK {
		t.Fatal("a different IP must not be affected")
	}
}

func TestRateLimit_BucketKeyFormat(t *testing.T) {
	store, reached := newFakeStore(), 0
	r := limitedRouter(store, v1.RuleLogin, v1.ByIP, &reached)
	do(r, http.MethodPost, "/login", fromAddr("203.0.113.9:1", nil))
	if len(store.keys) != 1 || store.keys[0] != "login:ip:203.0.113.9" {
		t.Errorf("keys = %v, want login:ip:203.0.113.9 (the format in 05-database-design.md)", store.keys)
	}
}

func TestRateLimit_IPv6IsGroupedBy64Prefix(t *testing.T) {
	store, reached := newFakeStore(), 0
	r := limitedRouter(store, v1.RateRule{Name: "login", Limit: 1, Window: time.Minute}, v1.ByIP, &reached)
	do(r, http.MethodPost, "/login", fromAddr("[2001:db8:1:2::1]:1", nil))
	w := do(r, http.MethodPost, "/login", fromAddr("[2001:db8:1:2:ffff:ffff:ffff:ffff]:1", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("two addresses in one /64 must share a bucket, got %d", w.Code)
	}
	if w := do(r, http.MethodPost, "/login", fromAddr("[2001:db8:1:3::1]:1", nil)); w.Code != http.StatusOK {
		t.Errorf("a different /64 must have its own bucket, got %d", w.Code)
	}
	if store.keys[0] != "login:ip:2001:db8:1:2::/64" {
		t.Errorf("key = %q", store.keys[0])
	}
}

func TestRateLimit_UnknownIPSharesOneStrictBucket(t *testing.T) {
	store, reached := newFakeStore(), 0
	r := limitedRouter(store, v1.RuleLogin, v1.ByIP, &reached)
	do(r, http.MethodPost, "/login", fromAddr("garbage", nil))
	if store.keys[0] != "login:ip:unknown" {
		t.Errorf("key = %q", store.keys[0])
	}
}

func TestRateLimit_ByUser(t *testing.T) {
	store := newFakeStore()
	uid := uuid.NewString()
	r := gin.New()
	r.POST("/pin", func(c *gin.Context) { c.Set(v1.CtxUserID, uid) }, v1.RateLimit(store, v1.RulePINVerify, v1.ByUser), okHandler)
	if w := do(r, http.MethodPost, "/pin", nil); w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if store.keys[0] != "pin:user:"+uid {
		t.Errorf("key = %q", store.keys[0])
	}
}

func TestRateLimit_ByUserWithoutAuthIs401(t *testing.T) {
	store, reached := newFakeStore(), 0
	r := limitedRouter(store, v1.RulePINVerify, v1.ByUser, &reached)
	w := do(r, http.MethodPost, "/login", nil)
	if w.Code != http.StatusUnauthorized || reached != 0 || len(store.keys) != 0 {
		t.Errorf("status=%d reached=%d keys=%v", w.Code, reached, store.keys)
	}
}

func TestRateLimit_ByUserRejectsMalformedUserID(t *testing.T) {
	r := gin.New()
	r.POST("/pin", func(c *gin.Context) { c.Set(v1.CtxUserID, "x:evil:key") }, v1.RateLimit(newFakeStore(), v1.RulePINVerify, v1.ByUser), okHandler)
	if w := do(r, http.MethodPost, "/pin", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("a non-UUID user id must not be usable as a bucket key, status = %d", w.Code)
	}
}

// If the counter cannot be reached the request is refused, never waved through.
func TestRateLimit_FailsClosedWhenStoreFails(t *testing.T) {
	store, reached := newFakeStore(), 0
	store.err = errors.New("connection refused to db.abc.supabase.co")
	r := limitedRouter(store, v1.RuleLogin, v1.ByIP, &reached)
	w := do(r, http.MethodPost, "/login", fromAddr("203.0.113.9:1", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
	if reached != 0 {
		t.Error("the handler must not run when the limiter fails")
	}
	if strings.Contains(w.Body.String(), "supabase") {
		t.Errorf("the store error leaked to the client: %s", w.Body.String())
	}
}

func TestRateLimit_PanicsOnInvalidRule(t *testing.T) {
	good := v1.RateRule{Name: "x", Limit: 1, Window: time.Minute}
	tests := map[string]func(){
		"nil store":     func() { v1.RateLimit(nil, good, v1.ByIP) },
		"nil key func":  func() { v1.RateLimit(newFakeStore(), good, nil) },
		"empty name":    func() { v1.RateLimit(newFakeStore(), v1.RateRule{Limit: 1, Window: time.Minute}, v1.ByIP) },
		"zero limit":    func() { v1.RateLimit(newFakeStore(), v1.RateRule{Name: "x", Window: time.Minute}, v1.ByIP) },
		"zero window":   func() { v1.RateLimit(newFakeStore(), v1.RateRule{Name: "x", Limit: 1}, v1.ByIP) },
		"negative rule": func() { v1.RateLimit(newFakeStore(), v1.RateRule{Name: "x", Limit: -1, Window: -time.Second}, v1.ByIP) },
	}
	for name, f := range tests {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected a panic at startup", name)
				}
			}()
			f()
		}()
	}
}
