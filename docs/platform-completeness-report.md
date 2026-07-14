# Platform Completeness Report

*Auto-updated by cron-16 platform completeness scanner.*

## Summary
- Total findings: 9
- Fixed: 0
- Remaining: 9
- Last scan: 2026-07-14 initial scan

## Findings

### HIGH Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 1 | DCR grant_types | oauth/server/server.go | DCR accepts grant_types param but doesn't persist. Registered clients can't use client_credentials. | [NEW] | - |
| 2 | MFA TOTP Secret | auth/server/mfa_factors_handler.go:75 | Hardcoded "JBSWY3DPEHPK3PXP" instead of otp.NewTOTP(). All users share same secret. | [NEW] | - |
| 3 | SAML SP-Initiated SSO | oauth/server/server.go:858 | Returns {"note":"SP-initiated SSO redirect placeholder"}. No actual SAML AuthnRequest. | [NEW] | - |

### MEDIUM Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 4 | SAML SLO | oauth/server/server.go | Returns {"status":"saml_slo_initiated"}. No LogoutRequest/Response handling. | [NEW] | - |
| 5 | Device-Bound SSO | oauth/service/device_bound_sso.go | 6 TODOs. Empty shell: no WebAuthn verify, no JWT sign, no device_id compare. | [NEW] | - |
| 6 | Backup Codes Storage | auth/service/backup_codes.go | Uses inMemBackupCodeRepo. Lost on restart. Needs DB implementation. | [NEW] | - |
| 7 | NoopIdentityClient | auth/service/identity_client.go | All methods return "not configured". Social login user creation fails when Identity Service unavailable. | [NEW] | - |
| 8 | CIBA backchannel route | oauth/server/server.go | CIBA service fully implemented but backchannel_auth endpoint not registered in routes. Only config endpoint routed. | [NEW] | - |

### LOW Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 9 | GeoIP | gateway/middleware/geoip.go:92 | lookupCountry is placeholder. "In production, use MaxMind GeoLite2 DB". | [NEW] | - |

## Previously Fixed (Prior Scans)

| Feature | Was | Fixed By | Commit | Date |
|---------|------|----------|--------|------|
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
