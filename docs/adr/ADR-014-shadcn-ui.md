# ADR-014: Adopt shadcn/ui

**Status:** Accepted

## Context
Needed accessible, unstyled component primitives that pair natively with Tailwind CSS without imposing a visual opinion — earlier mockup iterations suffered from a generic, templated look partly due to relying on ad-hoc styled elements rather than a consistent, well-built primitive set.

## Decision
Use shadcn/ui components (Radix primitives + Lucide icons), copied directly into `frontend/src/components/ui/` per shadcn's own distribution model — not installed as an npm dependency.

## Consequences
**Positive:** Free, MIT licensed, zero runtime dependency cost, full ownership/styling control since the source lives in-repo, accessible by default (Radix), pairs with the Inter/neutral design direction (ADR-015).
**Negative:** Slightly more initial setup than a pre-built component library; components must be added individually as needed.
**Cost:** $0.
