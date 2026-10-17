-- migrations/000005_create_pending_uploads.down.sql

DROP POLICY IF EXISTS pending_uploads_own ON pending_uploads;
DROP TABLE IF EXISTS pending_uploads CASCADE;