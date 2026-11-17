# Pikyon Security Architecture

> Full detail and rationale: `docs/planning/07-security-plan.md`. Decisions: ADR-013, ADR-016, ADR-017, ADR-018, ADR-019.

## Defense in depth

```
1 Transport      HTTPS, HSTS, strict CSP, X-Frame-Options, nosniff, Permissions-Policy
2 Authentication bcrypt(cost 12), JWT + pwd_ver, Google OAuth, TOTP 2FA, email verification
3 Authorization  JWT claims only, ownership checks, 404 for missing and not-yours
4 Validation     validator DTOs, parameterized SQL, server-side file checks
5 Database       RLS + restricted role + column grants, secrets unreadable by app role
6 Media          private bucket, UUID object keys, 5-minute signed URLs, PIN token
7 Hardening      Postgres-backed auth rate limits, safe errors, CORS allow-list, no PII logs
```

## Tenant isolation (ADR-018)

- `pikyon_app` is `NOBYPASSRLS` and holds only the grants it needs. All user-scoped requests run through `db.Pools.WithUser`, which sets `app.current_user_id` transaction-locally.
- Policies read `pikyon.current_user_id()` and are scoped `TO pikyon_app`; empty context returns no rows.
- The privileged owner pool is limited to flows with no user context or deliberate cross-tenant work.
- `anon` and `authenticated` have no grants on any table; the Supabase Data API is not used.
- Composite foreign keys (`memories (id, user_id)` and children) make cross-tenant child rows impossible even if application code is wrong.
- Cross-tenant behaviour is covered by `backend/migrations/tests/rls_isolation.sql` and the mandatory application tests (`pkg/testutil/tenancy.go`).

## Tokens

| Token | Storage | Lifetime | Notes |
|---|---|---|---|
| Access JWT | memory (React state) | 15m prod / 24h dev | HS256, issuer `pikyon-api`, `pwd_ver` claim |
| Refresh token | `localStorage` | 7 days | 256-bit random, SHA-256 hash stored (ADR-019), mandatory rotation |
| PIN token | client, per memory | 15 minutes | `X-Pin-Token`, scoped to one memory |
| Share token | share URL | owner-controlled | exact-match lookup on privileged pool |

Refresh reuse inside the 30-second grace window is treated as a harmless race; reuse after it revokes every session for the user.

## Known accepted tradeoffs

- `style-src 'unsafe-inline'` in CSP (ADR-013).
- Refresh token in `localStorage` rather than an httpOnly cookie (ADR-016) until a custom domain is affordable.
- Auth/PIN rate limits are Postgres-backed; general limits are in-memory and reset on restart.
