# Platform Completeness Report

*Authoritative source of truth for productization gaps. Maintained by the arch/PM role. Updated after every scan round and after each verified fix.*

## How to update this file

1. **Findings must be verified by code inspection or regression test** — not assumed from TODO comments.
2. **Status values:** `[NEW]`, `[PARTIAL]`, `[FIXED]` (code exists, needs verification), `[DONE]` (verified by test/build), `[ACCEPTABLE]` (known limitation, documented).
3. **Every status change requires a commit hash** in the Commit column.
4. **After updating, run `go build ./...` and `make test`.**
5. **Cross-reference with `docs/platform-scan-state.md`** — both files must agree on counts.

## Summary

- Total findings: 14
- Done: 12
- Fixed (pending verification): 0
- Partial: 1
- Remaining: 1
- Last scan: 2026-07-14 round 9 (focus: B — Route Wiring)

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
| 10 | CIBA backchannel route | oauth/server/server.go | CIBA backchannel endpoint `/api/v1/oauth/backchannel` registered and invokes BackchannelAuthentication. Service-layer tests in ciba_flow_test.go exercise the flow. | [DONE] | 2934fd98 |
| 11 | Client Branding persistence | oauth/internal/server/client_branding.go | `handleClientBranding` uses `brandingAdapterVar` (PG-backed adapter with mem fallback). Regression test `TestGapRegression_ClientBranding_UsesAdapter` passes. | [DONE] | 2934fd98 |

### LOW Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 12 | GeoIP | gateway/middleware/geoip.go | Detects private IPs (returns 'LOCAL'). MaxMind GeoLite2 DB integration pending. | [PARTIAL] | arch |
| 13 | Frontend page completeness | console/src/app/ | Key pages exist and are wired to APIs. | [DONE] | frontend |
| 14 | HSM/KMS key provider | pkg/crypto, services/auth, services/oauth | JWT/SAML signing keys stored in PEM files on disk. No PKCS#11, Cloud KMS, or Vault Transit provider. Research docs exist; implementation pending. | [NEW] | arch |

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
| 2026-07-14 | CIBA + Client Branding verification | 0 | 2 verified as DONE |
| 2026-07-14 | Round 8 — Focus A (Interface Integrity) | +4 route/handler interface gaps | 4 (gateway TODO, policy route aliases) |
| 2026-07-14 | Round 9 — Focus B (Route Wiring) | +3 missing gateway prefixes | 3 (/api/v1/oauth, /api/v1/identity, /api/v1/agents) |

## Remaining Real Gaps (post-audit)

1. **GeoIP MaxMind integration** (LOW, [PARTIAL]) — gateway/middleware/geoip.go
   - Private IP detection works; MaxMind GeoLite2 DB integration pending.
2. **HSM/KMS key provider** (HIGH, [NEW]) — pkg/crypto, services/auth, services/oauth
   - JWT/SAML signing keys still use disk PEM files; needs PKCS#11 / Cloud KMS / Vault Transit provider.

## Next Actions

- Round 9 (odd, Focus B): Route wiring scan
- Round 10 (even): E2E regression test run (`deploy/e2e-docker-test.sh`) — blocked by Docker infra, see docs/research/docker-e2e-infra-gap.md
- Research backlog: HSM/KMS key provider design, OAuth 2.1 enforcement, PQC migration, passkey health dashboard