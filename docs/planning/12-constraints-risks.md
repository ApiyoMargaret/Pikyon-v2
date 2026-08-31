# Section 12: Constraints & Risks

## 12.1 Technical Constraints

| Constraint | Impact | Mitigation |
|---|---|---|
| Supabase free tier: 500MB DB, 1GB storage, 60 max connections | Limits total users/memories before upgrade needed | pgxpool capped at 25 conns; monitor usage, documented upgrade path |
| Render free tier: sleeps after 15min inactivity | Cold starts add latency; SSE connections drop | Polling fallback for AI jobs (see 04-system-architecture.md §4.7); acceptable for MVP |
| Web Speech API: Chrome/Edge only | Firefox/Safari users can't use speech-to-text | Plain text input fallback shown when API unavailable |
| Gemini free tier: 15 req/min, 1500/day | AI features could hit ceiling with real usage | AI endpoints rate-limited at 20/hour/user, well under ceiling |
| Resend free tier: 3000 emails/month | Sharing/notification emails capped | Sufficient for MVP scale; monitor and upgrade if exceeded |
| No custom domain (budget decision) | Refresh token forced into localStorage instead of httpOnly cookie — weaker XSS posture, mitigated by mandatory rotation | Documented tradeoff (07-security-plan.md §7.11); revisit when budget allows (~$3-15/year) |
| Video compression not client-side (FFmpeg.wasm dropped) | Videos capped at 100MB raw upload until server-side worker ships | Server-side Go worker deferred to Sprint 2+ (04-system-architecture.md §4.6) |

## 12.2 Scope Constraints (explicitly out of MVP)
Mobile native app, multi-language support, semantic/vector search, video editing, face recognition, Apple OAuth, Redis-backed rate limiting/caching, custom domain.

## 12.3 Risk Register

| # | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| R1 | Video compression infra (server-side worker) delayed past Sprint 2 | Medium | Medium | 100MB raw upload cap as interim; ship without compression, add later |
| R2 | OAuth/2FA edge cases blow Sprint 1 timeline | Medium | Low (mitigated) | 2FA moved to Sprint 5 (see 10-sprint-plan.md) |
| R3 | AI SSE streaming unreliable on Render free tier | Medium | Low (mitigated) | Polling-first sequencing + fallback (see 04-system-architecture.md §4.7) |
| R4 | Solo/small team velocity vs 8-sprint estimate | High | Medium | Sprints are an estimate, not a contract; re-baseline after Sprint 1 actuals |
| R5 | Supabase connection exhaustion if Render scales beyond one instance | Low (MVP is single instance) | High if it happens | Documented in 05-database-design.md §5.5; revisit pooling (pgbouncer) before scaling instances |
| R6 | Untested migration rollback breaks production recovery | Low (mitigated) | High | UP/DOWN round-trip verification added to DoD (09-testing-strategy.md §9.4) |
| R7 | Cross-tenant data leak via missed RLS policy | Low (mitigated) | Critical | Mandatory automated cross-tenant tests (09-testing-strategy.md §9.2) |
| R8 | Scope creep — AI/feature wishlist growing mid-sprint | Medium | Medium | Section 1.6 scope boundary is the reference; new ideas → backlog, not current sprint |
| R9 | Refresh token in localStorage stolen via XSS | Low (CSP + validation in place) | High | Mandatory rotation limits blast radius to single-use window; revisit with custom domain when budget allows |
| R10 | Parallel FE/BE deploy causes contract-mismatch errors | Low (mitigated) | Medium | Sequential deploy, FE gated on BE health check (11-deployment-plan.md §11.2) |

## 12.4 Assumptions
- Team size: solo developer with AI pairing throughout
- No dedicated QA — testing responsibility falls on the Definition of Done checklist
- No paid infrastructure until real usage demands it
- English-only user base for MVP validation phase

## 12.5 Known Limitations Accepted for MVP (Full List)

```
1. Video compression: none at launch (raw upload, 100MB cap)
2. Refresh token: localStorage, not httpOnly cookie
3. Rate limiting: Postgres-backed for auth/PIN only, in-memory elsewhere
4. Caching: in-memory (Ristretto), not persistent/cross-instance
5. Background jobs: goroutines, not a true task queue (no retries/dead-letter)
6. Observability: Render built-in only, no Prometheus/Sentry
7. CSP: unsafe-inline for styles (Framer Motion/Tailwind dependency)
8. No custom domain — subdomain-based hosting only
9. English only, no i18n
10. SSE may drop on Render free-tier sleep — polling fallback covers this
```

Every item above has a documented reason, a mitigation, and a future-sprint path. None were silently dropped — see the corresponding architecture/security sections for full reasoning.
