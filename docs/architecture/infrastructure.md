# Pikyon Infrastructure

## Environments

| Aspect | Development | Staging | Production |
|---|---|---|---|
| Backend | `localhost:8080` | Render staging | Render production |
| Frontend | `localhost:5173` | Vercel preview | Vercel production |
| Database | Supabase | Supabase | Separate Supabase project |
| Storage bucket | `dev-memories` | `staging-memories` | `prod-memories` |
| Access token TTL | 24h | 15m | 15m |
| Auto-deploy | No | No | No |

Application code reads unsuffixed environment names. Each environment supplies its own values through `.env` (local), GitHub Environment secrets (deploy workflows) or the Render/Vercel dashboards (runtime). `VITE_*` values are public and must never contain secrets.

## Database connection

- Use the Supabase **session pooler** on port 5432. The direct connection is IPv6-only; the transaction pooler does not support the prepared statements the app uses.
- Pooler username format: `postgres.<project-ref>` or `pikyon_app.<project-ref>`.
- Two connection strings are required by ADR-018: `DATABASE_URL` (privileged) and `APP_DATABASE_URL` (restricted `pikyon_app` role).
- URL-encode special characters in the password.

## Deployment sequence (never parallel)

```
1. CI gate passed
2. Build backend/Dockerfile
3. Run migrations as an isolated job (up; staging also up→down→up)
4. Migration exit code must be 0
5. Deploy backend container to Render
6. GET /health must return 200
7. Only then trigger the Vercel deploy hook
8. Frontend builds and deploys
9. Staging smoke test: register → login → create memory
```

Automatic Git deploys must stay disabled on both Render and Vercel; otherwise a new backend or frontend can go live before migrations or health checks pass.

## CI/CD workflows

| Workflow | Trigger | Contents |
|---|---|---|
| `.github/workflows/ci.yml` | PR/push to `dev` or `main` | Go vet/test/build, migration up→down→up, frontend lint/typecheck/build |
| `.github/workflows/cd-staging.yml` | push to `dev` | Sequential staging deploy via GitHub Environment `staging` |
| `.github/workflows/cd-production.yml` | push to `main` | Sequential production deploy via Environment `production` (required reviewer) |

`ci.yml` needs no secrets: it uses a Postgres service container and mocks Gemini/Resend.

## Observability (MVP)

`slog` structured JSON with request IDs and no PII, Render metrics and log streaming, `GET /health` polled by Render. Prometheus/Sentry/OpenTelemetry are future-sprint items.
