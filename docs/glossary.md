# GGID Glossary

A comprehensive glossary of Identity and Access Management (IAM) terms used throughout the GGID platform.

---

## A

### ABAC (Attribute-Based Access Control)
An access control model where permissions are granted based on attributes of the user, resource, and environment (e.g., department, location, time of day). GGID implements ABAC alongside RBAC in its policy engine.

### Access Token
A credential that grants access to a protected resource. In GGID, access tokens are JWTs signed by the OAuth service, containing claims like `sub`, `tenant_id`, `roles`, and `scope`.

### ACID
A set of transaction properties (Atomicity, Consistency, Isolation, Durability) guaranteeing reliable database operations. GGID uses PostgreSQL with ACID-compliant transactions for all data modifications.

### ACR (Authentication Context Class Reference)
An OIDC parameter indicating the level of authentication assurance (e.g., `urn:mace:incommon:iap:silver`). GGID supports ACR-based step-up authentication.

### AES-256-GCM
Advanced Encryption Standard with 256-bit keys in Galois/Counter Mode. GGID uses AES-256-GCM for encrypting sensitive data at rest via `pkg/crypto`.

### Argon2id
The winner of the Password Hashing Competition (2015). A memory-hard hashing algorithm resistant to GPU/ASIC attacks. GGID uses Argon2id with parameters: 64MB memory, 3 iterations, 2 parallelism.

### Audit Event
A record of a security-relevant action (e.g., user.login, role.assign). GGID publishes audit events via NATS JetStream and persists them for compliance.

### Authentication
The process of verifying the identity of a user or service. GGID supports: password, MFA (TOTP), LDAP, OAuth/OIDC, SAML, WebAuthn, social login.

### Authorization
The process of determining what actions an authenticated user is allowed to perform. GGID implements RBAC + ABAC policy enforcement.

---

## B

### Back-Channel Logout
A server-to-server logout mechanism (OIDC Back-Channel Logout 1.0) where the OP sends a logout token to RPs via HTTP POST. GGID implements `POST /oauth/logout`.

### Backup Eligible
A WebAuthn flag indicating whether a credential can be synced via cloud services (iCloud Keychain, Google Password Manager). GGID stores this on the Credential struct.

---

## C

### CAEP (Continuous Access Evaluation Profile)
A Shared Signals Framework event type for real-time security events (session revoked, credential change). GGID's audit pipeline supports CAEP event publication.

### Circuit Breaker
A resilience pattern that prevents cascading failures by temporarily blocking calls to a failing service. GGID's gateway middleware implements circuit breaking with configurable thresholds.

### Claims
Key-value pairs in a JWT that provide information about the user (e.g., `sub`, `email`, `roles`). GGID allows custom claim customization via rules.

### Client Credentials Grant
An OAuth 2.0 grant type for machine-to-machine authentication. GGID supports `grant_type=client_credentials` with secret rotation.

### Clone Detection
A WebAuthn security feature that detects if a credential has been cloned by checking sign count monotonicity. GGID rejects authentication if `signCount <= stored signCount`.

### Consent
User approval for an application to access specific scopes. GGID implements an OAuth consent screen for interactive flows.

---

## D

### Device Flow (RFC 8628)
An OAuth 2.0 extension for devices with limited input capabilities (TVs, IoT). Users authorize on a separate device. GGID implements device authorization endpoints.

### DPoP (RFC 9449)
Demonstrating Proof-of-Possession at the Application Layer. Binds access tokens to a sender's cryptographic key. GGID research identifies DPoP as a future enhancement.

---

## E

### External ID
An identifier from an external system (e.g., SCIM `externalId`, LDAP `dn`). GGID's SCIM implementation supports externalId mapping.

---

## F

### FIDO2
An authentication standard developed by the FIDO Alliance, comprising WebAuthn (web API) and CTAP (client-to-authenticator protocol). GGID implements FIDO2 via WebAuthn.

---

## G

### Grant Type
The method by which a client obtains an OAuth access token (e.g., authorization_code, client_credentials, refresh_token, device_code).

---

## I

### IdP (Identity Provider)
A service that authenticates users and issues identity assertions. GGID acts as both an IdP and an SP (Service Provider).

### IdP-Initiated SSO
An SSO flow where authentication starts at the IdP, which then redirects to the SP. GGID supports both SP-initiated and IdP-initiated SAML SSO.

### IoT (Internet of Things)
Network-connected devices. GGID's device flow grant is designed for IoT authentication scenarios.

### Issuer
The URL that identifies the token-issuing party. In GGID, the issuer is the OAuth service URL (e.g., `https://auth.ggid.dev`).

### JIT Provisioning (Just-In-Time)
Automatic user account creation upon first SSO login. GGID creates user accounts on-the-fly from SAML/OIDC assertion data.

---

## J

### JWT (JSON Web Token)
A compact, URL-safe token format (RFC 7519). GGID issues JWTs for access tokens and ID tokens, signed with RS256.

### JWKS (JSON Web Key Set)
An endpoint exposing public keys for JWT verification (RFC 7517). GGID exposes `/.well-known/jwks.json`.

---

## K

### Key Rotation
The practice of periodically replacing cryptographic keys. GGID supports JWT signing key rotation and OAuth client secret rotation.

---

## L

### LDAP (Lightweight Directory Access Protocol)
A protocol for accessing directory services. GGID integrates with LDAP/Active Directory for credential verification and group-to-role mapping.

---

## M

### MFA (Multi-Factor Authentication)
Requiring more than one authentication factor. GGID supports TOTP (RFC 6238), WebAuthn as a second factor, and backup codes.

### Magic Link
A passwordless authentication method where a one-time-use link is sent via email. GGID implements magic link authentication.

---

## N

### NATS JetStream
A high-performance messaging system with persistence. GGID uses NATS JetStream for audit event delivery and webhooks.

### Nonce
A single-use random value to prevent replay attacks. GGID enforces nonce validation in OAuth flows.

---

## O

### OIDC (OpenID Connect)
An identity layer on top of OAuth 2.0. GGID implements OIDC Core 1.0 with ID tokens, userinfo, and discovery endpoints.

### OAuth 2.0
An authorization framework (RFC 6749) for delegated access. GGID implements authorization code, client credentials, refresh token, and device flows.

### OAuth 2.1
The upcoming consolidated OAuth specification. GGID's research identifies gaps (PKCE not enforced, refresh token rotation needed) for future compliance.

---

## P

### Passkey
A FIDO2/WebAuthn credential that replaces passwords. GGID implements passkey registration and authentication with backup eligibility tracking.

### PKCE (Proof Key for Code Exchange)
An OAuth 2.0 extension (RFC 7636) protecting authorization code flow from interception. GGID supports PKCE with S256 challenge method.

### Policy Engine
The component that evaluates access control decisions. GGID's policy engine supports RBAC role checks and ABAC attribute rules.

### PII (Personally Identifiable Information)
Data that can identify an individual. GGID includes `pkg/pii` for PII detection and redaction in logs.

### PRF Extension
A WebAuthn extension for deriving cryptographic secrets from authenticators. Useful for end-to-end encrypted data.

---

## R

### RBAC (Role-Based Access Control)
An access control model where permissions are assigned to roles, and users are assigned roles. GGID implements hierarchical RBAC with effective permission computation.

### Refresh Token
A long-lived token used to obtain new access tokens without re-authentication. GGID issues refresh tokens with rotation.

### Resident Key
A WebAuthn credential stored on the authenticator (discoverable credential), enabling usernameless login. GGID configures `residentKey: preferred`.

### RLS (Row-Level Security)
A PostgreSQL feature enforcing data isolation at the database level. GGID uses RLS policies to enforce tenant isolation.

### RP (Relying Party)
A service that relies on an IdP for authentication. In WebAuthn, the RP is the website. GGID's WebAuthn RP ID is configurable.

---

## S

### SAML 2.0
An XML-based SSO protocol (Security Assertion Markup Language). GGID implements SP-initiated and IdP-initiated SAML SSO with per-tenant metadata.

### SCIM 2.0
A provisioning protocol (System for Cross-domain Identity Management, RFC 7643/7644). GGID implements SCIM user and group CRUD with filter parsing.

### Sign Count
A WebAuthn authenticator counter incremented on each use. GGID checks sign count monotonicity for clone detection.

### SP-Initiated SSO
An SSO flow where authentication starts at the Service Provider, which redirects to the IdP. GGID supports SP-initiated SAML SSO.

### Step-Up Authentication
Requiring additional authentication for sensitive operations. GGID implements ACR-based step-up authentication.

### Subject Identifier
A unique identifier for the authenticated user (the `sub` claim in JWT).

---

## T

### Tenant
An isolated organizational unit in multi-tenant GGID deployments. Each tenant has independent users, roles, policies, and IdP configurations.

### TOTP (Time-based One-Time Password)
An RFC 6238 algorithm generating 6-digit codes from a shared secret and timestamp. GGID implements TOTP for MFA.

### Token Exchange (RFC 8693)
An OAuth 2.0 extension for exchanging one token type for another (e.g., JWT for access token). GGID research identifies this for future implementation.

### Token Revocation (RFC 7009)
An endpoint for invalidating access and refresh tokens. GGID implements `POST /oauth/revoke`.

---

## V

### Verification Flow
The process of verifying ownership of an email or phone number. GGID implements email verification with signed tokens.

---

## W

### WebAuthn
A W3C standard for passwordless authentication using public-key cryptography. GGID implements WebAuthn registration and authentication with backup flags, clone detection, and transport persistence.

### Webhook
An HTTP callback triggered by events. GGID's gateway supports configurable webhooks for audit events.

---

## A

### ABAC (Attribute-Based Access Control)
An access control model where authorization decisions are based on attributes (properties) of the user, resource, action, and environment. Unlike RBAC which uses roles, ABAC evaluates policies like "allow if user.department == resource.department AND user.clearance >= resource.classification." GGID supports both RBAC and ABAC in its policy engine.

### Access Token
A short-lived credential (typically 15 minutes) that grants access to protected resources. In GGID, access tokens are RS256-signed JWTs containing the user's identity, tenant, roles, and scopes.

### Account Lockout
A security mechanism that temporarily disables an account after a configurable number of failed login attempts (default: 5). Prevents brute-force password attacks. Lockouts auto-expire after a cooldown period (default: 15 minutes).

### Argon2id
The winner of the 2015 Password Hashing Competition. A memory-hard hashing algorithm resistant to GPU and ASIC attacks. GGID uses Argon2id as its recommended password hashing algorithm, alongside bcrypt.

---

## B

### bcrypt
A password hashing function based on the Blowfish cipher. Includes a configurable cost factor that controls computation time. GGID uses bcrypt with cost 12 by default.

### Bearer Token
An access token type where possession of the token is sufficient for authentication (no proof of key possession required). Sent in the `Authorization: Bearer <token>` header.

---

## C

### CAS (Central Authentication Service)
A single sign-on protocol developed by Yale University. GGID can act as a CAS-compatible identity provider for legacy integrations.

### Claim
A key-value pair in a JWT that represents an assertion about the subject. Common claims: `sub` (subject/user ID), `iss` (issuer), `exp` (expiration), `tenant_id`, `scope`.

### Client Credentials Grant
An OAuth 2.0 grant type for machine-to-machine authentication. A client exchanges its ID and secret for an access token without user involvement. Used by GGID for service-to-service authentication.

### CORS (Cross-Origin Resource Sharing)
A browser security mechanism that controls which origins can make requests to a server. GGID's Gateway enforces configurable CORS policies per tenant.

### CSP (Content Security Policy)
An HTTP response header that restricts which resources (scripts, styles, images) the browser is allowed to load. GGID sets strict CSP headers with per-request nonces to prevent XSS.

### CSRF (Cross-Site Request Forgery)
An attack where a malicious website tricks a user's browser into making an unwanted request to a different site. GGID mitigates CSRF via SameSite cookies and JWT-in-header authentication.

---

## D

### DPoP (Demonstrating Proof-of-Possession)
RFC 9449. An OAuth 2.0 extension that binds access tokens to a cryptographic key held by the client. Prevents token replay even if the token is stolen. GGID supports DPoP as an optional enhancement.

### Discoverable Credential
A WebAuthn term for credentials stored on the authenticator that can be used without the server providing a credential list. Also called "passkeys." Enables usernameless authentication.

---

## E

### End-User
In OIDC terminology, the human participant who authenticates with the identity provider.

---

## F

### FIDO2
A set of authentication standards from the FIDO Alliance, comprising WebAuthn (browser API) and CTAP (client-to-authenticator protocol). Enables passwordless authentication using security keys and platform authenticators.

### Forward-Only Migration
A database migration strategy where migrations can only be applied in the forward direction. No automatic rollback. GGID uses forward-only migrations to ensure schema consistency.

---

## G

### gRPC
A high-performance RPC framework using Protocol Buffers. GGID uses gRPC for internal service-to-service communication (identity, policy, org, audit).

---

## I

### IdP (Identity Provider)
A system that manages user identities and provides authentication services to relying parties. GGID functions as an IdP with support for OIDC, SAML, and CAS protocols.

### IdP-Initiated SSO
An SSO flow where authentication starts at the identity provider. The user logs in at the IdP and then selects a service provider to access. Contrast with SP-initiated SSO.

### Introspection (Token)
RFC 7662. An OAuth 2.0 endpoint that allows a resource server to check the validity and metadata of an access token. GGID exposes `POST /oauth/introspect`.

### IoT (Internet of Things)
Network of physical devices. GGID can authenticate IoT devices via device credentials and OAuth client credentials.

---

## J

### JIT Provisioning (Just-in-Time Provisioning)
A pattern where user accounts are created automatically on first login from an external identity provider (e.g., SAML, social login). GGID supports JIT provisioning for LDAP and social connectors.

### JWT (JSON Web Token)
RFC 7519. A compact, URL-safe token format for representing claims between parties. GGID uses RS256-signed JWTs for access and refresh tokens.

---

## K

### Key Rotation
The practice of periodically replacing cryptographic keys. GGID publishes public keys via JWKS and supports zero-downtime key rotation with an overlap window.

### PKCE (Proof Key for Code Exchange)
RFC 7636. An OAuth 2.0 extension that prevents authorization code interception attacks. The client proves possession of the code verifier. GGID requires PKCE for all public clients (SPAs, mobile apps).

---

## L

### LDAP (Lightweight Directory Access Protocol)
A protocol for accessing directory services. GGID can authenticate users against LDAP directories (OpenLDAP, Active Directory) as an alternative to local password storage.

### Lockout
See [Account Lockout](#account-lockout).

---

## M

### MFA (Multi-Factor Authentication)
Requiring two or more authentication factors: something you know (password), something you have (phone, security key), something you are (biometric). GGID supports TOTP, WebAuthn, and email-based MFA.

---

## N

### NATS JetStream
A persistent streaming system built into NATS. GGID uses JetStream for the audit event pipeline, providing at-least-once delivery and durability.

### Nonce
A single-use random value. Used in WebAuthn challenges, CSP script-src directives, and OIDC implicit flow to prevent replay attacks.

---

## O

### OAuth 2.0
RFC 6749. An authorization framework that allows third-party applications to obtain limited access to a user's account. GGID implements OAuth 2.0 as a provider.

### OAuth 2.1
A draft specification that consolidates OAuth 2.0 best practices. Deprecates implicit grant, requires PKCE for all authorization code flows, and mandates exact redirect_uri matching. GGID's OAuth implementation aligns with OAuth 2.1 recommendations.

### OIDC (OpenID Connect)
A layer on top of OAuth 2.0 that adds user authentication. Provides ID tokens, UserInfo endpoint, and standardized discovery. GGID is a certified OIDC provider.

---

## P

### Passkey
A WebAuthn discoverable credential that syncs across a user's devices via the platform's sync fabric (iCloud Keychain, Google Password Manager). Enables passwordless, phishing-resistant authentication.

### PII (Personally Identifiable Information)
Data that can identify an individual (email, phone, SSN). GGID automatically redacts PII in logs and audit events.

### Platform Authenticator
A WebAuthn authenticator built into the user's device (Touch ID, Face ID, Windows Hello, Android fingerprint). Contrast with roaming authenticators (security keys).

### Policy Engine
The GGID component that evaluates access decisions based on RBAC and ABAC rules. Exposed via REST and gRPC APIs.

---

## R

### RBAC (Role-Based Access Control)
An access control model where permissions are assigned to roles, and users are assigned roles. Simpler than ABAC but less flexible. GGID supports hierarchical RBAC with role inheritance.

### Refresh Token
A long-lived credential (typically 7 days) used to obtain new access tokens without requiring re-authentication. GGID refresh tokens are one-time-use and rotated on each refresh.

### Resident Key
A WebAuthn credential stored on the authenticator device. Enables discoverable credentials (passkeys). Contrast with non-resident keys, which require server-side credential lists.

### RLS (Row-Level Security)
A PostgreSQL feature that enforces per-row access control at the database level. GGID uses RLS to enforce tenant isolation — even if application code omits tenant_id filters.

### RP (Relying Party)
In WebAuthn/FIDO2, the application or service that the user is authenticating to. GGID acts as the RP for WebAuthn credential registration and verification.

---

## S

### SAML (Security Assertion Markup Language)
An XML-based SSO protocol. GGID can act as a SAML identity provider (IdP) for enterprise applications. Supports SP-initiated and IdP-initiated SSO.

### SCIM (System for Cross-domain Identity Management)
RFC 7643/7644. A standardized REST API for user provisioning and deprovisioning. GGID implements SCIM 2.0 for integration with HR systems and identity platforms.

### SP (Service Provider)
In SAML, the application that relies on the identity provider for authentication. The SP consumes SAML assertions issued by the IdP.

### SP-Initiated SSO
An SSO flow where authentication starts at the service provider. The user accesses the SP, which redirects to the IdP for login, then back to the SP with an assertion. Contrast with IdP-initiated SSO.

### Step-Up Authentication
Requiring a stronger authentication factor for sensitive operations (e.g., requiring MFA to change account settings, even if the initial login used only a password). GGID supports step-up auth via the `/api/v1/auth/stepup` endpoint.

### Subject Identifier (sub)
A unique, stable identifier for the end-user in a JWT or OIDC token. In GGID, this is the user's UUID.

---

## T

### TOTP (Time-based One-Time Password)
RFC 6238. A temporary code (typically 6 digits) generated from a shared secret and the current time. Used as a second factor in MFA. GGID implements TOTP compatible with Google Authenticator, Microsoft Authenticator, and Authy.

### Tenant
An isolated workspace within a multi-tenant GGID deployment. Each tenant has its own users, roles, organizations, policies, and branding. Tenant isolation is enforced at application, database (RLS), and Redis key levels.

### Token Binding
Associating an access token with a cryptographic proof of possession (e.g., DPoP, TLS certificate) so that stolen tokens cannot be replayed.

### Trace Context
W3C standard for distributed tracing. GGID propagates `traceparent` headers across all services for end-to-end request tracing.

---

## V

### Verifiable Credential
A W3C standard format for cryptographically verifiable digital credentials. GGID can issue verifiable credentials as a future enhancement.

---

## W

### WebAuthn (Web Authentication)
W3C standard for passwordless authentication using public-key cryptography. Implemented in all modern browsers. GGID supports WebAuthn for registration, authentication, and credential management.

### Webhook
An HTTP callback triggered by events. GGID's Gateway supports configurable webhooks for audit events, with HMAC signature verification.

---

## X

### X-API-Key
An HTTP header used by the GGID SDK for server-to-server authentication with management APIs.

### X-Correlation-ID
An HTTP header for distributed tracing correlation across microservices.

### X-Tenant-ID
An HTTP header identifying the tenant context for a request. Required for all API calls through the GGID gateway.

---

## J

### JAR (JWT-Secured Authorization Request)
An OAuth 2.0 extension (RFC 9101) that encodes authorization request parameters in a signed JWT instead of plain URL query parameters. Provides integrity protection, prevents parameter tampering, and enables request authentication. GGID supports JAR alongside PAR for high-security deployments.

---

## O

### OID4VC (OpenID for Verifiable Credentials)
A family of OpenID specifications enabling issuance and verification of W3C Verifiable Credentials (VCs) using OAuth 2.0 flows. Includes OID4VCI (issuance) and OID4VP (presentation). GGID's roadmap includes OID4VC support for wallet-based decentralized identity.

---

## P

### PAR (Pushed Authorization Requests)
An OAuth 2.0 extension (RFC 9126) where the client sends authorization parameters to a pushed authorization request endpoint (`/oauth/par`) and receives a `request_uri`. The authorization endpoint then receives only the `request_uri`, keeping sensitive parameters off the URL. PAR + JAR together provide maximum request security.

---

## R

### RISC (Risk and Incident Sharing via Event Streams)
An OpenID Foundation standard (now part of CAEP/SSF) for sharing security events between providers. GGID can publish and consume RISC events (e.g., account disabled, credential changed) to trigger cross-provider session revocation.

---

## S

### SSE (Server-Sent Events)
A standard HTTP protocol (HTML5 EventSource API) for streaming server-to-client updates over a persistent connection. GGID uses SSE for real-time audit event streaming at `GET /api/v1/audit/stream`, delivering events as they are published to NATS JetStream.

---

*Last updated: Phase 10 — Enterprise Features*
