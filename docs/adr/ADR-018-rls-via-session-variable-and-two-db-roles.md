# ADR-018: Database-level tenant isolation via a per-transaction session variable and two database roles

**Status:** Accepted
**Supersedes:** The `auth.uid()` RLS policy pattern in `05-database-design.md` §5.6 and the RLS statements in `07-security-plan.md`.

## Context
The original design wrote every RLS policy against Supabase's `auth.uid()`. That function reads a Supabase Auth JWT, but Pikyon issues its own JWTs (ADR-016) and the Go backend connects to Postgres directly. In that setup:

1. `auth.uid()` is always NULL for backend connections, so the policies protected nothing.
2. The backend connected as the table owner, which bypasses RLS entirely. Tenant isolation was therefore enforced only by `WHERE user_id = $1` in application code and by the cross-tenant tests.
3. `auth.uid()` does not exist on plain Postgres, so the migrations failed in the CI Postgres container.
4. Supabase exposes the `public` schema through its Data API (PostgREST) to the `anon` and `authenticated` roles by default. Tables without RLS or with default grants were readable and writable by anyone holding the public anon key. This was observed on the production project (Security Advisor: 4 errors).

## Decision
1. **Two database roles, two connection pools.**
   - `pikyon_app`: `NOBYPASSRLS`, least-privilege table and column grants. Used for all normal, user-scoped requests.
   - The owner/privileged role: used only for migrations, login/register/refresh (no user context exists yet), cron jobs (cross-tenant), resolving public share-link tokens, creating notifications for other users, and updating AI job results.
2. **Per-transaction user context.** Each user-scoped request runs in a transaction that begins with `SELECT set_config('app.current_user_id', $1, true)` (transaction-local). The user ID comes only from the verified JWT.
3. **Policies read the session variable** through `pikyon.current_user_id()` (NULL or empty means no rows), written as `user_id = (SELECT pikyon.current_user_id())` and scoped `TO pikyon_app`. The function sets `search_path = ''` (Supabase Security Advisor requirement).
4. **Ownership is also enforced structurally.** `memories` has `UNIQUE (id, user_id)`; `media_files`, `pending_uploads`, `ai_jobs` and `shared_access` reference `(memory_id, user_id)` / `(memory_id, owner_id)` with a composite foreign key. A child row can never point at another user's memory, even if application code is wrong.
5. **Secrets are unreadable by the app role.** Column-level grants exclude `password_hash`, `totp_secret`, `pin_hash` and `sessions.refresh_token_hash`.
6. **The Supabase Data API is not used.** All grants to `anon` and `authenticated` are revoked and default privileges are cleared (migration 000012, a no-op on plain Postgres). `rate_limit_buckets` and `schema_migrations` have deny-all RLS. This corrects the handoff's "no RLS required" for `rate_limit_buckets`.
7. **No public/anonymous policy on `shared_access`.** A policy cannot see the token the caller presents, so it would expose every active share to every user. Public-link lookup uses the privileged connection with an exact token match.
8. **Portability changes.** `gen_random_uuid()` replaces `uuid_generate_v4()` (no extension). The share token default uses two CSPRNG-backed UUIDs instead of `gen_random_bytes()` (no `pgcrypto`). Emails are stored lowercase with a `CHECK (email = lower(email))`. All migrations now run on plain Postgres.
9. **Verification.** `migrations/tests/rls_isolation.sql` asserts cross-tenant isolation at the database level. It passes on the production Supabase project and is intended to run in CI after migrating.

## Consequences
**Positive:** A missing `WHERE user_id = ...` in application code can no longer leak another user's rows; an SQL injection on the app connection cannot read password or token hashes or touch other tenants; the public Data API surface is closed; migrations are CI-portable.
**Negative:** Two connection pools and two connection strings to manage; every user-scoped repository method must run inside the user-scoped transaction (a small `Querier`/`WithUser` helper in `internal/db`); one extra `set_config` statement per request (negligible).
**Operational notes:** `pikyon_app` is created `NOLOGIN` by the migration. Enable login once by hand with a strong password (never in a migration). On Supabase's pooler the username becomes `pikyon_app.<project-ref>`. Use the session pooler (port 5432) for migrations.
**Related:** ADR-002 (pgx), ADR-012 (golang-migrate), ADR-017, ADR-019.

## Docs to update in the same commit
- `05-database-design.md` §5.6: replace the `auth.uid()` pattern with this one; update every table's policy; correct `rate_limit_buckets` to deny-all RLS; document `rotated_at` on `sessions`.
- `07-security-plan.md`: describe the two-role model and that `anon`/`authenticated` have no access.
- `12-constraints-risks.md`: note the two-pool operational requirement.