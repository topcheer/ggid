# Platform Completeness Report

*Authoritative source of truth for productization gaps. Maintained by the arch/PM role. Updated after every scan round and after each verified fix.*

## How to update this file

1. **Findings must be verified by code inspection or regression test** — not assumed from TODO comments.
2. **Status values:** `[NEW]`, `[PARTIAL]`, `[FIXED]` (code exists, needs verification), `[DONE]` (verified by test/build), `[ACCEPTABLE]` (known limitation, documented).
3. **Every status change requires a commit hash** in the Commit column.
4. **After updating, run `go build ./...` and `make test`.**
5. **Cross-reference with `docs/platform-scan-state.md`** — both files must agree on counts.

## Summary

- Total findings: 13
- Done: 8
- Fixed (pending verification): 1
- Partial: 1
- Remaining: 3
- Last scan: 2026-07-14 round 7 (focus: G — SDK Alignment)

## Findings

### HIGH Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 1 | DCR grant_types | oauth/service/oauth_service.go | DCR accepts grant_types param and persists via CreateClient → PG repo. Regression test: register client_credentials via DCR, then successfully obtain token. | [DONE] | ff6e2c0e |
| 2 | MFA TOTP Secret | auth/server/jit_mfa_handler.go | Hardcoded secret replaced with crypto/rand generated base32 secret. | [DONE] | backend |
| 3 | SAML SP-Initiated SSO | oauth/server/server.go | SP-initiated AuthnRequest generation and IdP redirect implemented. | [DONE] | backend/arch |
| 4 | Device-Bound SSO signing key | oauth/service/device_bound_sso.go | Hardcoded default HMAC key replaced with random 32-byte key. | [DONE] | backend |

### MEDIUM Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 5 | SAML SLO | oauth/server/server.go | `/saml/slo` and `/saml/idp/slo` handlers process LogoutRequest/Response. | [DONE] | arch |
| 6 | Device-Bound SSO | oauth/service/device_bound_sso.go | IssueDeviceBoundToken, VerifyDeviceBoundToken, signClaims, verifyClaims implemented. No remaining TODOs. | [DONE] | backend |
| 7 | Backup Codes Storage | auth/service/backup_codes.go, auth/cmd/main.go | `NewPgBackupCodeRepo(pool)` wired in main.go; table created via `EnsureSchema`. Falls back to in-memory only when pool is nil. | [DONE] | backend |
| 8 | Agent token scope enforcement | oauth/service/agent_identity.go | AgentTokenClaims carries scope; CheckAgentScope enforces it. | [DONE] | backend |
| 9 | NoopIdentityClient | auth/service/identity_client.go, auth/cmd/main.go | `NewHTTPIdentityClient` used when `IDENTITY_SERVICE_URL` is set; Noop is intentional degraded fallback. | [DONE] | backend |
| 10 | CIBA backchannel route | oauth/server/server.go | CIBA backchannel endpoint registered. | [FIXED] | arch (pending verification) |

### LOW Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 11 | GeoIP | gateway/middleware/geoip.go | Detects private IPs (returns 'LOCAL'). MaxMind GeoLite2 DB integration pending. | [PARTIAL] | arch |
| 12 | Frontend page completeness | console/src/app/ | Key pages exist and are wired to APIs. | [DONE] | frontend |

## Previously Fixed (Prior Scans)

| Feature | Was | Fixed By | Commit | Date |
|---------|------|----------|--------|------|
| CIBA backchannel route | Route not registered | arch | pending | 2026-07-14 |
| BotDetect not wired | BotDetect existed but not in Handler() chain | arch | - | 2026-07-12 |
| PII Obfuscate not called | pii.Obfuscate never called | backend | - | 2026-07-12 |
| CheckSessionTimeout not wired | Never invoked | backend | - | 2026-07-12 |
| i18n Translator not called | 0 call sites | backend | - | 2026-07-12 |
| Password Pepper not used | Not wired | arch/backend | - | 2026-07-12 |
| Hash Chain not enabled | Not wired | arch | - | 2026-07-12 |
| Go ERP NULL scan | products returned 0 | arch | 31fab80 | 2026-07-14 |
| Gateway DCR whitelist | DCR blocked by JWT middleware | arch | 8a446ab6 | 2026-07-14 |
| OAuth refresh_token | invalid_grant | backend | d949b958 | 2026-07-14 |
| OAuth Redis client init | Never called SetRedisClient | backend | 8c1b46d5 | 2026-07-14 |

## Scan History

| Date | Focus | New Findings | Fixed |
|------|-------|-------------|-------|
| 2026-07-14 | Initial manual scan | 9 | 0 |
| 2026-07-14 | Round 1 — Focus A | +1 | 1 |
| 2026-07-14 | Round 5 — Focus E | +2 | 3 |
| 2026-07-14 | Round 6 — Focus F | +3 coverage gaps | 3 |
| 2026-07-14 | Round 7 — Focus G | +3 SDK gaps | 3 |
| 2026-07-14 | Gap audit & deduplication | -5 false positives | 5 verified |

## Remaining Real Gaps (post-audit)

1. **GeoIP MaxMind integration** (LOW, PARTIAL) — gateway/middleware/geoip.go
2. **CIBA backchannel route verification** (MEDIUM, FIXED pending) — need functional test
3. **In-memory stores that should be persistent** (from scan-state):
   - Client Branding (MEDIUM) — `brandingStore` map in oauth service
   - Custom Scopes (MEDIUM) — already marked FIXED in scan-state, verify

## Next Actions

- Round 8 (even): E2E regression test run (`deploy/e2e-docker-test.sh`)
- Round 9 (odd, Focus A): Stub/placeholder scan with stricter verification rules
- Research backlog: ITDR fraud detection, OAuth 2.1 enforcement, PQC migration, passkey health dashboard