# ADR-010: Use swaggo/swag for API documentation generation

**Status:** Accepted

## Context
The project commits to a code-first pipeline: Go structs and handler comments generate the API spec, which generates frontend TypeScript types, so the frontend and backend contract cannot silently drift.

## Decision
Use `swaggo/swag` to generate `swagger.json` from annotated Go handlers, output exclusively to `backend/docs/` (the single source of truth). `openapi-typescript` generates `frontend/src/types/api.ts`. CI enforces that swagger is regenerated whenever handlers change.

## Consequences
**Positive:** Documentation is generated from the actual code, not maintained separately; frontend types follow automatically; the CI staleness check catches drift before merge.
**Negative:** Requires disciplined annotations on every handler — a missing annotation produces an incomplete spec.
**Related:** ADR-004; `docs/planning/03-tech-stack.md` §3.9.
