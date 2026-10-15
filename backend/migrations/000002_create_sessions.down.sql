-- Rollback for sessions table
DROP POLICY IF EXISTS sessions_delete_own ON sessions;
DROP POLICY IF EXISTS sessions_select_own ON sessions;
DROP TABLE IF EXISTS sessions CASCADE;