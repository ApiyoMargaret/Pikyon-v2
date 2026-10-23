-- 000001_foundation.down.sql
-- The pikyon_app role is cluster-level and is intentionally NOT dropped here.
DROP FUNCTION IF EXISTS pikyon.current_user_id();
DROP SCHEMA IF EXISTS pikyon;