-- 000005_create_media_files.up.sql
CREATE TYPE media_type AS ENUM ('image', 'video', 'audio', 'document');
CREATE TYPE media_status AS ENUM ('pending', 'confirmed', 'failed');

CREATE TABLE media_files (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id              UUID NOT NULL,
    user_id                UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_key             TEXT NOT NULL UNIQUE,   -- {user_id}/{memory_id}/{uuid}.{ext}
    media_type             media_type NOT NULL,
    mime_type              TEXT NOT NULL,          -- backend-verified, never client-trusted (ADR-017)
    file_size_bytes        BIGINT NOT NULL,        -- backend-verified, never client-trusted (ADR-017)
    compressed_size_bytes  BIGINT,
    width                  INTEGER,
    height                 INTEGER,
    duration_seconds       INTEGER,
    thumbnail_key          TEXT,
    status                 media_status NOT NULL DEFAULT 'pending',
    confirmed_at           TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT media_files_size_positive CHECK (file_size_bytes > 0),
    -- Owner of the media must be the owner of the memory.
    CONSTRAINT media_files_memory_owner_fk FOREIGN KEY (memory_id, user_id)
        REFERENCES memories (id, user_id) ON DELETE RESTRICT
);

CREATE INDEX idx_media_files_memory_id ON media_files(memory_id);
CREATE INDEX idx_media_files_user_id ON media_files(user_id);

ALTER TABLE media_files ENABLE ROW LEVEL SECURITY;

CREATE POLICY media_files_owner_all ON media_files FOR ALL TO pikyon_app
    USING (user_id = (SELECT pikyon.current_user_id()))
    WITH CHECK (user_id = (SELECT pikyon.current_user_id()));

GRANT SELECT, INSERT, UPDATE, DELETE ON media_files TO pikyon_app;