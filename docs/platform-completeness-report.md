# Platform Completeness Report

*Auto-updated by cron-16 platform completeness scanner.*

## Summary
- Total findings: 13
- Fixed: 4
- Partial: 1
- Remaining: 8
- Last scan: 2026-07-14 round 7 (focus: G — SDK Alignment)

## Findings

### HIGH Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 1 | DCR grant_types | oauth/server/server.go | DCR accepts grant_types param but doesn't persist. Registered clients can't use client_credentials. | [NEW] | - |
| 2 | MFA TOTP Secret | auth/server/jit_mfa_handler.go:59 | Hardcoded "JBSWY3DPEHPK3PXP" instead of crypto/rand generated secret. All JIT-enrolled users shared same secret. | [DONE] | arch |
| 3 | SAML SP-Initiated SSO | oauth/server/server.go:858 | Returns {"note":"SP-initiated SSO redirect placeholder"}. No actual SAML AuthnRequest. | [NEW] | - |
| 4 | Device-Bound SSO signing key | oauth/service/device_bound_sso.go:31 | Hardcoded default HMAC key in NewDeviceBoundSSO. Tokens could be forged by anyone knowing the package default. | [DONE] | arch |

### MEDIUM Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 5 | SAML SLO | oauth/server/server.go | Returns {"status":"saml_slo_initiated"}. No LogoutRequest/Response handling. | [NEW] | - |
| 6 | Device-Bound SSO | oauth/service/device_bound_sso.go | 6 TODOs. Empty shell: no WebAuthn verify, no JWT sign, no device_id compare. | [NEW] | - |
| 7 | Backup Codes Storage | auth/service/backup_codes.go | Uses inMemBackupCodeRepo. Lost on restart. Needs DB implementation. | [NEW] | - |
| 8 | Agent token scope enforcement | oauth/service/agent_identity.go:447 | CheckAgentScope is a no-op; issued agent tokens don't carry granted scope, so scope checks can't be enforced. | [DONE] | arch |
| 9 | NoopIdentityClient | auth/service/identity_client.go | All methods return "not configured". Social login user creation fails when Identity Service unavailable. | [NEW] | - |
| 10 | CIBA backchannel route | oauth/server/server.go | CIBA service fully implemented but backchannel_auth endpoint not registered in routes. Only config endpoint routed. | [FIXED] | arch (pending) |

### LOW Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 11 | GeoIP | gateway/middleware/geoip.go:92 | Was placeholder returning unknown. Now detects private IPs (returns 'LOCAL'). MaxMind GeoLite2 DB integration still pending. | [PARTIAL] | arch (pending) |
| 12 | Frontend page completeness | console/src/app/ | Verified 4 key pages exist and are fully functional: SAML settings, OAuth clients, API keys, Certificates. All have proper API wiring. | [DONE] | - |

## Previously Fixed (Prior Scans)

| Feature | Was | Fixed By | Commit | Date |
|---------|------|----------|--------|------|
| CIBA backchannel route | Route not registered, only config endpoint routed | arch | pending | 2026-07-14 |
| BotDetect not wired | BotDetect existed but not in Handler() chain | arch | - | 2026-07-12 |
| PII Obfuscate not called | pii.Obfuscate never called | backend | - | 2026-07-12 |
| CheckSessionTimeout not wired | Never invoked in request pipeline | backend | - | 2026-07-12 |
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
| 2026-07-14 | Initial manual scan (all dimensions) | 9 | 0 |
| 2026-07-14 | Round 1 — Focus A (Stub/Placeholder/TODO) | +1 (#10) | 1 (#8) + 1 partial (#9) |
| 2026-07-14 | Round 5 — Focus E (Security Config) | +2 (#4, #8) | 3 (#2, #4, #8) |
| 2026-07-14 | Round 6 — Focus F (Functional Depth Verification) | +3 coverage gaps | 3 (identity health/tenant, OAuth helpers, org tree build/prune) |
| 2026-07-14 | Round 7 — Focus G (SDK Alignment) | +3 SDK alignment gaps | 3 (auth /me, /mfa/status, /tokens wired to real service) |
