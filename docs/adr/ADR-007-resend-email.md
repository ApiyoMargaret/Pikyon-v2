# ADR-007: Use Resend for transactional email

**Status:** Accepted (pending the sender-domain decision, G2)

## Context
Pikyon needs transactional email: verification, password reset, share invitations and notification digests, without a paid email budget.

## Decision
Use Resend for all transactional email delivery. Email templates live in `backend/internal/email/templates/` and are compiled into the Go binary with `//go:embed`, so the running binary has no runtime file-path dependency.

## Consequences
**Positive:** Free tier (3000 emails/month) covers MVP scale; simple API; good deliverability compared with self-managed SMTP.
**Negative:** Resend's sandbox sender cannot deliver to arbitrary recipients until a sending domain is verified. A custom domain was ruled out on budget, so this decision is blocked by G2 in the handoff: choose a cheap domain, switch provider, or stub verification for Sprint 1.
**Related:** ADR-005; `docs/planning/11-deployment-plan.md`; handoff G2.
