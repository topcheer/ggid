# Multi-Factor Authentication Strategy

Factor types, risk-based step-up, enrollment, recovery codes, factor precedence, backup policy, service account bypass, and MFA fatigue mitigation.

## Factor Types

| Factor | Security | UX | Cost |
|--------|---------|-----|------|
| WebAuthn (platform) | Highest | Biometric (instant) | None |
| WebAuthn (cross-platform) | Highest | USB tap | Key ($20-50) |
| TOTP (authenticator app) | High | 6-digit code | None |
| Email OTP | Medium | Code in email | Low |
| SMS OTP | Low | Text message | Per-message |
| Push notification | Medium | Approve/deny | Per-push |

## Factor Precedence

```
WebAuthn (preferred) → TOTP → Email OTP → SMS (last resort)
```

## Risk-Based Step-Up

| Risk Level | Required Factor | Trigger |
|-----------|----------------|---------|
| Minimal (0-19) | None | Normal read |
| Low (20-39) | TOTP | Write ops |
| High (60-79) | WebAuthn | Admin/destructive |
| Critical (80+) | WebAuthn + approval | Break-glass |

## Enrollment Flow

```bash
# 1. User enrolls TOTP
POST /api/v1/auth/mfa/totp/enroll
# → {secret: "...", qr_url: "otpauth://..."}

# 2. User verifies with first code
POST /api/v1/auth/mfa/totp/verify
{"code": "123456"}
# → {verified: true, recovery_codes: [10 codes]}

# 3. User enrolls WebAuthn
POST /api/v1/auth/webauthn/register/begin
# → challenge
POST /api/v1/auth/webauthn/register/complete
# → {device_name: "iPhone", verified: true}
```

## Recovery Codes

- 10 single-use codes generated at enrollment
- Stored hashed (bcrypt) — never plaintext
- Displayed once, user must save securely
- Track usage count, alert when <3 remaining

## Service Account Bypass

```yaml
mfa_policy:
  service_accounts:
    bypass_mfa: true
    condition: "client_credentials grant only"
    require_mtls: true  # Compensating control
    audit: "always"
```

## MFA Fatigue Mitigation (Number Matching)

TOTP doesn't have fatigue risk, but push notifications do. GGID uses **number matching**:

```bash
# 1. Server generates random number
# 2. Shows number on login screen: "Approve with number: 42"
# 3. Push notification shows number pad
# 4. User must enter 42 to approve
# → Attacker can't just spam "Approve"
```

## Backup Factor Policy

```
Primary: WebAuthn
  ↓ Unavailable
Backup 1: TOTP (always enrolled alongside WebAuthn)
  ↓ Unavailable
Backup 2: Recovery codes (last resort)
  ↓ Used
Admin-assisted recovery (identity verification required)
```

**Minimum**: Every admin must have ≥2 enrolled factors.

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| MFA enrollment rate | >90% of users | <70% → enrollment campaign |
| MFA failure rate | <5% | >10% → clock sync or device issue |
| Recovery code usage | <5%/month | Spike → users losing devices |
| SMS usage | <10% of MFA | High → migrate to TOTP/WebAuthn |

## See Also

- [MFA Architecture](mfa-architecture.md)
- [Multi-Factor Step-Up Design](multi-factor-step-up-design.md)
- [Adaptive Authentication](adaptive-authentication.md)
- [Passkey Recovery Strategy](passkey-recovery-strategy.md)
