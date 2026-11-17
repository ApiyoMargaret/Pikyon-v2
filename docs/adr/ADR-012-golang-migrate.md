# ADR-012: Use golang-migrate for database migrations

**Status:** Accepted

## Context
Schema changes need version control, repeatable application across dev/staging/production, and a tested rollback path — especially with per-environment isolation and two database roles (ADR-018).

## Decision
Use `golang-migrate` with sequentially numbered `.up.sql`/`.down.sql` pairs in `backend/migrations/`. Migrations run as an **isolated step before the application binary starts** — never inside `main()`, which risks row-locking across replicas. Migrations must never be edited after merge; corrections are appended as new migrations. CI verifies an up → down → up round trip for changes under `backend/migrations/`.

## Consequences
**Positive:** Explicit, reviewable SQL history; works cleanly with pgx/Postgres; the round-trip check closes the untested-rollback gap.
**Negative:** Manual discipline is required; a skipped or renumbered file breaks the sequence for every environment.
**Related:** ADR-002, ADR-009, ADR-018.
