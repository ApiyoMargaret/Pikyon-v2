-- rls_isolation.sql — DB-level cross-tenant tests for ADR-018.
-- Run AFTER all migrations, as a superuser/owner:  psql "$ADMIN_URL" -v ON_ERROR_STOP=1 -f rls_isolation.sql
-- Everything runs inside one transaction that is rolled back, so it leaves no data behind.
-- Any failed ASSERT aborts with a message.
\set ON_ERROR_STOP on
BEGIN;

-- Fixtures (owner connection, RLS bypassed)
INSERT INTO users (id, email, name, password_hash) VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', 'a@test.dev', 'A', 'x'),
  ('bbbbbbbb-0000-0000-0000-000000000002', 'b@test.dev', 'B', 'x'),
  ('cccccccc-0000-0000-0000-000000000003', 'c@test.dev', 'C', 'x');
INSERT INTO memories (id, user_id, title) VALUES
  ('a1a1a1a1-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'A memory'),
  ('b2b2b2b2-0000-0000-0000-000000000002', 'bbbbbbbb-0000-0000-0000-000000000002', 'B memory'),
  ('c3c3c3c3-0000-0000-0000-000000000003', 'cccccccc-0000-0000-0000-000000000003', 'C memory');
INSERT INTO sessions (user_id, refresh_token_hash, expires_at) VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', repeat('a', 64), NOW() + INTERVAL '1 day');
INSERT INTO shared_access (id, memory_id, owner_id, recipient_email, recipient_id) VALUES
  ('5a5a5a5a-0000-0000-0000-000000000001', 'a1a1a1a1-0000-0000-0000-000000000001',
   'aaaaaaaa-0000-0000-0000-000000000001', 'b@test.dev', 'bbbbbbbb-0000-0000-0000-000000000002');
-- A third user's share that neither A nor B may ever see (catches over-permissive policies)
INSERT INTO shared_access (memory_id, owner_id, recipient_email) VALUES
  ('c3c3c3c3-0000-0000-0000-000000000003', 'cccccccc-0000-0000-0000-000000000003', 'z@test.dev');
INSERT INTO rate_limit_buckets (bucket_key) VALUES ('login:ip:1.2.3.4');

-- Helper: asserts that running a statement as the current role raises an error.
CREATE FUNCTION pg_temp.denied(stmt text) RETURNS boolean LANGUAGE plpgsql AS $$
BEGIN
  EXECUTE stmt;
  RETURN FALSE;
EXCEPTION WHEN insufficient_privilege OR foreign_key_violation OR check_violation OR unique_violation THEN
  RETURN TRUE;
END $$;
GRANT EXECUTE ON FUNCTION pg_temp.denied(text) TO pikyon_app;

SET LOCAL ROLE pikyon_app;

-- 0. No user context => sees nothing, and does not error
SELECT set_config('app.current_user_id', '', true);
DO $$ BEGIN
  ASSERT (SELECT count(*) FROM memories) = 0, 'no user context must see zero memories';
  ASSERT (SELECT count(*) FROM users) = 0, 'no user context must see zero users';
END $$;

-- 1. User A sees only A's data
SELECT set_config('app.current_user_id', 'aaaaaaaa-0000-0000-0000-000000000001', true);
DO $$ BEGIN
  ASSERT (SELECT count(*) FROM memories) = 1, 'A sees exactly one memory';
  ASSERT (SELECT title FROM memories) = 'A memory', 'A sees own memory';
  ASSERT (SELECT count(*) FROM users) = 1, 'A sees only own user row';
  ASSERT (SELECT count(*) FROM sessions) = 1, 'A sees own session';
  ASSERT (SELECT count(*) FROM shared_access) = 1, 'A (owner) sees the share';
END $$;

-- 2. A cannot read or touch B's memory
DO $$ BEGIN
  ASSERT (SELECT count(*) FROM memories WHERE id = 'b2b2b2b2-0000-0000-0000-000000000002') = 0, 'A cannot read B memory';
  UPDATE memories SET title = 'hacked' WHERE id = 'b2b2b2b2-0000-0000-0000-000000000002';
  ASSERT NOT FOUND, 'A cannot update B memory';
  DELETE FROM memories WHERE id = 'b2b2b2b2-0000-0000-0000-000000000002';
  ASSERT NOT FOUND, 'A cannot delete B memory';
END $$;

-- 3. A cannot forge ownership on insert
DO $$ BEGIN
  ASSERT pg_temp.denied($q$INSERT INTO memories (user_id, title) VALUES ('bbbbbbbb-0000-0000-0000-000000000002', 'forged')$q$),
    'A cannot insert a memory owned by B';
END $$;

-- 4. A cannot attach media / uploads / jobs to B's memory (composite FK), even as itself
DO $$ BEGIN
  ASSERT pg_temp.denied($q$INSERT INTO media_files (memory_id, user_id, object_key, media_type, mime_type, file_size_bytes)
    VALUES ('b2b2b2b2-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'k1', 'image', 'image/png', 10)$q$),
    'media on another user''s memory must be rejected';
  ASSERT pg_temp.denied($q$INSERT INTO pending_uploads (user_id, memory_id, object_key, media_type)
    VALUES ('aaaaaaaa-0000-0000-0000-000000000001', 'b2b2b2b2-0000-0000-0000-000000000002', 'k2', 'image')$q$),
    'pending upload on another user''s memory must be rejected';
  ASSERT pg_temp.denied($q$INSERT INTO ai_jobs (memory_id, user_id, job_type)
    VALUES ('b2b2b2b2-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'mood_detection')$q$),
    'ai job on another user''s memory must be rejected';
  ASSERT pg_temp.denied($q$INSERT INTO shared_access (memory_id, owner_id, recipient_email)
    VALUES ('b2b2b2b2-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'c@test.dev')$q$),
    'share of another user''s memory must be rejected';
END $$;

-- 5. A's legitimate writes work
INSERT INTO media_files (memory_id, user_id, object_key, media_type, mime_type, file_size_bytes)
  VALUES ('a1a1a1a1-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'ok-1', 'image', 'image/png', 10);
INSERT INTO pending_uploads (user_id, memory_id, object_key, media_type)
  VALUES ('aaaaaaaa-0000-0000-0000-000000000001', 'a1a1a1a1-0000-0000-0000-000000000001', 'ok-2', 'image');
UPDATE users SET name = 'A renamed' WHERE id = 'aaaaaaaa-0000-0000-0000-000000000001';
DO $$ BEGIN ASSERT (SELECT name FROM users) = 'A renamed', 'A can update own profile'; END $$;

-- 6. Secrets and privileged tables are unreachable for the app role
DO $$ BEGIN
  ASSERT pg_temp.denied('SELECT password_hash FROM users'), 'app role cannot read password_hash';
  ASSERT pg_temp.denied('SELECT pin_hash FROM users'), 'app role cannot read pin_hash';
  ASSERT pg_temp.denied('SELECT totp_secret FROM users'), 'app role cannot read totp_secret';
  ASSERT pg_temp.denied('SELECT refresh_token_hash FROM sessions'), 'app role cannot read refresh_token_hash';
  ASSERT pg_temp.denied($q$INSERT INTO users (email, name, password_hash) VALUES ('c@test.dev', 'C', 'x')$q$), 'app role cannot create users';
  ASSERT pg_temp.denied('SELECT * FROM rate_limit_buckets'), 'app role cannot touch rate_limit_buckets';
  ASSERT pg_temp.denied($q$UPDATE users SET email_verified = TRUE$q$), 'app role cannot change email_verified';
  ASSERT pg_temp.denied($q$UPDATE ai_jobs SET status = 'completed'$q$), 'app role cannot complete ai jobs';
  ASSERT pg_temp.denied($q$INSERT INTO notifications (user_id, type, title, message) VALUES ('aaaaaaaa-0000-0000-0000-000000000001', 'system', 't', 'm')$q$), 'app role cannot insert notifications';
END $$;

-- 7. User B: sees the share as recipient, but NOT A's memory rows or A's media
SELECT set_config('app.current_user_id', 'bbbbbbbb-0000-0000-0000-000000000002', true);
DO $$ BEGIN
  ASSERT (SELECT count(*) FROM memories) = 1, 'B sees only own memory';
  ASSERT (SELECT count(*) FROM media_files) = 0, 'B cannot see A media';
  ASSERT (SELECT count(*) FROM shared_access) = 1, 'B sees the share addressed to B';
  ASSERT (SELECT count(*) FROM sessions) = 0, 'B cannot see A sessions';
  ASSERT (SELECT count(*) FROM pending_uploads) = 0, 'B cannot see A pending uploads';
END $$;

-- 8. A view/reaction can only be recorded by the recipient of an active share
INSERT INTO memory_views (memory_id, viewer_id, share_id)
  VALUES ('a1a1a1a1-0000-0000-0000-000000000001', 'bbbbbbbb-0000-0000-0000-000000000002', '5a5a5a5a-0000-0000-0000-000000000001');
INSERT INTO memory_reactions (memory_id, user_id, share_id, reaction)
  VALUES ('a1a1a1a1-0000-0000-0000-000000000001', 'bbbbbbbb-0000-0000-0000-000000000002', '5a5a5a5a-0000-0000-0000-000000000001', 'heart');
DO $$ BEGIN
  ASSERT pg_temp.denied($q$INSERT INTO memory_views (memory_id, viewer_id, share_id)
    VALUES ('a1a1a1a1-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', '5a5a5a5a-0000-0000-0000-000000000001')$q$),
    'a viewer cannot record a view as someone else';
END $$;

-- 9. The owner of the share sees the view; B (viewer) cannot read the analytics
SELECT set_config('app.current_user_id', 'aaaaaaaa-0000-0000-0000-000000000001', true);
DO $$ BEGIN
  ASSERT (SELECT count(*) FROM memory_views) = 1, 'owner sees views on own shares';
  ASSERT (SELECT count(*) FROM memory_reactions) = 1, 'memory owner sees reactions';
END $$;
SELECT set_config('app.current_user_id', 'bbbbbbbb-0000-0000-0000-000000000002', true);
DO $$ BEGIN ASSERT (SELECT count(*) FROM memory_views) = 0, 'viewer cannot read owner analytics'; END $$;

RESET ROLE;

-- 10. Supabase API roles (if present) have no access at all
DO $$
DECLARE r text;
BEGIN
  FOREACH r IN ARRAY ARRAY['anon', 'authenticated'] LOOP
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = r) THEN
      ASSERT NOT has_table_privilege(r, 'public.users', 'SELECT'), r || ' must not read users';
      ASSERT NOT has_table_privilege(r, 'public.sessions', 'SELECT'), r || ' must not read sessions';
      ASSERT NOT has_table_privilege(r, 'public.memories', 'SELECT'), r || ' must not read memories';
      ASSERT NOT has_table_privilege(r, 'public.rate_limit_buckets', 'SELECT'), r || ' must not read rate_limit_buckets';
    END IF;
  END LOOP;
END $$;

-- 11. Every public table has RLS enabled
DO $$ BEGIN
  ASSERT (SELECT count(*) FROM pg_tables WHERE schemaname = 'public' AND NOT rowsecurity) = 0,
    'every public table must have RLS enabled';
END $$;

\echo 'ALL RLS ISOLATION CHECKS PASSED'
ROLLBACK;