-- migrations/000003_create_memories.down.sql

-- 1. Drop the security policy first (it depends on the table)
DROP POLICY IF EXISTS memories_owner_all ON memories;

-- 2. Drop the table itself (CASCADE ensures dependent indexes are also cleaned up)
DROP TABLE IF EXISTS memories CASCADE;

-- 3. Drop the custom ENUM types last (since the table columns no longer use them)
DROP TYPE IF EXISTS memory_status;
DROP TYPE IF EXISTS memory_visibility;