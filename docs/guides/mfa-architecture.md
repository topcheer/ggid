# MFA Architecture Guide

Comprehensive MFA architecture — factor types, priority, step-up flows, backup factors, recovery, risk-level matrix, anti-phishing.

## Factor Types

| Factor | Security | UX | GGID Support |
|--------|----------|-----|-------------|
| TOTP (authenticator app) | High | Medium | Done (RFC 6238) |
| WebAuthn platform (biometric) | Very High | High | Done |
| WebAuthn cross-platform (key) | Very High | Medium | Done |
| SMS OTP | Low | High | Via provider |
| Email OTP | Medium | Medium | Via provider |
| Push notification | High | High | Planned |
| Hardware token (FIDO2) | Very High | Low | Done |

## Factor Priority

```
1st: WebAuthn (passwordless or step-up)
2nd: TOTP (authenticator app)
3rd: Email OTP (fallback)
4th: SMS (last resort, being deprecated)
```

## Step-Up Flow

```
Low-risk action (read profile) → Password only
Medium-risk action (change settings) → Password + TOTP
High-risk action (admin operations) → Password + WebAuthn
Critical action (key rotation) → Password + WebAuthn + approval
```

### API

```bash
# Step-up: trigger TOTP
POST /api/v1/auth/mfa/verify
{"mfa_token":"temp_token","code":"123456"}

# Step-up: trigger WebAuthn
POST /api/v1/webauthn/auth/begin
POST /api/v1/webauthn/auth/finish
```

## Risk-Level Matrix

| Risk Level | Actions | Required Factor |
|-----------|---------|-----------------|
| Low | Read own profile, view dashboard | Password |
| Medium | Update email, change settings | Password + TOTP |
| High | Admin operations, user management | Password + WebAuthn |
| Critical | Key rotation, data export, delete | Password + WebAuthn + approval |

## Backup Factors

Users should enroll 2+ factors:
- Primary: WebAuthn (phone biometric)
- Backup: TOTP (authenticator app)
- Fallback: Recovery codes

## Anti-Phishing

WebAuthn is phishing-resistant because:
- Origin-bound (authenticator only responds to correct RP ID)
- User verification (biometric/PIN)
- No shared secret transmitted

For non-WebAuthn factors:
- Number-matching push notifications
- TOTP with short window (30s ± 1)
- Rate limit MFA attempts

## See Also

- [Password Policy Guide](password-policy-guide.md)
- [WebAuthn Deploy](webauthn-deploy.md)
- [Authentication Flows](authentication-flows.md)
- [Adaptive Authentication](../research/adaptive-authentication.md)
