# ADR-013: Accept `unsafe-inline` in CSP `style-src` for MVP

**Status:** Accepted (MVP), revisit in future sprint

## Context
Content Security Policy restricts what can execute/load on the page to prevent XSS. `style-src 'unsafe-inline'` weakens this by allowing any inline `style` attribute or `<style>` block to run, which is a known XSS vector.

Framer Motion (animation) and Tailwind CSS both rely on inline styles in normal usage. Removing `unsafe-inline` requires either nonce-based CSP (server generates a per-request nonce, every inline style tagged with it) or hash-based CSP (every inline style's content hashed at build time and allow-listed) — both are non-trivial build pipeline changes.

## Decision
Accept `'unsafe-inline'` in `style-src` for MVP.

## Consequences
**Positive:** No build pipeline complexity added before launch; ships on schedule.
**Negative:** Slightly weaker XSS defense-in-depth on the style vector specifically (script-src remains strict).
**Mitigation:** Documented explicitly as a known, accepted tradeoff — not an oversight. Revisit with nonce-based CSP once Framer Motion/Tailwind inline style usage is audited for static extraction.
