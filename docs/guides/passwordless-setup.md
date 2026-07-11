# Passwordless Authentication Setup

> Configure WebAuthn, passkeys, and biometric authentication for passwordless login.

---

## Overview

GGID supports multiple passwordless methods:

| Method | Experience | Device |
|--------|-----------|--------|
| WebAuthn (platform) | Touch ID / Face ID / Windows Hello | Built-in |
| WebAuthn (roaming) | YubiKey / Titan / FIDO2 security key | USB/NFC |
| Hybrid | Phone unlock via QR code | Cross-device |
| Magic Link | Email one-time login link | Any |

---

## Enable Passwordless

### Step 1: Configure Relying Party

```bash
export WEBAUTHN_RP_ID=auth.example.com
export WEBAUTHN_RP_NAME="My App"
export WEBAUTHN_RP_ORIGINS=https://auth.example.com
```

### Step 2: User Registers Passkey

```bash
# Begin
curl -X POST .../api/v1/auth/webauthn/register/begin \
  -H "Authorization: Bearer $JWT"

# User touches sensor → browser creates credential

# Finish
curl -X POST .../api/v1/auth/webauthn/register/finish \
  -H "Authorization: Bearer $JWT" \
  -d '{"credential": {...}, "label": "MacBook Touch ID"}'
```

### Step 3: Login Without Password

```bash
# Begin
curl -X POST .../api/v1/auth/webauthn/login/begin \
  -d '{"username":"alice"}'

# User touches sensor

# Finish
curl -X POST .../api/v1/auth/webauthn/login/finish \
  -d '{"credential": {...}}'
# → {"access_token":"eyJ..."}
```

---

## Frontend JavaScript

```javascript
// Registration
const options = await fetch('/api/v1/auth/webauthn/register/begin', {
  method: 'POST', headers: { Authorization: `Bearer ${jwt}` }
}).then(r => r.json());

const credential = await navigator.credentials.create({ publicKey: options });
await fetch('/api/v1/auth/webauthn/register/finish', {
  method: 'POST',
  headers: { Authorization: `Bearer ${jwt}`, 'Content-Type': 'application/json' },
  body: JSON.stringify({ credential, label: 'MacBook' })
});

// Login (no password!)
const assertion = await navigator.credentials.get({
  publicKey: await fetch('/api/v1/auth/webauthn/login/begin', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'alice' })
  }).then(r => r.json())
});
const { access_token } = await fetch('/api/v1/auth/webauthn/login/finish', {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ credential: assertion })
}).then(r => r.json());
```

---

## Passkey-First Policy

Force all users to register a passkey on next login:

```bash
curl -X POST .../api/v1/policies \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -d '{
    "name": "Require passkey",
    "effect": "deny",
    "actions": ["*"],
    "resources": ["*"],
    "priority": 500
  }'
```

---

*See: [WebAuthn Setup](webauthn-setup.md) | [MFA Enforcement](mfa-enforcement.md) | [Security Hardening](security-hardening.md)*

*Last updated: 2025-07-11*
