# GGID API Reference Guide

> Complete REST API endpoint list with examples. For developer quick lookup.

---

## Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/auth/register` | None | Register new user |
| POST | `/api/v1/auth/login` | None | Login → JWT + refresh |
| POST | `/api/v1/auth/refresh` | Refresh | Get new access token |
| POST | `/api/v1/auth/logout` | Bearer | Revoke session |
| POST | `/api/v1/auth/mfa/verify` | MFA token | Verify TOTP/WebAuthn |
| POST | `/api/v1/auth/mfa/totp/setup` | Bearer | Generate TOTP secret |
| POST | `/api/v1/auth/webauthn/register/begin` | Bearer | Start passkey registration |
| POST | `/api/v1/auth/webauthn/register/finish` | Bearer | Complete passkey registration |
| POST | `/api/v1/auth/webauthn/login/begin` | None | Start passkey login |
| POST | `/api/v1/auth/webauthn/login/finish` | None | Complete passkey login |

### Example: Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","password":"Secure123!"}'
```

---

## Users

| Method | Endpoint | Scope |
|--------|----------|-------|
| GET | `/api/v1/users` | `read:users` |
| GET | `/api/v1/users/{id}` | `read:users` |
| POST | `/api/v1/users` | `write:users` |
| PUT | `/api/v1/users/{id}` | `write:users` |
| DELETE | `/api/v1/users/{id}` | `delete:users` |
| POST | `/api/v1/users/{id}/roles` | `write:users` |
| GET | `/api/v1/users/{id}/permissions` | `read:users` |

---

## Roles & Policies

| Method | Endpoint | Scope |
|--------|----------|-------|
| GET | `/api/v1/roles` | `read:roles` |
| POST | `/api/v1/roles` | `write:roles` |
| POST | `/api/v1/roles/{id}/permissions` | `write:roles` |
| POST | `/api/v1/policies` | `write:policies` |
| POST | `/api/v1/policies/check` | Any auth |
| POST | `/api/v1/policies/evaluate` | Any auth |
| GET | `/api/v1/policies/templates` | `read:policies` |

---

## Organizations

| Method | Endpoint | Scope |
|--------|----------|-------|
| GET | `/api/v1/orgs` | `read:orgs` |
| POST | `/api/v1/orgs` | `write:orgs` |
| DELETE | `/api/v1/orgs/{id}` | `delete:orgs` |

---

## Audit

| Method | Endpoint | Scope |
|--------|----------|-------|
| GET | `/api/v1/audit/events` | `read:audit` |
| GET | `/api/v1/audit/verify` | `read:audit` |

---

## OAuth/OIDC

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/oauth/authorize` | Authorization code flow |
| POST | `/oauth/token` | Token exchange |
| GET | `/oauth/userinfo` | OIDC UserInfo |
| GET | `/.well-known/openid-configuration` | Discovery |
| GET | `/.well-known/jwks.json` | Public keys |
| POST | `/oauth/introspect` | Token introspection |

---

## AI Agents

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/agents/register` | Register agent |
| GET | `/api/v1/agents` | List agents |
| POST | `/api/v1/agents/token` | Exchange user→agent token |
| POST | `/api/v1/agents/verify` | Verify agent token |

---

## IGA Workflows

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/access-requests` | Create access request |
| GET | `/api/v1/access-requests` | List requests |
| POST | `/api/v1/access-requests/{id}/approve` | Approve |
| POST | `/api/v1/access-requests/{id}/deny` | Deny |

---

## SCIM 2.0

| Method | Endpoint |
|--------|----------|
| GET/POST | `/scim/v2/Users` |
| GET/PUT/PATCH/DELETE | `/scim/v2/Users/{id}` |
| GET/POST | `/scim/v2/Groups` |
| POST | `/scim/v2/Bulk` |

---

*See: [REST API Reference](../api/rest-api.md) | [Admin API](../api/admin-api.md) | [OpenAPI Spec](../openapi.yaml)*

*Last updated: 2025-07-11*
