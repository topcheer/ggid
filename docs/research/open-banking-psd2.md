# Open Banking & PSD2

> SCA requirements, AIS/PIS consent flows, and financial-grade API analysis.

---

## PSD2 Requirements

### Strong Customer Authentication (SCA)

PSD2 mandates two-factor authentication for electronic payments:

| Factor | Examples |
|-------|----------|
| Knowledge | Password, PIN |
| Possession | Phone, hardware token |
| Inherence | Fingerprint, Face ID |

**GGID support**: TOTP (possession), WebAuthn (possession+inherence), password (knowledge). SCA compliant.

### Account Information Services (AIS)

Read-only access to bank accounts with explicit consent:
```
1. TPP requests access to account data
2. PSU authenticates via SCA
3. Bank returns consent token (scoped, time-limited)
4. TPP accesses account data within consent scope
```

### Payment Initiation Services (PIS)

Initiate payments on behalf of user:
```
1. TPP submits payment request
2. PSU authenticates via SCA (mandatory)
3. Bank executes payment
4. PSU receives confirmation
```

---

## FAPI (Financial-grade API)

FAPI 2.0 builds on OAuth 2.1 with stricter security:

| Requirement | FAPI 2.0 | GGID Status |
|------------|----------|-------------|
| mTLS or DPoP | Required | DPoP done, mTLS partial |
| PAR (Pushed Authorization) | Recommended | Not implemented |
| JARM (JWT Authorization Response) | Optional | Implemented |
| Opaque tokens | Optional | JWT (transparent) |
| Sender-constrained tokens | Required | DPoP done |

---

## GGID for Open Banking

| Feature | Status | Gap |
|---------|--------|-----|
| SCA (MFA) | Done | — |
| OAuth 2.1 | Done | — |
| DPoP | Done | — |
| Consent management | Partial | Need consent endpoint |
| PAR | Not done | P2 |
| Transaction signing | Not done | P2 |

### Recommendation

GGID is 70% ready for Open Banking. Gaps:
1. **Consent endpoint** — explicit consent grant/revoke API (3 days)
2. **PAR** — Pushed Authorization Request (2 days)
3. **Transaction signing** — PSD2 requires dynamic linking for payments (5 days)

Priority: P3 (not a target market, but feasible).

---

*See: [OAuth Scopes Design](oauth-scopes-design.md) | [Token Binding & DPoP](token-binding-dpop.md)*

*Last updated: 2025-07-11*
