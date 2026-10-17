-- migrations/000006_create_ai_jobs.down.sql

DROP POLICY IF EXISTS ai_jobs_owner_all ON ai_jobs;
DROP TABLE IF EXISTS ai_jobs CASCADE;
DROP TYPE IF EXISTS ai_job_status;
DROP TYPE IF EXISTS ai_job_type;