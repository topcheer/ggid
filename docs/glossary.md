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

## X

### X-API-Key
An HTTP header used by the GGID SDK for server-to-server authentication with management APIs.

### X-Correlation-ID
An HTTP header for distributed tracing correlation across microservices.

### X-Tenant-ID
An HTTP header identifying the tenant context for a request. Required for all API calls through the GGID gateway.

---

*Last updated: Phase 10 — Enterprise Features*
