# OAuth 2.1 & Continuous Verification — GGID Gap Analysis

*Research date: 2026-07-12*

## Overview

### OAuth 2.1 (draft-ietf-oauth-v2-1-15)
OAuth 2.1 consolidates OAuth 2.0 + best practices into a single spec. Key changes:
- PKCE mandatory for all authorization code flows
- Implicit flow removed
- Token exchange (RFC 8693) referenced as best practice
- DPoP/mTLS recommended as sender-constrained tokens
- Refresh token rotation required

### Continuous Verification (Zero Trust Evolution)
NIST SP 800-207-22 (2025 update) emphasizes:
- Continuous authentication: re-verify identity throughout session
- Risk-adaptive access: step-up based on context changes
- Device posture attestation: verify device trust continuously
- Session federation: propagate auth state across services

## GGID Current State

### OAuth 2.1 Compliance: MOSTLY COMPLIANT
- [x] PKCE support (authorization code flow)
- [x] Token exchange (RFC 8693)
- [x] DPoP verification
- [x] Refresh token rotation
- [x] Client credentials grant
- [x] Device authorization (RFC 8628)
- [x] Dynamic client registration (RFC 7591/7592)
- [x] PAR support (RFC 9126)
- [ ] OAuth 2.1 compliance test suite
- [ ] Implicit flow explicitly disabled/rejected
- [ ] PKCE mandatory enforcement (currently optional per client)

### Continuous Verification: MISSING
- [ ] Continuous session validation (periodic re-check)
- [ ] Risk-score-based step-up triggers
- [ ] Device posture re-evaluation mid-session
- [ ] Session federation token (CIBA-style continuous)
- [ ] Geo-velocity anomaly detection (impossible travel)

## Gap Analysis

### P1: OAuth 2.1 Enforcement (Backend)
- Make PKCE mandatory for all new OAuth clients
- Explicitly reject `response_type=token` (implicit flow)
- Add compliance test suite verifying spec requirements
- Document OAuth 2.1 conformance in API docs

### P1: Continuous Session Validation (Backend)
- Background goroutine validating active sessions every N minutes
- Re-evaluate: user status, risk score, device trust, IP reputation
- Auto-step-up or terminate session on risk change
- `GET /api/v1/auth/sessions/validate` endpoint

### P2: Geo-Velocity Detection (Backend)
- Track successful auth IPs + timestamps per user
- Calculate travel time between consecutive auth events
- Flag impossible travel (e.g., NYC → Tokyo in 1 hour)
- Feed into risk engine + conditional access policies

### P2: Device Posture API (Backend)
- `POST /api/v1/devices/{id}/posture` — client reports device posture
- Fields: OS version, disk encryption, screen lock, antivirus, jailbreak status
- Feed into conditional access policy evaluation
- Console: device posture dashboard

## Backlog Items Generated
- [ ] **P1** Backend: OAuth 2.1 enforcement (mandatory PKCE, reject implicit)
- [ ] **P1** Backend: Continuous session validation goroutine
- [ ] **P2** Backend: Geo-velocity anomaly detection
- [ ] **P2** Backend: Device posture API + conditional access integration
- [ ] **P2** Frontend: Device posture dashboard (console/src/)
- [ ] **P2** Docs: OAuth 2.1 compliance statement (docs/)
