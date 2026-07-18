# OpenAPI Coverage Report

**Date:** July 2026

## Current State

| Metric | Value |
|--------|-------|
| In-memory spec paths (`openapi_spec.go`) | 704 |
| `deploy/openapi.yaml` paths | 823 |
| Total unique API endpoints | ~864 |
| Combined coverage | ~830+ paths (95%+) |
| Request body schemas | 0 |
| Response schemas | 0 |
| Component schemas | Security schemes only (bearer, apiKey, DPoP, mTLS) |

## Schema Gap Analysis

The OpenAPI spec has excellent **path coverage** (95%+) but **zero request/response body schemas**. This means:

- ✅ Swagger UI shows all endpoints with descriptions
- ✅ Security schemes are documented (Bearer, API Key, DPoP, mTLS)
- ❌ No request body examples for POST/PUT endpoints
- ❌ No response models shown in Swagger UI
- ❌ No auto-generated SDK types from the spec

## Priority Endpoints for Schema Addition (Top 50)

### Auth (10)
- `POST /api/v1/auth/login` — LoginRequest {username, password}
- `POST /api/v1/auth/register` — RegisterRequest {email, password, name, username}
- `POST /api/v1/auth/refresh` — RefreshRequest {refresh_token}
- `POST /api/v1/auth/logout` — empty
- `GET /api/v1/auth/profile` — UserProfile response
- `POST /api/v1/auth/password/change` — PasswordChangeRequest
- `POST /api/v1/auth/password/forgot` — ForgotRequest {email}
- `POST /api/v1/auth/password/reset` — ResetRequest {token, password}
- `POST /api/v1/auth/mfa/enroll` — MFAEnrollRequest
- `POST /api/v1/auth/mfa/verify` — MFAVerifyRequest {code}

### Identity (10)
- `GET /api/v1/users` — UserList response (paginated)
- `POST /api/v1/users` — CreateUserRequest
- `GET /api/v1/users/{id}` — UserProfile response
- `PUT /api/v1/users/{id}` — UpdateUserRequest
- `DELETE /api/v1/users/{id}` — empty
- `GET /api/v1/roles` — RoleList response
- `POST /api/v1/roles` — CreateRoleRequest {key, name, description}
- `POST /api/v1/roles/assign` — AssignRoleRequest {user_id, role_id}
- `GET /api/v1/policies` — PolicyList response
- `POST /api/v1/policies/check` — CheckRequest {user_id, resource, action}

### OAuth (8)
- `GET /api/v1/oauth/clients` — ClientList response
- `POST /api/v1/oauth/clients` — CreateClientRequest
- `POST /api/v1/oauth/token` — TokenRequest (form-urlencoded)
- `GET /api/v1/oauth/authorize` — AuthorizeRequest (query params)
- `POST /api/v1/oauth/introspect` — IntrospectRequest
- `GET /api/v1/oauth/userinfo` — UserInfo response
- `POST /api/v1/oauth/consent` — ConsentRequest
- `GET /api/v1/oauth/.well-known/openid-configuration` — OIDCDiscovery response

### Audit (5)
- `GET /api/v1/audit/events` — AuditEventList (paginated)
- `GET /api/v1/audit/integrity` — IntegrityResponse
- `GET /api/v1/audit/export` — binary (CSV/JSON)
- `GET /api/v1/audit/threat-intel/sources` — SourceList
- `POST /api/v1/audit/ccm/scan` — empty

### Sessions & Security (7)
- `GET /api/v1/auth/sessions` — SessionList
- `DELETE /api/v1/auth/sessions/{id}` — empty
- `POST /api/v1/auth/conditional-access/policies` — CreateCAPRequest
- `POST /api/v1/auth/break-glass/activate` — BreakGlassRequest
- `GET /api/v1/identity/privileged-operations` — PrivilegedOpList
- `POST /api/v1/webhooks` — CreateWebhookRequest
- `POST /api/v1/auth/cae/run` — CAERunRequest

### Other (10)
- `POST /api/v1/system/quickstart` — QuickstartRequest
- `POST /api/v1/system/bootstrap` — BootstrapRequest
- `GET /api/v1/system/status` — SystemStatus
- `POST /api/v1/auth/webauthn/register/begin` — WebAuthnBeginRequest
- `POST /api/v1/auth/tap` — TAPRequest
- `GET /api/v1/dashboard/stats` — DashboardStats
- `GET /healthz` — HealthResponse
- `GET /metrics` — Prometheus text
- `GET /docs` — SwaggerUI HTML
- `GET /swagger.json` — OpenAPI JSON

## Recommendations

1. **P0**: Add request body schemas to top 20 endpoints (auth + identity core)
2. **P1**: Add response schemas for list endpoints (pagination shape)
3. **P2**: Add error response schema (unified `{error: {code, message}}`)
4. **P3**: Enable `openapi-generator` for SDK auto-generation
5. **P3**: Add examples for complex request bodies
