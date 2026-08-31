# Section 4: System Architecture

## 4.1 Architecture Style

Client-Server with RESTful API, Service Layer Pattern internally.

```
Browser (React/TS, Vercel)
        │ HTTPS + JWT (Authorization: Bearer)
        ▼
API Server (Go/Gin, Render)
  Routes → Handler → Service → Repository
        │ pgx driver
        ▼
Supabase Platform
  PostgreSQL (RLS enforced) + Storage (private bucket)
        │
External services: Gemini API, Resend, Google OAuth, Web Speech API (browser-side)
```

## 4.2 Internal Backend Layering

```
HANDLER   → receives HTTP, validates input, calls service, formats response.
            Never contains business logic.

SERVICE   → all business logic, orchestrates repositories, calls external
            APIs (Gemini, Resend). Never knows about HTTP. Never writes SQL.

REPOSITORY → all database queries via pgx. Returns domain models.
            Never contains business logic. Never calls external APIs.
```

Every `internal/<domain>/` package is flat: `handler.go`, `service.go`, `repository.go`, `dto.go`, `<domain>_test.go`. No nested sub-directories — see `03-tech-stack.md` §3.11.

## 4.3 Request Lifecycle

```
1. React component calls service function (axios)
2. Axios interceptor attaches JWT: Authorization: Bearer <token>
3. Request hits Render → Gin global middleware (CORS, rate limit, logging)
4. api/v1/routes.go matches route → delegates to handler (wiring only)
5. Auth middleware validates JWT, extracts userID from claims,
   checks password_changed_at claim against DB (see §4.5 / Security §7)
6. Handler decodes + validates DTO → calls service
7. Service applies business rules → calls repository
8. Repository executes parameterized query via pgx (RLS enforces isolation)
9. Response flows back: Repository → Service → Handler
10. Handler wraps in standard envelope (see 06-api-design.md)
11. React Query caches, re-renders UI
```

## 4.4 Authentication Architecture

### Registration
Email + password → bcrypt hash (cost 12) → user saved → verification email via Resend → access + refresh tokens issued.

### Login
Credentials verified → 2FA prompt if enabled → tokens issued.

### Token Storage — REVISED (see ADR-016)
Originally planned as httpOnly cookie for the refresh token. **Corrected**: Vercel (`*.vercel.app`) and Render (`*.onrender.com`) are different top-level domains, so browser ITP/ETP third-party cookie partitioning can silently drop the cookie regardless of `SameSite` setting — this is a separate, newer browser mechanism than the SameSite policy itself and was initially missed.

```
FINAL TOKEN STRATEGY (free-tier compatible, no custom domain):

Access token:
  → Authorization: Bearer <token> header, as originally planned
  → Stored in memory (React context/state), NOT localStorage

Refresh token:
  → Stored in localStorage (accepted tradeoff — reopens XSS
    surface that httpOnly cookies exist to prevent)
  → Mitigated by:
    - Mandatory rotation: every refresh issues a NEW refresh
      token, invalidates the old one immediately
    - sessions.rotated_from column tracks the rotation chain
      for anomaly detection (see 05-database-design.md)
    - Strict CSP (see 07-security-plan.md)
  → This is the WEAKER of two options. The stronger option
    (custom domain + httpOnly cookie, ~$3-15/year) was declined
    per explicit no-budget constraint. Revisit when budget allows.
```

### Token Refresh Flow
Access token expires (15 min prod) → axios interceptor catches 401 → **mutex-protected** refresh (see below) → original request retried transparently.

### JWT Refresh Race Condition Fix
Original flow allowed multiple simultaneous 401s to fire multiple concurrent `/refresh` calls, invalidating tokens on 2nd/3rd attempt. Fixed with a request queue:

```typescript
let isRefreshing = false;
let failedQueue: Array<{resolve: (t: string) => void; reject: (e: unknown) => void}> = [];

// First 401 triggers refresh; subsequent 401s queue and wait
// On success: all queued requests retried with new token
// On failure: all queued requests rejected, user logged out
```
Lives in `frontend/src/services/api.ts`.

### 2FA (TOTP)
RFC 6238, Google Authenticator compatible. Secret AES-256 encrypted at rest. 8 bcrypt-hashed backup codes.

### Token Invalidation
Access tokens are stateless JWTs — logout/password-reset can't retroactively invalidate an already-issued token by deletion alone. Fixed via a claim comparison:

```
JWT includes: "pwd_ver": <unix timestamp of users.password_changed_at>

On every authenticated request:
  Compare JWT's pwd_ver claim vs current users.password_changed_at
  If JWT's value is older → reject (401), even if not expired

Result: password reset / forced logout instantly invalidates
ALL existing access tokens with zero new infrastructure
(no blocklist, no Redis) — just one DB column comparison.
```

## 4.5 Media Architecture — Zero Direct Serving

### Upload Flow
```
1. File selected → size checked
   - Images over limit: browser-image-compression (silent)
   - Video: raw upload, 100MB cap (server-side compression
     is a Sprint 2 deferred item, see §4.7)
2. Frontend requests pre-signed upload URL:
   POST /media/upload-url
3. Backend creates a pending_uploads record (status=pending,
   expires_at=+1hr) alongside issuing the signed URL
4. Frontend uploads DIRECTLY to Supabase Storage (backend never
   touches file bytes)
5. Frontend confirms: POST /media/confirm {upload_id, object_key}
   — NO file_size or mime_type sent by client
6. Backend fetches REAL metadata from Supabase Storage API
   directly (Content-Length, Content-Type) — client-reported
   values are never trusted (see ADR-017, closes a metadata-
   spoofing vulnerability caught in API design review)
7. media_files record created with backend-verified metadata
   pending_uploads record deleted
```

### Orphaned Upload Cleanup
If the user's connection drops after Supabase upload completes but before `/media/confirm` is called, the file becomes orphaned (in storage, no DB record).

```
Fix: pending_uploads table (see 05-database-design.md) +
     hourly Go cron job:
       DELETE pending_uploads WHERE expires_at < NOW()
       → also deletes the corresponding Supabase Storage object
       → every deletion logged for audit
```

### Streaming Flow
```
User clicks "Reveal Memory"
  → GET /media/:memoryId/stream (JWT + X-Pin-Token if locked)
  → Backend validates ownership (RLS) + PIN token if applicable
  → Backend generates 5-minute pre-signed URL from Supabase
  → Returns ONLY the signed URL — never proxies media bytes
  → Frontend streams directly from Supabase CDN
  → URL expires in 5 minutes, cannot be bookmarked or shared
```

## 4.6 Video Compression — Revised Approach

Original plan used FFmpeg.wasm client-side for all video compression. **Corrected**: FFmpeg.wasm requires WebAssembly threads and heavy memory allocation; mobile Safari's ~1.5GB WebKit memory cap routinely crashes the tab on high-resolution video.

```
PHASED APPROACH:

Images (MVP, Sprint 2):
  → browser-image-compression — safe on all browsers, no Wasm

Videos (MVP, Sprint 2):
  → REMOVE FFmpeg.wasm entirely
  → Raw upload with 100MB cap, user informed if exceeded
  → Known limitation, documented in 12-constraints-risks.md

Videos (Sprint 2+, server-side):
  → Raw video uploaded to a temporary staging bucket
  → Go background worker (goroutine, see §4.7) transcodes/
    compresses asynchronously
  → Final asset moved to permanent bucket
  → User sees "Processing video..." status, notified when ready
```

## 4.7 AI Features Architecture

### Processing Strategy — Revised for Render Free Tier
Synchronous Gemini calls (3–12 seconds) risk gateway timeouts on Render's tier. A full asynq + Redis queue was considered but deferred as infrastructure overkill for MVP.

```
MVP APPROACH — Go goroutines + polling-first, SSE layered on top:

1. POST /ai/analyze → backend returns 202 Accepted + job_id
   immediately (does not block on Gemini call)
2. Backend spawns a Go goroutine to call Gemini API
3. Result written to ai_jobs table (status: queued → processing
   → completed/failed)
4. Frontend polls GET /ai/jobs/:job_id first (built and verified
   BEFORE SSE — sequencing decision from Sprint 3 planning, so
   a working baseline exists independent of streaming reliability)
5. GET /ai/jobs/:job_id/stream (SSE) layered on top once polling
   is verified — frontend prefers SSE, falls back to polling
   every 3s if the SSE connection drops (Render free tier sleeps
   after 15 min inactivity, a known limitation)

FUTURE SPRINT: asynq + Redis if traffic demands lower latency
than goroutines + polling/SSE provide.
```

### Auto Caption Flow
```
User uploads media → clicks "Analyze with AI" (manual, never automatic)
  → POST /ai/analyze {memory_id, analyze: [caption, mood, tags, social_captions]}
  → Backend generates temp signed URL, sends to Gemini Vision
  → Gemini returns structured JSON (caption, mood, tags, social captions)
  → Frontend displays via SSE/poll, user edits before saving
```

### Speech-to-Text Flow
```
User clicks microphone → Web Speech API transcribes (free, browser-native)
  → User chooses:
      Summarize: transcript → Gemini → short summary
      Polish: transcript → Gemini → grammar-fixed, filler-removed,
              full journal entry (nothing summarized away)
  → User edits and saves either way
```

## 4.8 Sharing Architecture

```
Owner clicks Share on a PUBLIC memory → enters recipient email,
optional expiry/max_views
  ↓
Backend checks if recipient is registered:
  YES → share token generated, Resend email with "View Memory" link
  NO  → Resend invitation email, share honored after registration
  ↓
Recipient clicks link → must be authenticated → memory opens
  ↓
memory_views record created, owner notified (in-app + email)
  ↓
Recipient may react (heart/wow/sad), cannot download or re-share
```

## 4.9 Database Connection Pooling

Supabase free tier caps at 60 total connections. Explicit pgxpool limits prevent exhaustion:

```go
config.MaxConns          = 25   // per Render instance; safe under 60 cap for single instance
config.MinConns          = 5
config.MaxConnLifetime   = 30 * time.Minute
config.MaxConnIdleTime   = 5 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute
```

> ⚠️ If Render scales beyond one instance, 25×N conns will exceed Supabase's 60-connection cap. Documented as a known scaling constraint — revisit pooling strategy (e.g., pgbouncer transaction mode) before adding instances.

## 4.10 Rate Limiting — Revised for Persistence

Original plan used Ristretto (in-memory) for all rate limiting. **Corrected**: in-memory state is wiped on every Render container restart (deploys, health-check-triggered restarts), reopening a brute-force window, and doesn't work across horizontal scaling since requests load-balance across instances with separate memory.

```
REVISED APPROACH:

Non-critical routes (general API):
  → Ristretto in-memory (acceptable risk — brief permissive
    window after restart is a low-severity tradeoff)

Critical routes (auth, PIN verify — where brute force matters):
  → Postgres-backed token bucket, new table: rate_limit_buckets
    (see 05-database-design.md)
  → Persists across restarts, naturally consistent across
    instances since Supabase is shared state
  → No new infrastructure required (uses existing DB)

FUTURE SPRINT: Redis if latency demands beat Postgres round-trip cost.
```

## 4.11 In-Memory Caching Layer

Shared/public memory metadata (view counts, memory details on the shared-view page) risks repeated database hits during traffic spikes.

```
MVP: Ristretto in-memory cache for shared memory metadata
  → TTL: 5 minutes
  → Invalidated on share revocation
  → Free, zero new infrastructure

FUTURE SPRINT: Redis if traffic requires persistent, cross-instance cache
```

## 4.12 Migration Execution Strategy

```
RULE: Migrations run as an ISOLATED step BEFORE the app binary
      starts. NEVER inside main() — this risks locking table
      rows across multiple instance replicas.

CD pipeline order:
  1. Build Docker image
  2. Run migrate.sh (up migrations only in production;
     staging additionally verifies up→down→up round-trip)
  3. Health check: migration exit code 0
  4. Deploy app binary
  5. Health check: GET /health returns 200
  6. Automatic rollback of app container if health check fails

Naming: 000001_create_users.up.sql / .down.sql — sequential,
never skipped, never edited after merge (always add a new one).
```

## 4.13 Observability (MVP scope)

```
MVP:
  → slog structured JSON logging (no PII, ever)
  → Render built-in metrics dashboard (free)
  → Render log streaming (free)
  → GET /health polled by Render's own health checker

FUTURE SPRINT:
  → Prometheus + Grafana, or Sentry for error tracking
  → OpenTelemetry distributed tracing
```

## 4.14 Background Jobs Strategy

```
MVP: Go goroutines + channels (built-in, free)
  Used for: AI processing (§4.7), orphaned upload cleanup (§4.5),
  async social/notification dispatch

FUTURE SPRINT: hibiken/asynq + Redis, once true task-queue
  semantics (retries, dead-letter, delayed jobs) are needed
  at a scale goroutines can't cleanly handle
```

## 4.15 CI/CD Pipeline

```
Feature branch → PR to dev
  ↓
ci.yml runs:
  Backend: golangci-lint, go test ./... -cover, go build,
           swaggo generates swagger.json, git diff check
           (swagger must be current), migration up→down→up
           round-trip test (for PRs touching backend/migrations/)
  Frontend: eslint, prettier check, openapi-typescript,
            tsc --noEmit, vitest run --coverage, vite build
  ALL must pass → PR mergeable

Merge to dev
  ↓
cd-staging.yml:
  Build Docker image → run migrations (staging schema) →
  deploy backend to Render (staging) → WAIT for /health 200
  → THEN trigger Vercel preview deploy (sequential, not
  parallel — see 11-deployment-plan.md §11.2 for the race
  condition this prevents) → smoke tests

PR dev → main (after staging verified)
  ↓
cd-production.yml: same sequential pattern, production targets,
  automatic rollback on health check failure
```

## 4.16 Environment Architecture

See `03-tech-stack.md` §3.10 for the full environment matrix (dev/staging schema separation, production isolation).

## 4.17 Architecture Decisions Changelog

| Area | Original Plan | Revised Plan | Reason |
|---|---|---|---|
| Video compression | FFmpeg.wasm client-side | Images: browser-compression. Video: raw upload capped + server-side worker (Sprint 2+) | Mobile Safari memory crashes |
| Orphaned media | None specified | `pending_uploads` table + hourly cron | Confirm-call can be missed on connection drop |
| AI processing | Synchronous handler | Goroutines + 202 + job_id, polling-first then SSE | Render timeout risk, testability |
| JWT refresh | Naive interceptor | Mutex + request queue | Concurrent 401s invalidate tokens |
| Refresh token storage | httpOnly cross-site cookie | localStorage + mandatory rotation | Cross-domain cookie partitioning (ITP/ETP) |
| Rate limiting | Ristretto only | Postgres-backed for auth/PIN, Ristretto elsewhere | Restart wipes in-memory state |
| Migrations | Implied in-app | Isolated job before binary starts | Row-locking risk across replicas |
| Caching | None specified | Ristretto, 5min TTL | Avoid DB hit on every shared-view |
| Deploy sequencing | Parallel FE/BE deploy | Sequential, FE gated on BE health check | Contract-mismatch race window |
