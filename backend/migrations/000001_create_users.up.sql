-- Ensure UUID extension is available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email               TEXT NOT NULL UNIQUE,
    email_verified      BOOLEAN NOT NULL DEFAULT FALSE,
    name                TEXT NOT NULL,
    avatar_url          TEXT,
    password_hash       TEXT,             -- NULL if Google-OAuth-only
    google_id           TEXT UNIQUE,      -- NULL if password-only
    totp_secret         TEXT,             -- AES-256 encrypted
    totp_enabled        BOOLEAN NOT NULL DEFAULT FALSE,
    pin_hash            TEXT,             -- bcrypt, independent of password
    storage_used_bytes  BIGINT NOT NULL DEFAULT 0,
    password_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_must_have_auth CHECK (password_hash IS NOT NULL OR google_id IS NOT NULL)
);

-- Enable Row Level Security for Users
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Drop existing policies if re-running migration, then create them securely
DROP POLICY IF EXISTS users_select_own ON users;
DROP POLICY IF EXISTS users_update_own ON users;

-- SELECT policy restricts visibility to own record
CREATE POLICY users_select_own ON users 
    FOR SELECT 
    USING (auth.uid() = id);

-- UPDATE policy enforces both target row visibility AND post-mutation constraints (prevents ID/email hijacking)
CREATE POLICY users_update_own ON users 
    FOR UPDATE 
    USING (auth.uid() = id) 
    WITH CHECK (auth.uid() = id);
