# OpenAPI Spec Completeness Audit

**Date:** 2025-07-11
**Auditor:** Research Analyst
**Spec File:** `docs/openapi.yaml` (2,397 lines, OpenAPI 3.1.0)
**Scope:** All 7 microservices (gateway, auth, oauth, identity, policy, org, audit)

---

## Executive Summary

The GGID OpenAPI specification defines **~111 endpoint operations** across 9 tags.
The actual source code registers **~160+ endpoint operations** across 7 services.
The spec covers approximately **69%** of implemented endpoints, with **49+ endpoints
missing from the spec** and **1 endpoint in the spec not implemented in code**.
The spec is partially usable for SDK generation but would produce incomplete SDKs
missing OAuth/OIDC, SAML, SCIM Groups, and several operational endpoints.

---

## 1. OpenAPI Spec Inventory

### 1.1 All Paths and Methods Defined

The spec at `docs/openapi.yaml` defines the following paths (counted by HTTP method):

| # | Path | Method | Tag |
|---|------|--------|-----|
| 1 | `/healthz` | GET | Health |
| 2 | `/healthz/live` | GET | Health |
| 3 | `/healthz/ready` | GET | Health |
| 4 | `/api/v1/auth/register` | POST | Auth |
| 5 | `/api/v1/auth/login` | POST | Auth |
| 6 | `/api/v1/auth/logout` | POST | Auth |
| 7 | `/api/v1/auth/refresh` | POST | Auth |
| 8 | `/api/v1/auth/password/forgot` | POST | Auth |
| 9 | `/api/v1/auth/password/reset` | POST | Auth |
| 10 | `/api/v1/auth/password/change` | POST | Auth |
| 11 | `/api/v1/auth/mfa/setup` | POST | Auth |
| 12 | `/api/v1/auth/mfa/verify` | POST | Auth |
| 13 | `/api/v1/auth/mfa/disable` | POST | Auth |
| 14 | `/api/v1/auth/mfa/login` | POST | Auth |
| 15 | `/api/v1/auth/password/policy` | GET | Auth |
| 16 | `/api/v1/auth/magic-link` | POST | Auth |
| 17 | `/api/v1/auth/magic-link/verify` | POST | Auth |
| 18 | `/api/v1/auth/email/verify` | POST | Auth |
| 19 | `/api/v1/auth/email/resend` | POST | Auth |
| 20 | `/api/v1/auth/phone/send` | POST | Auth |
| 21 | `/api/v1/auth/phone/verify` | POST | Auth |
| 22 | `/api/v1/auth/stepup/challenge` | POST | Auth |
| 23 | `/api/v1/auth/stepup/verify` | POST | Auth |
| 24 | `/api/v1/auth/sessions` | GET | Auth |
| 25 | `/api/v1/auth/sessions` | DELETE | Auth |
| 26 | `/api/v1/auth/logout-all` | POST | Auth |
| 27 | `/api/v1/auth/hooks` | GET | Auth |
| 28 | `/api/v1/auth/hooks` | POST | Auth |
| 29 | `/api/v1/auth/passwordless/register` | POST | Auth |
| 30 | `/api/v1/auth/mfa/webauthn/begin` | POST | Auth |
| 31 | `/api/v1/auth/mfa/webauthn/finish` | POST | Auth |
| 32 | `/api/v1/auth/webauthn/register/begin` | POST | Auth |
| 33 | `/api/v1/auth/webauthn/register/finish` | POST | Auth |
| 34 | `/api/v1/auth/webauthn/login/begin` | POST | Auth |
| 35 | `/api/v1/auth/webauthn/login/finish` | POST | Auth |
| 36 | `/api/v1/auth/step-up-check` | GET | Auth |
| 37 | `/api/v1/auth/step-up` | POST | Auth |
| 38 | `/api/v1/idp/config` | GET | Auth |
| 39 | `/api/v1/idp/config` | POST | Auth |
| 40 | `/api/v1/auth/email/change` | POST | Auth |
| 41 | `/api/v1/auth/email/change/confirm` | POST | Auth |
| 42 | `/api/v1/auth/social/{provider}` | GET | Auth |
| 43 | `/api/v1/users` | GET | Users |
| 44 | `/api/v1/users` | POST | Users |
| 45 | `/api/v1/users/{id}` | GET | Users |
| 46 | `/api/v1/users/{id}` | DELETE | Users |
| 47 | `/api/v1/users/{id}` | PATCH | Users |
| 48 | `/api/v1/users/{id}/lock` | POST | Users |
| 49 | `/api/v1/users/{id}/unlock` | POST | Users |
| 50 | `/api/v1/users/{id}/deactivate` | POST | Users |
| 51 | `/api/v1/users/{id}/activate` | POST | Users |
| 52 | `/api/v1/users/import` | POST | Users |
| 53 | `/scim/v2/Users` | GET | SCIM |
| 54 | `/scim/v2/Users` | POST | SCIM |
| 55 | `/scim/v2/Users/{id}` | GET | SCIM |
| 56 | `/scim/v2/Users/{id}` | PUT | SCIM |
| 57 | `/scim/v2/Users/{id}` | DELETE | SCIM |
| 58 | `/api/v1/roles` | GET | Roles |
| 59 | `/api/v1/roles` | POST | Roles |
| 60 | `/api/v1/roles/{id}` | GET | Roles |
| 61 | `/api/v1/roles/{id}` | DELETE | Roles |
| 62 | `/api/v1/roles/{id}/permissions` | GET | Roles |
| 63 | `/api/v1/roles/{id}/permissions` | POST | Roles |
| 64 | `/api/v1/roles/{id}/parent` | POST | Roles |
| 65 | `/api/v1/permissions` | GET | Permissions |
| 66 | `/api/v1/policies` | GET | Policies |
| 67 | `/api/v1/policies` | POST | Policies |
| 68 | `/api/v1/policies/{id}` | GET | Policies |
| 69 | `/api/v1/policies/{id}` | DELETE | Policies |
| 70 | `/api/v1/policies/check` | POST | Policies |
| 71 | `/api/v1/policies/export` | GET | Policies |
| 72 | `/api/v1/policies/import` | POST | Policies |
| 73 | `/api/v1/policies/attribute-mapping` | GET | Policies |
| 74 | `/api/v1/policies/versions` | GET | Policies |
| 75 | `/api/v1/policies/versions` | POST | Policies |
| 76 | `/api/v1/policies/versions/rollback` | POST | Policies |
| 77 | `/api/v1/policies/templates` | GET | Policies |
| 78 | `/api/v1/policies/from-template/{template_id}` | POST | Policies |
| 79 | `/api/v1/policies/default-action` | GET | Policies |
| 80 | `/api/v1/policies/default-action` | PUT | Policies |
| 81 | `/api/v1/policies/time-conditions` | GET | Policies |
| 82 | `/api/v1/policies/time-conditions` | POST | Policies |
| 83 | `/api/v1/orgs` | GET | Organizations |
| 84 | `/api/v1/orgs` | POST | Organizations |
| 85 | `/api/v1/orgs/{id}` | GET | Organizations |
| 86 | `/api/v1/orgs/{id}` | PUT | Organizations |
| 87 | `/api/v1/orgs/{id}` | DELETE | Organizations |
| 88 | `/api/v1/orgs/{id}/members` | GET | Organizations |
| 89 | `/api/v1/orgs/{id}/members` | POST | Organizations |
| 90 | `/api/v1/orgs/{id}/tree` | GET | Organizations |
| 91 | `/api/v1/departments` | GET | Organizations |
| 92 | `/api/v1/departments` | POST | Organizations |
| 93 | `/api/v1/teams` | GET | Organizations |
| 94 | `/api/v1/teams` | POST | Organizations |
| 95 | `/api/v1/audit/events` | GET | Audit |
| 96 | `/api/v1/audit/events/{id}` | GET | Audit |
| 97 | `/api/v1/audit/stats` | GET | Audit |
| 98 | `/api/v1/audit/export` | GET | Audit |
| 99 | `/api/v1/audit/stream` | GET | Audit |
| 100 | `/api/v1/audit/retention` | GET | Audit |
| 101 | `/api/v1/audit/retention` | PUT | Audit |
| 102 | `/api/v1/audit/rules` | GET | Audit |
| 103 | `/api/v1/audit/rules` | POST | Audit |
| 104 | `/api/v1/audit/integrity` | GET | Audit |
| 105 | `/api/v1/audit/webhooks` | GET | Audit |
| 106 | `/api/v1/audit/webhooks` | POST | Audit |
| 107 | `/.well-known/jwks.json` | GET | OAuth |
| 108 | `/.well-known/openid-configuration` | GET | OAuth |
| 109 | `/oauth/authorize` | GET | OAuth |
| 110 | `/oauth/token` | POST | OAuth |

### 1.2 Total Endpoint Count

**Total: 111 endpoint operations** across 110 unique paths (sessions has GET + DELETE).

Breakdown by tag:
- Auth: 39 operations (largest section)
- Policies: 17 operations
- Users: 10 operations
- Organizations: 12 operations
- Audit: 12 operations
- SCIM: 5 operations
- Roles: 7 operations
- OAuth: 4 operations
- Health: 3 operations
- Permissions: 1 operation

---

## 2. Actual Endpoints in Source Code

### 2.1 Gateway Service (directly handled, not proxied)

Found in `services/gateway/internal/router/router.go`:

| Endpoint | Method | Handler |
|----------|--------|---------|
| `/healthz` | GET | inline healthz |
| `/healthz/live` | GET | liveness probe |
| `/healthz/ready` | GET | readiness probe |
| `/healthz/deep` | GET | deep health check |
| `/metrics` | GET | Prometheus metrics |
| `/docs` | GET | Swagger UI |
| `/api-docs` | GET | OpenAPI JSON spec |
| `/api/v1/admin/routes` | GET | handleAdminRoutes |
| `/api/v1/admin/stats` | GET | handleAdminStats |
| `/api/v1/admin/routes/{prefix}/toggle` | POST | handleAdminToggleRoute |
| `/api/v1/gateway/routes` | GET | handleGetRoutes |
| `/api/v1/gateway/routes/reload` | POST | handleReloadRoutes |
| `/api/v1/gateway/middleware` | GET | middleware chain info |
| `/api/v1/gateway/stats` | GET | stats handler |
| `/graphql` | POST | GraphQL resolver |

### 2.2 Gateway Proxy Routes (prefix-based forwarding)

Config in `services/gateway/internal/config/config.go`:

| Prefix | Backend Service |
|--------|----------------|
| `/api/v1/auth` | Auth (port 9001) |
| `/api/v1/users` | Identity (port 8081) |
| `/api/v1/roles` | Policy (port 8070) |
| `/api/v1/permissions` | Policy (port 8070) |
| `/api/v1/policies` | Policy (port 8070) |
| `/api/v1/orgs` | Org (port 8071) |
| `/api/v1/audit` | Audit (port 8072) |
| `/oauth` | OAuth (port 9005) |
| `/saml` | OAuth (port 9005) |

**Critical gap:** `/scim/v2` prefix is NOT in gateway config. SCIM endpoints
are unreachable through the gateway.

**Critical gap:** `/api/v1/departments` and `/api/v1/teams` do NOT match
the `/api/v1/orgs` prefix and are unreachable through the gateway.

### 2.3 Auth Service Endpoints

Found in `services/auth/internal/server/http.go`:

| Endpoint | Method | In Spec? |
|----------|--------|----------|
| `/healthz` | GET | gateway-only |
| `/readyz` | GET | no |
| `/metrics` | GET | no |
| `/api/v1/auth/login` | POST | YES |
| `/api/v1/auth/register` | POST | YES |
| `/api/v1/auth/logout` | POST | YES |
| `/api/v1/auth/refresh` | POST | YES |
| `/api/v1/auth/password/forgot` | POST | YES |
| `/api/v1/auth/password/reset` | POST | YES |
| `/api/v1/auth/forgot-password` | POST | alias (no) |
| `/api/v1/auth/reset-password` | POST | alias (no) |
| `/api/v1/auth/password/change` | POST | YES |
| `/api/v1/auth/sessions` | GET/POST | YES (GET/DELETE) |
| `/api/v1/auth/mfa/setup` | POST | YES |
| `/api/v1/auth/mfa/verify` | POST | YES |
| `/api/v1/auth/mfa/disable` | POST | YES |
| `/api/v1/auth/mfa/login` | POST | YES |
| `/api/v1/auth/password/policy` | GET | YES |
| `/api/v1/auth/password-policy` | GET | alias (no) |
| `/api/v1/auth/password-history` | GET | **MISSING** |
| `/api/v1/auth/lockout-policy` | GET | **MISSING** |
| `/api/v1/auth/webauthn/autofill` | GET | **MISSING** |
| `/api/v1/auth/magic-link` | POST | YES |
| `/api/v1/auth/magic-link/verify` | POST | YES |
| `/api/v1/auth/email/verify` | POST | YES |
| `/api/v1/auth/email/resend` | POST | YES |
| `/api/v1/auth/send-verification` | POST | alias (no) |
| `/api/v1/auth/verify-email` | POST | alias (no) |
| `/api/v1/auth/phone/send` | POST | YES |
| `/api/v1/auth/phone/verify` | POST | YES |
| `/api/v1/auth/stepup/challenge` | POST | YES |
| `/api/v1/auth/stepup/verify` | POST | YES |
| `/api/v1/auth/logout-all` | POST | YES |
| `/api/v1/auth/sessions/force-logout` | POST | **MISSING** |
| `/api/v1/auth/sessions/limit` | POST | **MISSING** |
| `/api/v1/auth/login-attempts` | GET | **MISSING** |
| `/api/v1/auth/risk-assess` | POST | **MISSING** |
| `/api/v1/auth/hooks` | GET/POST | YES |
| `/api/v1/auth/webauthn/register/begin` | POST | YES |
| `/api/v1/auth/webauthn/register/finish` | POST | YES |
| `/api/v1/auth/webauthn/login/begin` | POST | YES |
| `/api/v1/auth/webauthn/login/finish` | POST | YES |
| `/api/v1/auth/passwordless/register` | POST | YES |
| `/api/v1/auth/mfa/webauthn/begin` | POST | YES |
| `/api/v1/auth/mfa/webauthn/finish` | POST | YES |
| `/api/v1/idp/config` | GET/POST | YES |
| `/api/v1/auth/email/change` | POST | YES |
| `/api/v1/auth/email/change/confirm` | POST | YES |
| `/api/v1/auth/change-email` | POST | alias (no) |
| `/api/v1/auth/verify-email-change` | POST | alias (no) |
| `/api/v1/auth/device` | POST | **MISSING** |
| `/authorize` | POST | **MISSING** |
| `/usernamepassword/login` | POST | **MISSING** |
| `/dbconnections/signup` | POST | **MISSING** |
| `/api/v1/auth/social/` | any | YES |
| `/api/v1/auth/step-up-check` | GET | YES |
| `/api/v1/auth/step-up` | POST | YES |
| `/api/v1/security/password-policy` | GET | **MISSING** |

WebAuthn handler (`services/auth/internal/webauthn/handler.go`):

| Endpoint | Method | In Spec? |
|----------|--------|----------|
| `/api/v1/webauthn/register/begin` | POST | **MISSING** (spec has /auth/webauthn/ variant) |
| `/api/v1/webauthn/register/finish` | POST | **MISSING** |
| `/api/v1/webauthn/auth/begin` | POST | **MISSING** |
| `/api/v1/webauthn/auth/finish` | POST | **MISSING** |
| `/api/v1/webauthn/credentials` | GET | **MISSING** |
| `/api/v1/webauthn/credentials/` | DELETE | **MISSING** |
| `/.well-known/webauthn` | GET | **MISSING** |
| `/.well-known/assetlinks.json` | GET | **MISSING** |
| `/.well-known/apple-app-site-association` | GET | **MISSING** |

### 2.4 OAuth Service Endpoints

Found in `services/oauth/internal/server/server.go`:

| Endpoint | Method | In Spec? |
|----------|--------|----------|
| `/.well-known/openid-configuration` | GET | YES |
| `/oauth/jwks` | GET | **MISSING** (spec has /.well-known/jwks.json) |
| `/oauth/authorize` | GET | YES |
| `/oauth/token` | POST | YES |
| `/oauth/userinfo` | GET | **MISSING** |
| `/oauth/logout` | POST | **MISSING** |
| `/oauth/revoke` | POST | **MISSING** |
| `/api/v1/oauth/revoke` | POST | **MISSING** |
| `/api/v1/oauth/backchannel-logout` | POST | **MISSING** |
| `/api/v1/oauth/register` | POST | **MISSING** |
| `/oauth/introspect` | POST | **MISSING** |
| `/api/v1/oauth/introspect` | POST | **MISSING** |
| `/oauth/register` | POST | **MISSING** |
| `/oauth/consent` | POST | **MISSING** |
| `/saml/metadata` | GET | **MISSING** |
| `/saml/acs` | POST | **MISSING** |
| `/saml/sso` | GET | **MISSING** |
| `/saml/slo` | GET | **MISSING** |
| `/api/v1/oauth/clients` | GET | **MISSING** |
| `/api/v1/oauth/clients/` | GET/PUT/DELETE | **MISSING** |
| `/api/v1/oauth/device_authorization` | POST | **MISSING** |
| `/api/v1/oauth/device/approve` | POST | **MISSING** |

### 2.5 Identity Service Endpoints

Found in `services/identity/internal/server/http.go`:

| Endpoint | Method | In Spec? |
|----------|--------|----------|
| `/healthz` | GET | service-level |
| `/api/v1/users` | GET/POST | YES |
| `/api/v1/users/` | GET/DELETE/PATCH | YES ({id}) |
| `/api/v1/users/import` | POST | YES |

SCIM handler (`services/identity/internal/scim/handler.go`):

| Endpoint | Method | In Spec? |
|----------|--------|----------|
| `/scim/v2/Users` | GET/POST | YES |
| `/scim/v2/Users/` | GET/PUT/DELETE/PATCH | YES ({id}, but PATCH missing) |
| `/scim/v2/Groups` | GET/POST | **MISSING** |
| `/scim/v2/Groups/` | GET/PUT/DELETE/PATCH | **MISSING** |
| `/scim/v2/Bulk` | POST | **MISSING** |
| `/scim/v2/ServiceProviderConfig` | GET | **MISSING** |
| `/scim/v2/ResourceTypes` | GET | **MISSING** |

### 2.6 Policy Service Endpoints

Found in `services/policy/internal/server/http.go`:

| Endpoint | Method | In Spec? |
|----------|--------|----------|
| `/api/v1/roles` | GET/POST | YES |
| `/api/v1/roles/` | GET/DELETE/POST | YES ({id}) |
| `/api/v1/permissions` | GET | YES |
| `/api/v1/policies` | GET/POST | YES |
| `/api/v1/policies/` | GET/DELETE | YES ({id}) |
| `/api/v1/policies/check` | POST | YES |
| `/api/v1/policies/evaluate` | POST | **MISSING** |
| `/api/v1/policies/export` | GET | YES |
| `/api/v1/policies/import` | POST | YES |
| `/api/v1/policies/attribute-mapping` | GET | YES |
| `/api/v1/policies/versions` | GET/POST | YES |
| `/api/v1/policies/templates` | GET | YES |
| `/api/v1/policies/from-template/` | POST | YES |
| `/api/v1/policies/default-action` | GET/PUT | YES |
| `/api/v1/policies/time-conditions` | GET/POST | YES |
| `/api/v1/policies/dry-run` | POST | **MISSING** |
| `/api/v1/policies/diff` | GET | **MISSING** |
| `/api/v1/policies/analyze` | GET | **MISSING** |
| `/api/v1/policies/decision-log` | GET | **MISSING** |

### 2.7 Org Service Endpoints

Found in `services/org/internal/server/http.go`:

| Endpoint | Method | In Spec? |
|----------|--------|----------|
| `/api/v1/orgs` | GET/POST | YES |
| `/api/v1/orgs/tree` | GET | **MISSING** (spec has /orgs/{id}/tree only) |
| `/api/v1/orgs/` | GET/PUT/DELETE | YES ({id}) |
| `/api/v1/departments` | GET/POST | YES |
| `/api/v1/departments/` | GET/PUT/DELETE | **MISSING** (no /departments/{id} in spec) |
| `/api/v1/teams` | GET/POST | YES |
| `/api/v1/teams/` | GET/PUT/DELETE | **MISSING** (no /teams/{id} in spec) |

### 2.8 Audit Service Endpoints

Found in `services/audit/internal/server/http.go`:

| Endpoint | Method | In Spec? |
|----------|--------|----------|
| `/api/v1/audit/events` | GET | YES |
| `/api/v1/audit/events/` | GET | YES ({id}) |
| `/api/v1/audit/stats` | GET | YES |
| `/api/v1/audit/export` | GET | YES |
| `/api/v1/audit/stream` | GET | YES |
| `/api/v1/audit/ws` | GET | **MISSING** |
| `/api/v1/audit/metrics` | GET | **MISSING** |
| `/api/v1/audit/retention` | GET/PUT | YES |
| `/api/v1/audit/rules` | GET/POST | YES |
| `/api/v1/audit/correlate` | POST | **MISSING** |
| `/api/v1/audit/webhooks` | GET/POST | YES |
| `/api/v1/audit/verify-integrity` | GET | **MISSING** (spec has /audit/integrity) |
| `/api/v1/audit/search` | GET | **MISSING** |
| `/api/v1/audit/alerts/config` | GET/POST | **MISSING** |
| `/api/v1/audit/alerts/test` | POST | **MISSING** |
| `/api/v1/audit/reports` | GET | **MISSING** |
| `/api/v1/audit` | GET | alias for events |

---

## 3. Comparison Matrix (Summary)

### 3.1 Match Statistics

| Category | Count |
|----------|-------|
| MATCH (in both spec and code) | 111 |
| MISSING_FROM_SPEC (in code, not in spec) | 49+ |
| MISSING_FROM_CODE (in spec, not in code) | 1 |
| PATH MISMATCH (spec path != code path) | 2 |

### 3.2 Critical Discrepancies

| Endpoint | Spec Path | Code Path | Issue |
|----------|-----------|-----------|-------|
| JWKS | `/.well-known/jwks.json` | `/oauth/jwks` | Path mismatch |
| Audit integrity | `/api/v1/audit/integrity` | `/api/v1/audit/verify-integrity` | Path mismatch |
| SCIM Users PATCH | (not documented) | `/scim/v2/Users/{id}` PATCH | Missing from spec |
| SCIM Groups | (not documented) | `/scim/v2/Groups` | Entire resource missing |
| OAuth userinfo | (not documented) | `/oauth/userinfo` | OIDC standard, missing |
| OAuth introspect | (not documented) | `/oauth/introspect` | RFC 7662, missing |
| OAuth revoke | (not documented) | `/oauth/revoke` | RFC 7009, missing |
| SAML endpoints | (not documented) | `/saml/*` | All SAML missing |

---

## 4. Missing From OpenAPI (Endpoints in Code, Not in Spec)

### 4.1 OAuth/OIDC Standard Endpoints (Critical)

These are standard OAuth 2.0 / OIDC endpoints that any OIDC-compliant client expects:

| Endpoint | Method | RFC | Impact |
|----------|--------|-----|--------|
| `/oauth/userinfo` | GET | OIDC Core | Clients cannot get user info |
| `/oauth/introspect` | POST | RFC 7662 | Resource servers can't validate tokens |
| `/api/v1/oauth/introspect` | POST | RFC 7662 | API-prefixed alias missing |
| `/oauth/revoke` | POST | RFC 7009 | Clients can't revoke tokens |
| `/api/v1/oauth/revoke` | POST | RFC 7009 | API-prefixed alias missing |
| `/oauth/logout` | POST | OIDC Session | RP-initiated logout missing |
| `/api/v1/oauth/backchannel-logout` | POST | OIDC Backchannel | Back-channel logout missing |
| `/oauth/register` | POST | RFC 7591 | Dynamic client registration missing |
| `/api/v1/oauth/register` | POST | RFC 7591 | API-prefixed alias missing |
| `/oauth/consent` | POST | - | Consent flow missing |
| `/oauth/jwks` | GET | OIDC Discovery | Path differs from spec |
| `/api/v1/oauth/clients` | GET | - | Client management missing |
| `/api/v1/oauth/clients/{id}` | GET/PUT/DELETE | - | Client CRUD missing |
| `/api/v1/oauth/device_authorization` | POST | RFC 8628 | Device flow missing |
| `/api/v1/oauth/device/approve` | POST | RFC 8628 | Device approval missing |

**Why this matters:** SDK-generated clients will be missing the most commonly used
OAuth2/OIDC endpoints. Any developer building an OIDC integration will not find
the userinfo, introspect, or revoke endpoints. This blocks third-party OIDC client
registration and compliance testing.

### 4.2 SAML Endpoints (Critical)

| Endpoint | Method | Impact |
|----------|--------|--------|
| `/saml/metadata` | GET | SP metadata not documented |
| `/saml/acs` | POST | Assertion consumer service missing |
| `/saml/sso` | GET | SSO redirect binding missing |
| `/saml/slo` | GET | Single logout missing |

**Why this matters:** SAML federation partners cannot discover SP endpoints.
This blocks enterprise SAML integrations entirely.

### 4.3 SCIM Endpoints (Important)

| Endpoint | Method | Impact |
|----------|--------|--------|
| `/scim/v2/Users/{id}` | PATCH | SCIM PATCH for users missing from spec |
| `/scim/v2/Groups` | GET/POST | Entire Groups resource missing |
| `/scim/v2/Groups/{id}` | GET/PUT/DELETE/PATCH | Group lifecycle missing |
| `/scim/v2/Bulk` | POST | Bulk operations missing |
| `/scim/v2/ServiceProviderConfig` | GET | SCIM compliance info missing |
| `/scim/v2/ResourceTypes` | GET | Resource type discovery missing |

**Why this matters:** Identity provisioning tools (Okta, Azure AD, OneLogin) that
support SCIM Groups and Bulk operations cannot configure correctly. The spec
implies only User provisioning is supported.

### 4.4 Auth Security/Policy Endpoints

| Endpoint | Method | Impact |
|----------|--------|--------|
| `/api/v1/auth/password-history` | GET | Password history policy not documented |
| `/api/v1/auth/lockout-policy` | GET | Account lockout config missing |
| `/api/v1/auth/webauthn/autofill` | GET | WebAuthn autofill feature missing |
| `/api/v1/auth/sessions/force-logout` | POST | Admin session revocation missing |
| `/api/v1/auth/sessions/limit` | POST | Concurrent session limits missing |
| `/api/v1/auth/login-attempts` | GET | Login attempt history missing |
| `/api/v1/auth/risk-assess` | POST | Risk scoring endpoint missing |
| `/api/v1/auth/device` | POST | Device registration/remember missing |
| `/api/v1/security/password-policy` | GET | Security-branded policy endpoint missing |

### 4.5 Gateway Operational Endpoints

| Endpoint | Method | Impact |
|----------|--------|--------|
| `/healthz/deep` | GET | Deep health check (with dependencies) missing |
| `/metrics` | GET | Prometheus metrics endpoint missing |
| `/docs` | GET | Swagger UI missing |
| `/api-docs` | GET | OpenAPI JSON spec self-serving missing |
| `/api/v1/admin/routes` | GET | Route admin API missing |
| `/api/v1/admin/stats` | GET | Backend statistics missing |
| `/api/v1/admin/routes/{prefix}/toggle` | POST | Route toggle missing |
| `/api/v1/gateway/routes` | GET | Gateway route listing missing |
| `/api/v1/gateway/routes/reload` | POST | Hot route reload missing |
| `/api/v1/gateway/middleware` | GET | Middleware chain info missing |
| `/api/v1/gateway/stats` | GET | Gateway statistics missing |
| `/graphql` | POST | GraphQL endpoint missing |

### 4.6 Policy Advanced Endpoints

| Endpoint | Method | Impact |
|----------|--------|--------|
| `/api/v1/policies/evaluate` | POST | Alternative evaluation endpoint missing |
| `/api/v1/policies/dry-run` | POST | Dry-run policy evaluation missing |
| `/api/v1/policies/diff` | GET | Policy version diff missing |
| `/api/v1/policies/analyze` | GET | Policy analysis/audit missing |
| `/api/v1/policies/decision-log` | GET | Decision logging query missing |

### 4.7 Audit Advanced Endpoints

| Endpoint | Method | Impact |
|----------|--------|--------|
| `/api/v1/audit/ws` | GET | WebSocket real-time push missing |
| `/api/v1/audit/metrics` | GET | Audit metrics missing |
| `/api/v1/audit/correlate` | POST | Event correlation missing |
| `/api/v1/audit/verify-integrity` | GET | Integrity verification (path differs from spec) |
| `/api/v1/audit/search` | GET | Full-text search missing |
| `/api/v1/audit/alerts/config` | GET/POST | Alert configuration missing |
| `/api/v1/audit/alerts/test` | POST | Alert testing missing |
| `/api/v1/audit/reports` | GET | Compliance reports missing |

### 4.8 Org CRUD Completion

| Endpoint | Method | Impact |
|----------|--------|--------|
| `/api/v1/orgs/tree` | GET | Full org tree (spec has /orgs/{id}/tree only) |
| `/api/v1/departments/{id}` | GET/PUT/DELETE | Department CRUD incomplete |
| `/api/v1/teams/{id}` | GET/PUT/DELETE | Team CRUD incomplete |

---

## 5. Missing From Code (Endpoints in Spec, Not Implemented)

### 5.1 Path Mismatches

| Spec Path | Expected Code Path | Actual Code Path | Status |
|-----------|-------------------|-----------------|--------|
| `/.well-known/jwks.json` | JWKS at well-known | `/oauth/jwks` | PATH MISMATCH — spec says well-known path, code uses /oauth/ prefix |
| `/api/v1/audit/integrity` | Audit integrity check | `/api/v1/audit/verify-integrity` | PATH MISMATCH |

### 5.2 Missing Implementation

| Spec Path | Method | Status |
|-----------|--------|--------|
| `/api/v1/audit/integrity` | GET | **NOT IMPLEMENTED** (code uses different path) |

The `/api/v1/audit/integrity` endpoint from the spec returns 404 when accessed
through the gateway. The actual endpoint is at `/api/v1/audit/verify-integrity`.

### 5.3 Impact

Spec-promised endpoints that don't exist at the documented path cause:
- **Integration failures** when third-party tools follow the spec
- **SDK method errors** when generated SDKs call nonexistent paths
- **Broken contract trust** when developers discover the spec is unreliable

---

## 6. Spec Quality Issues

### 6.1 Missing Schemas

| Endpoint | Issue |
|----------|-------|
| `/api/v1/policies` GET | No response schema — returns "Policy list" with no schema definition |
| `/api/v1/policies/{id}` GET | No response schema for policy details |
| `/api/v1/policies/export` GET | Response schema is generic `type: object` |
| `/api/v1/policies/templates` GET | No response schema |
| `/api/v1/permissions` GET | No response schema — returns "Permission list" with nothing |
| `/api/v1/roles/{id}/permissions` GET | No response schema |
| `/.well-known/openid-configuration` GET | No response schema for OIDC discovery doc |
| `/oauth/authorize` GET | No response schema beyond "302 redirect" |
| `/scim/v2/Users/{id}` GET | Response schema is undefined (just "SCIM user") |
| `/scim/v2/Users/{id}` PUT | Request body schema is generic `type: object` |
| `/scim/v2/Users` POST | Request body schema is generic `type: object` |

### 6.2 Missing Error Responses

The spec defines reusable error responses (`BadRequest`, `Unauthorized`, `Forbidden`,
`NotFound`, `Conflict`, `RateLimited`) but many endpoints don't reference them:

| Endpoint | Missing Error Responses |
|----------|----------------------|
| All `/api/v1/users/{id}/*` endpoints | No 403 Forbidden |
| `/api/v1/orgs/*` endpoints | No 400/409 error responses |
| `/api/v1/audit/*` endpoints | No error responses at all |
| `/scim/v2/*` endpoints | No SCIM error response (RFC 7644 requires `urn:ietf:params:scim:api:messages:2.0:Error`) |
| `/oauth/token` POST | No 400 (invalid_grant), 401 (invalid_client) |
| `/oauth/authorize` GET | No 400 (invalid_request), 403 (access_denied) |

### 6.3 Missing Security Definitions

| Endpoint | Issue |
|----------|-------|
| `/api/v1/auth/logout` POST | Has no `security: []`, defaults to BearerAuth, but may not always require auth |
| `/api/v1/auth/email/resend` POST | Has no `security: []` but should be authenticated |
| `/api/v1/auth/hooks` GET/POST | Missing scope requirement documentation |
| `/scim/v2/*` | SCIM endpoints typically use Bearer token but spec doesn't document SCIM auth |
| No OAuth2 security scheme | The spec only defines BearerAuth (JWT), no OAuth2 password/client_credentials flow |

### 6.4 Outdated Descriptions

| Issue | Details |
|-------|---------|
| `/oauth/authorize` | Says `response_type: [code, token]` but code supports `code` only for authorization_code flow |
| `/oauth/token` | Missing `grant_type: password` and `urn:ietf:params:oauth:grant-type:device_code` |
| `/api/v1/auth/social/{provider}` | Lists `[google, github, microsoft, apple]` but code also supports discord, slack, linkedin, gitlab, oidc |
| Version `1.0.0` | No indication of API maturity or breaking change policy |

### 6.5 Wrong Content Types

| Endpoint | Issue |
|----------|-------|
| `/scim/v2/*` | Correctly uses `application/scim+json` but SCIM PATCH should document `application/scim+json` with specific schema |
| `/oauth/token` | Uses `application/x-www-form-urlencoded` (correct) but response should document `application/json` |
| `/api/v1/audit/stream` | Uses `text/event-stream` (correct for SSE) but no schema for individual event format |

### 6.6 Missing Tags

The spec defines no `SAML` tag despite gateway routing `/saml` prefix.
SAML endpoints are completely absent from the spec.

### 6.7 Global Security Ambiguity

The spec applies `security: - BearerAuth: []` globally, then overrides with
`security: []` on public endpoints. However, several endpoints that should be
public (like `/oauth/authorize`) or semi-public are inconsistent.

---

## 7. SDK Generation Readiness

### 7.1 Overall Assessment

**Readiness: ~60% — NOT READY for production SDK generation.**

The spec would generate SDKs that compile and have basic CRUD methods, but:
- OAuth2/OIDC client methods would be incomplete (no userinfo, introspect, revoke)
- SAML methods would not exist
- SCIM SDK would cover only Users (no Groups, Bulk, ServiceProviderConfig)
- Operational/admin methods would be undocumented
- Response schemas for many endpoints would be `any`/`interface{}`

### 7.2 Go SDK Impact

| Issue | Impact |
|-------|--------|
| Missing OAuth2 flows | `oapi-codegen` would generate no `UserInfo()`, `Introspect()`, `Revoke()` methods |
| Missing schemas | Many methods return `*http.Response` instead of typed structs |
| Path mismatches | JWKS endpoint would call wrong URL |
| No pagination schema | List endpoints lack `cursor`/`offset` standardized pattern |
| SCIM Groups missing | Go SDK cannot manage SCIM groups |

### 7.3 JavaScript/TypeScript SDK Impact

| Issue | Impact |
|-------|--------|
| Missing `application/x-www-form-urlencoded` helpers | Token endpoint requires manual form encoding |
| No discriminated unions | Policy and auth response types are too generic |
| Missing OAuth2 flows | OpenAPI Codegen would not generate OIDC helpers |
| No webhook event schemas | Audit webhook subscription has no event payload schema |
| SSE not modeled | `text/event-stream` has no typed event schema |

### 7.4 Python SDK Impact

| Issue | Impact |
|-------|--------|
| Missing schemas | Many methods return `Dict[str, Any]` instead of dataclasses |
| No OAuth2 security scheme | `openapi-python-client` cannot generate auth helpers |
| SCIM incomplete | Cannot manage groups, bulk operations |
| No async patterns | Long-running operations (export, import) lack async indication |

---

## 8. Recommendations

### P0 — Blocking SDK Generation

1. **Add all OAuth2/OIDC standard endpoints** to the spec:
   - `/oauth/userinfo` (GET)
   - `/oauth/introspect` (POST) + `/api/v1/oauth/introspect` alias
   - `/oauth/revoke` (POST) + `/api/v1/oauth/revoke` alias
   - `/oauth/logout` (POST)
   - `/oauth/jwks` (GET) or fix `/.well-known/jwks.json` path to match code
   - `/api/v1/oauth/backchannel-logout` (POST)
   - `/oauth/register` (POST) + `/api/v1/oauth/register` alias
   - `/api/v1/oauth/clients` (GET/POST) + `/api/v1/oauth/clients/{id}` (GET/PUT/DELETE)
   - `/api/v1/oauth/device_authorization` (POST)
   - `/api/v1/oauth/device/approve` (POST)

2. **Add all SAML endpoints** to the spec with a new `SAML` tag:
   - `/saml/metadata` (GET)
   - `/saml/acs` (POST)
   - `/saml/sso` (GET)
   - `/saml/slo` (GET)

3. **Fix path mismatches**:
   - JWKS: align spec path `/.well-known/jwks.json` with actual `/oauth/jwks`, or add both
   - Audit integrity: change spec from `/api/v1/audit/integrity` to `/api/v1/audit/verify-integrity`, or add both paths

4. **Add SCIM PATCH** for `/scim/v2/Users/{id}` (already implemented in code)

5. **Define response schemas** for all endpoints that currently lack them:
   - Policy list, policy details, policy templates
   - Permission list
   - Role permissions
   - OIDC discovery document

### P1 — Improving Developer Experience

6. **Add complete SCIM resource coverage**:
   - `/scim/v2/Groups` (GET/POST) and `/scim/v2/Groups/{id}` (GET/PUT/DELETE/PATCH)
   - `/scim/v2/Bulk` (POST)
   - `/scim/v2/ServiceProviderConfig` (GET)
   - `/scim/v2/ResourceTypes` (GET)

7. **Add error responses** to all endpoints — at minimum 400, 401, 403, 404, 500

8. **Add OAuth2 security scheme** (flows: authorizationCode, clientCredentials, refreshToken, deviceCode) alongside the existing BearerAuth

9. **Document social login providers** comprehensively — update the enum to include all supported providers (google, github, microsoft, apple, discord, slack, linkedin, gitlab)

10. **Add request/response schemas** for:
    - OAuth token request/response (all grant types)
    - OIDC userinfo response
    - Token introspection response
    - SCIM error response

11. **Add gateway operational endpoints** to spec under a `Gateway` tag:
    - `/api/v1/gateway/routes` (GET)
    - `/api/v1/gateway/routes/reload` (POST)
    - `/api/v1/gateway/stats` (GET)

12. **Document `/api/v1/orgs/tree`** (GET) — the full org tree endpoint that exists in code

13. **Add department and team CRUD** completion — `/api/v1/departments/{id}` and `/api/v1/teams/{id}`

### P2 — Nice-to-Have

14. **Add WebAuthn credential management endpoints** (`/api/v1/webauthn/credentials`)

15. **Add audit advanced endpoints**:
    - `/api/v1/audit/ws` (WebSocket)
    - `/api/v1/audit/search` (GET)
    - `/api/v1/audit/correlate` (POST)
    - `/api/v1/audit/alerts/config` (GET/POST)
    - `/api/v1/audit/reports` (GET)

16. **Add policy advanced endpoints**:
    - `/api/v1/policies/evaluate` (POST)
    - `/api/v1/policies/dry-run` (POST)
    - `/api/v1/policies/diff` (GET)
    - `/api/v1/policies/analyze` (GET)
    - `/api/v1/policies/decision-log` (GET)

17. **Add GraphQL endpoint** (`/graphql` POST) with introspection

18. **Add health deep check** (`/healthz/deep` GET) and metrics (`/metrics` GET)

19. **Add pagination schema** as a reusable component (cursor or offset pattern)

20. **Add webhook event payload schemas** for audit webhooks

21. **Implement CI validation** — add a step to `make test` that validates spec completeness against source code route registrations

22. **Add deprecation notes** for alias endpoints (e.g., `/api/v1/auth/forgot-password` is an alias for `/api/v1/auth/password/forgot`)

---

## Appendix A: Spec Schema Coverage

| Schema | Defined | Used By |
|--------|---------|---------|
| HealthStatus | Yes | healthz |
| Error | Yes | error responses |
| LoginRequest | Yes | auth/login |
| RegisterRequest | Yes | auth/register |
| RegisterResponse | Yes | auth/register |
| RefreshRequest | Yes | auth/refresh |
| PasswordResetRequest | Yes | auth/password/reset |
| PasswordChangeRequest | Yes | auth/password/change |
| TokenSet | Yes | login, refresh, token |
| MFALoginRequest | Yes | auth/mfa/login |
| MFASetupResponse | Yes | auth/mfa/setup |
| PasswordPolicy | Yes | auth/password/policy |
| User | Yes | users |
| CreateUserRequest | Yes | users POST |
| UpdateUserRequest | Yes | users PATCH |
| UserListResponse | Yes | users GET |
| Role | Yes | roles |
| CreateRoleRequest | Yes | roles POST |
| Organization | Yes | orgs |
| CreateOrgRequest | Yes | orgs POST |
| CreatePolicyRequest | Yes | policies POST |
| PolicyCheckRequest | Yes | policies/check POST |
| PolicyCheckResponse | Yes | policies/check POST |
| Session | Yes | auth/sessions |
| AuthHook | Yes | auth/hooks |
| IdPConfig | Yes | idp/config |
| AuditEvent | Yes | audit/events |
| AuditEventList | Yes | audit/events |
| AuditStats | Yes | audit/stats |
| SCIMListResponse | Yes | scim/v2/Users |
| JWKS | Yes | well-known/jwks |
| **Policy** | **NO** | policies GET/{id} |
| **Permission** | **NO** | permissions GET |
| **PolicyVersion** | **NO** | policies/versions |
| **PolicyTemplate** | **NO** | policies/templates |
| **Department** | **NO** | departments |
| **Team** | **NO** | teams |
| **OIDCDiscovery** | **NO** | openid-configuration |
| **SCIMUser** | **NO** | scim/v2/Users |
| **SCIMGroup** | **NO** | (missing entirely) |
| **SCIMErrror** | **NO** | scim error responses |
| **WebhookSubscription** | **NO** | audit/webhooks |
| **OAuthClient** | **NO** | oauth/clients |

---

## Appendix B: Methodology

1. Read the full OpenAPI spec (2,397 lines) and extracted all paths + methods
2. Grepped all 7 services for route registration patterns (`mux.HandleFunc`, `r.Get`, `r.Post`)
3. Read gateway config to determine proxy routing prefixes
4. Cross-referenced each spec endpoint against source code route registrations
5. Identified path mismatches, missing endpoints, and schema gaps
6. Assessed SDK generation impact for Go, JS/TS, and Python targets

---

*End of audit. Total spec endpoints: 111. Total code endpoints: ~160+. Coverage: ~69%.*
