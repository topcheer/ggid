# Platform Scan State

## Current round: 2
## Last scan focus: B (Route Wiring)
## Next scan focus: C (Middleware Chain)
## Total findings: 10
## Fixed: 7
## Remaining: 2
## Partial: 1

## Current top incomplete features:
1. DCR grant_types — HIGH — [FIXED] M2M token exchange PASS
2. MFA TOTP hardcoded secret — HIGH — [FIXED] commit 27db0fd9
3. SAML SP-Initiated SSO — HIGH — [FIXED] 302 redirect with SAMLRequest
4. SAML SLO — MEDIUM — [FIXED] Real SLO handler (processes LogoutRequest/Response)
5. SAML IdP — HIGH — [FIXED] Full IdP: BuildSAMLResponse+SignAssertion+IdPMetadata+3 endpoints (commit 8e099f93)
6. CIBA backchannel route — MEDIUM — [FIXED] Route registered + gateway whitelisted
7. GeoIP placeholder — LOW — [PARTIAL] Private IP detection, MaxMind DB pending
8. Device-Bound SSO — MEDIUM — [NEW] 6 TODOs, empty shell
9. Backup Codes in-memory — MEDIUM — [NEW] inMemBackupCodeRepo
10. NoopIdentityClient — MEDIUM — [NEW] stub fallback
11. Frontend pages — LOW — [DONE] 4 pages verified complete

## Scan rotation order:
A → B → C → D → E → F → G → A → ...

## Round mapping:
- Odd rounds (1,3,5,...): Workflow B (completeness scan)
- Even rounds (2,4,6,...): Workflow A (E2E tests)
- Scan focus rotates within odd rounds: A→B→C→D→E→F→G→A

## Round 1 (Scan A - Stub/Placeholder): COMPLETE
- Found: 9 issues (3 HIGH, 5 MEDIUM, 1 LOW)
- Fixed: 5 (DCR, MFA TOTP, SAML SP-SSO, CIBA route, GeoIP partial)
- SAML IdP implemented (commit 8e099f93, 422 lines + 12 tests)

## Round 2 (Scan B - Route Wiring): COMPLETE
- Found: SAML IdP routes not deployed (404) → FIXED by rebuilding OAuth image
- Found: SAML SLO placeholder → FIXED (replaced with real handler)
- SCIM routes: /api/v1/scim works through gateway, /scim/v2/ not proxied (acceptable - SCIM is on identity service)
- All gateway public paths verified correct
- OAuth regression: PASS (Discovery 200, JWKS 200)

## Commits this cycle:
- 0db7939d: CIBA backchannel route + GeoIP improvement (arch)
- ab3605ce: Gateway whitelist for CIBA (arch)
- 27db0fd9: MFA TOTP dynamic secret (backend)
- 8e099f93: SAML 2.0 IdP implementation (arch) — 422 lines + 12 tests
- 1f9a36e0: GeoIP test fix (arch)
