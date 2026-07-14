# SAML Federation Ecosystem

> SAML 2.0 federation landscape and GGID's SAML implementation status.

---

## SAML in 2026

Despite being a 2005 standard, SAML remains dominant in enterprise SSO:

| Segment | SAML Usage |
|---------|-----------|
| Enterprise B2B SSO | 70%+ market share |
| Higher Education (Shibboleth) | Near-universal |
| Government (PIV/CAC) | Widespread |
| Healthcare (HL7/SMART) | Growing |

### Federation Hubs

| Hub | Members |
|-----|---------|
| InCommon (US education) | 1,000+ institutions |
| UK Federation | 800+ |
| eduGAIN (global) | 60+ federations |

---

## GGID SAML Implementation

### Current Capabilities

| Feature | Status |
|---------|--------|
| SP metadata generation | Done (`GenerateSPMetadata`) |
| SAML ACS (Assertion Consumer Service) | Done |
| SAML SSO redirect | Done (`EncodeForRedirect`) |
| Signature verification | Done (RSA + ECDSA) |
| Signed assertion verification | Done |
| Multi-tenant SAML | Done (per-tenant IdP config) |
| SAML logout (SLO) | Partial |

### Coverage: 91.1%

SAML package coverage at 91.1% (see saml coverage analysis).

---

## Competitor Comparison

| Feature | GGID | Auth0 | Keycloak | Okta |
|--------|------|-------|----------|------|
| SAML SP | Yes | Yes | Yes (advanced) | Yes |
| SAML IdP | No | Yes | Yes | Yes |
| Federation hub | No | No | Yes | Limited |
| SAML logout | Partial | Yes | Yes | Yes |
| Encryption | Yes | Yes | Yes | Yes |

---

## Gaps and Recommendations

1. **SAML IdP role** — GGID is SP only. Enterprise customers may need GGID as IdP.
2. **SLO** — Single Logout needs completion.
3. **Federation metadata aggregation** — Import aggregate metadata from InCommon/eduGAIN.

Priority: P2 (enterprise demand, not blocking).

---

*See: [Per-Tenant IdP](per-tenant-idp.md) | [Gap Closure Report](gap-closure-report.md)*

*Last updated: 2025-07-11*
