# Pikyon API Design

> Full endpoint and error reference: `docs/planning/06-api-design.md`. The generated contract is `backend/docs/swagger.json`; frontend types in `frontend/src/types/api.ts` are generated from it and must never be hand-written.

## Conventions

- Base path: `/api/v1`; JSON only; `snake_case` fields; UUID v4 IDs.
- Auth: `Authorization: Bearer <access token>`.
- Pagination: cursor-based only (`meta.next_cursor`, `meta.has_more`, `meta.returned`). No `total`.
- Batch operations: maximum 50 items.
- Errors: RFC 7807-style code/message/details.

## Envelopes

```json
{ "success": true, "data": {}, "message": "..." }

{ "success": true, "data": { "items": [] },
  "meta": { "next_cursor": "...", "has_more": true, "returned": 20 } }

{ "success": true, "data": { "job_id": "...", "status": "queued",
  "stream_url": "/api/v1/ai/jobs/.../stream" } }

{ "success": false,
  "error": { "code": "VALIDATION_ERROR", "message": "...",
             "details": [{ "field": "title", "message": "Title is required" }] } }
```

## Endpoint groups

```
/auth        register, login, logout, refresh, verify-email, resend-verification,
             forgot-password, reset-password, Google OAuth, 2FA, PIN
/users/me    profile, password, avatar, sessions, storage
/memories    CRUD, soft delete/restore, permanent delete, trash, bulk trash, empty trash
/media       upload-url, confirm, stream, thumbnail, delete
/ai          analyze, speech summarize/polish, job polling, job SSE stream
/sharing     create/list/revoke, view log, public shared view, reactions
/search      full-text search, suggestions
/notifications list, read, read-all, delete, unread count
```

## Contract automation

```
Go handler annotations
  → swaggo → backend/docs/swagger.json (single source of truth)
  → openapi-typescript → frontend/src/types/api.ts
```

`make sync-api` runs both steps. CI fails a PR when `swagger.json` is stale relative to handler annotations.

## Rate limits

Auth and PIN routes are backed by Postgres `rate_limit_buckets` (survives restarts); everything else uses the in-memory Ristretto-style limiter. The full table is in `docs/planning/06-api-design.md` §6.13.
