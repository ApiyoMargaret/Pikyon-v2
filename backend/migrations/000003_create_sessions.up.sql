-- 000003_create_sessions.up.sql
CREATE TABLE sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- SHA-256 (hex) of the raw refresh token (ADR-019). Looked up directly by hash.
    refresh_token_hash  TEXT NOT NULL UNIQUE,
    rotated_from        UUID REFERENCES sessions(id) ON DELETE SET NULL,
    rotated_at          TIMESTAMPTZ,      -- set when this session's token was exchanged for a new one
    device_info         TEXT,
    ip_address          INET,
    expires_at          TIMESTAMPTZ NOT NULL,
    is_revoked          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sessions_token_hash_sha256 CHECK (refresh_token_hash ~ '^[0-9a-f]{64}$')
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_rotated_from ON sessions(rotated_from);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;

-- Session creation, token rotation and refresh happen on the privileged connection
-- (they run before a user context exists). The app role may only list/revoke/delete its own.
CREATE POLICY sessions_select_own ON sessions FOR SELECT TO pikyon_app
    USING (user_id = (SELECT pikyon.current_user_id()));
CREATE POLICY sessions_update_own ON sessions FOR UPDATE TO pikyon_app
    USING (user_id = (SELECT pikyon.current_user_id()))
    WITH CHECK (user_id = (SELECT pikyon.current_user_id()));
CREATE POLICY sessions_delete_own ON sessions FOR DELETE TO pikyon_app
    USING (user_id = (SELECT pikyon.current_user_id()));

GRANT SELECT (id, user_id, rotated_from, rotated_at, device_info, ip_address,
              expires_at, is_revoked, created_at, last_used_at) ON sessions TO pikyon_app;
GRANT UPDATE (is_revoked, last_used_at) ON sessions TO pikyon_app;
GRANT DELETE ON sessions TO pikyon_app;