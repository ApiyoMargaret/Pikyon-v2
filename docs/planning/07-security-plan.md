# Section 7: Security Plan

## 7.1 Philosophy — Defense in Depth
No single control is relied on alone; every layer assumes the layer above it already failed.

```
Layer 1: Transport      Layer 5: Database
Layer 2: Authentication Layer 6: Media
Layer 3: Authorization  Layer 7: Application Hardening
Layer 4: Input Validation
```

## 7.2 Layer 1 — Transport Security

```
HTTPS enforced (Render/Vercel managed), TLS 1.2+ (1.3 preferred)

Security headers on every response:
  Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
  Content-Security-Policy:
    default-src 'self';
    img-src 'self' https://*.supabase.co data:;
    script-src 'self';
    style-src 'self' 'unsafe-inline';   -- accepted MVP tradeoff, see ADR-013
    connect-src 'self' https://*.supabase.co
  X-Frame-Options: DENY
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin
  Permissions-Policy: camera=(), microphone=(self), geolocation=()
    -- microphone=(self) required for Web Speech API
```

> `style-src 'unsafe-inline'` is a known, accepted tradeoff for MVP — Framer Motion and Tailwind inject inline styles. Full removal requires nonce- or hash-based CSP, a non-trivial build change deferred to a future sprint (ADR-013).

## 7.3 Layer 2 — Authentication

### Password Security
bcrypt cost 12, min 8 chars with uppercase/lowercase/number, strength indicator shown client-side.

### Token Strategy — Revised (critical correction)
Original plan: refresh token as an httpOnly, `SameSite=Strict` cookie. **First correction** (mid-review): switched to `SameSite=None; Secure` after realizing Vercel and Render are cross-site domains and `Strict` cookies wouldn't be sent. **Second, deeper correction**: even `SameSite=None; Secure` isn't sufficient — modern browser ITP (Safari) and ETP (Firefox) partition or block third-party cookies by a *separate mechanism* from the SameSite policy entirely, regardless of flag.

**Final resolution** (free-tier compatible, no custom domain purchased):
```
Access token:  Authorization: Bearer <token>, stored in-memory only
               (React context, not localStorage)
Refresh token: stored in localStorage
               - Accepted tradeoff: reopens XSS-theft surface that
                 httpOnly cookies exist specifically to prevent
               - Mitigated by: mandatory rotation on every refresh
                 (old token invalidated immediately), rotation
                 chain tracked via sessions.rotated_from, strict CSP
               - This is explicitly the WEAKER of two options. The
                 stronger option (custom domain, ~$3-15/yr, restores
                 httpOnly cookie viability) was declined per budget
                 constraint. Documented for revisit.
```

### Access Token Claims
```json
{ "sub": "user-uuid", "email": "...", "pwd_ver": 1720000000, "iat": ..., "exp": ..., "iss": "pikyon-api" }
```
`pwd_ver` (unix timestamp of `users.password_changed_at`) is compared on every request — if older than the DB value, token is rejected even if not expired. This instantly invalidates all issued tokens on password reset or forced logout, with zero blocklist infrastructure.

### Access Token Expiry — Revised
15 minutes in production (was 1 hour) — shortened specifically because the token-invalidation gap (a stolen token staying valid until natural expiry) is more consequential without an httpOnly-cookie-protected refresh flow. 24hr in development for convenience.

### Google OAuth
Authorization Code flow, `state` parameter for CSRF, ID token verified against Google's public keys.

### 2FA (TOTP)
RFC 6238, Google Authenticator compatible, AES-256 encrypted secret at rest, 8 bcrypt-hashed backup codes, 30s window with 1-step tolerance.

### Email Verification / Password Reset
64-byte hex tokens, single-use, 24hr (verification) / 1hr (reset) expiry. Password reset invalidates all active sessions.

## 7.4 Layer 3 — Authorization

```
Every protected route:
  1. JWT signature + expiry + pwd_ver claim validated
  2. UserID extracted from claims — NEVER from request body/query
  3. All DB queries scoped to user_id = JWT.sub (RLS enforces this
     even if application code has a bug)
  4. 404 returned for both "not found" and "not yours" — prevents
     information leakage about resource existence
  5. PIN token (X-Pin-Token) required for locked memories,
     scoped to a single memory_id, 15 min expiry
  6. Share token required for shared memory access, checked for
     revocation/expiry/max_views
```

## 7.5 Layer 4 — Input Validation

```
Backend: go-playground/validator on all DTOs — email format,
  password complexity, UUID format, ISO 8601 dates, enum values,
  array length caps (max 50 for batch ops), string length limits.
  Parameterized queries ONLY, pgx prepared statements — no raw
  SQL concatenation anywhere.

Frontend: Zod schemas — UX convenience only, server is always
  authoritative.

File validation: MIME type checked server-side (not just
  extension — see §7.6), size checked before signed URL issued.
  No executables accepted under any circumstance.
```

## 7.6 Layer 5 — Database Security

RLS enabled on every table (see `05-database-design.md`). SSL required on all connections. Service-role key used only by backend cron jobs, never exposed to frontend. Sensitive data (passwords, refresh tokens, PINs, backup codes) stored as bcrypt hashes only; TOTP secrets AES-256 encrypted. No PII in logs, ever.

## 7.7 Layer 6 — Media Security

```
Private Supabase bucket only, no public access ever.
Object keys: UUID-based, {user_id}/{memory_id}/{uuid}.{ext}
Pre-signed streaming URLs: 5-minute expiry, ownership validated
  before signing, PIN token required for locked memories
Pre-signed upload URLs: 10-minute expiry, object key cannot be
  changed by client, backend verifies + fetches real metadata
  from Supabase on confirmation (see 06-api-design.md §6.7 —
  closes the metadata-spoofing vulnerability)
Orphaned uploads cleaned up after 1 hour (pending_uploads cron)
URLs never stored in DB, never appear in frontend HTML source
```

## 7.8 Layer 7 — Application Hardening

### Rate Limiting — Revised
Postgres-backed for auth/PIN routes (persists across Render restarts, works across future horizontal scaling), Ristretto in-memory for everything else. See `04-system-architecture.md` §4.10 and `06-api-design.md` §6.13 for the full reasoning — this was a correction from an initial all-in-memory plan that would have silently reset brute-force counters on every deploy.

### Error Handling
Production: generic safe messages only, no stack traces, no internal path exposure.

### Secrets Management
GitHub Secrets (CI/CD) + Render/Vercel environment variables. Never committed. `.env.production` gitignored everywhere; only `.env.example` (placeholders) is committed.

### CORS Policy
```
Development: allow localhost:5173 only
Staging:     allow Vercel preview URL only
Production:  allow pikyon.vercel.app only — no wildcard, ever
Credentials: true (still required for other cookie-based flows)
Allowed headers: Authorization, Content-Type, X-Pin-Token,
                 X-Confirm-Delete, X-Refresh-Token
```

### Logging Security
Structured JSON (slog), no PII, no tokens/secrets, request IDs for tracing. ERROR/WARN only in production.

### Dependency Security
`govulncheck` (Go) and `npm audit` in CI, Dependabot enabled.

### Account Security
Failed logins logged with IP. Suspicious activity (multiple failed logins from different IPs) flagged for a future-sprint notification. Password reset invalidates all sessions. Account deletion purges data within 30 days.

## 7.9 OWASP Top 10 Coverage

| Risk | Mitigation |
|---|---|
| A01 Broken Access Control | RLS everywhere, ownership checks, JWT-claims-only user identity |
| A02 Cryptographic Failures | bcrypt, AES-256, TLS 1.2+ |
| A03 Injection | Parameterized queries only, pgx prepared statements |
| A04 Insecure Design | 7-layer defense in depth |
| A05 Security Misconfiguration | Security headers, CORS whitelist, no swagger in prod |
| A06 Vulnerable Components | govulncheck, npm audit, Dependabot |
| A07 Auth Failures | bcrypt, JWT + pwd_ver invalidation, 2FA, Postgres-backed rate limiting |
| A08 Data Integrity Failures | Backend-verified upload metadata, parameterized queries |
| A09 Logging Failures | Structured logging, no PII |
| A10 SSRF | No user-controlled URLs fetched server-side, signed URLs only |

## 7.10 Security Checklist Per Sprint

```
□ No hardcoded secrets
□ New endpoints require JWT where appropriate + rate limiting
□ Input validation on all new fields
□ RLS policy written for every new table
□ Parameterized queries only, no raw SQL
□ No new public storage buckets
□ Generic error messages, no stack traces
□ CORS not widened
□ govulncheck / npm audit pass
```

## 7.11 Security Decisions Changelog

| Area | Original | Revised | Reason |
|---|---|---|---|
| Refresh token storage | httpOnly SameSite=Strict cookie | localStorage + mandatory rotation | Cross-domain (Vercel/Render) + browser ITP/ETP partitioning defeats cookie-based approach without a paid custom domain |
| Access token expiry (prod) | 1 hour | 15 minutes | Compensates for weaker refresh-token storage |
| Rate limiting (auth/PIN) | Ristretto in-memory | Postgres-backed `rate_limit_buckets` | Restart wipes in-memory counters, reopening brute-force window |
| CSP style-src | Not specified | `unsafe-inline` accepted, documented tradeoff (ADR-013) | Framer Motion/Tailwind inline style dependency |
