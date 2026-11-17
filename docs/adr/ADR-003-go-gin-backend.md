# ADR-003: Use Go with Gin for the backend

**Status:** Accepted

## Context
Pikyon needs a backend for a REST API handling authentication, media pre-signing, AI job orchestration and database access, with good performance on free-tier hosting (Render) where resource limits are tight.

## Decision
Use Go 1.22+ as the language, with the Gin HTTP framework for routing and middleware. The module currently targets the Go toolchain pinned in `backend/go.mod`; CI must use the same version.

## Consequences
**Positive:** Low memory footprint and fast startup suit free-tier hosting; native goroutines provide lightweight concurrency without extra infrastructure — used directly for AI job processing and background cleanup, avoiding a task queue at MVP scale; static typing catches errors at compile time; Gin is a minimal, production-proven framework.
**Negative:** Smaller ecosystem of pre-built packages than Node.js/Python for some integrations (for example, no official Gemini SDK for Go; direct HTTP calls are used).
**Related:** ADR-002, ADR-009.
