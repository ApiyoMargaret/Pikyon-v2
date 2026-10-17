-- migrations/000004_create_media_files.up.sql

CREATE TYPE media_type AS ENUM ('image', 'video', 'audio', 'document');
CREATE TYPE media_status AS ENUM ('pending', 'confirmed', 'failed');

CREATE TABLE media_files (
    id                     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id              UUID NOT NULL REFERENCES memories(id) ON DELETE RESTRICT,
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

-- Index for foreign key lookup on memory relationship
CREATE INDEX idx_media_files_memory_id ON media_files(memory_id);

-- Row Level Security with bulletproof tenant isolation and mutation checks
ALTER TABLE media_files ENABLE ROW LEVEL SECURITY;

CREATE POLICY media_files_owner_all ON media_files 
    FOR ALL 
    USING (auth.uid() = user_id)
    WITH CHECK (auth.uid() = user_id);