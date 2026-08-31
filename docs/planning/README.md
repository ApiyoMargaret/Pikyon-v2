# Pikyon — Planning Documentation

Complete project planning document, split by section. This is the authoritative reference for Sprint 0 onward — every architectural, security, and process decision made during planning is recorded here, including corrections made along the way and why.

## Sections

| # | File | Covers |
|---|---|---|
| 1 | [01-project-overview.md](./01-project-overview.md) | Description, problem statement, goals, scope, success metrics, timeline |
| 2 | [02-features-scope.md](./02-features-scope.md) | Full feature breakdown by category, MVP vs future |
| 3 | [03-tech-stack.md](./03-tech-stack.md) | Every technology chosen and why, environment matrix, monorepo structure rules |
| 4 | [04-system-architecture.md](./04-system-architecture.md) | Request lifecycle, auth flow, media flow, AI flow, sharing flow, all architecture revisions |
| 5 | [05-database-design.md](./05-database-design.md) | Full schema (11 tables), RLS, indexes, cron jobs |
| 6 | [06-api-design.md](./06-api-design.md) | Every endpoint, response envelope, error codes, rate limits |
| 7 | [07-security-plan.md](./07-security-plan.md) | 7-layer defense in depth, OWASP coverage, security changelog |
| 8 | [08-ui-ux-plan.md](./08-ui-ux-plan.md) | Page structures, design system, design process notes |
| 9 | [09-testing-strategy.md](./09-testing-strategy.md) | Testing pyramid, isolation strategy, mandatory cross-tenant tests |
| 10 | [10-sprint-plan.md](./10-sprint-plan.md) | 8-sprint breakdown, Definition of Ready/Done |
| 11 | [11-deployment-plan.md](./11-deployment-plan.md) | Deployment topology, sequencing, rollback strategy |
| 12 | [12-constraints-risks.md](./12-constraints-risks.md) | Constraints, risk register, accepted MVP limitations |

## How to Read This

Each section includes a **changelog** at the bottom documenting anything that was revised during planning review, with the original approach, the correction, and the reason — nothing was silently changed. Cross-references between sections use relative filenames (e.g. "see `05-database-design.md`").

## Related Documentation (created during implementation, not planning)

```
docs/
├── planning/          ← this folder (Sections 1-12)
├── architecture/       ← living system diagrams, updated per feature
│   ├── system-architecture.md
│   ├── data-flow.md
│   ├── infrastructure.md
│   ├── security.md
│   └── api-design.md
└── adr/                ← Architecture Decision Records, one per major decision
    ├── ADR-001-supabase.md
    ├── ADR-002-pgx-driver.md
    ├── ADR-003-go-gin-backend.md
    ├── ADR-004-react-typescript.md
    ├── ADR-005-render-deployment.md
    ├── ADR-006-gemini-ai.md
    ├── ADR-007-resend-email.md
    ├── ADR-008-web-speech-api.md
    ├── ADR-009-github-actions.md
    ├── ADR-010-swaggo.md
    ├── ADR-011-client-compression.md
    ├── ADR-012-golang-migrate.md
    ├── ADR-013-csp-unsafe-inline-tradeoff.md
    ├── ADR-014-shadcn-ui.md
    ├── ADR-015-inter-only-typography.md
    ├── ADR-016-refresh-token-localstorage.md
    └── ADR-017-backend-verified-upload-metadata.md
```

> The `architecture/` and `adr/` folders are populated during Sprint 0 setup (repo scaffold) and updated continuously per the rule established in planning: **every new feature or implementation gets its architecture doc and ADR updated in the same PR that ships it, not after.**
