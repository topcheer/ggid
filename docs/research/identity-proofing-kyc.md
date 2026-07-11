# Identity Proofing & KYC Research

> NIST 800-63A IAL levels, ID.me/Stripe Identity comparison, GGID roadmap.

---

## NIST 800-63A Identity Assurance Levels

| Level | Name | Requirement | Example |
|-------|------|------------|--------|
| IAL1 | Self-asserted | No proof | Email + password |
| IAL2 | Verified | 1+ strong evidence | Government ID + selfie |
| IAL3 | In-person | Biometric capture | Notarized, in-person |

---

## Competitor Analysis

### ID.me
- IAL2 verified identity (government ID + facial match)
- Used by IRS, VA, Social Security
- FIDO2 + NIST 800-63A IAL2 certified

### Stripe Identity
- Document verification (passport, driver's license)
- Selfie video liveness check
- Watchlist screening
- API-based integration

### Onfido / Jumio
- Document + biometric verification
- Global coverage (190+ countries)
- Real-time verification

---

## GGID Current State

| Feature | Status |
|---------|--------|
| IAL1 (email/password) | Done |
| WebAuthn (device binding) | Done |
| MFA (TOTP/WebAuthn) | Done |
| Document verification | Not implemented |
| Biometric liveness | Not implemented |
| Watchlist screening | Not implemented |

GGID handles **authentication** (IAL1 + MFA) but not **identity proofing** (IAL2/IAL3).

---

## Roadmap

GGID should integrate with external KYC providers rather than build identity proofing:

```bash
# Proposed: webhook-based KYC integration
curl -X POST https://ggid.example.com/api/v1/users/{id}/kyc \
  -d '{
    "provider": "stripe",
    "session_id": "stripe_identity_session_abc"
  }'
```

1. **Stripe Identity integration** — webhook callback updates user `kyc_verified` flag. Effort: 3 days.
2. **IAL2 claim in JWT** — add `ial: 2` to JWT after verification. Effort: 0.5 days.
3. **Policy enforcement** — ABAC policy denying high-risk actions without IAL2. Effort: 0.5 days.

Total: ~4 days. Priority: P3 (only needed for regulated industries).

---

*See: [Security Overview](../architecture/security-overview.md) | [Risk Scoring](risk-scoring-adaptive-access.md)*

*Last updated: 2025-07-11*
