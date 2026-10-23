-- 000011_create_rate_limit_buckets.up.sql
-- Persists rate-limit state for auth/PIN routes across restarts (04-system-architecture §4.10).
CREATE TABLE rate_limit_buckets (
    bucket_key    TEXT PRIMARY KEY,  -- e.g. "login:ip:1.2.3.4" or "pin:user:<uuid>"
    count         INTEGER NOT NULL DEFAULT 1,
    window_start  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '15 minutes')
);
CREATE INDEX idx_rate_limit_expires ON rate_limit_buckets(expires_at);

-- Deny-all RLS with no policies and no grants for any API/app role. Only the privileged
-- connection (table owner) can touch it. This corrects the handoff's "no RLS required":
-- on Supabase, a public table without RLS is readable and writable through the Data API.
ALTER TABLE rate_limit_buckets ENABLE ROW LEVEL SECURITY;