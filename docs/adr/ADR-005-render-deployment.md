# ADR-005: Use Render for backend hosting

**Status:** Accepted

## Context
Pikyon needs hosting for the Go backend that supports Docker-based deployment, has a usable free tier, and integrates with GitHub CI/CD, consistent with the no-paid-infrastructure MVP constraint.

## Decision
Use Render for backend hosting, deployed from the single canonical multi-stage `backend/Dockerfile`. Automatic Git deploys are disabled; deploys are triggered by a deploy hook only after the isolated migration step succeeds (see `docs/planning/11-deployment-plan.md` §11.2).

## Consequences
**Positive:** Free tier supports a containerized Go service; built-in health checks and log streaming provide MVP-level observability; far simpler than raw cloud infrastructure.
**Negative:** Free tier sleeps after 15 minutes of inactivity, causing cold-start latency and dropped SSE connections — directly informed the polling-first AI job design and is recorded as an accepted MVP limitation.
**Mitigation:** Health-check gated, sequential deploys; polling fallback for AI jobs.
**Related:** ADR-009, ADR-011, ADR-016.
