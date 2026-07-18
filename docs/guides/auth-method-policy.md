# Authentication Method Policy (KB-073)

## Overview

The Auth Method Policy engine controls which authentication methods are available per user, per group, per tenant, and per application. It enables progressive security posture — from passwordless-first to high-assurance enterprise SSO.

## Policy Dimensions

| Dimension | Example | Scope |
|-----------|---------|-------|
| User-level | `user:alice` → passkey only | Individual user |
| Group-level | `group:engineering` → password + TOTP | Department/team |
| Tenant-level | `tenant:enterprise` → require SSO | All users in tenant |
| App-level | `app:finance` → require MFA | Per protected application |

## Available Auth Methods

| Method | ID | Strength | Description |
|--------|----|----------|-------------|
| Password | `password` | Low | Username/password |
| TOTP | `totp` | Medium | Time-based OTP (Google Auth) |
| SMS OTP | `sms_otp` | Medium | SMS one-time code |
| WebAuthn/Passkey | `webauthn` | High | FIDO2 hardware/platform |
| Email Link | `email_link` | Medium | Magic link |
| SAML SSO | `saml` | High | Enterprise SSO |
| OIDC | `oidc` | High | OAuth/OIDC provider |
| Social (Google) | `social_google` | Medium | Google OAuth |
| Biometric | `biometric` | High | Platform biometrics |

## Policy Configuration

### Tenant-Level Policy
```yaml
auth_method_policy:
  enabled: true
  default_methods:
    - password
    - webauthn
    - totp
  required_mfa:
    - totp
    - webauthn
  forbidden_methods:
    - sms_otp
  max_session_duration: 8h
```

### Per-User Override
```http
PUT /api/v1/users/{id}/auth-policy
Content-Type: application/json

{
  "allowed_methods": ["webauthn", "totp"],
  "primary_method": "webauthn",
  "fallback_method": "totp",
  "require_step_up": true
}
```

## Decision Flow

```
1. User initiates login
2. Policy engine loads: tenant → group → user → app policies (merged)
3. Determine available methods (intersection of allowed)
4. Present available methods to user (conditional UI)
5. User authenticates
6. Risk engine evaluates (if enabled)
7. If risk > threshold → step-up required (even if auth succeeded)
8. Issue token with auth_strength claim
```

## Conditional UI

The policy engine returns available methods to the frontend for conditional rendering:

```http
GET /api/v1/auth/methods?user_id=alice&app=finance
```

```json
{
  "primary": "webauthn",
  "available": ["webauthn", "totp"],
  "fallback": "totp",
  "require_step_up": false,
  "risk_score": 12
}
```

## Step-Up Authentication

When `require_step_up` is true or risk exceeds threshold:

1. Initial auth succeeds (e.g., password)
2. Token issued with `auth_strength: 1` (low)
3. Accessing protected resource triggers 403 with `step_up_required: true`
4. User completes additional factor (e.g., passkey)
5. Token upgraded with `auth_strength: 2` (high)
6. Resource access granted

## Enforcement Points

| Layer | Enforcement |
|-------|-------------|
| Gateway | Rate limits, IP reputation, method availability |
| Auth service | Password/MFA validation, session creation |
| Policy engine | ABAC evaluation, method filtering |
| Application | Token strength checks |

## Best Practices

- **Passkey-first**: Set `webauthn` as primary where possible
- **Eliminate SMS**: Use `totp` instead of `sms_otp` (SIM swap attacks)
- **Progressive enforcement**: Start with optional MFA, then require for sensitive ops
- **Monitor**: Track method distribution and step-up trigger rates
