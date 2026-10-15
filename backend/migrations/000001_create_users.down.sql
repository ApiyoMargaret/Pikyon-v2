-- Rollback for users table
DROP POLICY IF EXISTS users_update_own ON users;
DROP POLICY IF EXISTS users_select_own ON users;
DROP TABLE IF EXISTS users CASCADE;