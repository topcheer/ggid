# MFA Strategy

Factor comparison, strength matrix, step-up triggers, MFA fatigue prevention, backup factors, recovery codes, per-risk policy, and migration strategy.

## Factor Comparison

| Factor | Security | UX | Cost | Phishing Resistant |
|--------|---------|-----|------|-------------------|
| WebAuthn (platform) | Highest | Biometric instant | None | ✅ |
| WebAuthn (cross-platform) | Highest | USB tap | $20-50 | ✅ |
| TOTP (authenticator app) | High | 6-digit code | None | ❌ |
| Email OTP | Medium | Code in email | Low | ❌ |
| SMS OTP | Low | Text message | Per-msg | ❌ |
| Push notification | Medium | Approve/deny | Per-push | ⚠️ (fatigue risk) |

## Factor Strength Matrix

| Operation | Min Factor | Additional |
|-----------|-----------|-----------|
| Read own profile | None | — |
| Write own profile | TOTP | — |
| Admin operations | WebAuthn | — |
| Delete users | WebAuthn | Admin approval |
| Break-glass | WebAuthn | Dual approval |

## Step-Up Triggers

| Trigger | Required Factor |
|---------|----------------|
| New device | TOTP |
| New country | TOTP |
| Write operation | TOTP |
| Admin operation | WebAuthn |
| Destructive operation | WebAuthn + approval |
| Impossible travel | WebAuthn |

## MFA Fatigue Prevention (Number Matching)

Push notifications are vulnerable to "MFA fatigue" — attacker spams push until user approves.

```
1. Server generates random number: 42
2. Login screen shows: "Enter 42 on your device to approve"
3. Push notification shows number pad
4. User must enter 42 (not just tap "Approve")
5. Attacker can't spam — needs the number
```

## Backup Factors

```
Primary: WebAuthn (Face ID / Touch ID)
  ↓ Unavailable
Backup 1: TOTP (always enrolled alongside)
  ↓ Unavailable
Backup 2: Recovery codes (10 single-use, hashed)
  ↓ Used up
Admin-assisted recovery (identity verification)
```

**Policy**: Every admin must have ≥2 enrolled factors.

## Recovery Codes

- 10 single-use codes generated at MFA enrollment
- Stored bcrypt-hashed (never plaintext)
- Displayed once — user must save securely
- Alert when <3 remaining → prompt re-generation

## Per-Risk MFA Policy

```yaml
mfa_policies:
  - name: "standard"
    condition: "true"
    min_factor: "totp"

  - name: "financial"
    condition: "user.department == 'Finance'"
    min_factor: "webauthn"

  - name: "admin"
    condition: "user.has_role('admin')"
    min_factor: "webauthn"
    require_backup: true

  - name: "break_glass"
    condition: "operation == 'break_glass'"
    min_factor: "webauthn"
    require_dual_approval: true
```

## Migration Strategy

```
Phase 1: SMS/Email OTP (current for some users)
  → Add TOTP enrollment campaign
Phase 2: TOTP as primary, SMS as backup
  → Promote WebAuthn enrollment
Phase 3: WebAuthn as primary, TOTP as backup
  → Deprecate SMS for new enrollments
Phase 4: WebAuthn-only for admins
  → SMS deprecated entirely
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| MFA enrollment rate | >90% | <70% → campaign |
| MFA failure rate | <5% | >10% → clock/device issue |
| SMS usage | <10% | High → migrate to TOTP/WebAuthn |
| Recovery code usage | <5%/mo | Spike → users losing devices |
| Push fatigue attempts | 0 | Any → number matching not working |

## See Also

- [Multi-Factor Auth Strategy](multi-factor-auth-strategy.md)
- [MFA Architecture](mfa-architecture.md)
- [Multi-Factor Step-Up Design](multi-factor-step-up-design.md)
- [Adaptive Authentication](adaptive-authentication.md)
- [WebAuthn Deployment Guide](webauthn-deployment-guide.md)
