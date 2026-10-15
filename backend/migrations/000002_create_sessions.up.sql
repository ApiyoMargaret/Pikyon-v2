
CREATE TABLE IF NOT EXISTS sessions (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  TEXT NOT NULL UNIQUE,  -- bcrypt, raw token never stored
    rotated_from        UUID REFERENCES sessions(id) ON DELETE SET NULL,
    device_info         TEXT,
    ip_address          INET,
    expires_at          TIMESTAMPTZ NOT NULL,
    is_revoked          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enable Row Level Security for Sessions
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;

-- Drop existing policies if re-running migration, then create them securely
DROP POLICY IF EXISTS sessions_select_own ON sessions;
DROP POLICY IF EXISTS sessions_update_own ON sessions;
DROP POLICY IF EXISTS sessions_delete_own ON sessions;

-- RLS Policies
CREATE POLICY sessions_select_own ON sessions 
    FOR SELECT 
    USING (auth.uid() = user_id);

CREATE POLICY sessions_update_own ON sessions 
    FOR UPDATE 
    USING (auth.uid() = user_id) 
    WITH CHECK (auth.uid() = user_id);

CREATE POLICY sessions_delete_own ON sessions 
    FOR DELETE 
    USING (auth.uid() = user_id);

-- Indexes for performance and foreign key cascade optimization
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_rotated_from ON sessions(rotated_from);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);