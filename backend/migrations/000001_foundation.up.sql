-- 000001_foundation.up.sql
-- Foundation for database-level tenant isolation (ADR-018).
-- Creates: the `pikyon` helper schema, pikyon.current_user_id(), and the restricted
-- application role `pikyon_app` (NOBYPASSRLS). The backend sets
-- `app.current_user_id` per transaction; every RLS policy reads it via this function.

CREATE SCHEMA IF NOT EXISTS pikyon;

-- NULLIF handles the case where a transaction-local setting has been reset to ''
-- (Postgres leaves custom settings as '' rather than NULL after the first use).
CREATE OR REPLACE FUNCTION pikyon.current_user_id() RETURNS uuid
LANGUAGE sql STABLE
SET search_path = ''
AS $$ SELECT NULLIF(current_setting('app.current_user_id', true), '')::uuid $$;
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pikyon_app') THEN
    -- NOLOGIN here on purpose: a password must never live in a migration.
    -- Enable login once, manually:  ALTER ROLE pikyon_app WITH LOGIN PASSWORD '...';
    CREATE ROLE pikyon_app NOLOGIN NOBYPASSRLS;
  END IF;
END $$;

REVOKE ALL ON SCHEMA pikyon FROM PUBLIC;
GRANT USAGE ON SCHEMA pikyon TO pikyon_app;
REVOKE ALL ON FUNCTION pikyon.current_user_id() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION pikyon.current_user_id() TO pikyon_app;
GRANT USAGE ON SCHEMA public TO pikyon_app;