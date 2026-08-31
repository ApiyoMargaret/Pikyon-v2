# Section 3: Tech Stack

## 3.1 Overview

```
Pikyon System
├── Frontend        → React + TypeScript + shadcn/ui
├── Backend         → Go (Gin), feature-based architecture
├── Database        → Supabase (PostgreSQL + pgx)
├── Storage         → Supabase Storage
├── AI              → Google Gemini API
├── Auth            → Supabase Auth + Google OAuth + 2FA
├── Email           → Resend
├── Speech          → Web Speech API
├── CI/CD           → GitHub Actions
├── Deployment      → Render (backend) + Vercel (frontend)
└── Documentation   → Swaggo + ADR + Architecture Docs
```

## 3.2 Frontend

| Technology | Purpose | Notes |
|---|---|---|
| React 18+ | UI framework | |
| TypeScript 5+ | Type safety | Types auto-generated from swagger |
| Tailwind CSS 3+ | Styling | |
| **shadcn/ui** | Component primitives | Free, MIT licensed, copied into repo — see ADR-014 |
| **Inter** | Typography (only) | No serif — see ADR-015, benchmarked against Ente/Linear |
| Lucide icons | Icon set | Ships with shadcn |
| Framer Motion | Animation | Used sparingly, 0.15–0.3s |
| React Router 6+ | Routing | |
| React Query 5+ | Server state, caching | |
| Axios | HTTP client | Refresh-token mutex interceptor — see `04-system-architecture.md` §4.5 |
| React Hook Form + Zod | Forms + validation | |
| browser-image-compression | Client-side image compression | |
| MSW | API mocking for tests | Network-layer interception — see `09-testing-strategy.md` |
| Vite | Build tool | |

## 3.3 Backend

| Technology | Purpose |
|---|---|
| Go 1.22+ | Backend language |
| Gin 1.9+ | HTTP framework |
| pgx v5 | PostgreSQL driver — connects to Supabase's underlying Postgres |
| golang-migrate | Schema migrations (up/down, isolated job, never in `main()`) |
| swaggo/swag | Generates `swagger.json` from Go comments |
| golang-jwt v5 | JWT handling |
| bcrypt | Password/PIN/refresh-token hashing |
| go-playground/validator | Request DTO validation |
| slog (stdlib) | Structured logging |
| testify | Assertions/mocking |

## 3.4 Database & Storage

| Technology | Purpose |
|---|---|
| Supabase | Managed Postgres platform — **is** Postgres, not a separate system; pgx is the Go driver used to connect to it |
| PostgreSQL 15+ | Relational database |
| pgxpool | Connection pooling — explicit limits, see `04-system-architecture.md` §4.9 |
| Supabase Storage | Media storage, pre-signed URLs |
| Row Level Security | Enforced on every table |

## 3.5 AI & External Services (all free tier)

| Service | Purpose | Limit |
|---|---|---|
| Google Gemini API | Caption, mood, tags, story polish, social captions | 15 req/min, 1500/day |
| Web Speech API | Browser-native speech-to-text | Free always, Chrome/Edge only |
| Resend | Transactional email | 3000 emails/month |
| Google OAuth 2.0 | Social login | Free |

## 3.6 Authentication & Security

See `07-security-plan.md` for full detail. Summary:
- JWT access token (15 min production / 24hr dev) + refresh token with mandatory rotation
- Refresh token stored in **localStorage** (not httpOnly cookie — cross-domain cookie limitation, documented tradeoff)
- bcrypt password/PIN hashing, TOTP 2FA
- RLS + PIN token + share token authorization layers

## 3.7 DevOps & Infrastructure

| Technology | Purpose |
|---|---|
| GitHub + GitHub Projects | Source control, Scrum board |
| GitHub Actions | CI/CD |
| Render | Backend hosting (free tier) |
| Vercel | Frontend hosting (free tier) |
| Docker | Single canonical multi-stage build — `backend/Dockerfile` only |
| GitHub Secrets | Secrets management |

## 3.8 Documentation & Code Quality

| Technology | Purpose |
|---|---|
| swaggo/swag | Single source of truth: `backend/docs/swagger.json` |
| openapi-typescript | Generates `frontend/src/types/api.ts` |
| ADR (Markdown) | `docs/adr/` — one per major decision |
| Architecture Docs | `docs/architecture/` — updated with every new feature |
| ESLint + Prettier | Frontend linting/formatting |
| golangci-lint | Backend linting |
| govulncheck / npm audit | Dependency vulnerability scanning in CI |

## 3.9 Type Generation Automation Pipeline

```
Write Go structs + handler comments
        ↓
swaggo/swag → backend/docs/swagger.json (ONLY location, no duplication)
        ↓
openapi-typescript → frontend/src/types/api.ts
        ↓
Frontend uses generated types directly, no manual type writing

CI enforces staleness check:
  make generate-swagger && git diff --exit-code backend/docs/swagger.json
  → PR blocked if swagger wasn't regenerated after handler changes

Run manually: make sync-api
```

## 3.10 Development vs Production Environment

| Aspect | Development | Staging | Production |
|---|---|---|---|
| Backend URL | localhost:8080 | Render (staging) | pikyon-api.onrender.com |
| Frontend URL | localhost:5173 | Vercel (preview) | pikyon.vercel.app |
| Database | Supabase, `dev` schema | Supabase, `staging` schema (same project) | Supabase prod project (fully isolated) |
| Storage bucket | dev-memories | staging-memories | prod-memories |
| Logging | Verbose debug | Structured JSON | Structured JSON only |
| Error messages | Full detail | Generic safe | Generic safe |
| Access token expiry | 24 hours | 15 minutes | 15 minutes |
| CORS | Allow localhost | Allow preview URL | Allow production domain only |

> Dev and staging share one free-tier Supabase project via **separate schemas** (`dev` / `staging`) to avoid a second paid project while preventing local experiments from polluting staging CI runs.

## 3.11 Monorepo File Structure

See `docs/architecture/system-architecture.md` for the complete, corrected file tree. Key structural rules locked in during planning:

- **One canonical Dockerfile** at `backend/Dockerfile` (multi-stage). `deployment/docker/` contains compose files only, referencing it — never a second Dockerfile.
- **One swagger source of truth**: `backend/docs/swagger.json`. Never duplicated into `backend/api/`.
- `backend/api/v1/routes.go` **wires routes and middleware only** — zero business logic. All logic lives in `internal/*/handler.go`.
- Email templates use Go's `//go:embed templates/*.html` — compiled into the binary, no runtime file-path dependency.
- Root `Makefile` orchestrates: `make dev`, `make sync-api`, `make test`, `make migrate`, `make build`, `make deploy`.
- `internal/` is feature-based, **flat** — one package per domain, no nested sub-directories. Split horizontally into new domains, never nest vertically.
- `frontend/src/features/` — self-contained, no cross-feature imports. Shared logic promotes to `components/`, `hooks/`, or `utils/`.
- Root level capped at 8–12 top-level items.
- Soft limits: 400 lines per Go file, 300 lines per React component — signal to split, not a hard rule.

## 3.12 Stack Additions Log

| ADR | Addition | Reason |
|---|---|---|
| ADR-014 | shadcn/ui | Free, accessible primitives pairing natively with Tailwind; avoids imposing a visual opinion |
| ADR-015 | Inter-only typography (no Fraunces/serif) | Consistency with Ente/Linear benchmark; avoids mixed design personality across the app |
