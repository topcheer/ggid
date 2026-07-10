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
