# ADR-001: Use Supabase as the database and storage provider

**Status:** Accepted

## Context
Pikyon needs a relational database, file storage for media, and basic infrastructure without taking on the operational overhead of managing a database server directly. The team is a solo developer with no dedicated DevOps capacity, and the project has an explicit no-paid-infrastructure constraint for the MVP.

## Decision
Use Supabase as the managed platform. Supabase **is** PostgreSQL underneath — a hosted Postgres instance plus a dashboard, Storage service and other bundled services. The Go backend connects to it exactly as it would connect to any PostgreSQL server (see ADR-002 for the driver choice). Pikyon does **not** use Supabase Auth (G3 correction); authentication is the custom Go implementation described in the security plan and ADR-016/ADR-019.

## Consequences
**Positive:** No database server to provision or patch; Storage supports pre-signed URLs, which the zero-direct-serving media architecture depends on; free tier sufficient for the MVP; standard PostgreSQL underneath means no proprietary query lock-in.
**Negative:** Free-tier caps (500MB DB, 1GB storage, 60 max connections) constrain scale before an upgrade is needed — documented in `docs/planning/12-constraints-risks.md`.
**Mitigation:** Two explicit pgxpool connection pools (restricted app role plus privileged role) stay within the connection cap; usage is monitored; upgrading is a plan change, not a migration.
**Related:** ADR-002, ADR-018.
