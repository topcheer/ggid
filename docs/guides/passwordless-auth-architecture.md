# Passwordless Authentication Architecture

Design guide for passwordless flows in GGID: magic links, passkeys/WebAuthn, and FIDO2.

## Overview

Passwordless eliminates the password as an attack surface. GGID supports three primary passwordless methods:

| Method | Channel | Phishing-Resistant | UX |
|--------|---------|-------------------|-----|
| Magic Link | Email | No (phishable link) | Frictionless |
| Passkey (WebAuthn) | Device biometric/PIN | Yes | Best |
| FIDO2 Security Key | Hardware key | Yes | High trust |

## Magic Link Flow

```
User enters email → GGID sends one-time link → User clicks → Authenticated
```

```bash
# Request magic link
POST /api/v1/auth/passwordless/magic-link
{"email": "user@corp.com"}
# → 202 (always, even if email unknown — prevent enumeration)

# Link contains: https://auth.ggid.dev/magic?token=abc123
# Token: single-use, TTL 10 min, bound to email + IP range

# Verify magic link token
POST /api/v1/auth/passwordless/magic-link/verify
{"token": "abc123"}
# → 200 {access_token, refresh_token}
```

Security:
- Token is single-use, invalidated after verification
- TTL 10 minutes maximum
- Rate limited: 3 requests per email per hour
- IP binding: token valid only from same /24 subnet
- If user clicks expired link → re-request

## Passkey / WebAuthn Flow

### Registration

```
1. User authenticates (password or existing factor)
2. Browser prompts for biometric/PIN (authenticator)
3. New credential created, stored on device
4. Public key sent to GGID, stored with user
```

```bash
# Begin registration
POST /api/v1/auth/webauthn/register/begin
{"user_id": "uuid", "device_name": "iPhone 15"}
# → {challenge, rp, user, pubKeyCredParams, excludeCredentials}

# Complete registration
POST /api/v1/auth/webauthn/register/complete
{"id":"...","rawId":"...","response":{...},"type":"public-key"}
# → 201 {credential_id, device_name}
```

### Authentication (Conditional UI)

```javascript
// Browser autofill shows passkey option on email field
const assertion = await navigator.credentials.get({
  publicKey: {
    challenge: serverChallenge,
    allowCredentials: [], // empty = discoverable credentials
    userVerification: "required"
  }
});
// Send assertion to server for verification
```

Conditional UI means the passkey prompt appears in the browser's native autofill — no extra clicks for the user.

### Cross-Device Authentication

User on desktop can use phone's passkey via:
1. QR code scan (hybrid transport)
2. Bluetooth proximity check (anti-phishing)
3. Phone authenticates locally
4. Assertion relayed to desktop browser

## FIDO2 Security Keys

| Key Type | Examples | Transport |
|----------|----------|-----------|
| USB | YubiKey 5, SoloKeys | USB-A/C |
| NFC | YubiKey NFC | Tap on mobile |
| Bluetooth | Google Titan | Wireless |

```bash
# Register FIDO2 key (same WebAuthn API)
POST /api/v1/auth/webauthn/register/begin
{"user_id":"uuid","authenticator_attachment":"cross-platform"}
```

## Fallback Chains

```
Passkey (preferred)
  ↓ not available
Magic Link (email)
  ↓ email unavailable
Backup recovery codes
  ↓ exhausted
Admin-assisted recovery (break-glass)
```

GGID never falls back to password — the fallback is always a different passwordless channel.

## Recovery Without Password

| Scenario | Recovery Method |
|----------|----------------|
| Lost phone with passkey | Backup authenticator → new passkey |
| No backup device | Recovery codes (10 codes, generated at enrollment) |
| Lost recovery codes | Admin-assisted identity verification + break-glass |
| Email compromised | Admin revokes all passwordless factors, re-enroll |

## Device Management

```bash
# List registered passwordless devices
GET /api/v1/auth/webauthn/credentials
# → [{credential_id, device_name, created_at, last_used_at, transports}]

# Revoke a device
DELETE /api/v1/auth/webauthn/credentials/{credential_id}

# Rename device
PATCH /api/v1/auth/webauthn/credentials/{credential_id}
{"device_name": "Work Laptop"}
```

Dashboard shows:
- All registered passkeys and their last-used time
- Unused devices >90 days flagged for review
- Quick revoke button per device

## UX Best Practices

1. **Conditional UI first**: Let browser show passkey in autofill — zero extra clicks
2. **Progressive onboarding**: Offer passkey after first successful login
3. **Clear device naming**: Prompt user to name each device ("iPhone", "Work Laptop")
4. **Backup prompt**: "Add a backup passkey" after first registration
5. **Graceful fallback**: If passkey fails, show magic link — never show password field
6. **No password field ever**: Once passwordless is enabled, password field is removed entirely

## Anti-Phishing

WebAuthn/FIDO2 are inherently phishing-resistant:
- Origin bound: credential only works on the real domain
- RP ID check: browser refuses to send credential to wrong origin
- No shared secret: nothing for phisher to intercept

Magic links are NOT phishing-resistant — treat as convenience, not high-security.

## Monitoring

| Metric | Alert |
|--------|-------|
| Passkey registration failures | >10% of attempts |
| Magic link enumeration | Same IP requesting >5 different emails |
| Recovery code usage | Alert security team |
| Stale devices | Last used >180 days |
| Cross-device auth failures | >20% rate |

## See Also

- [WebAuthn Recovery](webauthn-recovery.md)
- [MFA Architecture](mfa-architecture.md)
- [Multi-Factor Step-Up](multi-factor-step-up.md)
- [Authentication Flows](authentication-flows.md)
