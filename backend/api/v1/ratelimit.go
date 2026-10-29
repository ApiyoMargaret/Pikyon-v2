package v1

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperr "github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/errors"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/response"
)

// This file holds client-IP resolution and the Postgres-backed rate limiter for the routes where
// brute force matters (06-api-design.md §6.13): register, login, forgot-password and PIN verify.

const (
	ctxClientIP = "client_ip"

	// CtxUserID is the Gin context key where the auth middleware stores the authenticated user's
	// ID. It must match auth.AuthUserIDKey.
	CtxUserID = "user_id"
)

// Rules from the API design's rate limit table. Name becomes the first part of the bucket key,
// for example "login:ip:203.0.113.9" or "pin:user:<uuid>".
var (
	RuleRegister       = RateRule{Name: "register", Limit: 5, Window: time.Hour}
	RuleLogin          = RateRule{Name: "login", Limit: 10, Window: 15 * time.Minute}
	RuleForgotPassword = RateRule{Name: "forgot", Limit: 3, Window: time.Hour}
	RulePINVerify      = RateRule{Name: "pin", Limit: 5, Window: 15 * time.Minute}
)

// ClientIP resolves the caller's IP address once per request and stores it for the rate limiter
// and for handlers (see ClientIPFrom).
//
// trustedHeader names a single-IP header that the hosting platform overwrites on every request,
// for example "CF-Connecting-IP" behind Cloudflare. Leave it empty to use the TCP connection's
// address only. Never name a header that a client can set freely, and never a list header such as
// X-Forwarded-For: the leftmost entry of that header is attacker-controlled. If the header is
// missing or not exactly one valid IP, the TCP address is used.
func ClientIP(trustedHeader string) gin.HandlerFunc {
	header := strings.TrimSpace(trustedHeader)
	return func(c *gin.Context) {
		c.Set(ctxClientIP, resolveClientIP(c.Request, header))
		c.Next()
	}
}

// ClientIPFrom returns the address resolved by ClientIP, or "" if unknown.
func ClientIPFrom(c *gin.Context) string {
	if v, ok := c.Get(ctxClientIP); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func resolveClientIP(r *http.Request, header string) string {
	if header != "" {
		if ip, ok := parseSingleIP(r.Header.Get(header)); ok {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip, ok := parseSingleIP(host); ok {
		return ip.String()
	}
	return ""
}

// parseSingleIP accepts exactly one IP address. Lists, ports, zones and garbage are rejected.
func parseSingleIP(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Addr{}, false
	}
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, false
	}
	return ip.WithZone("").Unmap(), true
}

// ipRateKey turns an address into a rate limit key part. IPv6 addresses are grouped by their /64
// prefix, because one customer usually controls a whole /64 and could otherwise dodge the limit
// by rotating addresses.
func ipRateKey(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "ip:unknown"
	}
	if addr.Is6() {
		return "ip:" + netip.PrefixFrom(addr, 64).Masked().String()
	}
	return "ip:" + addr.String()
}

// RateLimitStore counts attempts per key within a window. db.RateLimitStore implements it.
type RateLimitStore interface {
	Hit(ctx context.Context, key string, window time.Duration) (count int, retryAfter time.Duration, err error)
}

// RateRule is a limit of Limit attempts per Window.
type RateRule struct {
	Name   string
	Limit  int
	Window time.Duration
}

// KeyFunc picks who a request is counted against. It returns false when it cannot.
type KeyFunc func(c *gin.Context) (string, bool)

// ByIP counts requests per client IP. Requires the ClientIP middleware to run first.
func ByIP(c *gin.Context) (string, bool) {
	return ipRateKey(ClientIPFrom(c)), true
}

// ByUser counts requests per authenticated user. The auth middleware must run first.
func ByUser(c *gin.Context) (string, bool) {
	id, err := uuid.Parse(c.GetString(CtxUserID))
	if err != nil {
		return "", false
	}
	return "user:" + id.String(), true
}

// RateLimit enforces rule using store, counting requests by key.
//
// Over the limit, it answers 429 RATE_LIMITED with a Retry-After header. If the counter store
// fails it answers 503 and does NOT let the request through: these routes protect against
// brute force, so they fail closed. A rule used before authentication with ByUser answers 401.
//
// It panics at startup if the rule is invalid, so a typo cannot silently disable protection.
func RateLimit(store RateLimitStore, rule RateRule, key KeyFunc) gin.HandlerFunc {
	if store == nil || key == nil || rule.Name == "" || rule.Limit < 1 || rule.Window <= 0 {
		panic(fmt.Sprintf("v1: invalid rate limit rule %+v", rule))
	}
	return func(c *gin.Context) {
		k, ok := key(c)
		if !ok {
			response.Error(c, apperr.Unauthorized())
			return
		}
		count, retryAfter, err := store.Hit(c.Request.Context(), rule.Name+":"+k, rule.Window)
		if err != nil {
			response.Error(c, apperr.ServiceUnavailable(err))
			return
		}
		remaining := rule.Limit - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(rule.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if count > rule.Limit {
			response.Error(c, apperr.RateLimited(retryAfter))
			return
		}
		c.Next()
	}
}
