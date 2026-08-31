# ADR-016: Store refresh token in localStorage instead of httpOnly cookie

**Status:** Accepted (MVP), revisit when custom domain budget available

## Context
The frontend (Vercel, `*.vercel.app`) and backend (Render, `*.onrender.com`) are hosted on different top-level domains. The original plan set the refresh token as an httpOnly cookie with `SameSite=Strict`, which browsers treat as cross-site in this topology and simply don't send.

A first correction switched to `SameSite=None; Secure`, which addresses the SameSite policy specifically — but browser Intelligent/Enhanced Tracking Prevention (Safari ITP, Firefox ETP) partitions or blocks third-party cookies via a *separate* mechanism, independent of the SameSite flag. This was identified as a deeper, unresolved issue: even `SameSite=None; Secure` cookies can be silently dropped or partitioned in this cross-domain setup.

The fully correct fix — a custom domain with `api.pikyon.app` / `app.pikyon.app` sharing an apex domain — was declined per an explicit no-budget constraint for MVP infrastructure.

## Decision
- Access token: kept in-memory (React state/context), sent via `Authorization: Bearer` header — unchanged from original plan.
- Refresh token: stored in `localStorage` instead of an httpOnly cookie.
- Mandatory refresh-token rotation: every refresh call issues a new refresh token and immediately invalidates the previous one (`sessions.rotated_from` tracks the chain).
- Access token expiry shortened to 15 minutes in production (from 1 hour) to reduce the exposure window of the weaker refresh-token storage.

## Consequences
**Positive:** Works correctly across the Vercel/Render cross-domain topology with zero infrastructure cost.
**Negative:** Reopens the XSS-theft attack surface that httpOnly cookies exist specifically to close — a successful XSS attack could read the refresh token from localStorage.
**Mitigation:** Strict CSP (`script-src 'self'`), mandatory token rotation limiting a stolen token's useful life to a single refresh cycle, and shortened access token TTL.
**Explicitly documented as the weaker of two valid options** — the custom-domain + httpOnly-cookie approach remains the stronger choice and should be revisited the moment budget allows (~$3–15/year for a domain).
