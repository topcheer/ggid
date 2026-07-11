# Keycloak 26 Analysis

> Keycloak 26 new features and how GGID compares.

---

## Keycloak 26 Highlights (2025)

### 1. Organization Support (GA)
Keycloak 26 promotes Organizations from preview to GA:
- Multi-tenant via organizations (similar to GGID tenants)
- Per-organization identity provider linking
- Domain-based org routing

**GGID status**: Superior — GGID uses PostgreSQL RLS for hard database-level isolation. Keycloak relies on realm separation (softer isolation).

### 2. Device Bound SSO (Preview)
Keycloak 26 introduced device-bound SSO using WebAuthn:
- Bind SSO session to specific device
- Prevents session theft

**GGID status**: Building blocks exist (WebAuthn + JWT claims). Implementation estimated 4.5 days. See [Device-Bound SSO Analysis](device-bound-sso-analysis.md).

### 3. Improved Admin UI
- New organization management UI
- Improved user federation UI
- Dark mode support

**GGID status**: Next.js 15 console has 30+ pages, dark mode (commit cb13e9b). Competitive.

### 4. OAuth 2.1 Alignment
- Removed implicit flow
- PKCE required for all authorization code flows
- DPoP support improved

**GGID status**: Already aligned — PKCE enforced, DPoP implemented, no implicit flow.

### 5. Performance Improvements
- 40% faster token issuance (optimized JWT signing)
- Reduced memory footprint
- Improved connection pooling

**GGID status**: Go's goroutine model + pgx connection pooling already efficient. Benchmark needed.

---

## Feature Gap Analysis

| Feature | Keycloak 26 | GGID | Gap |
|---------|------------|------|-----|
| Multi-tenant (orgs) | GA (realm-based) | RLS-based | GGID better |
| Device-bound SSO | Preview | Planned (4.5d) | Keycloak ahead |
| SCIM 2.0 | No | Yes | GGID better |
| ABAC | No | Yes | GGID better |
| Audit hash chain | No | Yes | GGID better |
| Compliance templates | No | PCI/HIPAA/SOC2/GDPR | GGID better |
| AI Agent Identity | No | Yes | GGID better |
| User federation (LDAP) | Advanced | Yes (basic) | Keycloak better |
| SAML SP | Advanced | Yes | Keycloak better |
| Protocol broker | Yes | No | Keycloak better |
| Marketplace | Yes | Planned | Keycloak better |

---

## GGID Competitive Advantages

1. **Hard multi-tenant isolation** (RLS vs realm separation)
2. **ABAC + compliance templates** (Keycloak has neither)
3. **Audit hash chain** (tamper-evident — unique in market)
4. **AI Agent Identity** (no competitor has this)
5. **Go performance** (Keycloak is Java/JBoss — heavier)
6. **SCIM 2.0** (enterprise HR sync — Keycloak lacks)

## GGID Gaps to Close

1. **Device-bound SSO** — 4.5 days (P1)
2. **LDAP federation** — improve user filter flexibility (P2)
3. **SAML SP** — improve SP metadata exchange (P2)
4. **Protocol broker** — low priority (P3)

---

*See: [Competitive Analysis](competitive-analysis.md) | [Gap Closure Report](gap-closure-report.md) | [Competitive Update 2026-07](competitive-update-2026-07.md)*

*Last updated: 2025-07-11*
