# Platform Scan State

## Current round: 1
## Last scan focus: A (Stub/Placeholder/TODO)
## Next scan focus: B (Route Wiring)
## Total findings: 10
## Fixed: 5
## Remaining: 4
## Partial: 1

## Current top incomplete features:
1. DCR grant_types — HIGH — [FIXED] backend verified persistence works, M2M token exchange PASS
2. MFA TOTP hardcoded secret — HIGH — [FIXED] commit 27db0fd9 (backend)
3. SAML SP-Initiated SSO — HIGH — [FIXED] 302 redirect with SAMLRequest (backend)
4. SAML SLO — MEDIUM — [NEW] still returns placeholder
5. Device-Bound SSO — MEDIUM — [NEW] 6 TODOs, empty shell
6. Backup Codes in-memory — MEDIUM — [NEW] inMemBackupCodeRepo
7. NoopIdentityClient — MEDIUM — [NEW] stub fallback
8. CIBA backchannel route — MEDIUM — [FIXED] route registered + gateway whitelisted
9. GeoIP placeholder — LOW — [PARTIAL] private IP detection added, MaxMind DB pending
10. Frontend pages — LOW — [DONE] 4 pages verified complete

## Scan rotation order:
A → B → C → D → E → F → G → A → ...

## Round mapping:
- Odd rounds (1,3,5,...): Workflow B (completeness scan)
- Even rounds (2,4,6,...): Workflow A (E2E tests)
- Scan focus rotates within odd rounds: A→B→C→D→E→F→G→A

## Commits this cycle:
- 0db7939d: CIBA backchannel route + GeoIP improvement (arch)
- ab3605ce: Gateway whitelist for CIBA (arch)
- 27db0fd9: MFA TOTP dynamic secret (backend)
- Backend: SAML SP-SSO AuthnRequest generation (backend)
