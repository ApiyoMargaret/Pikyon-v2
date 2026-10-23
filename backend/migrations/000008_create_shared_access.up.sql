-- 000008_create_shared_access.up.sql
CREATE TABLE shared_access (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id       UUID NOT NULL,
    owner_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_email TEXT NOT NULL,
    recipient_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    -- 64 hex chars from two CSPRNG-backed UUIDv4s (~244 bits). No extension required.
    share_token     TEXT NOT NULL UNIQUE
                    DEFAULT replace(gen_random_uuid()::text || gen_random_uuid()::text, '-', ''),
    expires_at      TIMESTAMPTZ,
    is_revoked      BOOLEAN NOT NULL DEFAULT FALSE,
    view_count      INTEGER NOT NULL DEFAULT 0,
    max_views       INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shared_access_not_self CHECK (owner_id != recipient_id),
    CONSTRAINT shared_access_email_lowercase CHECK (recipient_email = lower(recipient_email)),
    CONSTRAINT shared_access_memory_owner_fk FOREIGN KEY (memory_id, owner_id)
        REFERENCES memories (id, user_id) ON DELETE CASCADE
);

CREATE INDEX idx_shared_access_owner_id ON shared_access(owner_id);
CREATE INDEX idx_shared_access_recipient_id ON shared_access(recipient_id);
CREATE UNIQUE INDEX idx_shared_access_unique_share
    ON shared_access(memory_id, recipient_email) WHERE is_revoked = FALSE;

ALTER TABLE shared_access ENABLE ROW LEVEL SECURITY;

CREATE POLICY shared_access_owner ON shared_access FOR ALL TO pikyon_app
    USING (owner_id = (SELECT pikyon.current_user_id()))
    WITH CHECK (owner_id = (SELECT pikyon.current_user_id()));

CREATE POLICY shared_access_recipient_select ON shared_access FOR SELECT TO pikyon_app
    USING (recipient_id = (SELECT pikyon.current_user_id()));

-- NOTE: there is deliberately NO policy for public/anonymous token lookup. A policy cannot
-- see the token the caller presents, so it would expose every active share to every user.
-- Public-link resolution runs on the privileged connection with an exact token match.
GRANT SELECT, INSERT, UPDATE, DELETE ON shared_access TO pikyon_app;