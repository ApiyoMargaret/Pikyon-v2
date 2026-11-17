# ADR-002: Use pgx as the PostgreSQL driver

**Status:** Accepted

## Context
Go needs a driver to connect to the PostgreSQL instance underlying Supabase (ADR-001). The standard library's `database/sql` requires a third-party implementation; the two common choices are `lib/pq` (older, minimally maintained) and `pgx` (actively maintained, higher performance).

## Decision
Use `pgx` v5, via its `pgxpool` connection-pooling package.

## Consequences
**Positive:** Faster than `lib/pq`; native support for PostgreSQL-specific types; configurable pooling (`pgxpool`) — critical given Supabase's 60-connection cap; actively maintained; supports transaction-scoped session settings, which the RLS design relies on (ADR-018).
**Negative:** Slightly less universal tutorial coverage than `lib/pq`, a gap that has largely closed as pgx became the de facto standard.
**Related:** ADR-001 (Supabase is Postgres; pgx is how the backend talks to it), ADR-018 (two pools, user-scoped transactions).
