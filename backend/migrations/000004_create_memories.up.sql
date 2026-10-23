-- 000004_create_memories.up.sql
CREATE TYPE memory_visibility AS ENUM ('private', 'public');
CREATE TYPE memory_status AS ENUM ('active', 'trashed');

CREATE TABLE memories (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title               TEXT NOT NULL,
    story               TEXT,
    location            TEXT,
    memory_date         DATE,
    visibility          memory_visibility NOT NULL DEFAULT 'private',
    status              memory_status NOT NULL DEFAULT 'active',
    is_pin_locked       BOOLEAN NOT NULL DEFAULT FALSE,
    is_favorite         BOOLEAN NOT NULL DEFAULT FALSE,
    ai_caption          TEXT,
    ai_mood             TEXT,
    ai_tags             TEXT[],
    caption_twitter     TEXT,
    caption_instagram   TEXT,
    caption_linkedin    TEXT,
    trashed_at          TIMESTAMPTZ,
    permanent_delete_at TIMESTAMPTZ,      -- trashed_at + 30 days
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT memories_title_length CHECK (char_length(title) BETWEEN 1 AND 255),
    CONSTRAINT memories_trash_consistency CHECK (
        (status = 'trashed') = (trashed_at IS NOT NULL AND permanent_delete_at IS NOT NULL)
    ),
    -- Target for composite foreign keys on child tables: a child row can only reference a
    -- memory that has the SAME owner, enforced by the database itself (see ADR-018).
    CONSTRAINT memories_id_user_id_key UNIQUE (id, user_id)
);

CREATE INDEX idx_memories_user_status ON memories(user_id, status);
CREATE INDEX idx_memories_memory_date ON memories(memory_date DESC) WHERE status = 'active';
CREATE INDEX idx_memories_permanent_delete ON memories(permanent_delete_at) WHERE status = 'trashed';
CREATE INDEX idx_memories_search ON memories USING GIN(
    to_tsvector('english', COALESCE(title, '') || ' ' || COALESCE(story, ''))
);

ALTER TABLE memories ENABLE ROW LEVEL SECURITY;

CREATE POLICY memories_owner_all ON memories FOR ALL TO pikyon_app
    USING (user_id = (SELECT pikyon.current_user_id()))
    WITH CHECK (user_id = (SELECT pikyon.current_user_id()));

GRANT SELECT, INSERT, UPDATE, DELETE ON memories TO pikyon_app;