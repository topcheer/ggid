# Multi-Factor Authentication (MFA) Guide

Complete guide to MFA in GGID: TOTP, WebAuthn/Passkey, Email OTP, Magic Link,
and forced MFA policies.

---

## Overview

GGID supports multiple MFA methods:

| Method | Type | Setup | Use Case |
|--------|------|-------|----------|
| TOTP | App-based | QR code scan | Default, universal |
| WebAuthn/Passkey | Device-based | Browser API | Phishing-resistant |
| Email OTP | Email | Auto (uses account email) | Backup factor |
| Magic Link | Email | Auto | Passwordless login |

---

## TOTP (RFC 6238)

### Setup Flow

```
User                GGID Auth               Authenticator App
  │                     │                        │
  │ POST /auth/mfa/setup│                        │
  │  {method: "totp"}   │                        │
  ├────────────────────►│                        │
  │                     │ Generate secret key    │
  │                     │ Create QR code URI     │
  │   {secret, qr_code} │                        │
  │◄────────────────────┤                        │
  │                     │                        │
  │ Scan QR code ───────────────────────────────►│
  │                     │                   App stores secret │
  │                     │                        │
  │ POST /auth/mfa/verify                       │
  │  {code: "123456"}   │                        │
  ├────────────────────►│                        │
  │                     │ Verify TOTP code       │
  │                     │ Store secret permanently│
  │   200 OK (MFA enabled)                      │
  │◄────────────────────┤                        │
```

### API Calls

```bash
# Step 1: Initiate TOTP setup
POST /api/v1/auth/mfa/setup
{"method": "totp"}
# Response: {"secret": "JBSWY3DPEHPK3PXP", "qr_code": "otpauth://..."}

# Step 2: Verify with code from app
POST /api/v1/auth/mfa/verify
{"method": "totp", "code": "123456"}
# Response: 200 OK
```

### QR Code URI Format

```
otpauth://totp/GGID:john@example.com?secret=JBSWY3DPEHPK3PXP&issuer=GGID&algorithm=SHA1&digits=6&period=30
```

### Supported Apps

- Google Authenticator
- Microsoft Authenticator
- Authy
- 1Password
- Bitwarden
- Any RFC 6238-compatible app

### Backup Codes

After enabling TOTP, generate one-time backup codes:

```bash
POST /api/v1/auth/mfa/backup-codes
# Response: ["12345678", "23456789", ...]  (10 codes, single-use)
```

---

## WebAuthn / Passkey

### Registration

```bash
# Step 1: Begin registration
POST /api/v1/auth/webauthn/register/begin
{"username": "john", "device_name": "YubiKey 5C"}
# Response: PublicKeyCredentialCreationOptions (JSON)

# Step 2: Browser creates credential
# navigator.credentials.create({ publicKey: ... })

# Step 3: Finish registration
POST /api/v1/auth/webauthn/register/finish
# Body: AuthenticatorAttestationResponse from browser
# Response: 200 OK (credential stored)
```

### Login

```bash
# Step 1: Begin assertion
POST /api/v1/auth/webauthn/login/begin
{"username": "john"}
# Response: PublicKeyCredentialRequestOptions

# Step 2: Browser gets assertion
# navigator.credentials.get({ publicKey: ... })

# Step 3: Finish login
POST /api/v1/auth/webauthn/login/finish
# Body: AuthenticatorAssertionResponse
# Response: {access_token, refresh_token}
```

### Supported Authenticators

| Type | Examples |
|------|----------|
| Security Key | YubiKey 5, Google Titan, Feitian ePass |
| Platform | Touch ID (macOS), Face ID (iOS), Windows Hello |
| Passkey | Apple iCloud Keychain, Google Password Manager |

### Attestation

GGID verifies device attestation during registration. For enterprise deployments,
you can restrict to specific attestation roots (e.g., only FIDO-certified keys).

---

## Email OTP

### Configuration

Email OTP uses the SMTP settings from the Auth service:

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=noreply@example.com
SMTP_PASSWORD=app-password
SMTP_FROM_EMAIL=noreply@example.com
```

### Trigger

When a user with email OTP enabled logs in:

```bash
# Login returns MFA challenge
POST /api/v1/auth/login
{"username": "john", "password": "..."}

# Response:
{
  "mfa_required": true,
  "mfa_token": "temp_xxx",
  "mfa_methods": ["email"]
}

# Email is sent with 6-digit code

# Verify:
POST /api/v1/auth/mfa/login
{"mfa_token": "temp_xxx", "method": "email", "code": "123456"}
```

### OTP Characteristics

- 6-digit numeric code
- TTL: 5 minutes
- Rate limited: 3 verification attempts per code
- Code is single-use

---

## Magic Link (Passwordless)

### Request

```bash
POST /api/v1/auth/magic-link
{"email": "john@example.com"}
# Response: 200 OK (always, to prevent email enumeration)
```

### Verification

User clicks link in email → redirected to:

```bash
POST /api/v1/auth/magic-link/verify
{"token": "token-from-email"}
# Response: {access_token, refresh_token}
```

### Token Properties

- TTL: 15 minutes
- Single-use
- Stored in Redis with TTL

---

## Forced MFA Policy

### Tenant-Level Enforcement

Require MFA for all users in a tenant:

```bash
PUT /api/v1/settings/security
{
  "mfa_required": true,
  "mfa_methods": ["totp", "webauthn"],
  "mfa_grace_period_hours": 24
}
```

During grace period, users see a "Set up MFA" prompt but can skip. After grace
period, login is blocked until MFA is configured.

### Role-Level Enforcement

Require MFA only for privileged roles:

```bash
PUT /api/v1/roles/{role_id}
{
  "metadata": {"require_mfa": "true"}
}
```

Use a post-login hook to enforce:

```python
@app.route("/hooks/post-login")
def post_login():
    data = request.json["data"]
    if data.get("roles", []) and "admin" in data["roles"]:
        if not data.get("mfa_used", False):
            return {"action": "deny", "reason": "MFA required for admin role"}
    return {"action": "allow"}
```

### Step-Up MFA

For sensitive operations (already authenticated user):

```bash
# Check if step-up needed
GET /api/v1/auth/step-up-check?scope=sensitive_operation

# Trigger step-up challenge
POST /api/v1/auth/step-up
{"scope": "sensitive_operation", "methods": ["totp"]}

# Verify challenge
POST /api/v1/auth/stepup/verify
{"challenge_id": "xxx", "code": "123456"}
# Response: {access_token (elevated), elevated: true}
```

---

## MFA Method Priority

When multiple MFA methods are enabled, the user selects during login:

```bash
# Login response offers multiple methods
{
  "mfa_required": true,
  "mfa_token": "temp_xxx",
  "mfa_methods": ["totp", "webauthn", "email"]
}
```

The Auth service tries the method the user selects. If that method is unavailable
(e.g., no authenticator app), the user can try another method.

---

## MFA Disable / Reset

### User-Initiated

```bash
DELETE /api/v1/auth/mfa
{"method": "totp"}
# Requires current password verification
```

### Admin-Initiated (Account Recovery)

If a user loses their MFA device:

1. Admin verifies user identity (via out-of-band channel)
2. Admin resets MFA:

```bash
DELETE /api/v1/users/{user_id}/mfa
# Clears all MFA credentials for the user
# User must set up MFA again on next login (if required)
```

3. Audit event published: `mfa.disable` (actor: admin)
