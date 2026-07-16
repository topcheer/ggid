# Authentication Guide

> Master guide to all authentication methods supported by GGID: passwords, MFA, WebAuthn/Passkeys, LDAP, OAuth/OIDC, SAML, and social login. How they work, how to configure them, and how they chain together.

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication Architecture](#authentication-architecture)
3. [Password Authentication](#password-authentication)
4. [Multi-Factor Authentication (MFA)](#multi-factor-authentication-mfa)
5. [WebAuthn / Passkeys](#webauthn--passkeys)
6. [LDAP / Active Directory](#ldap--active-directory)
7. [OAuth 2.1 / OIDC](#oauth-21--oidc)
8. [SAML 2.0](#saml-20)
9. [Social Login](#social-login)
10. [Auth Provider Chain](#auth-provider-chain)
11. [JWT Token Lifecycle](#jwt-token-lifecycle)
12. [Session Management](#session-management)
13. [Step-Up Authentication](#step-up-authentication)

---

## Overview

GGID supports a comprehensive set of authentication methods that can be used individually or chained together:

| Method | Type | Use Case |
|--------|------|----------|
| Password + bcrypt | Knowledge factor | Default login |
| TOTP (Google Authenticator) | Possession factor | MFA second step |
| Email OTP | Possession factor | MFA alternative |
| WebAuthn / Passkeys | Possession + biometric | Passwordless, phishing-resistant |
| LDAP / Active Directory | Directory bind | Enterprise integration |
| OAuth 2.1 / OIDC | Delegated authorization | Third-party apps, SSO |
| SAML 2.0 | Federated SSO | Enterprise SSO |
| Social (Google, GitHub, etc.) | Social login | Consumer apps |

---

## Authentication Architecture

### Request Flow

```
┌────────┐     ┌──────────┐     ┌──────────────┐     ┌──────────┐
│ Client │────▶│  Gateway │────▶│  Auth Svc    │────▶│ Provider │
│        │     │  (:8080) │     │  (:9001)     │     │  Chain   │
└────────┘     └──────────┘     └──────┬───────┘     └────┬─────┘
                                      │                   │
                               ┌──────▼───────┐    ┌──────▼──────┐
                               │   Redis      │    │ PostgreSQL  │
                               │ (sessions,   │    │ (users,     │
                               │  rate limit) │    │  credentials)│
                               └──────────────┘    └─────────────┘
```

### Auth Service Endpoints

| Endpoint | Method | Auth Required | Purpose |
|----------|--------|---------------|---------|
| `/api/v1/auth/register` | POST | No | Create new user |
| `/api/v1/auth/login` | POST | No | Authenticate with credentials |
| `/api/v1/auth/refresh` | POST | Refresh token | Get new access token |
| `/api/v1/auth/logout` | POST | Bearer | Revoke session |
| `/api/v1/auth/mfa/setup` | POST | Bearer | Enroll MFA device |
| `/api/v1/auth/mfa/verify` | POST | Bearer | Verify MFA challenge |
| `/api/v1/auth/mfa/disable` | POST | Bearer | Remove MFA |
| `/api/v1/auth/webauthn/register` | POST | Bearer | Register passkey |
| `/api/v1/auth/webauthn/authenticate` | POST | No | Login with passkey |
| `/api/v1/auth/password/reset` | POST | No | Request reset email |
| `/api/v1/auth/password/change` | POST | Bearer | Change password |

---

## Password Authentication

### Registration

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "SecurePass123!",
    "first_name": "John",
    "last_name": "Doe"
  }'
```

Response: `201 Created`
```json
{
  "user_id": "usr_abc123",
  "username": "johndoe",
  "email": "john@example.com",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "created_at": "2025-07-11T12:00:00Z"
}
```

**Important**: The `username` field is the credential identifier (not `email`). A unique username is required — empty username causes 409 Conflict.

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "johndoe",
    "password": "SecurePass123!"
  }'
```

Response: `200 OK`
```json
{
  "access_token": "eyJhbGci...",
  "refresh_token": "eyJhbGci...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "usr_abc123",
    "username": "johndoe",
    "email": "john@example.com",
    "roles": ["end_user"],
    "scopes": "read:self write:self"
  }
}
```

### Password Storage

- **Algorithm**: bcrypt with cost factor 12
- **Pepper**: Optional server-side pepper (recommended for production)
- **Migration**: Old hashes upgraded on next login automatically

### Password Requirements

Configurable via password policy:
- Minimum 8 characters (configurable)
- At least 1 uppercase, 1 lowercase, 1 digit, 1 special character
- Password breach check (planned — integration with HaveIBeenPwned API)
- Password reuse prevention (last 5 passwords)

---

## Multi-Factor Authentication (MFA)

### MFA Methods

| Method | Setup | Verification | Security |
|--------|-------|-------------|----------|
| TOTP | Scan QR code | 6-digit code from app | High |
| Email OTP | Auto (uses email) | 6-digit code via email | Medium |
| SMS OTP | Provide phone | 6-digit code via SMS | Medium |
| WebAuthn | Register device | Biometric/PIN | Highest |

### TOTP Setup

```bash
# Step 1: Generate secret + QR code
curl -X POST http://localhost:8080/api/v1/auth/mfa/setup \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"method": "totp"}'
```

Response:
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code": "data:image/png;base64,..."
}
```

> **Note:** Backup codes are generated separately via the dedicated endpoints below.

```bash
# Step 2: Verify with code from authenticator app
curl -X POST http://localhost:8080/api/v1/auth/mfa/verify \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"method": "totp", "code": "123456"}'
```

### Backup Codes

Backup codes are one-time recovery codes used when the primary MFA factor
is unavailable. They are hashed at rest with Argon2id and displayed only once.

#### Generate Backup Codes

```bash
curl -X POST http://localhost:8080/api/v1/auth/mfa/backup-codes/generate \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: <tenant-id>" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "<user-uuid>"}'
```

Response (200 OK):

```json
{
  "codes": ["BCDFG-HJKLM", "NPQRS-TVWXY", "BCDFG-HJKLM", "..."],
  "count": 10,
  "warning": "Store these codes securely. They will not be shown again.",
  "expires_in": "until regenerated"
}
```

- 10 single-use codes generated per request
- Regenerating invalidates all previous codes
- Codes are Argon2id-hashed; plaintext is returned only once

#### Login with Backup Code (Alternative MFA Factor)

```bash
curl -X POST http://localhost:8080/api/v1/auth/mfa/backup-codes/verify \
  -H "X-Tenant-ID: <tenant-id>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "password": "secret",
    "backup_code": "BCDFG-HJKLM"
  }'
```

Alternatively, the standard MFA login endpoint accepts a `backup_code` field:

```bash
curl -X POST http://localhost:8080/api/v1/auth/mfa/login \
  -H "X-Tenant-ID: <tenant-id>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "password": "secret",
    "backup_code": "BCDFG-HJKLM"
  }'
```

Response: standard `TokenSet` (access_token, refresh_token, session_id).
Error: `401 Unauthorized` if backup code is invalid or already used.

#### Check Remaining Codes

```bash
curl http://localhost:8080/api/v1/auth/mfa/backup-codes/remaining?user_id=<uuid> \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: <tenant-id>"
```

Response:

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "remaining": 8,
  "total": 10
}
```

### Per-Role MFA Enforcement

Configure MFA requirements per role via ABAC rules:

```json
{
  "rule_name": "admin_mfa_required",
  "condition": "user.role = 'admin' AND user.mfa_enrolled = true",
  "action": "REQUIRE_MFA"
}
```

---

## WebAuthn / Passkeys

WebAuthn provides phishing-resistant, passwordless authentication using device biometrics (Touch ID, Face ID, Windows Hello) or hardware security keys (YubiKey).

### Registration

```bash
# Step 1: Get registration challenge
curl -X POST http://localhost:8080/api/v1/auth/webauthn/register/begin \
  -H "Authorization: Bearer <JWT>"
```

```bash
# Step 2: Complete registration with device response
curl -X POST http://localhost:8080/api/v1/auth/webauthn/register/finish \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"attestation": "<base64-device-response>"}'
```

### Authentication

```bash
# Step 1: Get authentication challenge
curl -X POST http://localhost:8080/api/v1/auth/webauthn/authenticate/begin \
  -H "Content-Type: application/json" \
  -d '{"username": "johndoe"}'
```

```bash
# Step 2: Complete authentication with device response
curl -X POST http://localhost:8080/api/v1/auth/webauthn/authenticate/finish \
  -H "Content-Type: application/json" \
  -d '{"assertion": "<base64-device-response>"}'
```

### Supported Attestation Formats

| Format | Status |
|--------|--------|
| `none` | Verified |
| `packed` | Verified |
| `tpm` | Verified |
| `android-key` | Verified |
| `android-safetynet` | Verified |
| `fido-u2f` | Verified |
| `apple` | Verified |

See: [WebAuthn Implementation Guide](webauthn-implementation-guide.md) for detailed setup.

---

## LDAP / Active Directory

LDAP integration allows authenticating users against existing directory infrastructure.

### Configuration

Environment variables in auth service:

```bash
LDAP_URL=ldap://localhost:389
LDAP_BIND_DN=cn=admin,dc=ggid,dc=io
LDAP_BIND_PASSWORD=admin
LDAP_BASE_DN=dc=ggid,dc=io
LDAP_USER_FILTER=(uid=%s)
LDAP_START_TLS=false
LDAP_AUTO_PROVISION=true
```

### How It Works

1. User submits username + password to `/api/v1/auth/login`
2. Auth service tries local provider first
3. If local fails, tries LDAP provider:
   a. Bind with service account
   b. Search for user by filter (`(uid=johndoe)`)
   c. Bind with user DN + submitted password
   d. If bind succeeds → authenticated
4. If `LDAP_AUTO_PROVISION=true`, user is created in local DB on first login

### LDAP + Local Chain

```
Login Request → Local Provider (check local DB)
                   ↓ (not found)
                LDAP Provider (bind to directory)
                   ↓ (success)
                Auto-provision (if enabled)
                   ↓
                JWT issued
```

See: [LDAP Directory Sync](ldap-directory-sync.md) for full configuration.

---

## OAuth 2.1 / OIDC

GGID acts as both an OAuth 2.1 authorization server and an OIDC provider.

### As Authorization Server (GGID issues tokens)

Third-party apps can request access to GGID-protected APIs:

```
GET /api/v1/oauth/authorize?
  response_type=code&
  client_id=CLIENT_ID&
  redirect_uri=https://app.example.com/callback&
  scope=read:users&
  code_challenge=CODE_CHALLENGE&
  code_challenge_method=S256&
  state=RANDOM_STATE
```

### As OIDC Provider (GGID provides identity)

Relying parties can use GGID for single sign-on:

```
GET /api/v1/oauth/authorize?
  response_type=code&
  client_id=CLIENT_ID&
  redirect_uri=https://app.example.com/callback&
  scope=openid profile email&
  code_challenge=CODE_CHALLENGE&
  code_challenge_method=S256
```

### Supported Flows

| Flow | Use Case | PKCE Required |
|------|----------|---------------|
| Authorization Code | Web apps, mobile apps | Yes (mandatory) |
| Client Credentials | Service-to-service | N/A |
| Device Authorization | IoT, CLI tools | N/A |
| Token Exchange (RFC 8693) | Delegation | N/A |

See: [OAuth Flows Guide](oauth-flows-guide.md) for details.

---

## SAML 2.0

SAML 2.0 integration enables enterprise federated single sign-on.

### Service Provider (SP) Mode

GGID acts as a SAML Service Provider, trusting an external Identity Provider (IdP):

1. User accesses GGID → redirected to IdP
2. User authenticates at IdP
3. IdP sends SAML assertion to GGID ACS endpoint
4. GGID validates assertion, creates session

### Configuration

```bash
SAML_IDP_METADATA_URL=https://idp.example.com/metadata
SAML_SP_ENTITY_ID=https://ggid.example.com
SAML_ACS_URL=https://ggid.example.com/api/v1/saml/acs
SAML_CERT=/path/to/idp-certificate.pem
```

See: [SAML Configuration](saml-configuration.md) for detailed setup.

---

## Social Login

GGID supports social login through OAuth 2.0 connectors:

| Provider | Supported | Scope |
|----------|-----------|-------|
| Google | Yes | `openid email profile` |
| GitHub | Yes | `user:email` |
| Microsoft | Yes | `openid email profile` |
| Discord | Yes | `identify email` |
| Slack | Yes | `identity.basic` |
| LinkedIn | Yes | `r_emailaddress r_liteprofile` |
| GitLab | Yes | `read_user` |
| Apple | Yes | `email name` |

### How It Works

1. User clicks "Sign in with Google"
2. Redirect to Google OAuth consent
3. Google redirects back with code
4. GGID exchanges code for user info
5. If user exists → login; if not → auto-provision

See: [Social Login Guide](social-login-guide.md) for setup.

---

## Auth Provider Chain

GGID uses a chain-of-responsibility pattern for authentication:

```go
// Auth service chains providers
chain := authprovider.NewChain(
    authprovider.NewLocalProvider(db, crypto),
    authprovider.NewLDAPProvider(ldapConfig),  // optional
)

// On login, tries each provider in order
result, err := chain.Authenticate(ctx, username, password)
```

### Chain Order

1. **Local Provider**: Check local database (bcrypt hashed passwords)
2. **LDAP Provider** (if configured): Try LDAP bind
3. **Social Connectors** (if OAuth flow): Exchange code, get user info

If any provider succeeds, authentication succeeds. If all fail, 401 Unauthorized.

---

## JWT Token Lifecycle

### Token Issuance

```
Login success
    ↓
Generate access token (15 min, HMAC-SHA256)
    + tenant_id from user record
    + scope from user roles
    + jti (unique ID for anti-replay)
    ↓
Generate refresh token (7 days)
    + Store in Redis with TTL
    ↓
Return both tokens to client
```

### Token Refresh

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "eyJhbGci..."}'
```

- Old refresh token is invalidated (rotation)
- New access + refresh token pair issued
- If old refresh token is used again → session revoked (replay detection)

### Token Revocation

```
POST /api/v1/auth/logout
  → Add access token JTI to Redis revocation list (TTL = remaining token life)
  → Delete refresh token from Redis
  → Invalidate session
```

See: [Token Lifecycle](token-lifecycle.md) for full details.

---

## Session Management

### Session Tracking

| Store | Key | TTL | Purpose |
|-------|-----|-----|---------|
| Redis | `session:{session_id}` | 8 hours | Active session data |
| Redis | `refresh:{token_hash}` | 7 days | Refresh token validity |
| Redis | `jti:{token_jti}` | 15 min | Anti-replay for access tokens |
| Redis | `revoked:{session_id}` | — | Revoked sessions |

### Concurrent Sessions

- Default: 5 concurrent sessions per user
- Configurable per tenant
- Oldest session evicted when limit exceeded
- All sessions revocable via admin API

### Session Information

Each session tracks:
- User ID
- IP address
- User-Agent
- Creation time
- Last activity time
- Expiration time

---

## Step-Up Authentication

Step-up authentication requires additional verification for sensitive operations:

```
User has valid JWT (basic auth)
    ↓
Attempts sensitive operation (delete user, change security settings)
    ↓
Gateway checks: does this endpoint require step-up?
    ↓ Yes
Returns 403 with "mfa_required" error
    ↓
Client prompts for MFA
    ↓
User completes MFA challenge
    ↓
New JWT issued with "mfa_verified" claim + short TTL (5 min)
    ↓
Sensitive operation allowed
```

### Configuration

Mark endpoints as requiring step-up:

```go
// In route configuration
routes map[string]bool{
    "/api/v1/users/*/delete": true,
    "/api/v1/admin/*":        true,
    "/api/v1/security/*":     true,
}
```

---

*Last updated: 2025-07-11*
