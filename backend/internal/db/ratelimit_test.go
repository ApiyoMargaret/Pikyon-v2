package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/db"
)

// --- Rate limit store (integration) ---

func newStore(t *testing.T) (*db.RateLimitStore, *db.Pools) {
	t.Helper()
	p := connect(t, 2)
	return db.NewRateLimitStore(p.Admin), p
}

func uniqueKey(t *testing.T, p *db.Pools) string {
	t.Helper()
	key := "test:" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = p.Admin.Exec(context.Background(), `DELETE FROM rate_limit_buckets WHERE bucket_key = $1`, key)
	})
	return key
}

func TestIntegration_RateLimit_CountsWithinWindow(t *testing.T) {
	s, p := newStore(t)
	key := uniqueKey(t, p)
	ctx := context.Background()
	for want := 1; want <= 4; want++ {
		n, retry, err := s.Hit(ctx, key, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if n != want {
			t.Errorf("hit %d: count = %d", want, n)
		}
		if retry <= 0 || retry > time.Minute {
			t.Errorf("retryAfter = %v, want within (0, 1m]", retry)
		}
	}
}

func TestIntegration_RateLimit_KeysAreIndependent(t *testing.T) {
	s, p := newStore(t)
	a, b := uniqueKey(t, p), uniqueKey(t, p)
	ctx := context.Background()
	s.Hit(ctx, a, time.Minute)
	s.Hit(ctx, a, time.Minute)
	n, _, err := s.Hit(ctx, b, time.Minute)
	if err != nil || n != 1 {
		t.Errorf("a different key must start at 1: n=%d err=%v", n, err)
	}
}

func TestIntegration_RateLimit_WindowResets(t *testing.T) {
	s, p := newStore(t)
	key := uniqueKey(t, p)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		s.Hit(ctx, key, 300*time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	n, _, err := s.Hit(ctx, key, 300*time.Millisecond)
	if err != nil || n != 1 {
		t.Errorf("after the window expired the count must restart at 1: n=%d err=%v", n, err)
	}
}

// Many simultaneous attempts must be counted exactly once each: no lost updates.
func TestIntegration_RateLimit_ConcurrentHitsAreAllCounted(t *testing.T) {
	s, p := newStore(t)
	key := uniqueKey(t, p)
	ctx := context.Background()
	const workers = 40

	counts := make(chan int, workers)
	for i := 0; i < workers; i++ {
		go func() {
			n, _, err := s.Hit(ctx, key, time.Minute)
			if err != nil {
				n = -1
			}
			counts <- n
		}()
	}
	seen := map[int]bool{}
	for i := 0; i < workers; i++ {
		n := <-counts
		if n < 1 || seen[n] {
			t.Fatalf("count %d returned twice or failed: concurrent updates were lost", n)
		}
		seen[n] = true
	}
	for want := 1; want <= workers; want++ {
		if !seen[want] {
			t.Errorf("count %d never returned", want)
		}
	}
}

func TestIntegration_RateLimit_DeleteExpired(t *testing.T) {
	s, p := newStore(t)
	ctx := context.Background()
	old, live := uniqueKey(t, p), uniqueKey(t, p)
	s.Hit(ctx, old, 100*time.Millisecond)
	s.Hit(ctx, live, time.Hour)
	time.Sleep(300 * time.Millisecond)

	if _, err := s.DeleteExpired(ctx); err != nil {
		t.Fatal(err)
	}
	var oldN, liveN int
	p.Admin.QueryRow(ctx, `SELECT count(*) FROM rate_limit_buckets WHERE bucket_key = $1`, old).Scan(&oldN)
	p.Admin.QueryRow(ctx, `SELECT count(*) FROM rate_limit_buckets WHERE bucket_key = $1`, live).Scan(&liveN)
	if oldN != 0 || liveN != 1 {
		t.Errorf("expired rows kept=%d (want 0), live rows kept=%d (want 1)", oldN, liveN)
	}
}

func TestIntegration_RateLimit_RejectsBadInput(t *testing.T) {
	s, _ := newStore(t)
	if _, _, err := s.Hit(context.Background(), "", time.Minute); err == nil {
		t.Error("an empty key must be rejected")
	}
	if _, _, err := s.Hit(context.Background(), "k", 0); err == nil {
		t.Error("a zero window must be rejected")
	}
}

// The restricted app role must have no access to the buckets (deny-all RLS, no grants).
func TestIntegration_RateLimit_AppRoleCannotTouchBuckets(t *testing.T) {
	p := connect(t, 2)
	if _, err := p.App.Exec(context.Background(), `SELECT count(*) FROM rate_limit_buckets`); err == nil {
		t.Error("the app role must not be able to read rate_limit_buckets")
	}
	appStore := db.NewRateLimitStore(p.App)
	if _, _, err := appStore.Hit(context.Background(), "x:app", time.Minute); err == nil {
		t.Error("a store built on the app pool must fail, proving it has to use the admin pool")
	}
}
