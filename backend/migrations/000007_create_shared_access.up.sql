-- migrations/000007_create_shared_access.up.sql

CREATE TABLE shared_access (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id       UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    owner_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_email TEXT NOT NULL,
    recipient_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    share_token     TEXT NOT NULL UNIQUE DEFAULT encode(gen_random_bytes(32), 'hex'),
    expires_at      TIMESTAMPTZ,
    is_revoked      BOOLEAN NOT NULL DEFAULT FALSE,
    view_count      INTEGER NOT NULL DEFAULT 0,
    max_views       INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shared_access_not_self CHECK (owner_id != recipient_id)
);

-- Standalone indexes for foreign key cascade sweeps and performance
CREATE INDEX idx_shared_access_owner_id ON shared_access(owner_id);
CREATE INDEX idx_shared_access_recipient_id ON shared_access(recipient_id);

CREATE UNIQUE INDEX idx_shared_access_unique_share
    ON shared_access(memory_id, recipient_email) WHERE is_revoked = FALSE;

-- Row Level Security
ALTER TABLE shared_access ENABLE ROW LEVEL SECURITY;

-- Policy 1: Owner full access
CREATE POLICY shared_access_owner ON shared_access 
    FOR ALL 
    USING (auth.uid() = owner_id)
    WITH CHECK (auth.uid() = owner_id);

-- Policy 2: Recipient select access
CREATE POLICY shared_access_recipient_select ON shared_access 
    FOR SELECT 
    USING (auth.uid() = recipient_id);

-- Policy 3: Public/Anonymous token lookup for valid, active, and view-available share links
CREATE POLICY shared_access_public_token_select ON shared_access 
    FOR SELECT 
    USING (
        is_revoked = FALSE 
        AND (expires_at IS NULL OR expires_at > NOW())
        AND (max_views IS NULL OR view_count < max_views)
    );