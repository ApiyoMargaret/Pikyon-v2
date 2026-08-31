# Section 6: API Design

## 6.1 Principles

```
Base URL (dev):   http://localhost:8080/api/v1
Base URL (prod):  https://pikyon-api.onrender.com/api/v1
Format:           JSON only
Auth:             Bearer JWT in Authorization header
Case:             snake_case JSON fields
IDs:              UUID v4
Pagination:       Cursor-based ONLY across every list/search endpoint
                  (standardized — see §6.2 changelog)
Errors:           RFC 7807 style
Batch:            Max 50 items per batch operation
```

## 6.2 Standard Response Envelope

```json
// Single resource
{ "success": true, "data": {}, "message": "Memory created successfully" }

// List resource — cursor-based, always (never offset/total)
{
  "success": true,
  "data": { "items": [] },
  "meta": { "next_cursor": "uuid-or-encoded-string", "has_more": true, "returned": 20 }
}

// Async job (202)
{
  "success": true,
  "data": { "job_id": "uuid", "status": "queued", "stream_url": "/api/v1/ai/jobs/uuid/stream" },
  "message": "AI analysis queued"
}

// Error
{
  "success": false,
  "error": { "code": "VALIDATION_ERROR", "message": "...", "details": [{"field":"title","message":"Title is required"}] }
}
```

> **Revised**: originally `GET /memories` used a `total` count while `GET /search` used cursor pagination — inconsistent, and `total` requires an expensive full table scan on `tsvector` searches. Standardized to cursor-only, `total` removed everywhere.

## 6.3 Error Codes

| Status | Code | Meaning |
|---|---|---|
| 400 | VALIDATION_ERROR | Request validation failed |
| 400 | OBJECT_KEY_MISMATCH | Upload key doesn't match issued key |
| 401 | UNAUTHORIZED / TOKEN_EXPIRED / INVALID_CREDENTIALS | Auth failures |
| 403 | FORBIDDEN / PIN_REQUIRED / EMAIL_NOT_VERIFIED / TWO_FA_REQUIRED | Authorization failures |
| 404 | NOT_FOUND / OBJECT_NOT_FOUND | Resource missing |
| 409 | ALREADY_EXISTS | Conflict |
| 410 | GONE | Share expired / max views reached |
| 413 | FILE_TOO_LARGE | Exceeds 100MB |
| 415 | UNSUPPORTED_MEDIA | Bad file type |
| 429 | RATE_LIMITED | Too many requests |
| 500 | INTERNAL_ERROR | Generic safe message only |
| 503 | SERVICE_UNAVAILABLE | External service down |

## 6.4 Authentication Endpoints

```
POST   /auth/register
POST   /auth/login
POST   /auth/logout
POST   /auth/refresh
POST   /auth/verify-email
POST   /auth/resend-verification
POST   /auth/forgot-password
POST   /auth/reset-password
GET    /auth/google
GET    /auth/google/callback
POST   /auth/2fa/setup
POST   /auth/2fa/verify
POST   /auth/2fa/disable
POST   /auth/pin/set
POST   /auth/pin/verify
POST   /auth/pin/remove
```

Key request/response shapes: see original detailed spec in project history; unchanged except:
- `POST /auth/refresh` now reads the refresh token from the request body (or an `X-Refresh-Token` header), **not** an httpOnly cookie — since the token now lives in localStorage (see `04-system-architecture.md` §4.4).
- `POST /auth/pin/verify`: rate limited 5/15min **per user, Postgres-backed** (not Ristretto) — see §6.13.

## 6.5 User Endpoints
```
GET    /users/me
PATCH  /users/me
DELETE /users/me
PATCH  /users/me/password
POST   /users/me/avatar
DELETE /users/me/avatar
GET    /users/me/sessions
DELETE /users/me/sessions/:session_id
DELETE /users/me/sessions
GET    /users/me/storage
```

## 6.6 Memory Endpoints — Revised

```
GET    /memories               ← active memories ONLY, no status param
POST   /memories
GET    /memories/:id
PATCH  /memories/:id
DELETE /memories/:id           ← soft delete (trash)
POST   /memories/:id/restore
DELETE /memories/:id/permanent
GET    /memories/trash         ← dedicated view, kept separate
POST   /memories/trash/empty   ← added: purge entire trash, needs X-Confirm-Delete header
POST   /memories/bulk-trash    ← added: batch move to trash, max 50 IDs
```

> **Revised**: `GET /memories?status=trashed` was removed entirely to avoid redundancy with `GET /memories/trash`, which returns richer trash-specific fields (`permanent_delete_at`, `days_remaining`). `GET /memories` now always returns active memories only.

### POST /memories/bulk-trash
```json
Request:  { "memory_ids": ["uuid-1", "uuid-2"] }  // max 50, atomic (any invalid ID rejects the whole batch)
Response: { "success": true, "data": { "trashed_count": 2, "permanent_delete_at": "..." } }
```

### POST /memories/trash/empty
```
Headers: X-Confirm-Delete: true  (required — safety gate)
Response: { "success": true, "data": { "deleted_count": 12, "storage_freed_bytes": 524288000 } }
```

## 6.7 Media Endpoints — Revised for Metadata Trust

```
POST   /media/upload-url
POST   /media/confirm
GET    /media/:memory_id/stream
GET    /media/:memory_id/thumbnail
DELETE /media/:memory_id
```

### POST /media/confirm — Revised
```
Request (client sends ONLY identifiers, no metadata):
{ "upload_id": "uuid", "object_key": "user-uuid/memory-uuid/uuid.jpg" }

Backend process (revised — was: trust client-reported file_size/mime_type):
  1. Verify upload_id belongs to authenticated user
  2. Verify object_key matches the one originally issued
  3. Fetch REAL metadata from Supabase Storage API directly
     (Content-Length, Content-Type, dimensions)
  4. Create media_files record with backend-verified metadata only
  5. Delete pending_uploads record

Errors: 400 OBJECT_KEY_MISMATCH, 404 OBJECT_NOT_FOUND,
        422 UNPROCESSABLE (upload window expired, >1hr)
```
> Fixes a metadata-spoofing vulnerability: client-reported `file_size`/`mime_type` could previously be forged to corrupt storage-usage tracking or inject mismatched records.

## 6.8 AI Endpoints

```
POST   /ai/analyze
POST   /ai/speech/summarize
POST   /ai/speech/polish
GET    /ai/jobs/:job_id            ← polling endpoint, built and verified FIRST
GET    /ai/jobs/:job_id/stream     ← SSE, layered on top second (see 04 §4.7)
```

All AI endpoints return `202 Accepted` with `job_id` immediately — never block on the Gemini call.

## 6.9 Sharing Endpoints
```
POST   /sharing
GET    /sharing
DELETE /sharing/:share_id
GET    /sharing/:share_id/views
GET    /memories/shared/:share_token
POST   /memories/shared/:share_token/react
```

## 6.10 Search Endpoints — Revised

```
GET /search
GET /search/suggestions
```
Standardized to cursor-based `meta` (see §6.2). Query params: `q`, `media_type`, `mood`, `from_date`, `to_date`, `cursor`, `limit`.

## 6.11 Notification Endpoints
```
GET    /notifications
PATCH  /notifications/:id/read
PATCH  /notifications/read-all
DELETE /notifications/:id
GET    /notifications/unread-count
```

## 6.12 Complete Endpoint Summary

| Method | Endpoint | Auth |
|---|---|---|
| POST | /auth/register | None |
| POST | /auth/login | None |
| POST | /auth/logout | JWT |
| POST | /auth/refresh | Refresh token (body/header, not cookie) |
| POST | /auth/verify-email | None |
| POST | /auth/resend-verification | None |
| POST | /auth/forgot-password | None |
| POST | /auth/reset-password | None |
| GET | /auth/google, /auth/google/callback | None |
| POST | /auth/2fa/setup, /verify, /disable | JWT |
| POST | /auth/pin/set, /verify, /remove | JWT |
| GET/PATCH/DELETE | /users/me | JWT |
| PATCH | /users/me/password | JWT |
| POST/DELETE | /users/me/avatar | JWT |
| GET/DELETE | /users/me/sessions | JWT |
| GET | /users/me/storage | JWT |
| GET | /memories | JWT |
| POST | /memories | JWT |
| GET/PATCH/DELETE | /memories/:id | JWT |
| POST | /memories/:id/restore | JWT |
| DELETE | /memories/:id/permanent | JWT |
| GET | /memories/trash | JWT |
| POST | /memories/trash/empty | JWT |
| POST | /memories/bulk-trash | JWT |
| POST | /media/upload-url | JWT |
| POST | /media/confirm | JWT |
| GET | /media/:memory_id/stream, /thumbnail | JWT |
| DELETE | /media/:memory_id | JWT |
| POST | /ai/analyze, /speech/summarize, /speech/polish | JWT |
| GET | /ai/jobs/:job_id, /stream | JWT |
| POST/GET/DELETE | /sharing | JWT |
| GET | /sharing/:share_id/views | JWT |
| GET | /memories/shared/:token | JWT |
| POST | /memories/shared/:token/react | JWT |
| GET | /search, /search/suggestions | JWT |
| GET/PATCH/DELETE | /notifications | JWT |
| GET | /notifications/unread-count | JWT |

## 6.13 Rate Limiting Strategy — Revised

| Endpoint Group | Limit | Window | Backing Store |
|---|---|---|---|
| POST /auth/register | 5 | Per hour per IP | Postgres |
| POST /auth/login | 10 | Per 15 min per IP | Postgres |
| POST /auth/forgot-password | 3 | Per hour per IP | Postgres |
| POST /auth/pin/verify | 5 | Per 15 min per user | Postgres |
| POST /memories/trash/empty | 3 | Per day per user | Ristretto |
| POST /memories/bulk-trash | 10 | Per hour per user | Ristretto |
| POST /ai/* | 20 | Per hour per user | Ristretto |
| GET /media/*/stream | 60 | Per hour per user | Ristretto |
| All other endpoints | 100 | Per minute per user | Ristretto |

> **Revised**: auth/PIN routes moved to Postgres-backed `rate_limit_buckets` (see `05-database-design.md`) since in-memory state is wiped on Render restart, which would otherwise reopen a brute-force window on the most security-sensitive endpoints.

## 6.14 Swagger Documentation
Single source of truth: `backend/docs/swagger.json`. Swagger UI exposed in development only (`/swagger/index.html`), never in production. `make sync-api` regenerates both swagger and frontend types.
