# Section 9: Testing Strategy

## 9.1 Testing Pyramid

```
        ┌──────────────┐
        │   E2E (few)  │  ← Future sprint (Playwright)
        ├──────────────┤
        │ Integration  │  ← API endpoint tests, real test DB
        ├──────────────┤
        │  Unit (most) │  ← Services, handlers, components
        └──────────────┘
```

## 9.2 Backend Testing (Go) — Revised

```
Tool:        testing (stdlib) + testify
Isolation:   schema-per-test-package, NOT transaction rollback
             — see rationale below
Coverage:    70% minimum on internal/*, enforced in CI
```

### Isolation Strategy — Revised
Original plan: wrap each test in a transaction, roll back after. **Corrected**: this breaks the moment a handler spawns an async goroutine (AI job processing, §4.7 of architecture doc) or calls a Supabase Storage RPC outside the wrapped transaction boundary — the goroutine can outlive the rollback or commit against a connection outside the test's tx entirely.

```
REVISED — schema-per-test-package:
  Before each test PACKAGE run:
    CREATE SCHEMA test_pkg_<name>;
    Run migrations against that schema
  After package run:
    DROP SCHEMA test_pkg_<name> CASCADE;

  This correctly isolates async goroutines and Storage RPC calls,
  and allows parallel test packages (go test ./... -parallel)
  without collision. Trade-off: slower than transaction rollback
  per-test — accepted in favor of correctness for async-heavy code.
```

### Mandatory Cross-Tenant Testing — Added
Originally "RLS policy manually verified" was a checklist item. **Corrected**: manual security verification is a liability under sprint pressure. Now a mandatory, automated pattern:

```go
// pkg/testutil/tenancy.go provides reusable helpers
func TestGetMemory_UnauthorizedUser_Returns404(t *testing.T) {
    userA_Token := setupUserA(t)
    userB_Token := setupUserB(t)
    memoryID := createMemoryForUserA(t, userA_Token)

    resp := makeGETRequest(t, "/api/v1/memories/"+memoryID, userB_Token)
    assert.Equal(t, http.StatusNotFound, resp.Code)
}
```

Every new resource-owning table (`memories`, `media_files`, `shared_access`, etc.) **must** ship with:
```
Test_Get_UnauthorizedUser_Returns404
Test_Update_UnauthorizedUser_Returns404
Test_Delete_UnauthorizedUser_Returns404
```
This is a hard PR-merge blocker, not a suggestion, for any handler touching a resource-owning table.

## 9.3 Frontend Testing (React) — Revised

```
Tool:        Vitest + React Testing Library + MSW
Coverage:    60% minimum on src/hooks, src/utils, src/features
```

### API Mocking — Revised
Originally unspecified how HTTP calls would be mocked. **Corrected**: standardized on **MSW (Mock Service Worker)**, which intercepts at the network layer rather than mocking `axios`/`fetch` directly — this means the real data-fetching pipeline (React Query + the axios refresh-token-mutex interceptor from `04-system-architecture.md` §4.4) actually runs during tests instead of being bypassed.

```
frontend/src/test/mocks/
  handlers.ts   ← MSW request handlers, mirrors frontend/src/types/api.ts
  server.ts     ← MSW server setup for Vitest
```

## 9.4 Migration Testing — Added
Originally the Definition of Done only verified `.up.sql` applied cleanly, never testing `.down.sql`. An untested rollback script is a false safety net.

```
□ Schema migration UP and DOWN scripts verified against
  local/CI test database (migrate up → migrate down → migrate up
  again — confirms idempotent round-trip)
```
CI runs this sequence automatically for any PR touching `backend/migrations/`.

## 9.5 Definition of Done Per Feature

```
□ Unit tests for new service/business logic
□ Integration test for new API endpoint (happy path + 1 error case)
□ Cross-tenant test (Unauthorized_Returns404) for any new
  resource-owning table — MANDATORY
□ Frontend unit test using MSW for new hook/utility involving API
□ Migration UP/DOWN round-trip verified, if schema changed
□ No reduction in existing coverage percentage
```

## 9.6 CI Test Gate
```
go test ./... -cover        → fails PR if coverage drops
vitest run --coverage       → fails PR if coverage drops
golangci-lint run / eslint .
migrate up → down → up      → for PRs touching backend/migrations/
```

## 9.7 Deferred to Future Sprint
E2E (Playwright), load testing (k6), visual regression testing (Chromatic/Percy).

## 9.8 Testing Strategy Changelog

| Area | Original | Revised | Reason |
|---|---|---|---|
| Test isolation | Transaction rollback per test | Schema-per-test-package | Async goroutines/Storage RPCs break the rollback boundary |
| RLS verification | Manual checklist item | Automated mandatory cross-tenant tests | Manual security checks get skipped under pressure |
| Frontend API mocking | Unspecified | MSW (network-layer interception) | Exercises the real fetch pipeline, including the refresh mutex |
| Migration testing | Up-only | Up→down→up round-trip verified | Untested rollback scripts are a false safety net |
