# Platform Scan State

## Current round: 1
## Last scan focus: (none)
## Next scan focus: A (Stub/Placeholder/TODO)
## Total findings: 0
## Fixed: 0
## Remaining: 0

## Current top 5 incomplete features:
1. DCR grant_types not persisted — HIGH — [NEW] M2M demo blocked
2. MFA TOTP hardcoded secret — HIGH — [NEW] Security risk
3. SAML SP-Initiated SSO placeholder — HIGH — [NEW] Enterprise feature missing
4. SAML SLO placeholder — MEDIUM — [NEW]
5. Device-Bound SSO all TODOs — MEDIUM — [NEW] Empty shell
6. Backup Codes in-memory only — MEDIUM — [NEW] Lost on restart
7. NoopIdentityClient fallback — MEDIUM — [NEW] Social login breaks
8. GeoIP placeholder — LOW — [NEW]
9. CIBA backchannel route not registered — MEDIUM — [NEW]

## Scan rotation order:
A → B → C → D → E → F → G → A → ...

## Round mapping:
- Odd rounds (1,3,5,...): Workflow B (completeness scan)
- Even rounds (2,4,6,...): Workflow A (E2E tests)
- Scan focus rotates within odd rounds: A→B→C→D→E→F→G→A
