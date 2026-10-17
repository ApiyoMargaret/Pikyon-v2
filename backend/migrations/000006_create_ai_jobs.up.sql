-- migrations/000006_create_ai_jobs.up.sql

CREATE TYPE ai_job_type AS ENUM (
    'caption_from_media',
    'speech_summarize',
    'speech_polish',
    'social_captions',
    'mood_detection',
    'tag_suggestion'
);

CREATE TYPE ai_job_status AS ENUM (
    'queued',
    'processing',
    'completed',
    'failed'
);

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

-- Index for efficient job status/queue lookups
CREATE INDEX idx_ai_jobs_user_status ON ai_jobs(user_id, status);

-- Row Level Security with tenant isolation and mutation safety
ALTER TABLE ai_jobs ENABLE ROW LEVEL SECURITY;

CREATE POLICY ai_jobs_owner_all ON ai_jobs 
    FOR ALL 
    USING (auth.uid() = user_id)
    WITH CHECK (auth.uid() = user_id);