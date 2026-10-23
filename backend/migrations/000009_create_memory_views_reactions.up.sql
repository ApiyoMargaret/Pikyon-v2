-- 000009_create_memory_views_reactions.up.sql

-- Part 1: memory_views (who viewed a shared memory)

CREATE TABLE memory_views (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    viewer_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_id    UUID NOT NULL REFERENCES shared_access(id) ON DELETE CASCADE,
    viewed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address  INET
);
CREATE INDEX idx_memory_views_memory_id ON memory_views(memory_id);
CREATE INDEX idx_memory_views_share_id ON memory_views(share_id);
CREATE INDEX idx_memory_views_viewer_id ON memory_views(viewer_id);

ALTER TABLE memory_views ENABLE ROW LEVEL SECURITY;

-- Owners see analytics for their own shares. (shared_access is itself RLS-filtered, so the
-- subquery only matches shares the current user owns.)
CREATE POLICY memory_views_owner_select ON memory_views FOR SELECT TO pikyon_app
    USING (EXISTS (
        SELECT 1 FROM shared_access sa
        WHERE sa.id = memory_views.share_id
          AND sa.owner_id = (SELECT pikyon.current_user_id())
    ));

-- A viewer may record a view only against an active share that is visible to them
-- (they are its recipient) and that matches the memory.
CREATE POLICY memory_views_insert_viewer ON memory_views FOR INSERT TO pikyon_app
    WITH CHECK (
        viewer_id = (SELECT pikyon.current_user_id())
        AND EXISTS (
            SELECT 1 FROM shared_access sa
            WHERE sa.id = memory_views.share_id
              AND sa.memory_id = memory_views.memory_id
              AND sa.recipient_id = (SELECT pikyon.current_user_id())
              AND sa.is_revoked = FALSE
              AND (sa.expires_at IS NULL OR sa.expires_at > NOW())
        )
    );

GRANT SELECT, INSERT ON memory_views TO pikyon_app;

-- Part 2: memory_reactions (heart / wow / sad on a shared memory)

CREATE TYPE reaction_type AS ENUM ('heart', 'wow', 'sad');

CREATE TABLE memory_reactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_id    UUID NOT NULL REFERENCES shared_access(id) ON DELETE CASCADE,
    reaction    reaction_type NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT memory_reactions_unique UNIQUE (memory_id, user_id, share_id)
);
CREATE INDEX idx_memory_reactions_user_id ON memory_reactions(user_id);
CREATE INDEX idx_memory_reactions_share_id ON memory_reactions(share_id);

ALTER TABLE memory_reactions ENABLE ROW LEVEL SECURITY;

CREATE POLICY memory_reactions_select ON memory_reactions FOR SELECT TO pikyon_app
    USING (
        user_id = (SELECT pikyon.current_user_id())
        OR EXISTS (
            SELECT 1 FROM memories m
            WHERE m.id = memory_reactions.memory_id
              AND m.user_id = (SELECT pikyon.current_user_id())
        )
    );

CREATE POLICY memory_reactions_insert ON memory_reactions FOR INSERT TO pikyon_app
    WITH CHECK (
        user_id = (SELECT pikyon.current_user_id())
        AND EXISTS (
            SELECT 1 FROM shared_access sa
            WHERE sa.id = memory_reactions.share_id
              AND sa.memory_id = memory_reactions.memory_id
              AND sa.recipient_id = (SELECT pikyon.current_user_id())
              AND sa.is_revoked = FALSE
              AND (sa.expires_at IS NULL OR sa.expires_at > NOW())
        )
    );

CREATE POLICY memory_reactions_delete_own ON memory_reactions FOR DELETE TO pikyon_app
    USING (user_id = (SELECT pikyon.current_user_id()));

GRANT SELECT, INSERT, DELETE ON memory_reactions TO pikyon_app;