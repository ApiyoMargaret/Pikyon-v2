-- migrations/000008_create_memory_views_reactions.up.sql

-- 1. Memory Views Table
CREATE TABLE memory_views (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    viewer_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_id    UUID NOT NULL REFERENCES shared_access(id) ON DELETE CASCADE,
    viewed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address  INET
);

-- Standalone indexes for foreign key performance
CREATE INDEX idx_memory_views_memory_id ON memory_views(memory_id);
CREATE INDEX idx_memory_views_share_id ON memory_views(share_id);

ALTER TABLE memory_views ENABLE ROW LEVEL SECURITY;

-- Memory owners can view analytics for their shared items
CREATE POLICY memory_views_owner_select ON memory_views 
    FOR SELECT 
    USING (auth.uid() = (SELECT owner_id FROM shared_access WHERE id = share_id));

-- Viewers can insert view records only if the share_id securely matches the memory_id and is active
CREATE POLICY memory_views_insert_viewer ON memory_views 
    FOR INSERT 
    WITH CHECK (
        auth.uid() = viewer_id
        AND EXISTS (
            SELECT 1 FROM shared_access
            WHERE shared_access.id = share_id
            AND shared_access.memory_id = memory_views.memory_id
            AND shared_access.is_revoked = FALSE
        )
    );


-- 2. Memory Reactions Table & Enum
CREATE TYPE reaction_type AS ENUM ('heart', 'wow', 'sad');

CREATE TABLE memory_reactions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_id    UUID NOT NULL REFERENCES shared_access(id) ON DELETE CASCADE,
    reaction    reaction_type NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT memory_reactions_unique UNIQUE (memory_id, user_id, share_id)
);

-- Standalone index for user lookups (composite unique index covers memory_id/share_id leftmost paths)
CREATE INDEX idx_memory_reactions_user_id ON memory_reactions(user_id);

ALTER TABLE memory_reactions ENABLE ROW LEVEL SECURITY;

-- Users and memory owners can read reactions
CREATE POLICY memory_reactions_select ON memory_reactions
    FOR SELECT
    USING (
        auth.uid() = user_id
        OR EXISTS (
            SELECT 1 FROM memories 
            WHERE memories.id = memory_reactions.memory_id 
            AND memories.user_id = auth.uid()
        )
    );

-- Recipients/users can insert reactions securely tied to a valid active share link
CREATE POLICY reactions_insert_recipient ON memory_reactions 
    FOR INSERT 
    WITH CHECK (
        auth.uid() = user_id
        AND EXISTS (
            SELECT 1 FROM shared_access
            WHERE shared_access.id = share_id
            AND shared_access.memory_id = memory_reactions.memory_id
            AND shared_access.is_revoked = FALSE
        )
    );

-- Users can delete their own reactions
CREATE POLICY reactions_delete_own ON memory_reactions 
    FOR DELETE 
    USING (auth.uid() = user_id);