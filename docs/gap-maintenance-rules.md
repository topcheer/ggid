# Gap Maintenance Rules (Team Agreement)

> This document defines how the team maintains productization gaps. All teammates must follow these rules when updating `docs/platform-completeness-report.md` or `docs/platform-scan-state.md`.

## 1. Source of Truth

- **Primary:** `docs/platform-completeness-report.md`
- **Secondary:** `docs/platform-scan-state.md`
- Counts and statuses must be kept in sync between both files.

## 2. Roles

- **arch/PM** owns gap maintenance:
  - Verifies every reported gap by code inspection or regression test
  - Decides whether a gap is real, false positive, or acceptable
  - Assigns fixes to backend/frontend/docs teammates
  - Fills backlog with research-driven gaps
- **backend** may only report gaps in `services/` files they own
- **frontend** may only report gaps in `console/src/` files they own
- **docs** may only report gaps in `docs/` files they own

## 3. Status Definitions

| Status | Meaning | When to use |
|--------|---------|-------------|
| `[NEW]` | Gap discovered, not yet fixed | First time a gap is reported |
| `[PARTIAL]` | Partially implemented, known remaining work | Some parts done but not production-ready |
| `[FIXED]` | Code change committed, but not yet verified | After commit, before verification |
| `[DONE]` | Verified by build, test, or E2E | After `make test` passes or regression test exists |
| `[ACCEPTABLE]` | Known limitation, documented and accepted | Short-lived in-memory stores, debug features |

## 4. Before Marking DONE

You MUST have one of the following:
- A regression test that exercises the feature end-to-end
- A passing `go test` on the affected package
- A passing E2E test or Docker compose test
- A screenshot/video of the console page working (for frontend)

## 5. False Positives

If a reported gap is found to be already implemented:
- Change status to `[DONE]`
- Add explanation: "Verified by code inspection / regression test"
- Add commit hash if a new test was added
- Do NOT delete the finding; keep it in the report for audit history

## 6. Communication

- When you change a gap status, announce in lanchat team channel with: `gap #X -> [STATUS] (commit, verification method)`
- Every task assignment from arch must include: "完成后用 lanchat DM 向 arch 回报：commit hash + make test 结果"
- Do not create `coverage_sprint*_test.go` files; use meaningful test names

## 7. Research Backlog

arch/PM must continuously research IAM/OAuth/security trends and fill backlog with:
- Competitive gaps
- Compliance requirements (PIPL, GDPR, CRA, NIS2)
- Emerging standards (OAuth 2.1, FAPI, DPoP, PQC, passkeys)
- New findings go to `docs/research/*.md` first, then `docs/team-backlog.md`

## 8. Weekly Review

Every cron-1 PM cycle includes:
1. Re-read both gap documents
2. Verify counts match
3. Check for stale `[FIXED]` items that need promotion to `[DONE]`
4. Close or update `[NEW]` items that have been resolved
5. Add research-driven items to backlog

## 9. Penalties

- Teammates who mark `[DONE]` without verification will have their status reverted to `[FIXED]`
- False positives that are not cleaned up within 2 rounds become arch's responsibility
- Unreported new gaps found by arch during research will be added to the current owner's backlog