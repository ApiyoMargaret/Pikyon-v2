-- migrations/000010_create_rate_limit_buckets.up.sql

-- Persists rate-limit state for auth/PIN routes across server restarts
-- with explicit TTL tracking to prevent unbounded table bloat.
CREATE TABLE rate_limit_buckets (
    bucket_key    TEXT PRIMARY KEY, -- e.g., "login:ip:1.2.3.4" or "pin:user:uuid"
    count         INTEGER NOT NULL DEFAULT 1,
    window_start  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '15 minutes')
);

-- Standalone index for lightweight cron sweepers/TTL garbage collection
CREATE INDEX idx_rate_limit_expires ON rate_limit_buckets(expires_at);

-- No RLS required: accessed only via service-role key from backend middleware