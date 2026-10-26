# ADR-019: Hash refresh tokens with SHA-256 instead of bcrypt

**Status:** Accepted
**Supersedes:** The statements in `05-database-design.md`, `07-security-plan.md` and the tech-stack table that refresh tokens are stored as bcrypt hashes.

## Context
Refresh tokens are 256-bit random values, not human-chosen secrets. bcrypt's slowness protects low-entropy passwords from guessing; it adds nothing for a 256-bit random token, but it has real costs: a salted bcrypt hash cannot be looked up by value, which forced a `sessionID.rawToken` composite format and a placeholder-session workaround, and every refresh burns ~100 ms of CPU, which is a denial-of-service lever on a free-tier instance.

## Decision
- The refresh token is 32 bytes from `crypto/rand`, sent to the client base64url-encoded.
- `sessions.refresh_token_hash` stores the lowercase hex SHA-256 of the token (a `CHECK` enforces 64 hex characters). Lookup is a direct indexed query on the hash.
- Passwords, PINs and backup codes stay on bcrypt (cost 12): they are low-entropy.
- Rotation is mandatory. On refresh the old session is marked `is_revoked = true, rotated_at = NOW()` and a new session is created with `rotated_from` pointing at it.
- Reuse detection: presenting a rotated token again within the 30-second grace window returns 401 without revoking anything (concurrent tabs); outside the window it revokes all of the user's sessions. Revoked sessions are kept until they expire so reuse remains detectable.

## Consequences
**Positive:** Simpler token format, no placeholder session, cheap constant-time verification, and reuse detection that works.
**Negative:** A database leak exposes hashes of high-entropy tokens only; this is considered safe, but the handoff's "bcrypt everywhere" statement no longer holds for refresh tokens.
**Related:** ADR-016, ADR-018.

## Docs to update in the same commit
- `05-database-design.md`: `sessions.refresh_token_hash` is a SHA-256 hex digest; add `rotated_at`.
- `07-security-plan.md`: describe rotation, the 30-second grace window and reuse detection.