# WebAuthn / Passkey Setup

> Configure passwordless authentication with WebAuthn (Face ID, Touch ID, security keys).

---

## How WebAuthn Works

```
1. Registration:
   User → Server: "Register passkey"
   Server → User: Challenge (random bytes)
   User → Authenticator: Touch sensor / Face ID
   Authenticator → Server: Public key + attestation + signed challenge

2. Authentication:
   User → Server: "Login with passkey"
   Server → User: Challenge (random bytes)
   User → Authenticator: Touch sensor / Face ID
   Authenticator → Server: Signed challenge (proves private key)
```

---

## API Endpoints

### Begin Registration

```bash
curl -X POST http://localhost:8080/api/v1/auth/webauthn/register/begin \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT"
```

**Response:**
```json
{
  "challenge": "base64random...",
  "rp": { "name": "GGID", "id": "localhost" },
  "user": { "id": "usr_abc", "name": "alice@test.com", "displayName": "Alice" },
  "pubKeyCredParams": [{ "type": "public-key", "alg": -7 }],
  "authenticatorSelection": { "userVerification": "preferred" },
  "timeout": 60000
}
```

### Finish Registration

```bash
curl -X POST http://localhost:8080/api/v1/auth/webauthn/register/finish \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"credential": {...}, "label": "MacBook Touch ID"}'
```

### Begin Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/webauthn/login/begin \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"username":"alice"}'
```

### Finish Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/webauthn/login/finish \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"credential": {...}}'
```

---

## Frontend Integration (JavaScript)

```javascript
// Registration
const beginResp = await fetch('/api/v1/auth/webauthn/register/begin', {
  method: 'POST',
  headers: { Authorization: `Bearer ${jwt}` }
});
const options = await beginResp.json();

// Create credential via browser WebAuthn API
const credential = await navigator.credentials.create({ publicKey: options });

// Send to server
await fetch('/api/v1/auth/webauthn/register/finish', {
  method: 'POST',
  headers: { Authorization: `Bearer ${jwt}`, 'Content-Type': 'application/json' },
  body: JSON.stringify({ credential, label: 'MacBook' })
});

// Login
const loginResp = await fetch('/api/v1/auth/webauthn/login/begin', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'alice' })
});
const assertion = await navigator.credentials.get({ publicKey: await loginResp.json() });

const authResp = await fetch('/api/v1/auth/webauthn/login/finish', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ credential: assertion })
});
const { access_token } = await authResp.json();
```

---

## Configuration

```bash
# Relying party (must match your domain)
WEBAUTHN_RP_ID=localhost
WEBAUTHN_RP_NAME=GGID
WEBAUTHN_RP_ORIGINS=http://localhost:8080

# Attestation
WEBAUTHN_ATTESTATION_PREFERRED=none  # none, indirect, direct
WEBAUTHN_USER_VERIFICATION=preferred   # required, preferred, discouraged
```

---

## Device Management

### List Registered Passkeys

```bash
curl http://localhost:8080/api/v1/auth/webauthn/credentials \
  -H "Authorization: Bearer $JWT"
```

**Response:**
```json
{
  "credentials": [
    { "id": "cred_1", "label": "MacBook Touch ID", "created_at": "2025-07-11T..." },
    { "id": "cred_2", "label": "YubiKey 5C", "created_at": "2025-07-10T..." }
  ]
}
```

### Remove a Passkey

```bash
curl -X DELETE http://localhost:8080/api/v1/auth/webauthn/credentials/cred_2 \
  -H "Authorization: Bearer $JWT"
```

---

## Supported Authenticators

| Type | Examples | User Verification |
|------|----------|-------------------|
| Platform | Touch ID, Face ID, Windows Hello | Biometric |
| Roaming | YubiKey, Google Titan, Feitian | PIN / touch |
| Hybrid | Phone (via QR code) | Biometric |

---

*See: [MFA Enforcement](mfa-enforcement.md) | [Security Overview](../architecture/security-overview.md) | [Social Login](social-login-setup.md)*

*Last updated: 2025-07-11*
