# Pikyon System Architecture

> Living document. Update it in the same PR as any feature or architectural change, together with a new ADR when the change is a genuine decision (project rule 5 in the handoff).

## 1. Shape

```
Browser (React + TypeScript, Vercel)
  │  HTTPS + Authorization: Bearer <access token>
  ▼
Go API (Gin, Render)
  Routes → Handler → Service → Repository
  │  pgx v5, two connection pools
  ▼
Supabase platform
  PostgreSQL (RLS + restricted app role) + Storage (private bucket)
  │
External: Gemini API, Resend, Google OAuth, Web Speech API (browser only)
```

## 2. Backend layering

```
HANDLER     HTTP only: decode/validate DTO, call service, write response envelope.
SERVICE     Business logic, orchestration, external APIs. No HTTP, no SQL.
REPOSITORY  SQL through pgx. Returns domain models. No business logic.
```

Every `backend/internal/<domain>/` package is flat — `handler.go`, `service.go`, `repository.go`, `dto.go`, `<domain>_test.go` — with no nested sub-directories. The HTTP wiring lives in `backend/api/v1/routes.go` and contains zero business logic.

## 3. Database access — two roles, two pools (ADR-018)

| Pool | Role | RLS | Used for |
|---|---|---|---|
| `Pools.App` | `pikyon_app` (`NOBYPASSRLS`) | enforced | All normal, user-scoped requests, through `WithUser` |
| `Pools.Admin` | owner (privileged) | bypassed | Migrations, registration/login/refresh, cron jobs, public share-link resolution, cross-user notifications, AI job results |

`WithUser` opens a transaction, runs `set_config('app.current_user_id', <uuid>, true)`, executes the repository callback, then commits or rolls back. The user ID comes only from the verified JWT. Policies read `pikyon.current_user_id()` and are scoped `TO pikyon_app`.

## 4. Request lifecycle

1. React service (axios) attaches the in-memory access token.
2. Gin runs request-ID, logging, recovery, security headers, CORS and body-limit middleware.
3. `api/v1/routes.go` matches the route and delegates to a handler.
4. Auth middleware validates the JWT signature, issuer, expiry and `pwd_ver` claim.
5. Handler validates the DTO with `pkg/validator` and calls the service.
6. Service applies business rules and calls the repository inside `Pools.WithUser`.
7. Repository executes parameterized SQL; RLS restricts rows to the session user.
8. Handler writes the standard envelope (`pkg/response`).

## 5. Source layout rules

- `backend/Dockerfile` is the single canonical multi-stage build. `deployment/docker/` holds compose files only.
- `backend/docs/swagger.json` is the only swagger source. It is never duplicated under `backend/api/`.
- Root `Makefile` orchestrates `dev`, `sync-api`, `test`, `migrate`, `build`, `deploy`.
- `backend/internal/` is feature-based and flat.
- `frontend/src/features/` is self-contained; shared code is promoted to `components/`, `hooks/` or `utils/`.
- Root is capped at 8–12 top-level entries.
- Soft limits: 400 lines per Go file, 300 lines per React component.

## 6. Related documents

- `docs/planning/04-system-architecture.md` — full rationale, changelog and every revised decision.
- `docs/architecture/data-flow.md` — auth, media, AI and sharing flows.
- `docs/architecture/infrastructure.md` — environments, deployment and CI/CD.
- `docs/architecture/security.md` — layered security model.
- `docs/architecture/api-design.md` — API contract conventions.
