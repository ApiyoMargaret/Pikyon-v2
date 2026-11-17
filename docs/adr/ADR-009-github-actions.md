# ADR-009: Use GitHub Actions for CI/CD

**Status:** Accepted

## Context
Pikyon follows a Scrum workflow with a monorepo, feature branches merging to `dev`, and `dev` merging to `main`, needing automated linting, tests, generated-contract checks and deployment gating at each stage.

## Decision
Use GitHub Actions for all CI/CD: `ci.yml` (PR gate), `cd-staging.yml` (merge to `dev`), `cd-production.yml` (merge to `main`). CI uses no secrets: integration tests use a Postgres service container and external services are mocked. Deploy workflows use GitHub Environments (`staging`, `production`) with the unsuffixed variable names defined in the handoff, Part 4.

## Consequences
**Positive:** Free at this scale; lives in the repository; PR status checks and branch protection directly support the workflow; environment-scoped secrets keep deploy credentials separated per stage.
**Negative:** Workflow YAML complexity grows over time — mitigated by keeping each file focused on one pipeline stage.
**Related:** ADR-005, ADR-012.
