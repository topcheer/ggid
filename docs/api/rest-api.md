# REST API Reference

Complete REST API reference for GGID. All endpoints require `X-Tenant-ID` header. Protected routes require `Authorization: Bearer <JWT>`.

**Base URL**: `https://api.ggid.example.com`

## Authentication

### Register

```
POST /api/v1/auth/register
```

| Field | Required | Description |
|-------|----------|-------------|
| `username` | Yes | Unique username |
| `email` | Yes | User email |
| `password` | Yes | Min 8 chars |

**Request**:
```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"alice","email":"alice@example.com","password":"SecurePass1!"}'
```

**Response** (201):
```json
{ "id": "uuid", "username": "alice", "email": "alice@example.com" }
```

**Errors**: 409 Conflict (duplicate username/email)

### Login

```
POST /api/v1/auth/login
```

**Request**:
```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"alice","password":"SecurePass1!"}'
```

**Response** (200):
```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "eyJhbG...",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

**MFA Response** (200, `mfa_required: true`):
```json
{
  "mfa_required": true,
  "mfa_token": "mfa_temp_token_xxx"
}
```

**Errors**: 401 Unauthorized (wrong password), 429 Too Many Requests (rate limited)

### Refresh Token

```
POST /api/v1/auth/refresh
```

**Request**:
```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"refresh_token":"eyJhbG..."}'
```

**Response** (200): Same as login response.

### Logout

```
POST /api/v1/auth/logout
```

Revokes the current session via `jti` blacklist.

```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Verify MFA

```
POST /api/v1/auth/mfa/verify
```

| Field | Required | Description |
|-------|----------|-------------|
| `mfa_token` | Yes | Temporary MFA token from login |
| `code` | Yes | 6-digit TOTP code |

```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/mfa/verify \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"mfa_token":"mfa_temp_xxx","code":"123456"}'
```

### Impersonate User

```
POST /api/v1/auth/impersonate
```

Requires `admin` scope. Issues a JWT impersonating the target user.

```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/impersonate \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"user_id":"target-uuid"}'
```

## Users

### List Users

```
GET /api/v1/users
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | int | Page number (default 1) |
| `page_size` | int | Per page (max 100, default 20) |
| `search` | string | Search by username/email |
| `status` | string | `active` / `locked` / `suspended` |
| `role` | string | Filter by role key |

```bash
curl https://api.ggid.example.com/api/v1/users?page=1\&page_size=20 \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response** (200):
```json
{
  "items": [
    { "id": "uuid", "username": "alice", "email": "alice@example.com", "status": "active" }
  ],
  "page": 1,
  "page_size": 20,
  "total": 142
}
```

### Get User

```
GET /api/v1/users/{id}
```

```bash
curl https://api.ggid.example.com/api/v1/users/$USER_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Get Current User

```
GET /api/v1/users/me
```

Returns the authenticated user's profile.

### Create User

```
POST /api/v1/users
```

Requires `users:write` scope.

```bash
curl -X POST https://api.ggid.example.com/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"bob","email":"bob@example.com","password":"SecurePass1!","name":"Bob Smith"}'
```

### Update User

```
PUT /api/v1/users/{id}
```

| Field | Required | Description |
|-------|----------|-------------|
| `email` | No | New email |
| `phone` | No | Phone number |
| `name` | No | Display name |

### Delete User

```
DELETE /api/v1/users/{id}
```

Soft-deletes the user (marks as inactive).

### Lock User

```
POST /api/v1/users/{id}/lock
```

### Unlock User

```
POST /api/v1/users/{id}/unlock
```

### Search Users

```
GET /api/v1/users?search={query}
```

### Import Users (Bulk)

```
POST /api/v1/users/import
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/users/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"users":[{"username":"u1","email":"u1@example.com","password":"Pass1!"}]}'
```

### Export Users

```
GET /api/v1/users/export?format=csv
```

### Get User Sessions

```
GET /api/v1/users/{id}/sessions
```

### Revoke User Sessions

```
DELETE /api/v1/users/{id}/sessions
```

## Roles

### List Roles

```
GET /api/v1/roles
```

### Create Role

```
POST /api/v1/roles
```

| Field | Required | Description |
|-------|----------|-------------|
| `key` | Yes | Unique role key (e.g., `admin`) |
| `name` | Yes | Display name |
| `description` | No | Role description |

```bash
curl -X POST https://api.ggid.example.com/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"key":"developer","name":"Developer"}'
```

**Errors**: 500 (empty key causes UNIQUE constraint violation)

### Get Role

```
GET /api/v1/roles/{id}
```

### Update Role

```
PUT /api/v1/roles/{id}
```

### Delete Role

```
DELETE /api/v1/roles/{id}
```

### Assign Role

```
POST /api/v1/users/{user_id}/roles
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/users/$USER_ID/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"role_id":"role-uuid"}'
```

### Revoke Role

```
DELETE /api/v1/users/{user_id}/roles/{role_id}
```

### Get User Roles

```
GET /api/v1/users/{user_id}/roles
```

### List Permissions

```
GET /api/v1/permissions
```

## Policies

### Check Permission

```
POST /api/v1/policies/check
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies/check \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"user_id":"uuid","resource":"document:report","action":"read"}'
```

**Response**:
```json
{ "allowed": true, "reason": "role:developer permits document:read" }
```

### Dry-Run Policy

```
POST /api/v1/policies/dry-run
```

Evaluates a policy without saving it.

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies/dry-run \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"rules":[{"effect":"allow","resource":"api:*","action":"*"}]}'
```

### Create Policy

```
POST /api/v1/policies
```

### List Policies

```
GET /api/v1/policies
```

### Delete Policy

```
DELETE /api/v1/policies/{id}
```

## Audit

### Query Events

```
GET /api/v1/audit/events
```

See [Audit API Reference](audit-api.md) for full details.

### Stream Events (SSE)

```
GET /api/v1/audit/events/stream
```

### Export Events

```
GET /api/v1/audit/export?format=csv|json
```

### Verify Integrity

```
GET /api/v1/audit/integrity/verify
```

### Compliance Report

```
GET /api/v1/audit/compliance/report?type=soc2|hipaa|gdpr
```

## Organizations

### List Organizations

```
GET /api/v1/organizations
```

### Create Organization

```
POST /api/v1/organizations
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/organizations \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"name":"Engineering","description":"Engineering team"}'
```

### Get Organization

```
GET /api/v1/organizations/{id}
```

### Delete Organization

```
DELETE /api/v1/organizations/{id}
```

### List Departments

```
GET /api/v1/organizations/{id}/departments
```

### Create Department

```
POST /api/v1/organizations/{id}/departments
```

### Add Member

```
POST /api/v1/organizations/{id}/members
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/organizations/$ORG_ID/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"user_id":"uuid","role":"developer"}'
```

### Remove Member

```
DELETE /api/v1/organizations/{id}/members/{user_id}
```

## AI Agents

### Register Agent

```
POST /api/v1/agents
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "my-service-agent",
    "type": "service",
    "scopes": ["users:read", "audit:read"],
    "max_delegation_depth": 3
  }'
```

### List Agents

```
GET /api/v1/agents
```

### Exchange Agent Token

```
POST /api/v1/agents/{id}/token
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents/$AGENT_ID/token \
  -H "Authorization: Bearer $SUBJECT_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"scopes":["users:read"]}'
```

### Verify Agent Token

```
POST /api/v1/agents/verify
```

### Suspend Agent

```
POST /api/v1/agents/{id}/suspend
```

## OAuth

### Authorization Endpoint

```
GET /oauth/authorize
```

| Parameter | Required | Description |
|-----------|----------|-------------|
| `response_type` | Yes | `code` |
| `client_id` | Yes | OAuth client ID |
| `redirect_uri` | Yes | Must match registered URI exactly |
| `scope` | Yes | Requested scopes |
| `state` | Yes | CSRF protection |
| `code_challenge` | Yes | PKCE challenge |
| `code_challenge_method` | Yes | `S256` |

### Token Endpoint

```
POST /oauth/token
```

```bash
curl -X POST https://api.ggid.example.com/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=xxx&redirect_uri=xxx&client_id=xxx&code_verifier=xxx"
```

### JWKS

```
GET /.well-known/jwks.json
```

### Discovery

```
GET /.well-known/openid-configuration
```

## Sessions

### List Sessions

```
GET /api/v1/sessions
```

### Revoke Session

```
DELETE /api/v1/sessions/{id}
```

### Revoke All User Sessions

```
DELETE /api/v1/sessions?user_id={user_id}
```

## Access Requests (IGA)

### Submit Access Request

```
POST /api/v1/access-requests
```

### List Access Requests

```
GET /api/v1/access-requests?status=pending
```

### Approve Access Request

```
POST /api/v1/access-requests/{id}/approve
```

### Deny Access Request

```
POST /api/v1/access-requests/{id}/deny
```

## Webhooks

### Register Webhook

```
POST /api/v1/webhooks
```

### List Webhooks

```
GET /api/v1/webhooks
```

### Delete Webhook

```
DELETE /api/v1/webhooks/{id}
```

## SCIM 2.0

### Get User (SCIM)

```
GET /scim/v2/Users/{id}
```

### Create User (SCIM)

```
POST /scim/v2/Users
```

### Patch User (SCIM)

```
PATCH /scim/v2/Users/{id}
```

### List Users (SCIM)

```
GET /scim/v2/Users?filter=userName eq "alice"
```

## Health & Discovery

### Health Check

```
GET /healthz
```

**Response**: `{"status":"ok"}`

## Error Format

All errors return a consistent JSON structure:

```json
{
  "error": {
    "code": "not_found",
    "message": "User not found",
    "details": { "user_id": "uuid" }
  }
}
```

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_argument` | Bad request |
| 401 | `unauthorized` | Missing/invalid token |
| 403 | `forbidden` | Insufficient scope |
| 404 | `not_found` | Resource not found |
| 409 | `already_exists` | Duplicate resource |
| 429 | `rate_limit_exceeded` | Too many requests |
| 500 | `internal` | Server error |

## See Also

- [Audit API Reference](audit-api.md)
- [Go SDK Guide](../guides/go-sdk-guide.md)
- [Node.js SDK Guide](../guides/node-sdk-guide.md)
- [Java SDK Guide](../guides/java-sdk-guide.md)
- [Error Reference](error-reference.md)
