# OAuth 2.1 Compliance Checklist

> Standards: RFC 9700 (OAuth 2.1), FAPI 2.0, NIS2/CRA | Last updated: 2026-07-15

## Overview

This document helps evaluators and administrators assess GGID's compliance with OAuth 2.1 (RFC 9700), FAPI 2.0 security profile, and relevant NIS2/CRA IAM requirements. Each item includes the RFC requirement, GGID implementation status, code references, and any gaps.

GGID includes a built-in OAuth 2.1 compliance audit endpoint:

```
GET /api/v1/oauth/oauth-2-1/audit
```

This endpoint scans all registered OAuth clients and reports per-client compliance issues, an overall compliance percentage, and remediation actions.

---

## 1. RFC 9700 Core Requirements

### 1.1 PKCE Enforcement for Authorization Code Grant

| Attribute | Value |
|-----------|-------|
| RFC Reference | RFC 9700 Section 4.1.1 (derived from RFC 7636) |
| GGID Status | **DONE** |
| Config | `RequirePKCE` per-client + `RequirePKCE` in oauth config (`conf.go`) |

**Implementation:**
- `OAuthClient.RequirePKCE` field enforces PKCE on a per-client basis.
- `OAuthClient.RequiresPKCE()` returns true for all public clients or when the flag is set.
- `AuthorizationCode.ValidatePKCE(verifier)` verifies S256 code challenge against stored value.
- Authorization codes store `code_challenge` and `code_challenge_method` (default S256).
- The OAuth 2.1 audit handler flags `public_client_without_pkce` for non-compliant clients.

**Code references:**
- `services/oauth/internal/domain/models.go` — `RequiresPKCE()`, `ValidatePKCE()`
- `services/oauth/internal/conf/conf.go` — `RequirePKCE` config flag
- `services/oauth/internal/server/oauth21_audit_handler.go` — PKCE compliance check

**Verification:**
```bash
# The audit endpoint checks PKCE enforcement
curl http://127.0.0.1:8080/api/v1/oauth/oauth-2-1/audit | jq '.compliance_checklist[0]'
```

---

### 1.2 Authorization Code Grant Retained

| Attribute | Value |
|-----------|-------|
| RFC Reference | RFC 9700 Section 4.1 (primary grant type) |
| GGID Status | **DONE** |

**Implementation:**
- Full authorization code flow: `/authorize` → code issuance → `/token` → code exchange.
- Authorization codes stored in `oauth_authorization_codes` table with hash-based lookup, expiry, and single-use enforcement.
- Supports PKCE, nonce (OIDC), and state parameters.
- `code_challenge_method` defaults to `S256` (plain is not recommended).

**Code references:**
- `services/oauth/internal/service/oauth_service.go` — `ExchangeToken()`
- `services/oauth/internal/repository/pg_repo.go` — authorization code CRUD
- `services/oauth/migrations/000001_initial_schema.up.sql` — `oauth_authorization_codes` table

---

### 1.3 Implicit Grant Removed

| Attribute | Value |
|-----------|-------|
| RFC Reference | RFC 9700 Section 7.2 (removed from OAuth 2.1) |
| GGID Status | **DONE** — Implicit grant is not implemented |

**Implementation:**
- No `response_type=token` (implicit flow) handler exists in the token endpoint.
- The OAuth 2.1 audit handler explicitly checks all clients for `implicit` in their `grant_types` and flags `implicit_grant_enabled` as a high-risk issue.
- Default client creation uses `{authorization_code, refresh_token}` — implicit is never the default.

**Verification:**
```bash
# Attempt implicit flow — should return error
curl "http://127.0.0.1:8080/api/v1/oauth/authorize?response_type=token&client_id=..."

# Audit checks for implicit grant
curl http://127.0.0.1:8080/api/v1/oauth/oauth-2-1/audit | jq '.compliance_checklist[1]'
```

**Code references:**
- `services/oauth/internal/server/oauth21_audit_handler.go` — `implicit_grant_enabled` detection
- `services/oauth/internal/repository/pg_repo.go` — default grant types exclude implicit

---

### 1.4 Resource Owner Password Credentials Grant Removed

| Attribute | Value |
|-----------|-------|
| RFC Reference | RFC 9700 Section 7.3 (removed from OAuth 2.1) |
| GGID Status | **DONE** — Password grant is not implemented in OAuth service |

**Implementation:**
- The OAuth service does not handle `grant_type=password` at the `/token` endpoint.
- Note: The Auth service (`/api/v1/auth/login`) handles username/password login directly, but this is a first-party admin console login, NOT an OAuth 2.0 password grant. It does not issue OAuth access tokens via `grant_type=password`.
- The OAuth 2.1 audit handler flags any client with `password` in `grant_types` as `password_grant_enabled` (high risk).

**Code references:**
- `services/oauth/internal/server/oauth21_audit_handler.go` — `password_grant_enabled` detection
- `services/oauth/internal/service/oauth_service.go` — no password grant handler

---

### 1.5 Refresh Token Rotation

| Attribute | Value |
|-----------|-------|
| RFC Reference | RFC 9700 Section 4.2.2 (rotating refresh tokens recommended) |
| GGID Status | **DONE** |

**Implementation:**
- Client lifecycle config defaults to `RefreshTokenRotation: "rotating"`.
- On each refresh token exchange, a new refresh token is issued and the old one is invalidated.
- Refresh tokens are scoped to the issuing client and tenant.

**Code references:**
- `services/oauth/internal/server/client_lifecycle_config_handler.go` — `RefreshTokenRotation: "rotating"`

---

### 1.6 DPoP (Demonstrating Proof-of-Possession)

| Attribute | Value |
|-----------|-------|
| RFC Reference | RFC 9449 (referenced by OAuth 2.1) |
| GGID Status | **DONE** |

**Implementation:**
- Full RFC 9449 DPoP proof JWT parsing and validation.
- `ParseDPoPHeader()` extracts and validates the DPoP proof from the `DPoP` HTTP header.
- Validates: JWT signature, `htm` (HTTP method), `htu` (HTTP URI), `jti` (unique ID), `iat` (issued at).
- DPoP config handler provides enforcement settings: `RequireDPoP`, `ProofMaxAgeSeconds` (default 60s).
- Token binding statistics track DPoP adoption per client.
- `cnf` claim in access tokens binds the token to the client's DPoP key.

**Code references:**
- `services/oauth/internal/service/dpop.go` — `ParseDPoPHeader()`, `DPoPProof` struct
- `services/oauth/internal/server/dpop_config_handler.go` — DPoP enforcement config
- `services/oauth/internal/server/token_binding_stats_handler.go` — DPoP adoption metrics

---

### 1.7 PAR (Pushed Authorization Requests)

| Attribute | Value |
|-----------|-------|
| RFC Reference | RFC 9126 (referenced by OAuth 2.1) |
| GGID Status | **DONE** |

**Implementation:**
- `PushedAuthorizationRequest` type stores request parameters server-side.
- `ValidateAuthorizationRequest()` resolves `request_uri` parameters (format: `urn:ietf:params:oauth:request_uri:<id>`).
- PAR store enables pre-validated authorization requests, reducing URL length and exposure of sensitive parameters.
- JAR (JWT-Secured Authorization Requests, RFC 9101) is also supported via the `request` parameter.

**Code references:**
- `services/oauth/internal/service/oauth_service.go` — `ValidateAuthorizationRequest()`, `PushedAuthorizationRequest`
- `services/oauth/internal/service/jar_mtls_cov_test.go` — JAR + PAR integration tests

---

### 1.8 Client Credentials Grant with Scope

| Attribute | Value |
|-----------|-------|
| RFC Reference | RFC 9700 Section 4.3 (retained from RFC 6749) |
| GGID Status | **DONE** |

**Implementation:**
- `ClientCredentials()` method issues machine-to-machine tokens.
- Validates client is confidential (not public), authenticates client, and enforces requested scopes.
- Integrates with the Policy service for scope-based authorization.

**Code references:**
- `services/oauth/internal/service/oauth_service.go` — `ClientCredentials()`, `ClientCredentialsRequest`
- `services/oauth/internal/service/coverage_boost4_test.go` — edge case tests (public client rejection, wrong grant type)

---

### 1.9 Additional Security Features

| Feature | RFC | GGID Status | Notes |
|---------|-----|-------------|-------|
| Exact redirect URI matching | RFC 9700 §4.1.3 | **DONE** | Audit checks for wildcard/insecure URIs |
| HTTPS redirect URIs required | RFC 9700 §4.1.3 | **DONE** | `non_https_redirect_uri` flagged in audit |
| Token endpoint auth methods | RFC 9700 §4.4 | **DONE** | Supports: `client_secret_basic`, `client_secret_post`, `private_key_jwt`, `tls_client_auth`, `self_signed_tls_client_auth` |
| mTLS client auth (RFC 8705) | RFC 8705 | **DONE** | `ValidateMTLSBinding()` validates `x5t#S256` thumbprint |
| JWT-Secured Auth Request (JAR) | RFC 9101 | **DONE** | `ValidateAuthorizationRequest()` handles `request` parameter |
| RFC 7523 JWT assertion profile | RFC 7523 | **DONE** | `ClientAssertionTypeRFC7523` constant + validation |
| Token Exchange (RFC 8693) | RFC 8693 | **DONE** | `TokenExchangeRequestRFC8693` struct + `ExchangeToken()` |
| Issuer identification | RFC 9700 §3.1 | **DONE** | `iss` claim in all tokens |

---

### Summary Table

| # | Requirement | RFC 9700 Reference | GGID Status |
|---|-------------|-------------------|-------------|
| 1 | PKCE enforced for authorization code | §4.1.1 | **DONE** |
| 2 | Authorization code grant retained | §4.1 | **DONE** |
| 3 | Implicit grant removed | §7.2 | **DONE** (never implemented) |
| 4 | Password grant removed | §7.3 | **DONE** (never implemented) |
| 5 | Refresh token rotation | §4.2.2 | **DONE** |
| 6 | DPoP sender-constrained tokens | RFC 9449 | **DONE** |
| 7 | Pushed Authorization Requests (PAR) | RFC 9126 | **DONE** |
| 8 | Client credentials with scope | §4.3 | **DONE** |
| 9 | mTLS client certificate auth | RFC 8705 | **DONE** |
| 10 | JWT-Secured Authorization Requests | RFC 9101 | **DONE** |
| 11 | Token Exchange | RFC 8693 | **DONE** |
| 12 | Exact HTTPS redirect URI matching | §4.1.3 | **DONE** |

**Overall RFC 9700 core compliance: 12/12 requirements met.**

---

## 2. FAPI 2.0 Security Profile Alignment

FAPI 2.0 is an OpenID Foundation security profile built on OAuth 2.1 for high-risk financial and government APIs. GGID's alignment:

| FAPI 2.0 Requirement | GGID Status | Implementation |
|----------------------|-------------|----------------|
| OAuth 2.1 as baseline | **DONE** | All requirements in Section 1 |
| PAR required for all auth requests | **DONE** | `ValidateAuthorizationRequest()` resolves `request_uri` |
| JAR required (signed authorization requests) | **DONE** | `request` parameter validation in auth endpoint |
| DPoP or mTLS required (sender-constrained tokens) | **DONE** | Both DPoP (RFC 9449) and mTLS (RFC 8705) implemented |
| No `none` algorithm for ID tokens | **DONE** | RS256 only for JWT signing |
| Authorization code must be single-use + short-lived | **DONE** | `used` flag + `expires_at` in `oauth_authorization_codes` |
| Issuer must include `iss` claim | **DONE** | Issuer identification in all tokens |
| Client metadata FAPI 2.0 flag | **DONE** | `OAuthClient.FAPI2_0()` / `SetFAPI2_0()` methods |
| FAPI config handler | **DONE** | `/api/v1/oauth/fapi-config` endpoint |
| Reject `request` + `request_uri` simultaneously | **DONE** | `ValidateAuthorizationRequest` returns error when both present |

### FAPI 2.0 Gaps

| Gap | Severity | Notes |
|-----|----------|-------|
| No mandatory JARM (JWT-Secured Authorization Response Mode) | Low | OIDC hybrid flow returns JWT fragment; standalone JARM not yet enforced |
| No FAPI 2.0 certification conformance test suite | Medium | OpenID certification testing not yet conducted |

---

## 3. NIS2 / CRA Compliance (IAM-Relevant)

### NIS2 Directive (EU 2022/2555)

NIS2 requires "essential" and "important" entities to implement risk-based cybersecurity measures. IAM-relevant articles:

| NIS2 Article | Requirement | GGID Coverage |
|--------------|-------------|---------------|
| Art. 21(1)(a) | Authentication and access control | **DONE** — RBAC + ABAC engine, MFA (TOTP/WebAuthn), password policies |
| Art. 21(1)(b) | Incident handling and audit trails | **DONE** — Audit service with hash-chain integrity, NATS JetStream events |
| Art. 21(1)(c) | Business continuity (backup/HA) | **PARTIAL** — Multi-DB support; HA depends on deployment topology |
| Art. 21(1)(f) | Secure communication (encryption in transit) | **DONE** — TLS, mTLS client auth (RFC 8705), HTTPS enforcement |
| Art. 21(1)(g) | Vulnerability disclosure and handling | **DONE** — SECURITY.md, responsible disclosure process |
| Art. 21(2)(c) | Multi-factor authentication | **DONE** — TOTP, WebAuthn (passkeys), conditional mediation |
| Art. 21(2)(j) | Human resources security (access reviews) | **PARTIAL** — User lifecycle management exists; automated periodic access review not yet built |
| Art. 23 | Incident notification (24h early warning) | **PARTIAL** — Audit events captured; automated 24h notification workflow not yet built |

### Cyber Resilience Act (CRA)

The CRA mandates security requirements for products with digital elements:

| CRA Article | Requirement | GGID Coverage |
|-------------|-------------|---------------|
| Art. 13(1)(a) | Secure by design (no known exploitable vulnerabilities) | **DONE** — `make test` with race detection, dependency scanning |
| Art. 13(1)(b) | Secure default configuration | **DONE** — PKCE on by default, HTTPS required, refresh rotation |
| Art. 13(2)(a) | Vulnerability and security issue reporting | **DONE** — SECURITY.md, CVE tracking |
| Art. 13(2)(c) | Security updates mechanism | **PARTIAL** — Rolling update scripts exist; automated OTA not built |
| Art. 13(3) | Free security updates for product lifetime | **DONE** — Open source, Apache 2.0, community-maintained |

---

## 4. Production Deployment Checklist

Before deploying GGID to production, verify the following:

### OAuth / OIDC Security

- [ ] **PKCE enforced for all public clients** — Run the audit endpoint and ensure 0 non-compliant clients
- [ ] **No implicit or password grant clients** — Audit endpoint must show `implicit: compliant` and `password: compliant`
- [ ] **All redirect URIs use HTTPS** — No wildcard or HTTP redirect URIs
- [ ] **Refresh token rotation enabled** — Verify `RefreshTokenRotation: "rotating"` in client config
- [ ] **DPoP or mTLS enforcement for high-risk clients** — Enable `RequireDPoP` or configure `tls_client_auth`
- [ ] **PAR enabled for authorization requests** — Use `request_uri` flow for sensitive clients
- [ ] **JWT signing key is RSA 2048+ or ECDSA P-256** — Verify `OAUTH_PRIVATE_KEY_PATH` key strength

### Authentication Security

- [ ] **MFA enforced for admin accounts** — TOTP or WebAuthn
- [ ] **Password policy configured** — Minimum length, complexity, expiration
- [ ] **Argon2id password hashing** — Verify default algorithm
- [ ] **Rate limiting active** — Login attempts per IP, token endpoint throttling
- [ ] **Session timeout configured** — Access + refresh token TTLs

### Network / Transport

- [ ] **TLS 1.2+ for all endpoints** — No plaintext HTTP in production
- [ ] **HTTPS-only redirect URIs** — Enforced by OAuth 2.1 audit
- [ ] **HSTS header on all responses** — Strict-Transport-Security
- [ ] **CORS properly scoped** — Only trusted origins

### Database / Data

- [ ] **Row-Level Security (RLS) enabled** — Non-superuser DB role for multi-tenant isolation
- [ ] **PII encryption at rest** — AES-256-GCM for sensitive fields
- [ ] **Audit log retention policy set** — Per compliance requirements (NIS2: minimum retention)
- [ ] **Backup and restore tested** — PostgreSQL backup + verification

### Infrastructure

- [ ] **Non-root container user** — Override Docker USER for production
- [ ] **Secrets managed externally** — Use Vault/KMS, not environment variables in containers
- [ ] **Health checks and monitoring** — Prometheus/Grafana or equivalent
- [ ] **Log aggregation** — Centralized logging for audit trail

### Compliance Audit

- [ ] **Run OAuth 2.1 audit** — `GET /api/v1/oauth/oauth-2-1/audit` — must be 100% compliant
- [ ] **Review FAPI 2.0 clients** — Verify `fapi_2_0` metadata on financial/government clients
- [ ] **Export audit events** — Verify NATS JetStream → SIEM pipeline
- [ ] **Test incident response** — Verify audit trail can reconstruct a security event

---

## 5. Audit Endpoint Reference

### OAuth 2.1 Compliance Audit

```
GET /api/v1/oauth/oauth-2-1/audit
```

**Response structure:**
```json
{
  "compliance_checklist": [
    {
      "requirement": "PKCE required for all public clients",
      "status": "compliant",
      "detail": "All public clients enforce PKCE",
      "remediation_url": "/docs/oauth-2-1-migration"
    },
    {
      "requirement": "Implicit grant disabled",
      "status": "compliant",
      "detail": "No clients use implicit flow"
    },
    {
      "requirement": "Password grant disabled",
      "status": "compliant",
      "detail": "No clients use password grant"
    }
  ],
  "overall_compliance_pct": 100.0,
  "non_compliant_clients": [],
  "total_clients_audited": 5
}
```

### Remediation

For any non-compliant client, the audit response includes:
- `non_compliant_clients[]` — per-client issues and risk level
- `remediation_actions[]` — actionable steps to fix each issue

**Common remediations:**
| Issue | Action |
|-------|--------|
| `public_client_without_pkce` | Enable `RequirePKCE` on the client |
| `implicit_grant_enabled` | Remove `implicit` from client `grant_types` |
| `password_grant_enabled` | Remove `password` from client `grant_types` |
| `non_https_redirect_uri` | Change redirect URI to HTTPS |
| `wildcard_redirect_uri` | Use exact URI, no wildcards |
| `invalid_token_endpoint_auth_method` | Use `client_secret_post`, `client_secret_basic`, or `private_key_jwt` |

---

## 6. Code References

| Component | File | Purpose |
|-----------|------|---------|
| OAuth 2.1 audit handler | `services/oauth/internal/server/oauth21_audit_handler.go` | Runtime compliance scanning |
| OAuth client domain | `services/oauth/internal/domain/models.go` | `OAuthClient`, `AuthorizationCode`, PKCE validation |
| OAuth service | `services/oauth/internal/service/oauth_service.go` | Token exchange, client credentials, authorization |
| DPoP | `services/oauth/internal/service/dpop.go` | RFC 9449 proof parsing and validation |
| JAR / PAR | `services/oauth/internal/service/oauth_service.go` | `ValidateAuthorizationRequest()`, `PushedAuthorizationRequest` |
| mTLS binding | `services/oauth/internal/service/oauth_service.go` | `ValidateMTLSBinding()` |
| RFC 7523 | `services/oauth/internal/service/rfc7523.go` | JWT assertion profile |
| FAPI 2.0 config | `services/oauth/internal/server/fapi_config_handler.go` | FAPI profile settings |
| DPoP config | `services/oauth/internal/server/dpop_config_handler.go` | DPoP enforcement settings |
| Client lifecycle | `services/oauth/internal/server/client_lifecycle_config_handler.go` | Refresh token rotation config |
| DB schema | `services/oauth/migrations/000001_initial_schema.up.sql` | Tables, RLS policies, indexes |

---

## 7. Related Documents

- `docs/research/oauth-security-recommendations-bcp.md` — OAuth security best practices
- `docs/research/edge-computing-iam.md` — Edge deployment IAM patterns
- `docs/research/pqc-post-quantum-cryptography.md` — Post-quantum readiness
- `docs/operations-runbook.md` — Production operations guide
- `docs/deployment-guide.md` — Deployment options (Compose, K8s, Helm)
