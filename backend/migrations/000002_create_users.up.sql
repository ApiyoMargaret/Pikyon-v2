-- 000002_create_users.up.sql
CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email               TEXT NOT NULL UNIQUE,
    email_verified      BOOLEAN NOT NULL DEFAULT FALSE,
    name                TEXT NOT NULL,
    avatar_url          TEXT,
    password_hash       TEXT,             -- bcrypt cost 12; NULL if Google-OAuth-only
    google_id           TEXT UNIQUE,      -- NULL if password-only
    totp_secret         TEXT,             -- AES-256 encrypted
    totp_enabled        BOOLEAN NOT NULL DEFAULT FALSE,
    pin_hash            TEXT,             -- bcrypt, independent of password
    storage_used_bytes  BIGINT NOT NULL DEFAULT 0,
    password_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_must_have_auth CHECK (password_hash IS NOT NULL OR google_id IS NOT NULL),
    -- Emails are normalised to lowercase by the application; the DB rejects anything else
    -- so "A@x.com" and "a@x.com" can never both register.
    CONSTRAINT users_email_lowercase CHECK (email = lower(email)),
    CONSTRAINT users_storage_nonnegative CHECK (storage_used_bytes >= 0)
);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- The app role can only ever see/modify its own row. Registration and login run on the
-- privileged connection (no user exists yet), so there is deliberately no INSERT/DELETE policy.
CREATE POLICY users_select_own ON users FOR SELECT TO pikyon_app
    USING (id = (SELECT pikyon.current_user_id()));
CREATE POLICY users_update_own ON users FOR UPDATE TO pikyon_app
    USING (id = (SELECT pikyon.current_user_id()))
    WITH CHECK (id = (SELECT pikyon.current_user_id()));

-- Column-level grants: secrets (password_hash, totp_secret, pin_hash) are unreadable by the
-- app role, so an injection bug on that connection cannot exfiltrate them.
GRANT SELECT (id, email, email_verified, name, avatar_url, google_id, totp_enabled,
              storage_used_bytes, password_changed_at, is_active, created_at, updated_at)
    ON users TO pikyon_app;
GRANT UPDATE (name, avatar_url, storage_used_bytes, updated_at) ON users TO pikyon_app;