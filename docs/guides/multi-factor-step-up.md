# Multi-Factor Step-Up Authentication Guide

## Trigger Conditions

| Risk Level | Trigger | Required Factor |
|-----------|---------|-----------------|
| Low | Read profile, dashboard | Password only |
| Medium | Change settings, update email | Password + TOTP |
| High | Admin operations, user management | Password + WebAuthn |
| Critical | Key rotation, data export, delete | Password + WebAuthn + approval |

## Adaptive Flow Design

```
Request → Gateway evaluates risk score
  ↓
  Score < 20 → Proceed (no step-up)
  Score 20-50 → Require TOTP
  Score > 50 → Require WebAuthn
  Score > 80 → Deny + alert
```

## Session Elevation

```bash
# Low-risk session has scopes: [users:read]
# User clicks "Delete User" (requires users:delete)
# Gateway detects scope mismatch → returns 403 with step-up challenge

# Step 1: Verify TOTP
POST /api/v1/auth/mfa/verify
{"mfa_token":"temp","code":"123456"}

# Step 2: Elevated JWT issued with expanded scopes
```

## UX Patterns

- **Inline challenge**: Show MFA input without page reload
- **Progressive disclosure**: Only ask for step-up when needed
- **Remember device**: Skip step-up for trusted device (30 days)
- **Timeout**: Step-up session expires after 10 minutes

## Fallback Chain

```
WebAuthn (preferred) → TOTP → Email OTP → SMS (last resort)
```

If primary factor unavailable, GGID automatically offers next in chain.

## API

```bash
# Check if step-up needed
GET /api/v1/auth/step-up/check?action=users:delete
# → {"required": true, "factor": "webauthn"}

# Complete step-up
POST /api/v1/auth/step-up/complete
{"factor":"totp","code":"123456"}
# → {"elevated_token":"eyJ...","expires_in":600}
```

## See Also

- [MFA Architecture](mfa-architecture.md)
- [Adaptive Authentication](../research/adaptive-authentication.md)
- [Authentication Flows](authentication-flows.md)
