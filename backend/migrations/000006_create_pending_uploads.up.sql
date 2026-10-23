-- 000006_create_pending_uploads.up.sql
-- Prevents orphaned Storage objects when the client's confirm call never arrives (ADR-017).
CREATE TABLE pending_uploads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    memory_id       UUID,
    object_key      TEXT NOT NULL UNIQUE,
    media_type      media_type NOT NULL,
    file_size_bytes BIGINT,
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 hour',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- If memory_id is set, it must belong to user_id. (NULL memory_id skips the check.)
    CONSTRAINT pending_uploads_memory_owner_fk FOREIGN KEY (memory_id, user_id)
        REFERENCES memories (id, user_id) ON DELETE CASCADE
);

CREATE INDEX idx_pending_uploads_expires ON pending_uploads(expires_at);
CREATE INDEX idx_pending_uploads_user_id ON pending_uploads(user_id);

ALTER TABLE pending_uploads ENABLE ROW LEVEL SECURITY;

CREATE POLICY pending_uploads_owner_all ON pending_uploads FOR ALL TO pikyon_app
    USING (user_id = (SELECT pikyon.current_user_id()))
    WITH CHECK (user_id = (SELECT pikyon.current_user_id()));

-- The hourly orphan-cleanup cron is cross-tenant and runs on the privileged connection.
GRANT SELECT, INSERT, UPDATE, DELETE ON pending_uploads TO pikyon_app;