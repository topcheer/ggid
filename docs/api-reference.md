# GGID IAM Platform — API Reference

> Base URL: `http://localhost:8080`
> All requests require `X-Tenant-ID` header.
> Protected routes require `Authorization: Bearer <JWT>`.

## Authentication

### Register
```http
POST /api/v1/auth/register
Content-Type: application/json
X-Tenant-ID: <tenant-uuid>

{
  "username": "newuser",
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```
**Response 201:** `{ "user_id": "<uuid>" }`
**Response 409:** `{ "error": "username or email already registered" }`

### Login
```http
POST /api/v1/auth/login
Content-Type: application/json
X-Tenant-ID: <tenant-uuid>

{
  "username": "admin",
  "password": "Admin@123456"
}
```
**Response 200:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "session_id": "<uuid>"
}
```

### Refresh Token
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{ "refresh_token": "eyJ..." }
```

### Logout
```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
```

### Social Login — Begin
```http
GET /api/v1/auth/social/{provider}?redirect_uri=/
X-Tenant-ID: <tenant-uuid>
```
Providers: `google`, `github`, `oidc` (generic)

**Response 200:**
```json
{
  "provider": "google",
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth?...",
  "state": "<uuid>"
}
```

### Social Login — Callback
```http
GET /api/v1/auth/social/{provider}/callback?code=<auth_code>&state=<state>
X-Tenant-ID: <tenant-uuid>
```
**Response 200:**
```json
{
  "provider": "google",
  "external_id": "123456789",
  "email": "user@gmail.com",
  "name": "John Doe",
  "avatar_url": "https://..."
}
```

### MFA — Setup TOTP
```http
POST /api/v1/auth/mfa/setup
Authorization: Bearer <token>
```
Returns TOTP secret + QR code URL.

### MFA — Verify
```http
POST /api/v1/auth/mfa/verify
Authorization: Bearer <token>
Content-Type: application/json

{ "code": "123456" }
```

---

## User Management

### List Users
```http
GET /api/v1/users
Authorization: Bearer <token>
X-Tenant-ID: <tenant-uuid>
```
**Response 200:**
```json
{
  "users": [
    {
      "id": "<uuid>",
      "username": "admin",
      "email": "admin@ggid.dev",
      "status": "active",
      "display_name": "System Administrator",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Get User by ID
```http
GET /api/v1/users/{id}
Authorization: Bearer <token>
```

### Lock/Unlock User
```http
POST /api/v1/users/{id}/lock
POST /api/v1/users/{id}/unlock
Authorization: Bearer <token>
```

### Delete User
```http
DELETE /api/v1/users/{id}
Authorization: Bearer <token>
```

---

## Roles & Permissions

### List Roles
```http
GET /api/v1/roles
Authorization: Bearer <token>
X-Tenant-ID: <tenant-uuid>
```

### Create Role
```http
POST /api/v1/roles
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Developer",
  "key": "developer",
  "description": "Development team role"
}
```

### Delete Role
```http
DELETE /api/v1/roles/{id}
Authorization: Bearer <token>
```

### List Permissions
```http
GET /api/v1/permissions
Authorization: Bearer <token>
```

### Check Policy
```http
POST /api/v1/policies/check
Authorization: Bearer <token>
Content-Type: application/json

{
  "principal": "user:<uuid>",
  "action": "iam:users:read",
  "resource": "tenant:<uuid>"
}
```
**Response 200:** `{ "allowed": true }`

---

## Organizations

### List Organizations
```http
GET /api/v1/orgs
Authorization: Bearer <token>
X-Tenant-ID: <tenant-uuid>
```

### Create Organization
```http
POST /api/v1/orgs
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Engineering",
  "slug": "engineering"
}
```

### Departments
```http
GET /api/v1/departments
POST /api/v1/departments
```

### Teams
```http
GET /api/v1/teams
POST /api/v1/teams
```

---

## Audit

### Query Audit Events
```http
GET /api/v1/audit?limit=20&offset=0
Authorization: Bearer <token>
X-Tenant-ID: <tenant-uuid>
```
**Response 200:**
```json
{
  "events": [
    {
      "id": "<uuid>",
      "event_type": "user.login",
      "principal": "admin",
      "resource": "auth:login",
      "action": "login",
      "result": "success",
      "ip_address": "192.168.1.1",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 42
}
```

---

## OAuth2 / OIDC

### OIDC Discovery
```http
GET /oauth/.well-known/openid-configuration
```

### JWKS
```http
GET /oauth/jwks
```

### Authorize
```http
GET /oauth/authorize?client_id=...&redirect_uri=...&response_type=code&scope=openid+profile+email&state=...
```

### Token
```http
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&code=...&redirect_uri=...&client_id=...&client_secret=...
```

### UserInfo
```http
GET /oauth/userinfo
Authorization: Bearer <oauth-access-token>
```

---

## Health Checks

```http
GET /healthz                    # Gateway
GET /healthz (port 8081)       # Identity
GET /healthz (port 9001)       # Auth
GET /healthz (port 9005)       # OAuth
GET /healthz (port 8070)       # Policy
GET /healthz (port 8071)       # Org
GET /healthz (port 8072)       # Audit
```
All return: `{ "status": "ok" }`

---

## Error Format

All errors follow this structure:
```json
{
  "error": "human-readable error message",
  "code": "MACHINE_CODE"
}
```

| HTTP Status | Meaning |
|-------------|---------|
| 400 | Bad request — invalid input |
| 401 | Unauthorized — missing/invalid JWT |
| 403 | Forbidden — insufficient permissions |
| 404 | Not found |
| 409 | Conflict — duplicate resource |
| 429 | Too many requests — rate limited |
| 500 | Internal server error |

---

## WebAuthn (Passkeys)

GGID implements the WebAuthn (Web Authentication) API for passwordless
authentication using passkeys, security keys, and platform authenticators
(Face ID, Touch ID, Windows Hello).

### Begin Registration

Initiates a WebAuthn credential registration flow.

```http
POST /api/v1/auth/webauthn/register/begin
Authorization: Bearer <JWT>
Content-Type: application/json
X-Tenant-ID: <tenant-uuid>

{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My iPhone",
  "authenticator_selection": {
    "authenticator_attachment": "platform",
    "user_verification": "preferred",
    "resident_key": "preferred",
    "require_resident_key": false
  }
}
```

**Response 200:**
```json
{
  "challenge": "base64url-challenge-string",
  "rp": {
    "name": "GGID",
    "id": "example.com"
  },
  "user": {
    "id": "base64url-user-handle",
    "name": "john.doe@example.com",
    "displayName": "John Doe"
  },
  "pubKeyCredParams": [
    { "type": "public-key", "alg": -7 },
    { "type": "public-key", "alg": -257 }
  ],
  "timeout": 60000,
  "excludeCredentials": [
    {
      "type": "public-key",
      "id": "base64url-existing-credential-id"
    }
  ],
  "attestation": "none",
  "extensions": {
    "credProps": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `challenge` | base64url | Random challenge to prevent replay attacks |
| `rp.id` | string | Relying Party ID (domain). Must match origin |
| `rp.name` | string | Display name shown in browser prompt |
| `user.id` | base64url | User handle (opaque, not PII) |
| `user.displayName` | string | Human-readable name shown in browser prompt |
| `pubKeyCredParams` | array | Supported algorithms: ES256 (-7), RS256 (-257) |
| `timeout` | int | Milliseconds before challenge expires |
| `excludeCredentials` | array | Credentials already registered (prevents duplicates) |
| `attestation` | string | Attestation conveyance: `none`, `indirect`, `direct` |

### Finish Registration

Completes registration by verifying the attestation from the authenticator.

```http
POST /api/v1/auth/webauthn/register/finish
Authorization: Bearer <JWT>
Content-Type: application/json
X-Tenant-ID: <tenant-uuid>

{
  "id": "base64url-credential-id",
  "rawId": "base64url-credential-id",
  "type": "public-key",
  "response": {
    "attestationObject": "base64url-attestation-object",
    "clientDataJSON": "base64url-client-data-json",
    "transports": ["internal", "hybrid"]
  }
}
```

**Response 200:**
```json
{
  "credential_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My iPhone",
  "public_key": "base64url-spki-public-key",
  "aaguid": "00000000-0000-0000-0000-000000000000",
  "backup_eligible": true,
  "backup_state": true,
  "transports": ["internal", "hybrid"],
  "registered": true
}
```

| Field | Type | Description |
|-------|------|-------------|
| `credential_id` | UUID | Internal credential identifier |
| `name` | string | Human-readable credential name |
| `aaguid` | UUID | Authenticator Attestation GUID (identifies device model) |
| `backup_eligible` | bool | Whether credential supports multi-device sync |
| `backup_state` | bool | Whether credential is currently synced |
| `transports` | array | How the credential can communicate: `usb`, `nfc`, `ble`, `internal`, `hybrid` |

### Begin Authentication

Initiates a WebAuthn authentication (assertion) flow.

```http
POST /api/v1/auth/webauthn/login/begin
Content-Type: application/json
X-Tenant-ID: <tenant-uuid>

{
  "username": "john.doe@example.com",
  "user_verification": "preferred"
}
```

**Response 200:**
```json
{
  "challenge": "base64url-challenge-string",
  "rpId": "example.com",
  "allowCredentials": [
    {
      "type": "public-key",
      "id": "base64url-credential-id",
      "transports": ["internal", "hybrid"]
    }
  ],
  "userVerification": "preferred",
  "timeout": 60000
}
```

| Field | Type | Description |
|-------|------|-------------|
| `challenge` | base64url | Random challenge for this assertion |
| `rpId` | string | Relying Party ID that the credential was registered with |
| `allowCredentials` | array | Matching credentials for this user. Empty for discoverable credentials (passkeys) |
| `userVerification` | string | `required`, `preferred`, or `discouraged` |

### Finish Authentication

Completes authentication by verifying the signature from the authenticator.

```http
POST /api/v1/auth/webauthn/login/finish
Content-Type: application/json
X-Tenant-ID: <tenant-uuid>

{
  "id": "base64url-credential-id",
  "rawId": "base64url-credential-id",
  "type": "public-key",
  "response": {
    "authenticatorData": "base64url-auth-data",
    "clientDataJSON": "base64url-client-data",
    "signature": "base64url-signature",
    "userHandle": "base64url-user-handle"
  }
}
```

**Response 200:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "rt-550e8400-e29b-41d4-...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### List Credentials

```http
GET /api/v1/auth/webauthn/credentials?user_id=<uuid>
Authorization: Bearer <JWT>
X-Tenant-ID: <tenant-uuid>
```

**Response 200:**
```json
{
  "credentials": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "My iPhone",
      "aaguid": "adce0002-35bc-c60a-648b-0b25f1f05503",
      "backup_eligible": true,
      "backup_state": true,
      "transports": ["internal", "hybrid"],
      "created_at": "2024-01-15T10:30:00Z",
      "last_used_at": "2024-07-10T08:15:00Z",
      "sign_count": 142
    }
  ]
}
```

### Rename Credential

```http
PATCH /api/v1/auth/webauthn/credentials/:id
Authorization: Bearer <JWT>
Content-Type: application/json
X-Tenant-ID: <tenant-uuid>

{ "name": "Work Laptop" }
```

### Delete Credential

```http
DELETE /api/v1/auth/webauthn/credentials/:id
Authorization: Bearer <JWT>
X-Tenant-ID: <tenant-uuid>
```

**Response 204:** _(empty body)_

### Supported Algorithms

| COSE Algorithm ID | Name | Key Type | Recommended |
|-------------------|------|----------|-------------|
| -7 | ES256 (ECDSA w/ SHA-256) | ECDSA P-256 | Yes (preferred) |
| -257 | RS256 (RSASSA-PKCS1-v1_5 w/ SHA-256) | RSA 2048 | Yes (fallback) |
| -8 | EdDSA | Ed25519 | Optional |
| -37 | PS256 (RSASSA-PSS w/ SHA-256) | RSA 2048 | Optional |

### WebAuthn Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `WEBAUTHN_INVALID_CHALLENGE` | 400 | Challenge expired or mismatched |
| `WEBAUTHN_INVALID_ORIGIN` | 400 | Origin not in allowed list |
| `WEBAUTHN_INVALID_RP_ID` | 400 | RP ID doesn't match configuration |
| `WEBAUTHN_INVALID_ATTESTATION` | 400 | Attestation verification failed |
| `WEBAUTHN_INVALID_ASSERTION` | 400 | Assertion signature verification failed |
| `WEBAUTHN_USER_NOT_FOUND` | 404 | No user with matching credential |
| `WEBAUTHN_DUPLICATE_CREDENTIAL` | 409 | Credential ID already registered |
| `WEBAUTHN_RATE_LIMITED` | 429 | Too many WebAuthn attempts |

---

## SCIM 2.0 Endpoints

```http
GET    /scim/v2/Users                  # List (paginated)
POST   /scim/v2/Users                  # Create
GET    /scim/v2/Users/:id              # Get by ID
PUT    /scim/v2/Users/:id              # Replace
PATCH  /scim/v2/Users/:id              # Partial update
DELETE /scim/v2/Users/:id              # Deactivate
GET    /scim/v2/Groups                 # List groups
POST   /scim/v2/Groups                 # Create group
GET    /scim/v2/Groups/:id             # Get group
```

**Headers:** `Authorization: Bearer <SCIM-API-Key>`, `X-Tenant-ID: <tenant-uuid>`

**Example: Create User via SCIM**
```bash
curl -X POST http://localhost:8080/scim/v2/Users \
  -H "Authorization: Bearer $SCIM_KEY" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
    "userName": "jane.doe@example.com",
    "name": { "givenName": "Jane", "familyName": "Doe" },
    "emails": [{ "value": "jane.doe@example.com", "primary": true }],
    "active": true
  }'
```

---

## gRPC Endpoints

GGID services expose gRPC APIs for high-performance internal communication.

| Service | Port | Proto Package |
|---------|------|---------------|
| Identity | 50051 | `ggid.identity.v1` |
| Policy | 9070 | `ggid.policy.v1` |
| Org | 9071 | `ggid.org.v1` |
| Audit | 9072 | `ggid.audit.v1` |

**Example (grpcurl):**
```bash
# List identity service methods
grpcurl -plaintext localhost:50051 list ggid.identity.v1.IdentityService

# Call GetUser
grpcurl -plaintext -d '{"tenant_id":"00000000-0000-0000-0000-000000000001","user_id":"..."}' \
  localhost:50051 ggid.identity.v1.IdentityService/GetUser

# Policy check
grpcurl -plaintext -d '{"subject":"user-uuid","action":"read","resource":"users"}' \
  localhost:9070 ggid.policy.v1.PolicyService/Check
```

**Proto files:** `api/proto/` — use `make proto` to generate Go code.

---

## Rate Limits

| Endpoint Category | Rate Limit | Key |
|-------------------|------------|-----|
| `/api/v1/auth/login` | 10 req/min | Per IP + per username |
| `/api/v1/auth/register` | 5 req/min | Per IP |
| `/api/v1/auth/refresh` | 30 req/min | Per token |
| All other authenticated endpoints | 60 req/min | Per user (JWT sub) |
| SCIM endpoints | 100 req/min | Per API key |
| Health checks | No limit | — |

Rate limit headers:
```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 42
X-RateLimit-Reset: 1699999999
```

When exceeded: `429 Too Many Requests` with `Retry-After: <seconds>` header.

---

## Audit SSE Stream

```http
GET /api/v1/audit/stream?tenant_id=<uuid>
Authorization: Bearer <JWT>
Accept: text/event-stream
```

**Response (SSE):**
```
event: connected
data: {"message":"connected to audit stream"}

event: audit_event
data: {"event_id":"...","action":"user.login","actor_id":"...","timestamp":"..."}

event: audit_event
data: {"event_id":"...","action":"role.create","actor_id":"...","timestamp":"..."}
```

Events are pushed in real-time as they are published to NATS JetStream.

---

## Curl Quick Reference

```bash
# Set common variables
export API="http://localhost:8080"
export TENANT="00000000-0000-0000-0000-000000000001"

# Register
curl -sX POST $API/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","email":"alice@test.com","password":"StrongPass123!"}'

# Login and capture JWT
export TOKEN=$(curl -sX POST $API/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","password":"StrongPass123!"}' | jq -r .access_token)

# Authenticated request
curl -s $API/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# Create role
curl -sX POST $API/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"key":"viewer","name":"Viewer","permissions":["read:users"]}'

# Check policy
curl -sX POST $API/api/v1/policies/check \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"subject":"alice","action":"read","resource":"users"}'

# Query audit events
curl -s "$API/api/v1/audit/events?limit=10&tenant_id=$TENANT" \
  -H "Authorization: Bearer $TOKEN"
```

---

## OAuth 2.0 / OIDC Endpoints

### Discovery
```http
GET /.well-known/openid-configuration
```
Returns the OpenID Connect discovery document:
```json
{
  "issuer": "http://localhost:8080",
  "authorization_endpoint": "http://localhost:8080/oauth/authorize",
  "token_endpoint": "http://localhost:8080/oauth/token",
  "userinfo_endpoint": "http://localhost:8080/oauth/userinfo",
  "jwks_uri": "http://localhost:8080/.well-known/jwks.json",
  "revocation_endpoint": "http://localhost:8080/oauth/revoke",
  "introspection_endpoint": "http://localhost:8080/oauth/introspect",
  "end_session_endpoint": "http://localhost:8080/oauth/logout",
  "grant_types_supported": [
    "authorization_code", "refresh_token",
    "client_credentials", "password",
    "urn:ietf:params:oauth:grant-type:device_code"
  ],
  "response_types_supported": ["code", "token", "id_token"],
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic", "client_secret_post", "private_key_jwt"
  ],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256", "EdDSA"]
}
```

### JWKS
```http
GET /.well-known/jwks.json
```
Returns the JSON Web Key Set for JWT verification:
```json
{
  "keys": [{
    "kid": "2024-01-key",
    "kty": "RSA",
    "alg": "RS256",
    "use": "sig",
    "n": "...base64url...",
    "e": "AQAB"
  }]
}
```

### Authorization
```http
GET /oauth/authorize?response_type=code&client_id=CLIENT_ID&redirect_uri=URL&scope=openid%20profile&state=RANDOM
```
**Response 302:** Redirect to `redirect_uri` with `code` and `state` params.

### Token
```http
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&code=AUTH_CODE&client_id=CLIENT_ID&client_secret=SECRET&redirect_uri=URL
```
**Response 200:**
```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "rt-uuid...",
  "id_token": "eyJhbG...",
  "scope": "openid profile"
}
```

### Token Refresh
```http
POST /oauth/token
grant_type=refresh_token&refresh_token=rt-uuid...&client_id=CLIENT_ID&client_secret=SECRET
```

### Client Credentials
```http
POST /oauth/token
grant_type=client_credentials&client_id=CLIENT_ID&client_secret=SECRET&scope=policy:check
```

### Token Introspection
```http
POST /oauth/introspect
Authorization: Basic <client_credentials>
token=eyJhbG...&token_hint=access_token
```
**Response 200:**
```json
{
  "active": true,
  "scope": "openid profile",
  "client_id": "my-app",
  "username": "john.doe",
  "exp": 1700000000,
  "iat": 1699996400,
  "sub": "user-uuid",
  "tenant_id": "tenant-uuid"
}
```

### Token Revocation
```http
POST /oauth/revoke
token=rt-uuid...&token_type_hint=refresh_token
```
**Response 200:** _(empty body)_

### UserInfo
```http
GET /oauth/userinfo
Authorization: Bearer <access_token>
```
**Response 200:**
```json
{
  "sub": "550e8400-...",
  "name": "John Doe",
  "preferred_username": "john.doe",
  "email": "john@example.com",
  "email_verified": true
}
```

### Client Management
```http
POST   /api/v1/oauth/clients           # Register client
GET    /api/v1/oauth/clients           # List clients
GET    /api/v1/oauth/clients/:id       # Get client
PUT    /api/v1/oauth/clients/:id       # Update client
DELETE /api/v1/oauth/clients/:id       # Delete client
```

**Register Client:**
```json
{
  "name": "My App",
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "scope": "openid profile email",
  "token_endpoint_auth_method": "client_secret_basic"
}
```
**Response 201:**
```json
{
  "client_id": "client-uuid",
  "client_secret": "secret-shown-once",
  "name": "My App",
  "redirect_uris": ["https://app.example.com/callback"]
}
```

---

## MFA Endpoints

### Enable TOTP
```http
POST /api/v1/auth/mfa/totp/enable
Authorization: Bearer <JWT>
```
**Response 200:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code": "data:image/png;base64,...",
  "uri": "otpauth://totp/GGID:user@example.com?secret=..."
}
```

### Verify TOTP Setup
```http
POST /api/v1/auth/mfa/totp/verify
Authorization: Bearer <JWT>

{ "code": "123456" }
```
**Response 200:** `{ "enabled": true }`

### Disable TOTP
```http
DELETE /api/v1/auth/mfa/totp
Authorization: Bearer <JWT>

{ "password": "current-password" }
```

### List MFA Factors
```http
GET /api/v1/auth/mfa/factors
Authorization: Bearer <JWT>
```
**Response 200:**
```json
{
  "factors": [
    { "type": "totp", "enabled": true, "created_at": "2024-01-15T10:00:00Z" },
    { "type": "webauthn", "enabled": true, "credentials": 2 }
  ]
}
```

### Backup Codes — Generate
```http
POST /api/v1/auth/mfa/backup-codes/generate
Authorization: Bearer <JWT>
Content-Type: application/json

{ "user_id": "<user-uuid>" }
```
**Response 200:**
```json
{
  "codes": ["BCDFG-HJKLM", "NPQRS-TVWXY", "..."],
  "count": 10,
  "warning": "Store these codes securely. They will not be shown again.",
  "expires_in": "until regenerated"
}
```
Generates 10 new single-use backup codes. Previous codes are invalidated.
Codes are Argon2id-hashed at rest; plaintext is returned only once.

### Backup Codes — Verify (Login)
```http
POST /api/v1/auth/mfa/backup-codes/verify
Content-Type: application/json

{ "username": "alice", "password": "secret", "backup_code": "BCDFG-HJKLM" }
```
**Response 200:** Standard `TokenSet` (access_token, refresh_token, session_id).
**Response 401:** `invalid or used backup code` — the code is invalid, already consumed, or credentials are wrong.

Alternatively, `POST /api/v1/auth/mfa/login` accepts a `backup_code` field instead of `mfa_code`.

### Backup Codes — Remaining
```http
GET /api/v1/auth/mfa/backup-codes/remaining?user_id=<uuid>
Authorization: Bearer <JWT>
```
**Response 200:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "remaining": 8,
  "total": 10
}
```

### Step-Up Authentication
```http
POST /api/v1/auth/stepup
Authorization: Bearer <JWT>

{ "code": "123456", "method": "totp" }
```
**Response 200:**
```json
{
  "access_token": "eyJhbG...",
  "elevated": true,
  "expires_in": 300
}
```

---

## Social Login

### Initiate Social Login
```http
GET /api/v1/auth/social/:provider?redirect_uri=URL
```
Providers: `google`, `github`, `microsoft`, `discord`, `slack`, `linkedin`, `gitlab`, `apple`.

**Response 302:** Redirect to provider's OAuth consent screen.

### Social Login Callback
```http
GET /api/v1/auth/social/:provider/callback?code=OAUTH_CODE&state=RANDOM
```
**Response 302:** Redirect to application with JWT in query or fragment.

---

## API Conventions

- **Versioning:** All endpoints under `/api/v1/`. Breaking changes require `/api/v2/`.
- **Pagination:** `?page=1&limit=20`. Response includes `total` and `items`.
- **Sorting:** `?sort=created_at` (ascending) or `?sort=-created_at` (descending).
- **Filtering:** Query params map to filters (e.g., `?active=true&role=admin`).
- **ID format:** All resource IDs are UUID v4 strings.
- **Timestamps:** ISO 8601 UTC (`2024-01-15T10:30:00Z`).
- **Content-Type:** `application/json` for REST, `application/scim+json` for SCIM.
- **Tenant scope:** `X-Tenant-ID` header required on all non-health endpoints.

---

## References

- [OpenAPI Spec](./openapi.yaml) — Machine-readable API definition
- [Postman Collection](./postman-collection.json) — Importable Postman collection
- [API Conventions](./api-conventions.md) — Detailed API design conventions
- [Error Codes](./error-codes.md) — Complete error code reference
- [Rate Limiting](./rate-limiting.md) — Rate limit configuration

---

## Admin Endpoints

> All admin endpoints require `admin` role JWT and `X-Tenant-ID` header.

### User Management (Admin)

```http
POST   /api/v1/admin/users                    # Create user (admin)
GET    /api/v1/admin/users                    # List users (paginated)
GET    /api/v1/admin/users/{id}               # Get user details
PATCH  /api/v1/admin/users/{id}               # Update user
DELETE /api/v1/admin/users/{id}               # Delete user
POST   /api/v1/admin/users/{id}/deactivate    # Deactivate user
POST   /api/v1/admin/users/{id}/activate      # Activate user
POST   /api/v1/admin/users/{id}/unlock        # Unlock account
POST   /api/v1/admin/users/{id}/reset-password # Reset password
GET    /api/v1/admin/users/{id}/sessions      # List active sessions
DELETE /api/v1/admin/users/{id}/sessions      # Revoke all sessions
GET    /api/v1/admin/users/{id}/mfa           # View MFA enrollment
DELETE /api/v1/admin/users/{id}/mfa           # Reset MFA factors
POST   /api/v1/admin/users/{id}/roles         # Assign role
DELETE /api/v1/admin/users/{id}/roles/{rid}   # Revoke role
```

### Bulk Operations

```http
POST   /api/v1/admin/users/import             # CSV import
GET    /api/v1/admin/users/export             # Export CSV/JSON
POST   /api/v1/admin/roles/{id}/assign-bulk   # Bulk role assign
```

### Organization Management

```http
POST   /api/v1/admin/orgs                     # Create organization
GET    /api/v1/admin/orgs                     # List organizations
GET    /api/v1/admin/orgs/tree                # Org tree view
PATCH  /api/v1/admin/orgs/{id}                # Update organization
DELETE /api/v1/admin/orgs/{id}                # Delete organization
POST   /api/v1/admin/users/{id}/orgs          # Assign user to org
```

### Tenant Configuration

```http
GET    /api/v1/admin/tenant                   # Get tenant settings
PATCH  /api/v1/admin/tenant/settings/password-policy  # Update password policy
PATCH  /api/v1/admin/tenant/settings/session-policy   # Update session policy
PATCH  /api/v1/admin/tenant/settings/mfa-policy       # Update MFA policy
```

### Security & Audit

```http
GET    /api/v1/admin/security/failed-logins   # Failed login dashboard
POST   /api/v1/admin/impersonate              # Start impersonation
POST   /api/v1/admin/impersonate/end          # End impersonation
GET    /api/v1/audit/events                   # Query audit events
GET    /api/v1/audit/events/export            # Export audit logs
GET    /api/v1/audit/events/stream            # SSE real-time stream
```

### Webhook Management

```http
POST   /api/v1/auth/hooks                     # Register webhook
GET    /api/v1/auth/hooks                     # List webhooks
PATCH  /api/v1/auth/hooks/{id}                # Update webhook
DELETE /api/v1/auth/hooks/{id}                # Delete webhook
POST   /api/v1/auth/hooks/{id}/test           # Send test event
GET    /api/v1/auth/hooks/{id}/deliveries     # Delivery history
```

### Webhook Registration Example

```http
POST /api/v1/auth/hooks
Authorization: Bearer <admin-jwt>
X-Tenant-ID: <tenant-uuid>
Content-Type: application/json

{
  "url": "https://your-app.example.com/webhooks/ggid",
  "secret": "your-hmac-secret",
  "events": ["user.created", "user.login", "user.deleted"],
  "active": true
}
```

**Response 201:**
```json
{
  "id": "hook-uuid",
  "url": "https://your-app.example.com/webhooks/ggid",
  "events": ["user.created", "user.login", "user.deleted"],
  "active": true,
  "created_at": "2024-01-15T10:00:00Z"
}
```

---

## Policy & ABAC Endpoints

### Check Permission

Check if a user has a specific permission.

```
POST /api/v1/policies/check
```

**Request:**
```json
{
  "user_id": "usr_abc123",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "action": "write",
  "resource": "users"
}
```

**Response 200:**
```json
{
  "allowed": true,
  "reason": "role_permission_match",
  "matched_role": "user_admin"
}
```

### List ABAC Rules

```
GET /api/v1/policies/rules?tenant_id={uuid}
```

**Response 200:**
```json
{
  "rules": [
    {
      "id": "rule_001",
      "name": "business_hours_only",
      "condition": "env.time.hour >= 9 AND env.time.hour <= 17",
      "action": "ALLOW",
      "priority": 100,
      "active": true
    }
  ],
  "total": 1
}
```

### Create ABAC Rule

```
POST /api/v1/policies/rules
```

**Request:**
```json
{
  "name": "office_ip_only",
  "description": "Admin operations only from office IP range",
  "condition": "user.role = 'admin' AND env.ip IN ['10.0.0.0/8']",
  "action": "ALLOW",
  "priority": 200
}
```

---

## Tenant Management Endpoints

> **Note**: These endpoints require super-admin scope.

### Create Tenant

```
POST /api/v1/tenants
```

**Request:**
```json
{
  "name": "Acme Corporation",
  "plan": "enterprise",
  "max_users": 10000
}
```

**Response 201:**
```json
{
  "id": "55000000-0000-0000-0000-000000000002",
  "name": "Acme Corporation",
  "plan": "enterprise",
  "active": true,
  "created_at": "2025-07-11T12:00:00Z"
}
```

### Get Tenant

```
GET /api/v1/tenants/{tenant_id}
```

### List Tenants

```
GET /api/v1/tenants
```

### Update Tenant

```
PUT /api/v1/tenants/{tenant_id}
```

**Request:**
```json
{
  "name": "Acme Corp (Updated)",
  "max_users": 50000
}
```

### Suspend Tenant

```
POST /api/v1/tenants/{tenant_id}/suspend
```

**Request:**
```json
{
  "reason": "Non-payment"
}
```

### Delete Tenant

```
DELETE /api/v1/tenants/{tenant_id}
```

**Warning**: Cascades to all tenant data. Irreversible.

---

## Session Management Endpoints

### List Active Sessions

```
GET /api/v1/sessions?user_id={user_id}
```

**Response 200:**
```json
{
  "sessions": [
    {
      "id": "sess_abc123",
      "user_id": "usr_abc123",
      "ip_address": "192.168.1.50",
      "user_agent": "Mozilla/5.0...",
      "created_at": "2025-07-11T08:00:00Z",
      "last_active": "2025-07-11T11:30:00Z",
      "expires_at": "2025-07-11T20:00:00Z"
    }
  ]
}
```

### Revoke Session

```
DELETE /api/v1/sessions/{session_id}
```

### Revoke All User Sessions

```
DELETE /api/v1/sessions?user_id={user_id}
```

**Response 200:**
```json
{
  "revoked_count": 5
}
```

---

## Notification Endpoints

### List Notifications

```
GET /api/v1/notifications?user_id={user_id}
```

**Response 200:**
```json
{
  "notifications": [
    {
      "id": "notif_001",
      "type": "security_alert",
      "title": "New device login",
      "message": "Login from new device at 192.168.1.50",
      "read": false,
      "created_at": "2025-07-11T12:00:00Z"
    }
  ],
  "unread_count": 3
}
```

### Mark Notification Read

```
PUT /api/v1/notifications/{notification_id}/read
```

### Send Test Notification

```
POST /api/v1/notifications/test
```

**Request:**
```json
{
  "user_id": "usr_abc123",
  "channel": "email",
  "template": "welcome"
}
```

---

## Webhook Management Endpoints

### List Webhook Deliveries

```
GET /api/v1/webhooks/{webhook_id}/deliveries?limit=20
```

**Response 200:**
```json
{
  "deliveries": [
    {
      "id": "dlv_001",
      "event_id": "evt_abc123",
      "event_type": "user.created",
      "status": "delivered",
      "response_code": 200,
      "latency_ms": 45,
      "attempt": 1,
      "delivered_at": "2025-07-11T12:00:00Z"
    }
  ]
}
```

### Replay Failed Delivery

```
POST /api/v1/webhooks/{webhook_id}/deliveries/{delivery_id}/replay
```

### List Dead Letter Queue

```
GET /api/v1/webhooks/{webhook_id}/failures
```

---

## System & Multi-Tenant APIs

### Check System Initialization

```
GET /api/v1/system/initialized
```

Unauthenticated. Returns whether the system has any users (first-run detection).

**Response:**
```json
{"initialized": true}
```

### Resolve Tenant by Slug

```
GET /api/v1/tenants/resolve?slug=default
```

Unauthenticated. Resolves a workspace slug to tenant ID for multi-tenant login.

**Response:**
```json
{
  "id": "00000000-0000-0000-0000-000000000001",
  "name": "Default",
  "slug": "default",
  "plan": "enterprise",
  "status": "active"
}
```

### Bootstrap System (First Run)

```
POST /api/v1/system/bootstrap
```

Unauthenticated (bootstrap-only). Creates the initial tenant and admin user. Self-disables after initialization.

**Request:**
```json
{
  "tenant_slug": "acme",
  "admin_email": "admin@acme.com"
}
```

### Multi-Tenant Login

```
POST /api/v1/auth/login
```

Login now supports an optional `tenant_slug` field as an alternative to the `X-Tenant-ID` header:

```json
{
  "username": "admin",
  "password": "Password123!",
  "tenant_slug": "default"
}
```

If both `tenant_slug` and `X-Tenant-ID` header are provided, the header takes precedence.

---

*Last updated: 2026-07-15*
