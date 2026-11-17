# Pikyon Data Flows

> Living document. See `docs/planning/04-system-architecture.md` for the full reasoning behind each flow.

## Authentication

```
Register:  email/password → bcrypt(cost 12) → users row (lowercase email)
           → verification email → access + refresh token pair
Login:     constant-time bcrypt check (dummy hash for unknown accounts)
           → session row → access JWT (15m prod / 24h dev) + refresh token (7d)
Refresh:   token from body or X-Refresh-Token header
           → SHA-256 lookup → atomic rotate (old revoked, new inserted)
           → concurrent tab within 30s grace: harmless 401
           → replay after grace: revoke ALL user sessions
Access:    Authorization: Bearer <JWT>; claims sub, email, pwd_ver, iat, exp, iss
           pwd_ver compared with users.password_changed_at on each request
```

The access token lives in memory only. The refresh token lives in `localStorage` and is rotated on every use (ADR-016, ADR-019). The axios interceptor in `frontend/src/services/api.ts` uses a mutex plus a request queue so concurrent 401s trigger exactly one refresh.

## Media — zero direct serving

```
1. Select file; images compressed client-side
2. POST /media/upload-url → pending_uploads row + signed upload URL
3. Browser uploads directly to Supabase Storage (backend never touches bytes)
4. POST /media/confirm {upload_id, object_key} — no client metadata
5. Backend verifies ownership and key, fetches real metadata from Storage
6. media_files row created; pending_uploads row deleted
```

Orphaned uploads are deleted by an hourly job after `pending_uploads.expires_at`. Streaming always returns a 5-minute signed URL after ownership and PIN checks; media bytes are never proxied.

## AI — manually triggered

```
POST /ai/analyze → 202 + job_id → goroutine calls Gemini
                 → ai_jobs row: queued → processing → completed/failed
Frontend: GET /ai/jobs/:job_id polling first, SSE /stream layered on top,
          polling fallback every 3s if SSE drops
```

Nothing saves without user review, and AI is never triggered automatically on upload.

## Sharing

```
Owner shares a public memory → recipient email
  registered   → share token + Resend email
  unregistered → invitation email; share honoured after signup
Recipient opens link → authenticated → memory_view recorded → owner notified
Recipient may react (heart/wow/sad); no download, no re-share
```

Public share links resolve on the privileged pool by exact token match, because RLS cannot see the presented token. `shared_access` deliberately has no anonymous SELECT policy.
