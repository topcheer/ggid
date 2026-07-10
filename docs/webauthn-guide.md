# WebAuthn / FIDO2 Admin Guide

Practical guide for administrators managing WebAuthn (passkey)
authentication: registration and authentication ceremonies, credential
management, recovery procedures, and platform vs roaming authenticators.

> **See also**: [WebAuthn Implementation Guide](webauthn-implementation-guide.md)
> for protocol-level details, attestation formats, and COSE algorithms.

---

## Table of Contents

- [Authenticator Types](#authenticator-types)
- [Registration Ceremony](#registration-ceremony)
- [Authentication Ceremony](#authentication-ceremony)
- [Credential Management](#credential-management)
- [Recovery Procedures](#recovery-procedures)
- [Admin Operations](#admin-operations)

---

## Authenticator Types

| Type | Examples | Transport | Biometric | Backed Up |
|------|----------|-----------|-----------|:---------:|
| Platform | Touch ID, Face ID, Windows Hello | internal | Yes | No |
| Roaming | YubiKey, SoloKey, Titan | USB/NFC/BLE | Some | Yes |
| Hybrid | Phone as passkey (cross-device) | internal+hybrid | Yes | Yes |

### Platform vs Roaming Decision

| Factor | Platform | Roaming |
|--------|:--------:|:------:|
| Convenience | High (biometric) | Medium (carry key)
| Portability | Single device | Any device with USB |
| Phishing resistance | Strong | Strong |
| Backup | No (per-device) | Yes (multi-device) |
| Recovery | Re-register on new device | Use backup key |

**Recommendation**: Register at least 2 authenticators (1 platform + 1
roaming) per user for redundancy.

---

## Registration Ceremony

### Flow

```
User                  Client (Browser)           GGID Auth Server
  │ 1. Start registration       │                       │
  ├────────────────────────────►│                       │
  │                              │ 2. POST /webauthn/    │
  │                              │    register/begin      │
  │                              ├──────────────────────►│
  │                              │ 3. challenge,         │
  │                              │    rp.id, user info   │
  │                              │◄──────────────────────┤
  │                              │                       │
  │ 4. Browser prompts           │                       │
  │    biometric/PIN             │                       │
  ├────────────────────────────►│                       │
  │                              │ 5. POST /webauthn/    │
  │                              │    register/complete   │
  │                              │    (attestation)      │
  │                              ├──────────────────────►│
  │                              │                       │
  │                              │ 6. Verify attestation │
  │                              │    Store credential   │
  │                              │                       │
  │                              │ 7. Success            │
  │                              │◄──────────────────────┤
  │ 8. Done                      │                       │
  │◄────────────────────────────┤                       │
```

### Admin-Initiated Registration

```bash
curl -X POST https://iam.example.com/api/v1/admin/users/{user_id}/webauthn/register \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "authenticator_attachment": "platform",
    "user_verification": "required",
    "attestation": "none"
  }'
```

Returns a challenge that the user must complete in their browser.

### Registration Options

| Option | Values | Effect |
|--------|--------|--------|
| `authenticatorAttachment` | `platform`, `cross-platform`, null | Restrict authenticator type |
| `userVerification` | `required`, `preferred`, `discouraged` | Require biometric/PIN |
| `attestation` | `none`, `indirect`, `direct` | Attestation depth |
| `excludeCredentials` | `[{id, type}]` | Prevent duplicate registrations |

---

## Authentication Ceremony

### Flow

```
User                  Client (Browser)           GGID Auth Server
  │ 1. Login with passkey        │                       │
  ├────────────────────────────►│                       │
  │                              │ 2. POST /webauthn/    │
  │                              │    auth/begin          │
  │                              ├──────────────────────►│
  │                              │ 3. challenge,         │
  │                              │    credentialIds      │
  │                              │◄──────────────────────┤
  │                              │                       │
  │ 4. Browser prompts           │                       │
  │    biometric/PIN             │                       │
  ├────────────────────────────►│                       │
  │                              │ 5. POST /webauthn/    │
  │                              │    auth/complete       │
  │                              │    (assertion)        │
  │                              ├──────────────────────►│
  │                              │                       │
  │                              │ 6. Verify signature   │
  │                              │    Check sign_count   │
  │                              │    Issue JWT           │
  │                              │                       │
  │                              │ 7. access_token       │
  │                              │◄──────────────────────┤
  │ 8. Logged in                 │                       │
  │◄────────────────────────────┤                       │
```

### Clone Detection

GGID tracks `signCount` per credential. If the counter decreases between
authentications, the credential may have been cloned:
```
Authentication 1: signCount = 5  → OK
Authentication 2: signCount = 7  → OK (increased)
Authentication 3: signCount = 6  → ALERT (decreased — possible clone)
```

On clone detection, GGID:
1. Logs a security alert
2. Notifies the admin via webhook
3. Optionally requires step-up authentication

---

## Credential Management

### List User Credentials

```bash
curl https://iam.example.com/api/v1/admin/users/{user_id}/webauthn/credentials \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

```json
{
  "credentials": [
    {
      "id": "cred-uuid",
      "name": "MacBook Pro Touch ID",
      "device_type": "platform",
      "attestation_format": "none",
      "transports": ["internal"],
      "sign_count": 42,
      "created_at": "2024-01-15T10:00:00Z",
      "last_used_at": "2024-01-20T09:00:00Z"
    }
  ]
}
```

### Rename Credential

```bash
curl -X PATCH https://iam.example.com/api/v1/admin/users/{user_id}/webauthn/credentials/{cred_id} \
  -d '{ "name": "Work Laptop Touch ID" }'
```

### Remove Credential

```bash
curl -X DELETE https://iam.example.com/api/v1/admin/users/{user_id}/webauthn/credentials/{cred_id} \
  -d '{ "reason": "device_lost" }'
```

> Removing the last credential disables WebAuthn for that user. Ensure at
> least one MFA factor remains.

---

## Recovery Procedures

### Lost Device

1. User loses the device with their passkey
2. Admin removes the credential:
   ```bash
   curl -X DELETE .../webauthn/credentials/{cred_id}
   ```
3. User re-registers a new authenticator
4. If all credentials lost: admin resets MFA, user re-enrolls

### Recovery Codes

During WebAuthn registration, GGID generates one-time recovery codes:

```bash
curl -X POST .../users/{user_id}/webauthn/recovery-codes \
  -d '{ "count": 10 }'
```

```json
{
  "recovery_codes": [
    "ABCDE-FGHIJ",
    "KLMNO-PQRST",
    ...
  ]
}
```

> Recovery codes are hashed at rest. Displayed once during generation.
> Each code can be used exactly once.

### Account Recovery Flow

```
1. User reports lost passkey + no recovery codes
2. Admin verifies user identity (out-of-band)
3. Admin resets WebAuthn credentials:
   DELETE /users/{id}/webauthn/mfa
4. Admin generates temporary recovery code
5. User logs in with password + temp code
6. User registers new passkey
7. Admin revokes temp code
```

---

## Admin Operations

### Enforce WebAuthn Tenant-Wide

```bash
curl -X PATCH .../admin/tenant/settings/mfa-policy \
  -d '{
    "required": true,
    "allowed_methods": ["webauthn"],
    "enrollment_grace_period_days": 14,
    "excluded_roles": ["service-account"]
  }'
```

### RP ID Configuration

```yaml
webauthn:
  rp_id: "iam.example.com"           # Must match origin domain
  rp_name: "GGID IAM"                  # Display name
  rp_origins:
    - "https://iam.example.com"       # Exact origin
    - "https://console.example.com"   # Admin console
  timeout: 60000                       # 60 seconds
  user_verification: "required"       # Biometric/PIN mandatory
```

### Monitoring

| Metric | Alert Threshold |
|--------|----------------|
| Registration failures > 10% | Warning |
| Authentication failures > 5% | Warning |
| Clone detection (any) | Critical |
| Users without WebAuthn (after grace) | Warning |

### Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| `NotAllowedError` | User cancelled biometric | Retry |
| `SecurityError` | Origin doesn't match RP ID | Fix rp_origins config |
| `InvalidStateError` | Authenticator already registered | Exclude existing creds |
| `AbortError` | Timeout (60s) | Increase timeout |
| `NotSupportedError` | No authenticator available | Connect a security key |
