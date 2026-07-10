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

### Begin Registration
```http
POST /api/v1/auth/webauthn/register/begin
Authorization: Bearer <JWT>
X-Tenant-ID: <tenant-uuid>
```
**Response 200:** `{ "challenge": "...", "rp": {...}, "user": {...} }`

### Finish Registration
```http
POST /api/v1/auth/webauthn/register/finish
Authorization: Bearer <JWT>
Content-Type: application/json
X-Tenant-ID: <tenant-uuid>

{
  "id": "...",
  "rawId": "...",
  "response": { "attestationObject": "...", "clientDataJSON": "..." }
}
```
**Response 200:** `{ "credential_id": "...", "registered": true }`

### Begin Authentication
```http
POST /api/v1/auth/webauthn/login/begin
X-Tenant-ID: <tenant-uuid>

{ "username": "user@example.com" }
```
**Response 200:** `{ "challenge": "...", "allowCredentials": [...] }`

### Finish Authentication
```http
POST /api/v1/auth/webauthn/login/finish
X-Tenant-ID: <tenant-uuid>

{
  "id": "...",
  "rawId": "...",
  "response": { "authenticatorData": "...", "signature": "...", "clientDataJSON": "..." }
}
```
**Response 200:** `{ "access_token": "...", "refresh_token": "..." }`

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
