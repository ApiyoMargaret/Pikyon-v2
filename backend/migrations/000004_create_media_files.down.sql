-- migrations/000004_create_media_files.down.sql

DROP POLICY IF EXISTS media_files_owner_all ON media_files;
DROP TABLE IF EXISTS media_files CASCADE;
DROP TYPE IF EXISTS media_status;
DROP TYPE IF EXISTS media_type;