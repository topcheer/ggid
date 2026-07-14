# STRIDE Threat Model Analysis for GGID

## Overview

This document provides a comprehensive STRIDE threat model analysis of the GGID IAM Suite architecture. STRIDE is a Microsoft threat classification framework covering six threat categories: **S**poofing, **T**ampering, **R**epudiation, **I**nformation Disclosure, **D**enial of Service, and **E**levation of Privilege.

## Architecture Attack Surface

```
Internet → [API Gateway] → [Identity/Auth/OAuth/Policy/Org/Audit Services]
               ↓
         [PostgreSQL] [Redis] [NATS JetStream] [OpenLDAP]
```

### Trust Boundaries

| Boundary | Components                                    | Protocol          |
|----------|-----------------------------------------------|-------------------|
| Internet ↔ Gateway | Client → API Gateway                | HTTPS/TLS         |
| Gateway ↔ Services | Gateway → Microservices              | gRPC (TLS)        |
| Service ↔ Data     | Services → PostgreSQL/Redis/NATS    | TCP (TLS optional)|
| Service ↔ LDAP     | Auth → OpenLDAP                     | LDAP (START_TLS)  |

---

## S — Spoofing

### S1: JWT Token Forgery

**Threat**: Attacker forges or tampers with a JWT to impersonate another user.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | API Gateway, Auth Service                           |
| Attack Vector | Crafting JWT with forged signature                  |
| Likelihood    | Low                                                 |
| Impact        | Critical — full account takeover                    |
| Mitigation    | RS256/ES256 signing with asymmetric keys; JWT secret must be non-empty (`log.Fatal` on empty) |
| Status        | **MITIGATED** — JWT secret validation on startup (commit fc20c41 era), jti anti-replay with Redis SETNX |
| Remaining     | Consider short-lived access tokens (5 min) + refresh token rotation |

### S2: OAuth State CSRF

**Threat**: Attacker tricks user into completing OAuth flow initiated by attacker.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | OAuth Service                                       |
| Attack Vector | OAuth authorization code interception via CSRF     |
| Mitigation    | Redis-backed state parameter validation on token exchange; `iss` parameter in auth redirect |
| Status        | **MITIGATED** — State validation (commit 72edaa5)   |

### S3: LDAP Credential Spoofing

**Threat**: Rogue LDAP server spoofs directory to capture credentials.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Auth Service, LDAP Provider                         |
| Mitigation    | START_TLS support; configurable LDAP_URL; LDAP_BIND_DN verification |
| Status        | **PARTIALLY MITIGATED** — START_TLS supported but not enforced by default |

### S4: Tenant ID Spoofing

**Threat**: User sets `X-Tenant-ID` header to access another tenant's data.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Mitigation    | JWT `tenant_id` claim takes priority over `X-Tenant-ID` header; RLS enforcement at DB level |
| Status        | **MITIGATED** — Tenant claim priority (commit 5bcbfce) |

---

## T — Tampering

### T1: Database Tampering via SQL Injection

**Threat**: Attacker injects SQL through API parameters.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | All services (identity, auth, policy, org, audit)   |
| Mitigation    | Parameterized queries (pgx v5); `SET LOCAL` uses validated UUID input |
| Status        | **MITIGATED** — All queries use parameterized pgx   |

### T2: Audit Log Tampering

**Threat**: Attacker modifies audit events to cover tracks.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Audit Service                                       |
| Attack Vector | Direct DB modification of audit_events table        |
| Mitigation    | Audit hash chain (commit with hash_chain.go); append-only table; hash verification on read |
| Status        | **MITIGATED** — Hash chain implemented              |

### T3: Webhook Payload Tampering

**Threat**: MITM attacker modifies webhook delivery payload.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Webhook HTTP deliverer                              |
| Mitigation    | HMAC-SHA256 signature in webhook header             |
| Status        | **MITIGATED** — Signature included in all deliveries |

### T4: Configuration Tampering

**Threat**: Attacker modifies service configuration at runtime.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Mitigation    | Configuration loaded at startup from env vars; no runtime config API |
| Status        | **MITIGATED** — Immutable config after startup      |

---

## R — Repudiation

### R1: User Denies Authentication Event

**Threat**: User denies logging in or performing an action.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Auth Service, Audit Service                         |
| Mitigation    | Every auth event (login, logout, MFA) publishes to NATS → audit log; JWT jti tracking; hash chain |
| Status        | **MITIGATED** — Full audit trail with non-repudiation via hash chain |

### R2: Admin Denies Policy Change

**Threat**: Administrator denies modifying RBAC/ABAC policies.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Policy Service                                      |
| Mitigation    | All policy CRUD operations logged to audit service with actor ID, timestamp, and diff |
| Status        | **MITIGATED** — Policy changes audited              |

### R3: API Call Attribution

**Threat**: Caller denies making an API request.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Mitigation    | JWT contains subject (sub), issued-at (iat), and jti (unique ID); all logged |
| Status        | **MITIGATED** — JWT jti provides per-request attribution |

---

## I — Information Disclosure

### I1: PII Exposure in Audit Logs

**Threat**: Sensitive user data (email, phone) exposed in audit events.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Audit Service                                       |
| Mitigation    | `pii.Obfuscate` masks email/phone/SSN in audit payloads |
| Status        | **PARTIALLY MITIGATED** — Obfuscation code exists, must verify it is wired into the audit pipeline |

### I2: Cross-Tenant Data Leakage

**Threat**: User accesses data belonging to another tenant.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | All tenant-scoped services                          |
| Mitigation    | PostgreSQL Row-Level Security (RLS) with `FORCE ROW LEVEL SECURITY`; tenant_id in JWT claim; gateway-level tenant injection |
| Status        | **MITIGATED** — Multi-layer (JWT claim + RLS + gateway injection) |

### I3: Error Message Information Leakage

**Threat**: Stack traces or internal details leaked in API error responses.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Mitigation    | Structured error responses with standard error codes; no stack traces in production |
| Status        | **MITIGATED** — Error reference catalog (docs/api/error-reference.md) |

### I4: SSRF via Webhook URLs

**Threat**: Attacker configures webhook URL pointing to internal services (e.g., `http://169.254.169.254/`).

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Webhook HTTP deliverer                              |
| Mitigation    | Private IP range blocking; URL allowlist; SSRF protection middleware |
| Status        | **MITIGATED** — SSRF protection implemented         |

### I5: Credential Exposure in Transit

**Threat**: Attacker intercepts credentials via network sniffing.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Mitigation    | TLS for all external traffic; gRPC TLS between services (commit 6a0eced) |
| Status        | **MITIGATED** — gRPC TLS implemented for policy/org; remaining services should follow |

---

## D — Denial of Service

### D1: Login Rate Limiting Bypass

**Threat**: Brute-force attack overwhelming the auth service.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Auth Service, API Gateway                           |
| Mitigation    | Rate limiter (token bucket) wired into production handler chain; account lockout after N failures |
| Status        | **MITIGATED** — Rate limiter wired (commit fc20c41); account lockout implemented |

### D2: NATS JetStream Exhaustion

**Threat**: Attacker floods NATS with events, exhausting JetStream storage.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Audit Service, NATS                                 |
| Mitigation    | JetStream max-age and max-bytes limits; rate limiting at gateway |
| Status        | **PARTIALLY MITIGATED** — Gateway rate limiting covers most vectors; JetStream retention limits should be configured |

### D3: Database Connection Exhaustion

**Threat**: Attacker opens many concurrent requests, exhausting connection pool.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Mitigation    | pgxpool MaxConns limits per service; PgBouncer for multi-instance; gateway rate limiting |
| Status        | **MITIGATED** — Connection pooling + rate limiting  |

### D4: DNS Rebinding Attack

**Threat**: Attacker bypasses host-based access controls via DNS rebinding.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Mitigation    | Host header validation middleware (`host_validation.go`) |
| Status        | **MITIGATED** — Host validation implemented          |

---

## E — Elevation of Privilege

### E1: JWT Scope Escalation

**Threat**: User modifies their JWT to include admin scopes.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | API Gateway, all services                           |
| Mitigation    | RS256/ES256 asymmetric signing — private key never on gateway; `HasScope()` enforces actual scope check (fixed from always-true); admin API scope check via `hasAdminScope()` |
| Status        | **MITIGATED** — Scope enforcement (commit 66ef1db, 72edaa5) |

### E2: OAuth Introspection Without Authentication

**Threat**: Unauthenticated introspection endpoint leaks token validity and claims.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | OAuth Service                                       |
| Attack Vector | Direct introspection endpoint call without client auth |
| Mitigation    | Require client authentication on introspection endpoint |
| Status        | **OPEN** — Introspection endpoint lacks authentication (P0 outstanding) |

### E3: Privilege Escalation via SCIM

**Threat**: SCIM client modifies user roles or group membership outside their scope.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | Identity Service (SCIM)                             |
| Mitigation    | SCIM PATCH operations validated against RBAC; user provisioning limited by tenant scope |
| Status        | **PARTIALLY MITIGATED** — SCIM PATCH fixed (URN colon notation, sub-attribute parsing); full RBAC enforcement on SCIM operations needed |

### E4: AI Agent Token Delegation Abuse

**Threat**: AI agent exceeds delegated scope or delegation depth.

| Field         | Value                                               |
|---------------|-----------------------------------------------------|
| Component     | OAuth Service (Agent Identity)                      |
| Mitigation    | `max_delegation_depth` in agent token claims; delegation chain verification; per-agent scope enforcement |
| Status        | **MITIGATED** — Agent identity with delegation chain (commit 55ffd6f) |

---

## Summary Scorecard

| Category             | Threats | Mitigated | Partial | Open |
|----------------------|---------|-----------|---------|------|
| Spoofing (S)         | 4       | 3         | 1       | 0    |
| Tampering (T)        | 4       | 4         | 0       | 0    |
| Repudiation (R)      | 3       | 3         | 0       | 0    |
| Information Disclosure (I) | 5 | 4         | 1       | 0    |
| Denial of Service (D) | 4      | 3         | 1       | 0    |
| Elevation (E)        | 4       | 2         | 1       | 1    |
| **Total**            | **24**  | **19**    | **4**   | **1**|

**Overall STRIDE Score: ~7.9/10** (target: 8.0+)

## Priority Remediation

| Priority | Threat                          | Action                                     |
|----------|---------------------------------|--------------------------------------------|
| P0       | E2: Introspection auth          | Add client authentication to introspect EP |
| P1       | I1: PII obfuscation wiring      | Verify pii.Obfuscate is called in pipeline |
| P1       | D2: NATS retention limits       | Configure JetStream max-age/max-bytes      |
| P2       | S3: LDAP START_TLS enforcement  | Make START_TLS required by default         |
| P2       | E3: SCIM RBAC enforcement       | Full scope check on all SCIM operations    |

## See Also

- Security Audit Checklist
- [Security Overview Architecture](../architecture/security-overview.md)
- P0 Security Fixes
- Rate Limiting Guide
