# Section 10: Sprint Plan & Backlog

## 10.1 Sprint Structure

```
Duration:      1 week per sprint
Ceremonies:    Sprint Planning (Mon) → async daily standup notes
               → Sprint Review (Fri) → Retrospective (Fri)
Tool:          GitHub Projects (Kanban, linked to Issues)
Board columns: Backlog → Sprint Backlog → In Progress → In Review → Done
Labels:        sprint-0..7, type:feature, type:bug, type:chore,
               priority:must-have
```

## 10.2 Product Backlog → Sprint Mapping (Final, Revised)

```
SPRINT 0 — Planning & Foundation
  □ Complete planning document (this document, Sections 1–12)
  □ GitHub repo, branch protection, monorepo scaffold
  □ GitHub Projects board with backlog populated
  □ CI pipeline skeleton (lint + build, no deploy yet)

SPRINT 1 — Auth & Database Foundation  [REVISED — 2FA moved out]
  □ Database migrations (all tables from Section 5)
  □ pgxpool connection + config
  □ Auth: register, login, JWT, refresh token mutex
  □ Google OAuth
  □ Email verification flow (Resend)
  □ Cross-tenant test helper (pkg/testutil/tenancy.go)
  □ Frontend: Login/Register pages
  ~~2FA setup/verification~~ → moved to Sprint 5
  Reason: combining 10 migrations + JWT + refresh mutex + OAuth
  + 2FA + email verification + frontend auth pages in one week
  is over-scoped — OAuth edge cases and 2FA secret encryption
  routinely surface unexpected friction that threatens the deadline.

SPRINT 2 — Memory CRUD & Media Upload
  □ Memory create/read/update/soft-delete/restore
  □ Bulk-trash, trash/empty endpoints
  □ Pre-signed upload URL + confirm (real metadata fetch from
    Supabase, not client-trusted — see 06-api-design.md §6.7)
  □ pending_uploads + orphan cleanup cron
  □ Client-side image compression
  □ Frontend: Dashboard grid, Create/Edit memory form
  Known limitation shipped: video compression deferred (raw
  upload, 100MB cap) — see 12-constraints-risks.md

SPRINT 3 — AI Features & Speech  [REVISED — polling before SSE]
  Day 1-2: □ AI job queue (Go goroutines)
           □ GET /ai/jobs/:job_id (polling endpoint) — built and
             verified FIRST as a working baseline
           □ Gemini integration: caption, mood, tags, social captions
  Day 3:   □ GET /ai/jobs/:job_id/stream (SSE) layered on top,
             only once polling is verified reliable
  Day 4-5: □ Web Speech API integration (frontend)
           □ Summarize + Polish endpoints
           □ Frontend: AI assist UI, speech recorder
  Reason: building goroutines + Gemini + SSE simultaneously means
  a bug in SSE transport is indistinguishable from a bug in job
  processing. Sequencing gives a testable baseline before adding
  streaming complexity, with polling as the frontend's fallback
  if SSE drops (Render free tier sleep).

SPRINT 4 — Sharing & Notifications
  □ Share creation + email delivery (Resend)
  □ Shared memory view + reactions
  □ View log
  □ Notification system (in-app + email)
  □ Frontend: Share modal, notification bell, shared memory page

SPRINT 5 — Search, PIN Lock, 2FA & Settings  [REVISED — 2FA added]
  □ Full-text search + filters
  □ PIN set/verify/remove
  □ 2FA setup/verify/disable  ← moved here from Sprint 1
  □ Profile/Settings pages
  □ Session management (list/revoke)
  □ Storage usage breakdown
  Reason for grouping 2FA with PIN: both are "secondary
  verification" features, natural to build together once the
  base auth flow (Sprint 1) is stable and battle-tested.

SPRINT 6 — Testing, CI/CD & Hardening
  □ Close coverage gaps (70%/60% targets)
  □ Cross-tenant tests for all resource tables (mandatory pattern)
  □ Full CD pipeline — sequential deploy (backend health check
    gates frontend deploy, see 11-deployment-plan.md §11.2)
  □ Postgres-backed rate limiting live for auth/PIN routes
  □ Security headers, CSP, cron jobs live
  □ Migration up/down round-trip verification in CI

SPRINT 7 — Polish & Launch
  □ Bug bash
  □ Performance pass against Section 1.7 metrics
  □ Final ADR + architecture doc sweep
  □ Production deploy + smoke test
  □ README + onboarding docs finalized
```

## 10.3 Definition of Ready
```
□ User story written (As a / I want / So that)
□ Acceptance criteria listed
□ API endpoint(s) identified from 06-api-design.md
□ DB table(s) identified from 05-database-design.md, or confirmed
  no schema change
□ UI reference identified from 08-ui-ux-plan.md (or N/A backend-only)
```

## 10.4 Definition of Done
```
□ Code merged to dev via PR (passed CI gate)
□ Tests written per 09-testing-strategy.md §9.5
□ Migration UP/DOWN round-trip verified, if schema changed
□ Swagger regenerated if API changed (make sync-api)
□ ADR written/updated if architectural decision made
□ Architecture docs updated if new component added
□ Deployed to staging and manually verified
```

## 10.5 Sprint Plan Changelog

| Sprint | Change | Reason |
|---|---|---|
| Sprint 1 | 2FA removed from scope | Auth scope was over-compressed for one week |
| Sprint 3 | Polling built and verified before SSE, not simultaneously | Isolates job-processing bugs from streaming-transport bugs |
| Sprint 5 | 2FA added (moved from Sprint 1) | Natural pairing with PIN lock as "secondary verification" |
| Sprint 6 | Migration round-trip check added to CI scope | Closes an untested-rollback gap in the original Definition of Done |
