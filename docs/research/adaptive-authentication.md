# Adaptive Authentication: Research and GGID Roadmap

## Overview

Adaptive authentication (also called context-aware or risk-based authentication) dynamically adjusts authentication requirements based on contextual signals — device, location, time, behavior, and threat intelligence. Instead of a one-size-fits-all approach, the system evaluates risk in real-time and may require step-up authentication for high-risk sessions.

## Core Concepts

### 1. Context Signals

Modern adaptive auth systems collect and evaluate the following signals:

| Signal Category  | Examples                                   | Weight (Typical) |
|------------------|--------------------------------------------|------------------|
| Device           | Known device, new device, device fingerprint| High             |
| Location         | Geo-IP, impossible travel, new country      | High             |
| Network          | VPN/Tor detection, ASN reputation           | Medium           |
| Time             | Off-hours login, first-time time window      | Medium           |
| Behavioral       | Keystroke dynamics, mouse patterns           | Low (emerging)   |
| Threat Intel     | Known bad IPs, credential breach databases   | High             |
| Session          | Concurrent sessions, session age             | Low              |

### 2. Risk Scoring Models

**Rule-Based Scoring** (deterministic):
```
risk_score = 0
if new_device:       risk_score += 30
if new_country:      risk_score += 40
if impossible_travel: risk_score += 50
if off_hours:        risk_score += 10
if vpn_detected:     risk_score += 15

decision:
  score < 20  → ALLOW (no challenge)
  20-50       → STEP-UP (require MFA)
  > 50        → DENY (block + alert)
```

**ML-Based Scoring** (adaptive):
- Trains on historical login patterns per user
- Detects anomalies using isolation forests, autoencoders
- Adapts thresholds over time
- Higher accuracy but requires training data and explainability tooling

### 3. Step-Up Authentication Challenges

| Challenge Type        | UX Friction | Security Level | Use Case                        |
|-----------------------|-------------|----------------|---------------------------------|
| Email OTP             | Low         | Medium         | Low-risk step-up                |
| SMS OTP               | Low         | Medium         | Legacy fallback                 |
| TOTP (authenticator)  | Medium      | High           | Standard MFA                    |
| Push notification     | Low         | High           | Mobile-first users              |
| WebAuthn / Passkey    | Low         | Very High      | Passwordless step-up            |
| Hardware key (FIDO2)  | Medium      | Very High      | High-assurance, regulated       |

## Industry Landscape (2025-2026)

### Okta Adaptive MFA

- **Risk engine**: Combines device, IP, location, and behavioral signals
- **Step-up**: Push, OTP, WebAuthn, SMS (configurable per policy)
- **Zone-based**: IP allowlists/denylists by geographic zone
- **Behavioral**: Keystroke dynamics (optional, enterprise tier)

### Auth0 Actions / Rules

- **Custom risk logic**: JavaScript functions executed during authentication
- **Signal sources**: IP reputation, breached password detection, anomaly detection
- **Bot detection**: Cloudflare integration, CAPTCHA challenge
- **Limitation**: No built-in ML risk model — rule-based only

### Microsoft Entra ID

- **Conditional Access**: Policy engine evaluating user, device, location, app, risk
- **Identity Protection**: ML-based risk scoring (user risk, sign-in risk)
- **Session controls**: Limited session, app-enforced restrictions
- **Continuous Access Evaluation (CAE)**: Real-time policy re-evaluation

### Ping Identity

- **PingOne Neo**: Risk engine with device intelligence
- **Policy engine**: Attribute-based access control (ABAC) with risk as attribute
- **Integration**: WebAuthn, FIDO2, biometric step-up

## GGID Current Capabilities

### Already Implemented

| Capability                    | Status   | Location                              |
|-------------------------------|----------|---------------------------------------|
| MFA TOTP                      | Done     | services/auth (TOTP RFC 6238)         |
| WebAuthn / FIDO2              | Done     | services/auth (7 attestation formats) |
| Login rate limiting           | Done     | services/gateway (token bucket)       |
| Account lockout               | Done     | services/auth (N failures → lockout)  |
| Password history              | Done     | services/auth (CheckHistory)          |
| JWT jti anti-replay           | Done     | services/auth (Redis SETNX)           |
| Tenant isolation (RLS)        | Done     | PostgreSQL FORCE ROW LEVEL SECURITY   |
| Audit hash chain              | Done     | services/audit (hash_chain.go)        |
| Session timeout               | Done     | services/auth (CheckSessionTimeout)   |
| Geo-IP middleware             | Partial  | services/gateway (geoip.go)           |
| IP allowlist                  | Done     | services/gateway (ipallowlist.go)     |
| Circuit breaker               | Done     | services/gateway (circuitbreaker.go)  |

### Gaps for Full Adaptive Auth

| Gap                            | Priority | Effort   |
|--------------------------------|----------|----------|
| Risk scoring engine            | P1       | Large    |
| Device fingerprinting          | P1       | Medium   |
| Impossible travel detection    | P1       | Small    |
| Step-up challenge orchestration| P1       | Medium   |
| Policy-based step-up rules     | P1       | Medium   |
| Behavioral biometrics          | P2       | Large    |
| ML risk model                  | P2       | Large    |
| Push notification MFA          | P2       | Medium   |
| Breached password check        | P2       | Small    |

## Proposed GGID Adaptive Auth Architecture

### Phase 1: Risk Scoring Engine

```
Login Request
     ↓
┌─────────────────────────────┐
│    Context Collector         │
│  - IP → Geo (existing geoip) │
│  - Device fingerprint (new)  │
│  - User agent / headers      │
│  - Time of day               │
│  - Known device? (DB lookup) │
│  - Previous login location   │
└──────────┬──────────────────┘
           ↓
┌─────────────────────────────┐
│    Risk Scorer               │
│  - Rule-based weights        │
│  - Produces 0-100 score      │
│  - Returns decision:         │
│    ALLOW / STEP_UP / DENY    │
└──────────┬──────────────────┘
           ↓
┌─────────────────────────────┐
│    Decision Executor         │
│  ALLOW   → Issue JWT         │
│  STEP_UP → Trigger MFA flow  │
│  DENY    → 403 + audit event │
└─────────────────────────────┘
```

### Phase 2: Device Registry

```sql
CREATE TABLE device_fingerprints (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    tenant_id UUID NOT NULL,
    fingerprint_hash VARCHAR(64) NOT NULL,  -- SHA-256 of device signals
    user_agent TEXT,
    first_seen TIMESTAMPTZ DEFAULT NOW(),
    last_seen TIMESTAMPTZ DEFAULT NOW(),
    trusted BOOLEAN DEFAULT FALSE,
    UNIQUE(tenant_id, user_id, fingerprint_hash)
);
```

### Phase 3: Step-Up Orchestration

```go
func (s *AuthService) EvaluateRisk(ctx context.Context, loginReq *LoginRequest) RiskDecision {
    score := 0

    // Device check
    if !s.isKnownDevice(loginReq.UserID, loginReq.DeviceFingerprint) {
        score += 30
    }

    // Geo check
    geo := s.geoIP.Lookup(loginReq.ClientIP)
    if s.isNewCountry(loginReq.UserID, geo.Country) {
        score += 40
    }

    // Impossible travel
    if s.detectImpossibleTravel(loginReq.UserID, geo) {
        score += 50
    }

    // Time check
    if s.isOffHours(loginReq.UserID, time.Now()) {
        score += 10
    }

    // VPN/Tor
    if s.isVPN(loginReq.ClientIP) {
        score += 15
    }

    return RiskDecision{
        Score:    score,
        Action:   scoreToAction(score),
        Reasons:  collectReasons(),
    }
}
```

### Phase 4: Policy-Based Step-Up Rules

Administrators define policies via the Console:

```yaml
adaptive_policies:
  - name: "High-risk operations"
    resource: "admin:*"
    conditions:
      risk_score_gt: 20
    step_up: "webauthn"

  - name: "New device"
    conditions:
      new_device: true
    step_up: "totp"

  - name: "New country"
    conditions:
      new_country: true
    step_up: "email_otp"

  - name: "Impossible travel"
    conditions:
      impossible_travel: true
    action: "deny"
    alert: true
```

## Competitive Comparison

| Feature                  | Okta  | Auth0 | Entra ID | GGID (Proposed) |
|--------------------------|-------|-------|----------|-----------------|
| Risk scoring             | ML    | Rule  | ML       | Rule (Phase 1)  |
| Device fingerprinting    | Yes   | Yes   | Yes      | Phase 1          |
| Impossible travel        | Yes   | Add-on| Yes      | Phase 1          |
| Step-up: TOTP            | Yes   | Yes   | Yes      | Already done     |
| Step-up: WebAuthn        | Yes   | Yes   | Yes      | Already done     |
| Step-up: Push            | Yes   | Add-on| Yes      | Phase 2          |
| Behavioral biometrics    | Ent   | No    | No       | Phase 3          |
| Breached password check  | Yes   | Yes   | Yes      | Phase 2          |
| Zone-based policies      | Yes   | No    | Yes      | Via IP allowlist |
| Continuous re-evaluation | Yes   | No    | Yes (CAE)| Phase 3          |

## References

- [NIST SP 800-63B: Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [FIDO Alliance White Papers](https://fidoalliance.org/whitepapers/)

## See Also

- [MFA Setup Guide](mfa-setup.md)
- [Passwordless Setup](passwordless-setup.md)
- [Risk Scoring & Adaptive Access](risk-scoring-adaptive-access.md)
- [Security Overview](../architecture/security-overview.md)
