# Platform Scan State

## Current round: 4
## Last scan focus: D (Data Persistence) + G (SDK Alignment)
## Next scan focus: E (Security Config)
## Total findings: 22
## Fixed: 11
## Remaining: 9
## Partial: 2

## Current top incomplete features:
1. Backup Codes in-memory — MEDIUM — [NEW] inMemBackupCodeRepo, lost on restart
2. PAR store in-memory — LOW — [NEW] parStore map, short-lived (expires in 60s)
3. Device Code store in-memory — LOW — [NEW] deviceCodeStore map, short-lived
4. Custom Scopes in-memory — MEDIUM — [NEW] customScopes map, admin config lost
5. DPoP bindings in-memory — LOW — [NEW] dpopBindings map
6. Delegation chains in-memory — LOW — [NEW] delegationChains map
7. Agent consent/review in-memory — LOW — [NEW] sync.Map stores
8. Client branding in-memory — MEDIUM — [NEW] brandingStore map
9. CIBA store in-memory — LOW — [NEW] cibaStore sync.Map (fallback)
10. GeoIP — LOW — [PARTIAL] Private IP detection, MaxMind DB pending
11. Scope i18n in-memory — LOW — [NEW] scopeDescStore (static defaults)

## SDK Feature Matrix: 9/9 × 10/10 = 100% COMPLETE
All 9 SDKs (Go, Rust, Python, Node, Java, Ruby, C#, Dart, PHP) have:
login, refresh, userinfo, jwks, rbac, abac, webhook, introspect, revoke, discovery

## Scan rotation order:
A → B → C → D → E → F → G → A → ...

## Round mapping:
- Odd rounds (1,3,5,...): Workflow B (completeness scan)
- Even rounds (2,4,6,...): Workflow A (E2E tests)
- Scan focus rotates within odd rounds: A→B→C→D→E→F→G→A

## Round 1 (Scan A - Stub/Placeholder): COMPLETE
## Round 2 (Scan B - Route Wiring): COMPLETE
## Round 3 (Scan C+F - Middleware+Test Gaps): COMPLETE (by frontend)
## Round 3 (Scan D+G - Persistence+SDK Matrix): COMPLETE (by arch)

## Risk Assessment of In-Memory Stores:
HIGH (must fix for production):
- Backup Codes (user-facing, security-critical)
- Custom Scopes (admin config, lost on restart)

MEDIUM (should fix):
- Client Branding (admin config)
- Introspection Cache (performance, safe to lose)

LOW (acceptable for now — short-lived or fallback):
- PAR store (60s expiry)
- Device Code (15min expiry)
- DPoP bindings (session-scoped)
- Delegation chains (debug/audit)
- Agent consent/review (session-scoped)
- CIBA store (15min expiry, Redis fallback exists)
- Scope i18n (static defaults)
- OAuth state store (short-lived CSRF)

## Commits this cycle:
- 0db7939d: CIBA backchannel route + GeoIP (arch)
- ab3605ce: Gateway whitelist for CIBA (arch)
- 27db0fd9: MFA TOTP dynamic secret (backend)
- 8e099f93: SAML 2.0 IdP implementation (arch)
- 1f9a36e0: GeoIP test fix (arch)
- a19495e4: Device-Bound SSO + NoopIdentityClient (backend)
- 8343bde3: SAML IdP roundtrip tests (arch)
- 7be8355c: Frontend loading/error states (frontend)
- eec3a7bd: Docs code block fixes (docs)
