# REST API Reference

> Complete REST API for GGID. All endpoints require `X-Tenant-ID` header. Protected routes require `Authorization: Bearer <JWT>`.

---

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

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","email":"alice@test.com","password":"Secure123!"}'
```

**201:**
```json
{"id":"usr_abc","username":"alice","email":"alice@test.com","status":"active"}
```

### Login

```
POST /api/v1/auth/login
```

```bash
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","password":"Secure123!"}' | jq -r .access_token)
```

**200:**
```json
{"access_token":"eyJ...","refresh_token":"rft...","token_type":"Bearer","expires_in":900}
```

**401:** `{"error":"invalid_credentials"}`

### Refresh Token

```
POST /api/v1/auth/refresh
```
```json
{"refresh_token":"rft..."}
```

### Logout

```
POST /api/v1/auth/logout
```

### MFA Verify

```
POST /api/v1/auth/mfa/verify
```
```json
{"code":"123456"}
```

---

## Users

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/users` | `read:users` | List users (paginated) |
| `GET` | `/api/v1/users/{id}` | `read:users` | Get user |
| `POST` | `/api/v1/users` | `write:users` | Create user |
| `PUT` | `/api/v1/users/{id}` | `write:users` | Update user |
| `DELETE` | `/api/v1/users/{id}` | `delete:users` | Delete user |
| `POST` | `/api/v1/users/{id}/roles` | `write:users` | Assign role |
| `GET` | `/api/v1/users/{id}/permissions` | `read:users` | User's permissions |

### List Users

```bash
curl -s http://localhost:8080/api/v1/users?page=1&page_size=20 \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT"
```

**200:**
```json
{"users":[{"id":"usr_1","username":"alice","email":"alice@test.com","status":"active"}],"total":1,"page":1}
```

---

## Roles

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/roles` | `read:roles` | List roles |
| `POST` | `/api/v1/roles` | `write:roles` | Create role |
| `DELETE` | `/api/v1/roles/{id}` | `delete:roles` | Delete role |
| `POST` | `/api/v1/roles/{id}/permissions` | `write:roles` | Assign permissions |
| `POST` | `/api/v1/roles/{id}/parent` | `write:roles` | Set parent (inheritance) |
| `GET` | `/api/v1/permissions` | `read:roles` | List all permissions |

### Create Role

```bash
curl -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"name":"Editor","key":"editor","description":"Read+write users"}'
```

---

## Organizations

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/orgs` | `read:orgs` | List orgs |
| `POST` | `/api/v1/orgs` | `write:orgs` | Create org |
| `GET` | `/api/v1/orgs/{id}` | `read:orgs` | Get org |
| `PUT` | `/api/v1/orgs/{id}` | `write:orgs` | Update org |
| `DELETE` | `/api/v1/orgs/{id}` | `delete:orgs` | Delete org |

---

## Policies

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/policies` | `write:policies` | Create ABAC policy |
| `GET` | `/api/v1/policies` | `read:policies` | List policies |
| `DELETE` | `/api/v1/policies/{id}` | `delete:policies` | Delete policy |
| `POST` | `/api/v1/policies/check` | Any auth | Check permission (RBAC) |
| `POST` | `/api/v1/policies/evaluate` | Any auth | Evaluate with attributes (ABAC) |
| `POST` | `/api/v1/policies/dry-run` | `write:policies` | Test without affecting prod |
| `GET` | `/api/v1/policies/templates` | `read:policies` | List compliance templates |
| `POST` | `/api/v1/policies/from-template/{id}` | `write:policies` | Apply template |
| `GET` | `/api/v1/policies/export` | `read:policies` | Export policies JSON |
| `POST` | `/api/v1/policies/import` | `write:policies` | Import policies |

### Check Permission

```bash
curl -X POST http://localhost:8080/api/v1/policies/check \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"usr_abc","action":"write","resource":"users"}'
```

**200:**
```json
{"allowed":true,"reason":"role_permission_match"}
```

---

## Audit

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/audit/events` | `read:audit` | Query events (paginated) |
| `GET` | `/api/v1/audit/verify` | `read:audit` | Verify hash chain |

### Query Events

```bash
curl -s "http://localhost:8080/api/v1/audit/events?limit=10&action=login" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT"
```

---

## OAuth 2.1 / OIDC

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/oauth/authorize` | Session | Authorization code flow |
| `POST` | `/oauth/token` | Client creds | Token endpoint (code→token, refresh, exchange) |
| `GET` | `/oauth/userinfo` | Bearer JWT | OIDC UserInfo |
| `GET` | `/.well-known/openid-configuration` | None | OIDC Discovery |
| `GET` | `/.well-known/jwks.json` | None | JWKS (public keys) |
| `POST` | `/oauth/introspect` | Client creds | Token introspection |
| `POST` | `/oauth/revoke` | Client creds | Token revocation |

---

## SCIM 2.0

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/scim/v2/Users` | Bearer JWT | List/filter users |
| `POST` | `/scim/v2/Users` | Bearer JWT | Create user |
| `GET` | `/scim/v2/Users/{id}` | Bearer JWT | Get user |
| `PUT` | `/scim/v2/Users/{id}` | Bearer JWT | Replace user |
| `PATCH` | `/scim/v2/Users/{id}` | Bearer JWT | Patch user |
| `DELETE` | `/scim/v2/Users/{id}` | Bearer JWT | Delete user |
| `GET/POST` | `/scim/v2/Groups[/{id}]` | Bearer JWT | Group CRUD |
| `POST` | `/scim/v2/Bulk` | Bearer JWT | Bulk operations |

---

## Admin

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/admin/stats` | `admin` role | System stats |
| `GET` | `/api/v1/admin/routes` | `admin` role | Route config |
| `POST` | `/api/v1/admin/routes/{prefix}/toggle` | `admin` role | Toggle route |

---

## AI Agent Identity

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/agents/register` | Bearer JWT | Register a new AI agent |
| `GET` | `/api/v1/agents` | Bearer JWT | List agents by tenant |
| `POST` | `/api/v1/agents/token` | Bearer JWT | Exchange user token for agent token (RFC 8693) |
| `POST` | `/api/v1/agents/verify` | None | Verify an agent token |

See [AI Agent Identity Guide](../guides/ai-agent-identity.md) for delegation chains, MCP auth, and JWT claims.

---

## Health

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/healthz` | None | Health check |
| `GET` | `/docs` | None | Swagger UI |

---

## Conventions

- **Pagination:** `?page=1&page_size=20` → `{page, page_size, total}`
- **Tenant:** All requests need `X-Tenant-ID` header
- **Auth:** `Authorization: Bearer <jwt>` (except health, discovery, JWKS)
- **Content-Type:** `application/json` (SCIM: `application/scim+json`)
- **Errors:** `{"error":"code","message":"detail"}`

---

*See: [Admin API](admin-api.md) | [SCIM API](scim-api.md) | [Error Codes](error-codes.md)*

*Last updated: 2025-07-11*
