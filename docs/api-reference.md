# GGID API Reference

> Complete API reference for all GGID services.
> Base URL: `https://ggid.iot2.win`
> All requests require `Authorization: Bearer <token>` and `X-Tenant-ID: <tenant-uuid>` headers unless noted.

---

## Table of Contents

1. [Auth Service](#1-auth-service)
2. [Identity Service](#2-identity-service)
3. [OAuth Service](#3-oauth-service)
4. [Policy Service](#4-policy-service)
5. [Audit Service](#5-audit-service)
6. [Gateway](#6-gateway)

---

## 1. Auth Service

> Port: 9001 (HTTP) / 50052 (gRPC)
> Handles authentication, MFA, sessions, passkeys, biometrics.

### Authentication

#### POST `/api/v1/auth/login`
Authenticate a user with username/password.

```bash
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@123456","tenant_id":"00000000-0000-0000-0000-000000000001"}'
```

**Response:** `{"access_token":"...","refresh_token":"...","expires_in":3600}`

#### POST `/api/v1/auth/register`
Register a new user.

```bash
curl -k -X POST "https://ggid.iot2.win/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"newuser","email":"newuser@example.com","password":"SecurePass123!"}'
```

#### POST `/api/v1/auth/refresh`
Refresh an access token.

```bash
curl -k -X POST "https://ggid.iot2.win/api/v1/auth/refresh" \
  -H "Authorization: Bearer <token>" \
  -d '{"refresh_token":"..."}'
```

#### POST `/api/v1/auth/logout`
Invalidate the current session.

#### POST `/api/v1/auth/password/change`
Change password for the authenticated user.

```bash
curl -k -X POST "https://ggid.iot2.win/api/v1/auth/password/change" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"current_password":"OldPass","new_password":"NewPass123!"}'
```

#### POST `/api/v1/auth/password/forgot`
Request a password reset link. No auth required.

#### POST `/api/v1/auth/password/reset`
Reset password using a reset token.

### MFA

#### POST `/api/v1/auth/mfa/setup`
Initialize TOTP MFA enrollment. Returns a QR code secret.

#### POST `/api/v1/auth/mfa/verify`
Verify a TOTP code to complete MFA enrollment.

```bash
curl -k -X POST "https://ggid.iot2.win/api/v1/auth/mfa/verify" \
  -H "Authorization: Bearer <token>" \
  -d '{"code":"123456","secret":"BASE32SECRET"}'
```

#### POST `/api/v1/auth/mfa/login`
Submit MFA code during login flow.

#### POST `/api/v1/auth/mfa/disable`
Disable MFA for the authenticated user.

#### POST `/api/v1/auth/mfa/backup-codes/generate`
Generate one-time backup codes.

#### POST `/api/v1/auth/mfa/backup-codes/verify`
Verify a backup code.

### Sessions

#### GET `/api/v1/auth/sessions`
List active sessions for the authenticated user.

#### DELETE `/api/v1/auth/sessions`
Revoke all sessions except the current one.

### DLP

#### GET `/api/v1/dlp/policies`
List DLP policies.

#### POST `/api/v1/dlp/scan`
Scan content for PII.

```bash
curl -k -X POST "https://ggid.iot2.win/api/v1/dlp/scan" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"content":"My SSN is 123-45-6789"}'
```

---

## 2. Identity Service

> Port: 8081 (HTTP) / 50051 (gRPC)
> Handles users, groups, organizations, SCIM, federation, consent.

### Users

#### GET `/api/v1/users`
List all users (paginated).

```bash
curl -k "https://ggid.iot2.win/api/v1/users?page=1&page_size=20" \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

#### POST `/api/v1/users`
Create a new user.

#### GET `/api/v1/users/:id`
Get a user by ID.

#### PUT `/api/v1/users/:id`
Update a user.

#### DELETE `/api/v1/users/:id`
Delete a user.

#### POST `/api/v1/users/import`
Bulk import users via CSV.

#### GET `/api/v1/users/export`
Export users as CSV.

#### GET `/api/v1/users/search`
Search users by name, email, or attributes.

### SCIM 2.0

#### GET `/api/v1/scim/Groups`
List groups (SCIM 2.0 compliant).

#### POST `/api/v1/scim/Groups`
Create a group.

#### GET `/api/v1/scim/Groups/:id`
Get a group by ID.

#### PATCH `/api/v1/scim/Groups/:id`
Update group membership.

### Organizations

#### GET `/api/v1/organizations`
List organizational hierarchy.

#### POST `/api/v1/organizations`
Create an organization.

### Consent Management

#### GET `/api/v1/identity/consent/registry`
List consent records.

```bash
curl -k "https://ggid.iot2.win/api/v1/identity/consent/registry?status=active" \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

#### POST `/api/v1/identity/consent/registry`
Grant consent.

#### DELETE `/api/v1/identity/consent/registry`
Withdraw consent.

### Federation

#### GET `/api/v1/identity/federation/entities`
List federation entities.

#### POST `/api/v1/identity/federation/entities`
Create a federation entity (SAML/OIDC).

#### DELETE `/api/v1/identity/federation/entities?id=:id`
Delete a federation entity.

### Threat Intelligence

#### GET `/api/v1/audit/threat-intel/sources`
List configured intel sources.

#### GET `/api/v1/audit/threat-intel/indicators`
Query threat indicators.

#### POST `/api/v1/audit/threat-intel/check`
Real-time threat check.

#### GET `/api/v1/audit/threat-intel/stats`
Threat intel statistics.

---

## 3. OAuth Service

> Port: 9005
> Handles OAuth 2.1, OIDC, SAML, DPoP, consent, device flow.

### Authorization

#### GET `/api/v1/oauth/authorize`
Start the authorization code flow (with PKCE).

```
GET /api/v1/oauth/authorize?response_type=code&client_id=...&redirect_uri=...&code_challenge=...&code_challenge_method=S256&scope=openid+profile
```

#### POST `/api/v1/oauth/token`
Exchange authorization code or refresh token for an access token.

```bash
curl -k -X POST "https://ggid.iot2.win/api/v1/oauth/token" \
  -d 'grant_type=authorization_code&code=...&redirect_uri=...&code_verifier=...&client_id=...'
```

**DPoP:** Include `DPoP` header with a signed JWT proof to bind the token to the client's key.

#### GET `/api/v1/oauth/userinfo`
Get user info from an access token (OIDC UserInfo endpoint).

#### POST `/api/v1/oauth/revoke`
Revoke a token.

#### POST `/api/v1/oauth/introspect`
Introspect a token (RFC 7662).

### Client Registration

#### GET `/api/v1/oauth/clients`
List OAuth clients.

#### POST `/api/v1/oauth/clients`
Register a new OAuth client.

```bash
curl -k -X POST "https://ggid.iot2.win/api/v1/oauth/clients" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"client_name":"My App","redirect_uris":["https://app.example.com/callback"],"grant_types":["authorization_code"],"scope":"openid profile"}'
```

#### GET `/api/v1/oauth/clients/:id`
Get a client by ID.

#### PUT `/api/v1/oauth/clients/:id`
Update a client.

#### DELETE `/api/v1/oauth/clients/:id`
Delete a client.

### Device Flow

#### POST `/api/v1/oauth/device`
Initiate device authorization flow (RFC 8628).

### Consent

#### GET/POST `/oauth/consent`
Get or submit user consent for an authorization request.

### SAML 2.0

#### GET `/saml/metadata`
Service Provider metadata.

#### POST `/saml/acs`
Assertion Consumer Service endpoint.

#### GET `/saml/sso`
Initiate SP-initiated SSO.

#### GET `/saml/slo`
Single Logout endpoint.

#### GET `/saml/idp/metadata`
IdP metadata (GGID as Identity Provider).

### DPoP (RFC 9449)

#### GET `/api/v1/oauth/dpop/config`
Get DPoP configuration.

#### POST `/api/v1/oauth/dpop/verify`
Verify a DPoP proof.

#### POST `/api/v1/oauth/token/dpop-bind`
Bind an access token to a DPoP key.

---

## 4. Policy Service

> Port: 8070 (HTTP) / 9070 (gRPC)
> Handles roles, permissions, policies, ABAC, ReBAC, risk scoring.

### Roles & Permissions

#### GET `/api/v1/roles`
List all roles.

#### POST `/api/v1/roles`
Create a role.

#### GET `/api/v1/roles/:id`
Get a role by ID.

#### PUT `/api/v1/roles/:id`
Update a role.

#### DELETE `/api/v1/roles/:id`
Delete a role.

#### GET `/api/v1/permissions`
List all permissions.

### Policies

#### GET `/api/v1/policies`
List policies.

#### POST `/api/v1/policies`
Create a policy.

#### POST `/api/v1/policies/check`
Check if a subject has permission for an action.

```bash
curl -k -X POST "https://ggid.iot2.win/api/v1/policies/check" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"subject":"user:admin","action":"read","resource":"document:123"}'
```

#### POST `/api/v1/policies/evaluate`
Evaluate a policy with full context (ABAC).

### Risk Scoring

#### GET `/api/v1/policy/risk-score/summary`
Get organization-wide risk summary.

#### GET `/api/v1/policy/risk-score/users`
List users with risk scores.

#### POST `/api/v1/policy/risk-score/recalculate`
Recalculate risk for a specific user.

### JIT Elevation

#### POST `/api/v1/policies/jit/request`
Request just-in-time privilege elevation.

---

## 5. Audit Service

> Port: 8072 (HTTP) / 9072 (gRPC)
> Handles audit events, ITDR, compliance, reports.

### Events

#### GET `/api/v1/audit/events`
List audit events (paginated, filterable).

```bash
curl -k "https://ggid.iot2.win/api/v1/audit/events?page_size=50&action=login" \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

#### GET `/api/v1/audit/events/:id`
Get a specific event.

#### GET `/api/v1/audit/stats`
Get audit statistics (event counts, trends).

#### GET `/api/v1/audit/export`
Export audit events as CSV/JSON.

#### GET `/api/v1/audit/stream`
Server-Sent Events stream of real-time audit events.

#### GET `/api/v1/audit/ws`
WebSocket endpoint for real-time push.

### ITDR (Identity Threat Detection)

#### GET `/api/v1/audit/itdr/detections`
List ITDR detections.

#### GET `/api/v1/audit/itdr/stats`
ITDR statistics.

#### GET/POST `/api/v1/audit/itdr/rules`
Manage ITDR detection rules.

### Compliance

#### GET `/api/v1/audit/compliance-dashboard`
Get compliance framework summaries.

#### GET `/api/v1/audit/compliance-report`
Generate compliance report.

### Anomaly Detection

#### GET/POST `/api/v1/audit/rules`
Manage anomaly detection rules.

#### POST `/api/v1/audit/correlate`
Correlate events across sources.

### Integrity

#### GET `/api/v1/audit/verify-integrity`
Verify audit log integrity (hash chain).

### Threat Intelligence

#### GET `/api/v1/audit/threat-intel/sources`
List intel sources.

#### GET `/api/v1/audit/threat-intel/indicators`
Query indicators.

#### POST `/api/v1/audit/threat-intel/check`
Real-time IOC check.

#### GET `/api/v1/audit/threat-intel/stats`
Coverage and indicator statistics.

---

## 6. Gateway

> Port: 8080
> API gateway with reverse proxy, middleware, rate limiting.

### Health

#### GET `/healthz`
Health check endpoint.

```bash
curl -k "https://ggid.iot2.win/healthz"
```

#### GET `/readyz`
Readiness check.

### OIDC Discovery

#### GET `/.well-known/openid-configuration`
OpenID Connect discovery document.

```bash
curl -k "https://ggid.iot2.win/.well-known/openid-configuration" | python3 -m json.tool
```

#### GET `/oauth/jwks`
JSON Web Key Set for token verification.
