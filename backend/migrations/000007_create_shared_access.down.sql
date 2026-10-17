-- migrations/000007_create_shared_access.down.sql

DROP POLICY IF EXISTS shared_access_public_token_select ON shared_access;
DROP POLICY IF EXISTS shared_access_recipient_select ON shared_access;
DROP POLICY IF EXISTS shared_access_owner ON shared_access;
DROP TABLE IF EXISTS shared_access CASCADE;