# Round 65 Deep E2E Findings — tx.Rollback Pattern Across Services

## Date: 2026-07-16

## Critical Finding: tx.Rollback Data Loss Pattern

**Pattern:** Multiple repository files across services used `tx.Rollback(ctx)` instead of `tx.Commit(ctx)` before returning success. This silently discarded ALL database writes.

**Root cause analysis:**
Developers copied a pattern that included `defer tx.Rollback(ctx)` as an error safety net, but then also explicitly called `tx.Rollback(ctx)` in the success path — rolling back before the function returned.

**Affected files (total 16 instances):**
- `services/identity/internal/repository/pg_repo.go` — 10 methods (fixed in 78114833)
- `services/auth/internal/repository/mfa_pg_repo.go` — 3 methods (fixed in 47bddbd6)
- `services/oauth/internal/repository/pg_repo.go` — 3 methods (fixed in 47bddbd6)

**Impact:**
- User updates (display_name, email) were lost
- Lock/unlock status changes were lost
- MFA setup/verify/disable were lost
- OAuth client create/update/delete were lost

**Lesson:** This bug pattern is invisible in unit tests (mocks don't use real transactions) and only manifests with real DB connections. Always verify write operations with integration tests or curl E2E.

## User DELETE: Working Correctly

DELETE /users/{id} performs soft delete:
- Sets `status='deleted'` and `deleted_at=NOW()`
- ListUsers query has `deleted_at IS NULL` filter (correct)
- GET by ID still returns the record (correct for audit trail)

## Audit Events: Not Published (P1)

All write operations produce zero audit events because no service has a NATS publisher wired up. This is an architectural gap requiring cross-service integration.

## Recommendations
1. Add a CI integration test that creates a resource, then queries it back to verify persistence
2. grep for `tx.Rollback(ctx)` without `defer` as a lint rule
3. Wire audit publishers into all 7 services
