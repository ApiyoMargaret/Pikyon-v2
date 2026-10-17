-- migrations/000008_create_memory_views_reactions.down.sql

DROP POLICY IF EXISTS reactions_delete_own ON memory_reactions;
DROP POLICY IF EXISTS reactions_insert_recipient ON memory_reactions;
DROP POLICY IF EXISTS memory_reactions_select ON memory_reactions;
DROP TABLE IF EXISTS memory_reactions CASCADE;
DROP TYPE IF EXISTS reaction_type;

DROP POLICY IF EXISTS memory_views_insert_viewer ON memory_views;
DROP POLICY IF EXISTS memory_views_owner_select ON memory_views;
DROP TABLE IF EXISTS memory_views CASCADE;