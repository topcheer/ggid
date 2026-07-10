# GGID Glossary

Key terms used throughout the GGID IAM Platform documentation.

---

## A

**ABAC (Attribute-Based Access Control)**
Access control model where permissions are granted based on attributes of the user, resource, action, and environment (e.g., department, clearance level, time of day). GGID supports ABAC via the policy engine with JSON conditions.

**Access Token**
A short-lived JWT (default 1 hour) containing user identity claims (`sub`, `tenant_id`, `roles`, `scopes`). Used to authenticate API requests via `Authorization: Bearer <token>`.

**ACR (Authentication Context Class Reference)**
An OIDC parameter that specifies the level of authentication required (e.g., `phr` for phishing-resistant, `mfa` for multi-factor).

**Argon2id**
Memory-hard password hashing algorithm (RFC 9106) resistant to GPU/ASIC brute-force. GGID uses it for all password storage.

**Attestation**
In WebAuthn, the cryptographic proof from an authenticator device that it created a credential. Verified during registration.

**Audit Event**
A structured record of a security-relevant action (e.g., `user.login`, `role.create`). Published via NATS JetStream and persisted to PostgreSQL.

**Auth Hook**
A configurable webhook that intercepts authentication flows at specific points (e.g., `pre-registration`, `post-login`, `pre-token-issue`). See [Plugin Development](./plugin-development.md).

**Authenticator**
A software or hardware component that performs WebAuthn authentication (e.g., YubiKey, Touch ID, Windows Hello).

---

## B

**Bearer Token**
A token type where possession of the token grants access (no additional proof required). GGID JWTs are bearer tokens.

---

## C

**CORS (Cross-Origin Resource Sharing)**
Browser security mechanism that controls which origins can make requests to the API. GGID Gateway enforces a configurable CORS whitelist.

**Circuit Breaker**
A resilience pattern that stops sending requests to a failing backend after an error threshold is reached, allowing it to recover. GGID Gateway implements per-backend circuit breakers.

**Claim**
A key-value pair in a JWT payload (e.g., `sub`, `email`, `roles`). Claims represent assertions about the authenticated user.

**Client Credentials**
An OAuth2 grant type for machine-to-machine authentication. A service exchanges its `client_id` + `client_secret` for an access token without user interaction.

---

## D

**Deny-All Default**
A policy engine mode where all access is denied unless an explicit allow rule matches. GGID supports configurable default-deny.

**Device Auth**
Authentication using a registered device credential (WebAuthn/FIDO2) instead of a password.

---

## E

**E2E (End-to-End) Test**
Tests that verify the full request path from client through Gateway to backend services and back. GGID's E2E suite runs via `deploy/e2e-docker-test.sh`.

**Event-Driven Architecture**
A design where services communicate via events (messages) rather than direct calls. GGID's audit pipeline uses NATS JetStream for event-driven audit logging.

---

## F

**FIDO2**
An open authentication standard by the FIDO Alliance. Encompasses WebAuthn (browser API) and CTAP (client-to-authenticator protocol). Enables passwordless and phishing-resistant authentication.

**Flows (OAuth2)**
Standardized authorization flows: Authorization Code, Client Credentials, Refresh Token, Device Code, PKCE.

---

## G

**Gateway**
The single entry point for all API requests. Verifies JWTs, applies rate limiting, routes to backend services. See [Gateway Architecture](./design/gateway-architecture.md).

---

## I

**IdP (Identity Provider)**
A service that authenticates users and issues identity assertions. GGID can act as an IdP (via OIDC/SAML) or consume external IdPs (via federation).

**IdP Federation**
Configuring GGID to trust an external identity provider (e.g., Azure AD, Okta) for authentication. Users authenticate at the external IdP and GGID accepts their assertion.

**Idempotency**
A property where performing an operation multiple times produces the same result as performing it once. Important for retry-safe API calls.

**Idempotency Key**
A unique identifier sent by the client to ensure a request is processed only once, even if retried. Use the `Idempotency-Key` header for POST/PUT requests.

**IP Allowlist**
A security feature restricting API access to specified IP ranges (CIDR notation). Configured at the Gateway level.

**ISS (Issuer)**
A JWT claim (`iss`) identifying who issued the token. GGID sets `iss: ggid-auth` by default.

---

## J

**JIT (Just-In-Time) Provisioning**
Automatic user account creation on first login via SSO/LDAP. The user record is created from IdP attributes when they first authenticate, rather than being pre-created.

**JWKS (JSON Web Key Set)**
A JSON document at `/.well-known/jwks.json` containing RSA public keys for JWT signature verification. Cached by the Gateway and SDKs.

**JWT (JSON Web Token)**
A compact, signed token format (RFC 7519) used for authentication. GGID signs JWTs with RS256 (RSA 2048-bit).

---

## L

**LDAP (Lightweight Directory Access Protocol)**
A protocol for accessing directory services (e.g., Active Directory). GGID integrates LDAP as an auth provider in the chain (Local + LDAP).

**LTREE**
A PostgreSQL extension for hierarchical tree data. Used by GGID for organization tree structures.

---

## M

**Magic Link**
A passwordless login method where a one-time-use link is sent to the user's email. Clicking the link authenticates the user without a password.

**MFA (Multi-Factor Authentication)**
Requiring two or more authentication factors. GGID supports TOTP, Email OTP, and WebAuthn as second factors.

**Middleware**
Software that intercepts requests in the Gateway pipeline (e.g., JWT verification, rate limiting, CORS). See [Gateway Middleware Chain](./design/gateway-architecture.md#middleware-chain).

---

## N

**NATS JetStream**
A persistent streaming system built into NATS. GGID uses it as the durable transport for audit events. See [Event-Driven Audit Design](./design/event-driven-audit.md).

**Nonce**
A single-use random value to prevent replay attacks. Used in OAuth2/OIDC flows.

---

## O

**OIDC (OpenID Connect)**
An identity layer on top of OAuth2. Provides user authentication and identity assertions via ID tokens. GGID is a full OIDC provider with discovery and JWKS.

**OTLP (OpenTelemetry Protocol)**
The standard protocol for exporting traces and metrics. GGID Gateway exports traces via OTLP HTTP.

---

## P

**Passkey**
A FIDO2 credential synced across a user's devices via cloud (e.g., Apple iCloud Keychain). Enables passwordless login without a hardware key.

**Passwordless**
Authentication without a traditional password. GGID supports magic links and WebAuthn-only accounts.

**PKCE (Proof Key for Code Exchange)**
An OAuth2 extension (RFC 7636) that prevents authorization code interception. Recommended for SPAs and mobile apps.

**Policy Engine**
The GGID component that evaluates RBAC + ABAC rules to make allow/deny decisions. See [Policy Engine Design](./design/policy-engine.md).

**Provider Chain**
GGID's auth provider architecture: Local (password) → LDAP → Social → WebAuthn. Each is tried in order until one succeeds.

---

## R

**RBAC (Role-Based Access Control)**
Access control model where permissions are assigned to roles, and users inherit permissions through role assignment. GGID supports role hierarchy and wildcard matching.

**Rate Limiting**
Restricting the number of API requests per time window. GGID Gateway enforces per-IP and per-tenant rate limits. See [API Rate Limits](./api-rate-limits.md).

**ReBAC (Relationship-Based Access Control)**
An access model based on relationships between entities (e.g., "user X is editor of document Y"). Planned for GGID Phase 16.

**Refresh Token**
A long-lived token (default 30 days) used to obtain new access tokens. GGID rotates refresh tokens on each use.

**Refresh Token Rotation**
A security practice where each refresh produces a new refresh token, invalidating the old one. Detects token theft.

**RLS (Row-Level Security)**
A PostgreSQL feature that filters rows based on a session variable. GGID uses RLS for multi-tenant isolation. See [RLS Design](./design/multi-tenant-rls.md).

---

## S

**SAML 2.0 (Security Assertion Markup Language)**
An XML-based SSO protocol. GGID acts as a SAML Service Provider.

**SCIM 2.0 (System for Cross-domain Identity Management)**
A standard protocol (RFC 7643/7644) for automated user provisioning. GGID exposes `/scim/v2/Users`.

**Scope**
A string defining the level of access requested in OAuth2/OIDC (e.g., `openid profile email`). Included in the JWT `scope` claim.

**Server-Sent Events (SSE)**
A one-way streaming protocol from server to browser. GGID uses SSE for real-time audit event streaming.

**Session**
A period of authenticated access. GGID sessions are JWT-based (stateless) with optional Redis blocklist for revocation.

**Signing Key**
The RSA private key used to sign JWTs. Must be kept secret and rotated regularly.

**Single Sign-On (SSO)**
A system where one login provides access to multiple applications. GGID supports SSO via SAML 2.0 and OIDC.

**Step-Up Authentication**
Requiring additional verification (e.g., MFA) for sensitive operations, even when the user is already logged in. GGID provides `/auth/step-up` endpoints.

---

## T

**Tenant**
An isolated customer or organization within GGID. Each tenant has its own users, roles, and data, separated by RLS.

**Tenant ID**
A UUID identifying a tenant. Required as `X-Tenant-ID` header on all API requests. Default: `00000000-0000-0000-0000-000000000001`.

**TOTP (Time-Based One-Time Password)**
A 6-digit code generated by an authenticator app (RFC 6238). Changes every 30 seconds. GGID's default MFA method.

**Token Blocklist**
A Redis-based list of revoked JWT IDs (`jti`). Checked at refresh time to enforce immediate logout.

**Token Theft Detection**
When refresh token rotation detects reuse of an already-rotated token, indicating theft. GGID revokes all tokens for that user.

---

## W

**WebAuthn**
A browser API (W3C standard) for FIDO2 authentication. Enables registration and login with passkeys, security keys, and platform authenticators. GGID implements full WebAuthn flows.

**Wildcard Matching**
In RBAC, resource patterns with `*` (e.g., `documents:*` matches `documents:drafts`, `documents:sensitive`). GGID supports wildcard suffixes.

---

## X

**X-Tenant-ID**
An HTTP header containing the tenant UUID. Required on all GGID API requests. The Gateway also injects it as a query param and JSON body field for backend services.

---

## References

- [RFC 7519 — JWT](https://datatracker.ietf.org/doc/html/rfc7519)
- [RFC 6238 — TOTP](https://datatracker.ietf.org/doc/html/rfc6238)
- [RFC 7636 — PKCE](https://datatracker.ietf.org/doc/html/rfc7636)
- [RFC 9100 — Argon2](https://datatracker.ietf.org/doc/html/rfc9100)
- [W3C WebAuthn](https://www.w3.org/TR/webauthn/)
- [OWASP Top 10](https://owasp.org/Top10/)

---

## IAM Core Terminology

### OIDC (OpenID Connect)

**OIDC** is an identity layer built on top of OAuth 2.0. It enables clients to
verify the identity of an end-user and obtain basic profile information via an
ID Token (JWT).

GGID implements OIDC with:
- `/.well-known/openid-configuration` — Discovery endpoint
- Authorization Code + PKCE flow
- ID Token (RS256 signed JWT with user claims)
- UserInfo endpoint

### SCIM (System for Cross-domain Identity Management)

**SCIM 2.0** (RFC 7643/7644) is a standardized REST API for user provisioning.
It enables automated user lifecycle management across systems.

GGID exposes SCIM endpoints:
- `/scim/v2/Users` — CRUD with filtering (`?filter=userName eq "alice"`)
- `/scim/v2/Groups` — Group management
- Compatible with Okta, Azure AD, and other SCIM providers

### SAML (Security Assertion Markup Language)

**SAML 2.0** is an XML-based SSO protocol used primarily in enterprise
environments. GGID supports both SP-initiated and IdP-initiated SSO flows
with signed and encrypted assertions.

### WebAuthn / Passkey

**WebAuthn** (W3C standard) enables passwordless authentication using
public-key cryptography. Users authenticate with biometrics (Touch ID, Face ID)
or hardware security keys (YubiKey).

A **Passkey** is a WebAuthn credential synchronized across devices via cloud
( iCloud Keychain, Google Password Manager).

GGID supports:
- Platform authenticators (Touch ID, Windows Hello)
- Roaming authenticators (YubiKey, Titan)
- Registration and authentication flows

### RBAC (Role-Based Access Control)

**RBAC** assigns permissions to roles, and roles to users. Access decisions
are based on the user's assigned roles.

```
User → Role → Permission → Resource
alice → admin → write:users → /api/v1/users
```

### ABAC (Attribute-Based Access Control)

**ABAC** makes access decisions based on attributes of the user, resource,
action, and environment. More fine-grained than RBAC.

```
IF user.department == "HR"
   AND resource.type == "salary"
   AND action == "read"
   AND time.weekday IN [Mon-Fri]
THEN allow
```

GGID's policy engine supports both RBAC and ABAC, composable in a single policy.

### Federation

**Federation** allows users to authenticate through a trusted external Identity
Provider (IdP) instead of maintaining local credentials.

Examples:
- Social login (Google, GitHub, Microsoft) — OAuth 2.0 federation
- Enterprise SSO (Azure AD, Okta) — SAML federation
- LDAP/AD — Directory federation

### JIT (Just-In-Time) Provisioning

**JIT provisioning** automatically creates a local user account the first time
a user authenticates via federation (OAuth/SAML/LDAP), eliminating the need for
pre-registration.

GGID flow: User clicks "Login with Google" → Google authenticates → GGID checks
if user exists → If not, creates account with data from Google → Issues JWT.

### Step-up Authentication

**Step-up authentication** requires additional verification for sensitive
operations (e.g., changing password, deleting users) even when the user is
already logged in.

GGID implements step-up via:
1. Short-lived step-up JWT (5 min TTL)
2. Re-authentication required (password, MFA, or WebAuthn)
3. `auth_time` claim in JWT verified by Policy engine

### Back-channel Logout

**Back-channel logout** (OIDC Session Management) sends a server-to-server
logout notification to all registered applications when a user logs out,
ensuring sessions are invalidated across all clients without relying on
browser redirects.

GGID publishes logout events via:
- NATS JetStream (`session.expired` event)
- Webhook notification to registered logout URLs
- RFC-compliant back-channel logout token (JWT)
