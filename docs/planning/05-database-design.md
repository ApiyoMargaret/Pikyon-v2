# Section 5: Database Design

## 5.1 Overview

```
Database:     PostgreSQL 15+ (via Supabase)
Driver:       pgx v5
Migrations:   golang-migrate (isolated job, never in main())
Pool:         pgxpool (MaxConns=25, MinConns=5)
Security:     Row Level Security on ALL tables
Convention:   snake_case tables/columns, UUID v4 primary keys,
              TIMESTAMPTZ everywhere (UTC)
```

## 5.2 Entity Relationship Overview

```
users
 ├── memories
 │    ├── media_files
 │    ├── ai_jobs
 │    └── shared_access
 │         ├── memory_views
 │         └── memory_reactions
 ├── sessions
 ├── notifications
 ├── pending_uploads
 └── rate_limit_buckets
```

## 5.3 Tables

### users
```sql
CREATE TABLE users (
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
    -- ^ added: compared against JWT "pwd_ver" claim to invalidate
    --   all issued access tokens instantly on password reset/logout-all
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_must_have_auth
        CHECK (password_hash IS NOT NULL OR google_id IS NOT NULL)
);
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY users_select_own ON users FOR SELECT USING (auth.uid() = id);
CREATE POLICY users_update_own ON users FOR UPDATE USING (auth.uid() = id);
```

### sessions
```sql
CREATE TABLE sessions (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  TEXT NOT NULL UNIQUE,  -- bcrypt, raw token never stored
    rotated_from        UUID REFERENCES sessions(id) ON DELETE SET NULL,
    -- ^ added: tracks refresh-token rotation chain for anomaly
    --   detection, required since refresh tokens now live in
    --   localStorage (see 04-system-architecture.md §4.4)
    device_info         TEXT,
    ip_address          INET,
    expires_at          TIMESTAMPTZ NOT NULL,
    is_revoked          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY sessions_select_own ON sessions FOR SELECT USING (auth.uid() = user_id);
CREATE POLICY sessions_delete_own ON sessions FOR DELETE USING (auth.uid() = user_id);
```

### memories
```sql
CREATE TYPE memory_visibility AS ENUM ('private', 'public');
CREATE TYPE memory_status AS ENUM ('active', 'trashed');

CREATE TABLE memories (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title                TEXT NOT NULL,
    story                TEXT,
    location             TEXT,
    memory_date          DATE,
    visibility           memory_visibility NOT NULL DEFAULT 'private',
    status               memory_status NOT NULL DEFAULT 'active',
    is_pin_locked        BOOLEAN NOT NULL DEFAULT FALSE,
    is_favorite          BOOLEAN NOT NULL DEFAULT FALSE,
    -- added post-planning-review: supports the Favorites nav
    -- section (08-ui-ux-plan.md §8.8) — no new table needed,
    -- a simple toggle on the existing memory record
    ai_caption           TEXT,
    ai_mood              TEXT,
    ai_tags              TEXT[],
    caption_twitter      TEXT,
    caption_instagram    TEXT,
    caption_linkedin     TEXT,
    trashed_at           TIMESTAMPTZ,
    permanent_delete_at  TIMESTAMPTZ,   -- trashed_at + 30 days
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT memories_title_length CHECK (char_length(title) BETWEEN 1 AND 255)
);
CREATE INDEX idx_memories_user_status ON memories(user_id, status);
CREATE INDEX idx_memories_memory_date ON memories(memory_date DESC) WHERE status = 'active';
CREATE INDEX idx_memories_permanent_delete ON memories(permanent_delete_at) WHERE status = 'trashed';
CREATE INDEX idx_memories_search ON memories USING GIN(
    to_tsvector('english', COALESCE(title,'') || ' ' || COALESCE(story,'') || ' ' || COALESCE(location,'')));
ALTER TABLE memories ENABLE ROW LEVEL SECURITY;
CREATE POLICY memories_owner_all ON memories FOR ALL USING (auth.uid() = user_id);
```

### media_files
```sql
CREATE TYPE media_type AS ENUM ('image', 'video', 'audio', 'document');
CREATE TYPE media_status AS ENUM ('pending', 'confirmed', 'failed');

CREATE TABLE media_files (
    id                     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id              UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    user_id                UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_key             TEXT NOT NULL UNIQUE,   -- {user_id}/{memory_id}/{uuid}.{ext}
    media_type             media_type NOT NULL,
    mime_type              TEXT NOT NULL,           -- backend-verified, never client-trusted
    file_size_bytes        BIGINT NOT NULL,          -- backend-verified, never client-trusted
    compressed_size_bytes  BIGINT,
    width                  INTEGER,
    height                 INTEGER,
    duration_seconds       INTEGER,
    thumbnail_key          TEXT,
    status                 media_status NOT NULL DEFAULT 'pending',
    confirmed_at           TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE media_files ENABLE ROW LEVEL SECURITY;
CREATE POLICY media_files_owner_all ON media_files FOR ALL USING (auth.uid() = user_id);
```

### pending_uploads
```sql
-- Prevents orphaned Supabase Storage files when a client's
-- upload confirmation call never arrives (connection drop,
-- closed browser). See 04-system-architecture.md §4.5.
CREATE TABLE pending_uploads (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    memory_id       UUID REFERENCES memories(id) ON DELETE CASCADE,
    object_key      TEXT NOT NULL UNIQUE,
    media_type      media_type NOT NULL,
    file_size_bytes BIGINT,
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 hour',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_pending_uploads_expires ON pending_uploads(expires_at);
ALTER TABLE pending_uploads ENABLE ROW LEVEL SECURITY;
CREATE POLICY pending_uploads_own ON pending_uploads FOR ALL USING (auth.uid() = user_id);
```

### ai_jobs
```sql
CREATE TYPE ai_job_type AS ENUM
    ('caption_from_media','speech_summarize','speech_polish','social_captions','mood_detection','tag_suggestion');
CREATE TYPE ai_job_status AS ENUM ('queued','processing','completed','failed');

CREATE TABLE ai_jobs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id     UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_type      ai_job_type NOT NULL,
    status        ai_job_status NOT NULL DEFAULT 'queued',
    input_data    JSONB,
    result_data   JSONB,
    error_message TEXT,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE ai_jobs ENABLE ROW LEVEL SECURITY;
CREATE POLICY ai_jobs_owner_all ON ai_jobs FOR ALL USING (auth.uid() = user_id);
```

### shared_access
```sql
CREATE TABLE shared_access (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id       UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    owner_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_email TEXT NOT NULL,
    recipient_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    share_token     TEXT NOT NULL UNIQUE DEFAULT encode(gen_random_bytes(32),'hex'),
    expires_at      TIMESTAMPTZ,
    is_revoked      BOOLEAN NOT NULL DEFAULT FALSE,
    view_count      INTEGER NOT NULL DEFAULT 0,
    max_views       INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shared_access_not_self CHECK (owner_id != recipient_id)
);
CREATE UNIQUE INDEX idx_shared_access_unique_share
    ON shared_access(memory_id, recipient_email) WHERE is_revoked = FALSE;
ALTER TABLE shared_access ENABLE ROW LEVEL SECURITY;
CREATE POLICY shared_access_owner ON shared_access FOR ALL USING (auth.uid() = owner_id);
CREATE POLICY shared_access_recipient_select ON shared_access FOR SELECT USING (auth.uid() = recipient_id);
```

### memory_views / memory_reactions
```sql
CREATE TABLE memory_views (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    viewer_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_id    UUID NOT NULL REFERENCES shared_access(id) ON DELETE CASCADE,
    viewed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address  INET
);
ALTER TABLE memory_views ENABLE ROW LEVEL SECURITY;
CREATE POLICY memory_views_owner_select ON memory_views FOR SELECT
    USING (auth.uid() = (SELECT owner_id FROM shared_access WHERE id = share_id));
CREATE POLICY memory_views_insert_viewer ON memory_views FOR INSERT WITH CHECK (auth.uid() = viewer_id);

CREATE TYPE reaction_type AS ENUM ('heart','wow','sad');
CREATE TABLE memory_reactions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_id    UUID NOT NULL REFERENCES shared_access(id) ON DELETE CASCADE,
    reaction    reaction_type NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT memory_reactions_unique UNIQUE (memory_id, user_id, share_id)
);
ALTER TABLE memory_reactions ENABLE ROW LEVEL SECURITY;
CREATE POLICY reactions_insert_recipient ON memory_reactions FOR INSERT WITH CHECK (auth.uid() = user_id);
CREATE POLICY reactions_delete_own ON memory_reactions FOR DELETE USING (auth.uid() = user_id);
```

### notifications
```sql
CREATE TYPE notification_type AS ENUM
    ('memory_viewed','memory_reacted','storage_warning','share_accepted','system');
CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        notification_type NOT NULL,
    title       TEXT NOT NULL,
    message     TEXT NOT NULL,
    metadata    JSONB,
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_unread ON notifications(user_id, created_at DESC) WHERE is_read = FALSE;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY notifications_owner_all ON notifications FOR ALL USING (auth.uid() = user_id);
```

### rate_limit_buckets
```sql
-- Added: persists rate-limit state for auth/PIN routes across
-- Render restarts (in-memory Ristretto is wiped on every restart,
-- reopening a brute-force window). See 04-system-architecture.md §4.10.
CREATE TABLE rate_limit_buckets (
    bucket_key   TEXT PRIMARY KEY,   -- e.g. "login:ip:1.2.3.4" or "pin:user:<uuid>"
    count        INTEGER NOT NULL DEFAULT 1,
    window_start TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- No RLS: accessed only via service-role key from backend middleware
```

## 5.4 Migration Execution Strategy
See `04-system-architecture.md` §4.12. Summary: isolated job before app binary starts, never inside `main()`, sequential numbering, up/down round-trip verified in staging CI.

## 5.5 Connection Pool Configuration
```go
config.MaxConns          = 25
config.MinConns          = 5
config.MaxConnLifetime   = 30 * time.Minute
config.MaxConnIdleTime   = 5 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute
```

## 5.6 Row Level Security Strategy
Every table has RLS enabled. Pattern: SELECT/INSERT/UPDATE/DELETE scoped to `auth.uid()`. Service-role key (bypasses RLS) used only by backend cron jobs — never exposed to frontend. Cross-tenant access is verified with **mandatory automated tests**, not manual checks — see `09-testing-strategy.md`.

## 5.7 Scheduled Cron Jobs

| Job | Schedule | Action |
|---|---|---|
| Orphaned upload cleanup | Hourly | Delete expired `pending_uploads` + Storage objects |
| Permanent trash deletion | Daily 02:00 UTC | Delete memories past `permanent_delete_at` |
| Expired session cleanup | Daily 03:00 UTC | Delete expired/revoked sessions |
| Expired share cleanup | Daily 04:00 UTC | Revoke expired share links |
| Storage warning | Daily 06:00 UTC | Notify users at ≥80% quota |
