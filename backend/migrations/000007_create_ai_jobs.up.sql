-- 000007_create_ai_jobs.up.sql
CREATE TYPE ai_job_type AS ENUM (
    'caption_from_media', 'speech_summarize', 'speech_polish',
    'social_captions', 'mood_detection', 'tag_suggestion'
);
CREATE TYPE ai_job_status AS ENUM ('queued', 'processing', 'completed', 'failed');

CREATE TABLE ai_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id     UUID NOT NULL,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_type      ai_job_type NOT NULL,
    status        ai_job_status NOT NULL DEFAULT 'queued',
    input_data    JSONB,
    result_data   JSONB,
    error_message TEXT,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_jobs_memory_owner_fk FOREIGN KEY (memory_id, user_id)
        REFERENCES memories (id, user_id) ON DELETE CASCADE
);

CREATE INDEX idx_ai_jobs_user_status ON ai_jobs(user_id, status);
CREATE INDEX idx_ai_jobs_memory_id ON ai_jobs(memory_id);

ALTER TABLE ai_jobs ENABLE ROW LEVEL SECURITY;

CREATE POLICY ai_jobs_owner_all ON ai_jobs FOR ALL TO pikyon_app
    USING (user_id = (SELECT pikyon.current_user_id()))
    WITH CHECK (user_id = (SELECT pikyon.current_user_id()));

-- Request handlers may enqueue and read jobs. Only the background worker (privileged
-- connection) updates status/result, so the app role gets no UPDATE/DELETE.
GRANT SELECT, INSERT ON ai_jobs TO pikyon_app;