# ADR-004: Use React with TypeScript for the frontend

**Status:** Accepted

## Context
Pikyon needs a frontend for a media-heavy, interactive single-page application with near-real-time AI feedback (SSE/polling) and complex forms (memory creation, sharing, settings).

## Decision
Use React 18+ with TypeScript, built with Vite.

## Consequences
**Positive:** Large ecosystem and community support; TypeScript types are generated from the backend's swagger spec via openapi-typescript, keeping the API contract in sync automatically; Vite gives fast local iteration; strict typing catches integration breaks at compile time.
**Negative:** None significant relative to alternatives at this scale — React was chosen primarily for ecosystem maturity and familiarity.
**Related:** ADR-010 (swagger source of truth), ADR-014, ADR-015.
