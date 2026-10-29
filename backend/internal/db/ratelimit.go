package db

import (
	"context"
	"fmt"
	"time"
)

// RateLimitStore is a Postgres-backed fixed-window rate limiter for the routes where brute
// force matters (register, login, forgot-password, PIN verify). State lives in the
// rate_limit_buckets table, so it survives restarts and is shared by every instance
// (06-api-design.md §6.13, ADR-018).
//
// It must be built on the ADMIN pool: rate_limit_buckets has deny-all RLS and no grants for the
// restricted app role.
type RateLimitStore struct {
	q Querier
}

// NewRateLimitStore returns a store that uses q, which should be Pools.Admin.
func NewRateLimitStore(q Querier) *RateLimitStore {
	return &RateLimitStore{q: q}
}

// hitSQL records one attempt atomically in a single statement, so concurrent requests can never
// both read the same count. If the bucket's window has expired it restarts at 1. All times come
// from the database clock (now()), never from the app server's clock.
const hitSQL = `
INSERT INTO rate_limit_buckets AS b (bucket_key, count, window_start, expires_at)
VALUES ($1, 1, now(), now() + make_interval(secs => $2::double precision))
ON CONFLICT (bucket_key) DO UPDATE SET
	count        = CASE WHEN b.expires_at <= now() THEN 1 ELSE b.count + 1 END,
	window_start = CASE WHEN b.expires_at <= now() THEN now() ELSE b.window_start END,
	expires_at   = CASE WHEN b.expires_at <= now() THEN now() + make_interval(secs => $2::double precision) ELSE b.expires_at END
RETURNING b.count, GREATEST(EXTRACT(EPOCH FROM (b.expires_at - now())), 0)::double precision`

// Hit records one attempt for key and returns how many attempts the current window has seen
// (including this one) and how long until the window resets.
func (s *RateLimitStore) Hit(ctx context.Context, key string, window time.Duration) (count int, retryAfter time.Duration, err error) {
	if key == "" || window <= 0 {
		return 0, 0, fmt.Errorf("db: rate limit hit needs a key and a positive window")
	}
	var secs float64
	if err := s.q.QueryRow(ctx, hitSQL, key, window.Seconds()).Scan(&count, &secs); err != nil {
		return 0, 0, fmt.Errorf("db: rate limit hit: %w", err)
	}
	return count, time.Duration(secs * float64(time.Second)), nil
}

// DeleteExpired removes buckets whose window has ended. Run it periodically so the table does not
// grow without bound. It returns the number of rows removed.
func (s *RateLimitStore) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := s.q.Exec(ctx, `DELETE FROM rate_limit_buckets WHERE expires_at <= now()`)
	if err != nil {
		return 0, fmt.Errorf("db: delete expired rate limit buckets: %w", err)
	}
	return tag.RowsAffected(), nil
}
