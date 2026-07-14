# Platform Scan State

## Current round: 8
## Last scan focus: A (Interface Integrity)
## Next scan focus: B (Route Wiring)
## Total findings: 22
## Fixed: 22
## Remaining: 4
## Partial: 2

## Current top incomplete features:
1. Backup Codes in-memory — MEDIUM — [FIXED] PostgreSQL impl (backend, bc6c23ee)
2. Custom Scopes in-memory — MEDIUM — [FIXED] PostgreSQL impl (backend, bc6c23ee)
3. PAR store in-memory — LOW — [ACCEPTABLE] 60s expiry
4. Device Code store in-memory — LOW — [ACCEPTABLE] 15min expiry
5. DPoP bindings in-memory — LOW — [ACCEPTABLE] session-scoped
6. Delegation chains in-memory — LOW — [ACCEPTABLE] debug/audit
7. Agent consent/review in-memory — LOW — [ACCEPTABLE] session-scoped
8. Client branding in-memory — MEDIUM — [NEW] brandingStore map
9. GeoIP — LOW — [PARTIAL] Private IP detection, MaxMind DB pending

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
- (round 5): MFA JIT TOTP random secret, Device-Bound SSO random signing key, agent token scope enforcement (arch)
- (round 6): Server coverage tests for identity health/tenant, OAuth helpers, org tree build/prune (arch)
- (round 7): Auth missing handlers wired to real service: /api/v1/auth/me, /api/v1/auth/mfa/status, /api/v1/auth/tokens (arch)
- (round 8): Auth handler interface integrity — login-security, device-bindings, rate-limits wired to auth service; gateway sysconfig store wired into router (arch)
