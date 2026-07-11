# MFA Enforcement Guide

> Configure Multi-Factor Authentication: TOTP setup, forced MFA for admins, recovery codes.

---

## MFA Methods

| Method | Type | Setup |
|--------|------|-------|
| TOTP | Authenticator app (Google Authenticator, Authy) | Scan QR code |
| WebAuthn | Passkey (Touch ID, security key) | [WebAuthn Guide](webauthn-setup.md) |

---

## TOTP Setup

### Enable TOTP for a User

```bash
# Step 1: Generate TOTP secret + QR code
curl -X POST http://localhost:8080/api/v1/auth/mfa/totp/setup \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT"
```

**Response:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code": "data:image/png;base64,iVBOR...",
  "issuer": "GGID",
  "account": "alice@test.com"
}
```

### Verify TOTP

```bash
# Step 2: User enters 6-digit code from authenticator app
curl -X POST http://localhost:8080/api/v1/auth/mfa/totp/verify \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"code":"123456"}'
```

**Response (200):**
```json
{ "status": "enabled", "recovery_codes": ["abc123", "def456", ...] }
```

### Login with MFA

```bash
# Step 1: Normal login returns MFA challenge
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","password":"Secure123!"}'
# → {"mfa_required": true, "mfa_token": "mfa_xyz"}

# Step 2: Verify MFA code
curl -X POST http://localhost:8080/api/v1/auth/mfa/verify \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"mfa_token":"mfa_xyz","code":"123456"}'
# → {"access_token":"eyJ...","refresh_token":"rft..."}
```

---

## Disable TOTP

```bash
curl -X DELETE http://localhost:8080/api/v1/auth/mfa/totp \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"code":"123456"}'
```

Requires current TOTP code to prevent accidental removal.

---

## Recovery Codes

When TOTP is enabled, 10 one-time recovery codes are generated. Store them safely.

### Regenerate Recovery Codes

```bash
curl -X POST http://localhost:8080/api/v1/auth/mfa/recovery/regenerate \
  -H "Authorization: Bearer $JWT"
```

### Login with Recovery Code

```bash
curl -X POST http://localhost:8080/api/v1/auth/mfa/verify \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"mfa_token":"mfa_xyz","recovery_code":"abc123"}'
```

---

## Forced MFA (Admin Policy)

### Require MFA for All Users

```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Force MFA for all users",
    "effect": "deny",
    "actions": ["*"],
    "resources": ["*"],
    "priority": 500
  }'
```

Then evaluate with `user.mfa_verified: false` attribute to deny access until MFA is set up.

### Require MFA for Admin Role Only

```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "name": "Force MFA for admin role",
    "effect": "deny",
    "actions": ["*"],
    "resources": ["admin:*"],
    "priority": 400
  }'
```

---

## Configuration

```bash
# TOTP settings
TOTP_ISSUER=GGID
TOTP_ALGORITHM=SHA1
TOTP_DIGITS=6
TOTP_PERIOD=30

# Recovery codes
RECOVERY_CODE_COUNT=10
```

---

*See: [WebAuthn Setup](webauthn-setup.md) | [Security Hardening](security-hardening.md) | [ABAC Policy](abac-policy.md)*

*Last updated: 2025-07-11*
