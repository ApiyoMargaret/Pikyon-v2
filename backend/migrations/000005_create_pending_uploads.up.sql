-- migrations/000005_create_pending_uploads.up.sql

CREATE TABLE pending_uploads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    memory_id       UUID REFERENCES memories(id) ON DELETE CASCADE,
    object_key      TEXT NOT NULL UNIQUE,
    media_type      media_type NOT NULL,
    file_size_bytes BIGINT,
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 hour',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficient hourly orphan cleanup cron querying
CREATE INDEX idx_pending_uploads_expires ON pending_uploads(expires_at);

-- Row Level Security
ALTER TABLE pending_uploads ENABLE ROW LEVEL SECURITY;

CREATE POLICY pending_uploads_own ON pending_uploads 
    FOR ALL 
    USING (auth.uid() = user_id)
    WITH CHECK (auth.uid() = user_id);