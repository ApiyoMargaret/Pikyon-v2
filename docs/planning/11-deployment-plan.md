# Section 11: Deployment Plan

## 11.1 Deployment Topology

```
GitHub repo
  ├── dev branch  → staging: Render (backend) + Vercel preview (frontend)
  └── main branch → production: Render (backend) + Vercel (frontend)
                        │
                        ▼
        Supabase (single free-tier project, schema-separated:
                   dev / staging schemas; production is a
                   fully separate, isolated Supabase project)
```

## 11.2 Deployment Sequence — Revised for Race Condition

Original plan deployed backend (Render) and frontend (Vercel) **in parallel** on every merge. **Corrected**: if a migration adds a required column or an API contract changes, Vercel can finish deploying new frontend code before Render finishes deploying the matching backend — during that window, users hit new UI calling old backend handlers, causing client-side failures.

```
REVISED — sequential, frontend gated on backend health:

Step 1: CI gate already passed (lint, test, build)
Step 2: Build Docker image (backend/Dockerfile — single
        canonical multi-stage build, see 03-tech-stack.md §3.11)
Step 3: Push image to Render
Step 4: Run migrate.sh as an ISOLATED job (golang-migrate up,
        never inside app main()) — staging additionally verifies
        up→down→up round-trip
Step 5: Health check: migration job exit code 0
Step 6: Deploy new backend container to Render
Step 7: Render health check: GET /health returns 200
Step 8: ⚠ ONLY THEN trigger the Vercel deploy hook
        (was: parallel — now: sequential, gated on Step 7)
Step 9: Vercel builds and deploys frontend
Step 10: Post-deploy smoke test (staging only): register → login
         → create memory → basic assertions
```
This adds a few minutes to total deploy time but eliminates the contract-mismatch window entirely — an acceptable trade given deploys aren't so frequent that the extra time matters, but a broken production window does.

## 11.3 Environment Matrix — Revised for DB Isolation

Original plan: staging shared the same Supabase *project* as local development. **Corrected**: running staging CI integration tests against the same project a developer uses locally risks corrupted test state and schema collisions between the two.

```
                  DEVELOPMENT       STAGING              PRODUCTION
Backend host      localhost:8080    Render (staging)     Render (prod)
Frontend host     localhost:5173    Vercel (preview)     Vercel (prod)
Database          Supabase shared   Supabase shared       Supabase prod
                  project, "dev"    project, "staging"    project
                  SCHEMA            SCHEMA (same free-    (fully separate
                                    tier project, isolated project — this
                                    via schema, not a       isolation matters
                                    second paid project)    most)
Storage bucket    dev-memories      staging-memories      prod-memories
Auto-deploy       No                Merge to dev,          Merge to main,
                                    sequential (BE→FE)     sequential (BE→FE)
Access token TTL  24 hours          15 minutes             15 minutes
```

```
Connection strings differ only by search_path:
  DATABASE_URL_DEV:
    postgresql://...?options=-c%20search_path=dev
  DATABASE_URL_STAGING:
    postgresql://...?options=-c%20search_path=staging

Migrations run per-schema independently via golang-migrate's
schema targeting — dev and staging can sit at different
migration versions if needed (e.g. testing a new migration in
dev before promoting to staging).
```

## 11.4 Rollback Strategy

```
Application rollback:
  Render retains previous container image
  Automatic rollback if post-deploy health check fails
  Manual rollback documented as a runbook step (Render dashboard),
  not tribal knowledge

Database rollback:
  golang-migrate down for the specific failed migration only
  NEVER automatic — always a deliberate manual action

Order of operations for a full rollback:
  1. Roll back application container FIRST
  2. Confirm app is stable on old code against new schema
     (migrations should stay backward-compatible for one version)
  3. Only then consider schema rollback, if truly necessary
     (reversed order breaks the running app)
```

## 11.5 Secrets & Configuration Management

```
GitHub Secrets (CI/CD):
  DATABASE_URL_STAGING, DATABASE_URL_PRODUCTION
  SUPABASE_SERVICE_KEY_STAGING, SUPABASE_SERVICE_KEY_PRODUCTION
  GEMINI_API_KEY, RESEND_API_KEY
  JWT_SECRET_STAGING, JWT_SECRET_PRODUCTION
  RENDER_DEPLOY_HOOK, VERCEL_DEPLOY_HOOK

Render env vars: mirror production secrets, set via dashboard
Vercel env vars: VITE_API_BASE_URL per environment

Rule: .env.production gitignored everywhere. Only .env.example
(placeholder values) is ever committed.
```

## 11.6 Monitoring (MVP level)
Render built-in metrics dashboard, Render log streaming (structured JSON via slog), `GET /health` polled by Render's own checker, manual smoke test checklist after every production deploy. Full observability (Prometheus/Sentry) deferred — see `04-system-architecture.md` §4.13.

## 11.7 Deployment Plan Changelog

| Area | Original | Revised | Reason |
|---|---|---|---|
| FE/BE deploy order | Parallel | Sequential, FE gated on BE `/health` | Race window where new frontend hits old backend contract |
| Dev/staging database | Shared Supabase project | Shared project, separate schemas (`dev`/`staging`) | Local dev experiments were at risk of corrupting staging CI state |
