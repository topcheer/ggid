# Security Architecture

> This document describes the security architecture of the GGID IAM platform: trust boundaries, defense-in-depth layers, authentication flows, tenant isolation, audit pipeline, and incident response integration.

---

## Table of Contents

1. [Threat Model Overview](#threat-model-overview)
2. [Trust Boundaries](#trust-boundaries)
3. [Defense in Depth Layers](#defense-in-depth-layers)
4. [JWT Verification Flow](#jwt-verification-flow)
5. [Tenant Isolation Architecture](#tenant-isolation-architecture)
6. [Audit Pipeline Security](#audit-pipeline-security)
7. [Secrets Management](#secrets-management)
8. [Transport Security](#transport-security)
9. [Input Validation](#input-validation)
10. [Incident Response Integration](#incident-response-integration)
11. [Security Hardening Checklist](#security-hardening-checklist)

---

## Threat Model Overview

GGID uses the STRIDE threat modeling framework:

| Threat Category | Mitigation |
|----------------|------------|
| **Spoofing** | JWT verification, mTLS (planned), password hashing |
| **Tampering** | RLS policies, input validation, audit trail |
| **Repudiation** | Audit logging on all requests, JTI anti-replay |
| **Information Disclosure** | Tenant isolation, RLS, scoped tokens |
| **Denial of Service** | Rate limiting, circuit breakers, request size limits |
| **Elevation of Privilege** | Scope checks, admin API guards, RBAC + ABAC |

### Trust Levels

```
┌─────────────────────────────────────────────────────┐
│                  UNTRUSTED ZONE                      │
│  (Internet, unauthenticated requests)                │
│                                                      │
│  ┌─────────────────────────────────────────────┐    │
│  │              API GATEWAY (:8080)             │    │
│  │  • Rate limiting                             │    │
│  │  • JWT verification                          │    │
│  │  • CORS policy                               │    │
│  │  • Security headers                          │    │
│  │  • Request size limits                       │    │
│  └──────────────────┬──────────────────────────┘    │
└─────────────────────┼───────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────┐
│              SEMI-TRUSTED ZONE                        │
│  (Authenticated users, internal network)             │
│                                                      │
│  ┌──────────────────▼──────────────────────────┐    │
│  │  Identity  Auth  OAuth  Policy  Org  Audit   │    │
│  │  • Input validation                           │    │
│  │  • Scope enforcement                          │    │
│  │  • Tenant context from JWT                    │    │
│  └──────────────────┬──────────────────────────┘    │
└─────────────────────┼───────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────┐
│               TRUSTED ZONE                            │
│  (Database, Redis, NATS — internal only)             │
│                                                      │
│  ┌──────────────────▼──────────────────────────┐    │
│  │  PostgreSQL    Redis    NATS    OpenLDAP     │    │
│  │  • RLS policies                               │    │
│  │  • Connection pooling                         │    │
│  │  • TLS (planned)                              │    │
│  └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

---

## Trust Boundaries

### Boundary 1: Internet to Gateway

- **Threats**: DDoS, brute force, injection, CSRF
- **Controls**: Rate limiting (token bucket), CORS policy, security headers (X-Content-Type-Options, X-Frame-Options, Strict-Transport-Security), request body size limits, SQL injection prevention (parameterized queries)

### Boundary 2: Gateway to Microservices

- **Threats**: Service impersonation, tenant spoofing
- **Controls**: JWT verification at gateway, tenant_id from JWT claim (authoritative over X-Tenant-ID header), internal network isolation

### Boundary 3: Microservices to Database

- **Threats**: Data leakage across tenants, SQL injection
- **Controls**: PostgreSQL RLS policies, parameterized queries, connection pooling, least-privilege database roles

### Boundary 4: Microservices to Redis/NATS

- **Threats**: Session hijacking, message tampering
- **Controls**: Network isolation, Redis ACLs (planned), NATS authentication (planned)

---

## Defense in Depth Layers

### Layer 1: Network Edge

```
Client → [CDN/WAF] → Load Balancer → Gateway
```

- Rate limiting: token bucket per IP and per user
- Connection limits: max concurrent connections per IP
- Request timeout: configurable per route (default 30s)
- Body size limit: 10MB default

### Layer 2: Gateway Middleware Chain

The gateway applies middleware in a specific order:

```
Request →
  1. CORS Middleware
  2. Rate Limiter
  3. Security Headers
  4. Request Logging
  5. JWT Verification
  6. Tenant Context Extraction
  7. Scope Check
  8. Audit Log
  → Backend Service
```

### Layer 3: Service-Level Authorization

Each service independently validates:
- Tenant context (from JWT, not from header — prevents spoofing)
- Required scopes for the endpoint
- RBAC role permissions
- ABAC attribute conditions (where applicable)

### Layer 4: Database Enforcement

- **RLS policies**: Every tenant-scoped table has RLS enabled and forced
- **Parameterized queries**: All SQL uses `$1, $2, ...` placeholders
- **Connection-level tenant**: `SET LOCAL app.tenant_id` per transaction
- **Least privilege**: Database roles have minimal required permissions

---

## JWT Verification Flow

```
┌─────────┐     ┌──────────────────┐     ┌──────────────────┐
│ Client  │────▶│  Gateway:        │────▶│  Backend Service │
│ sends   │     │  VerifyJWT()     │     │  (trusts gateway │
│ JWT     │     │                  │     │   context)       │
└─────────┘     └────────┬─────────┘     └──────────────────┘
                         │
                ┌────────▼─────────┐
                │ Step 1: Parse    │
                │ header + payload │
                └────────┬─────────┘
                         │
                ┌────────▼─────────┐
                │ Step 2: Verify   │──── exp not expired?
                │ signature        │──── iat in the past?
                │ (HMAC-SHA256)    │──── iss matches config?
                └────────┬─────────┘
                         │
                ┌────────▼─────────┐
                │ Step 3: Check    │──── jti in replay set?
                │ Redis for:       │──── session revoked?
                │ • JTI anti-replay│──── refresh valid?
                │ • Session revoke │
                └────────┬─────────┘
                         │
                ┌────────▼─────────┐
                │ Step 4: Extract  │
                │ tenant_id from   │
                │ JWT claim        │
                │ (overrides header)│
                └────────┬─────────┘
                         │
                ┌────────▼─────────┐
                │ Step 5: Check    │
                │ required scopes  │
                │ (hasAdminScope,  │
                │  HasScope)       │
                └──────────────────┘
```

### JWT Claims

| Claim | Description | Example |
|-------|-------------|---------|
| `sub` | Subject (user ID) | `"usr_abc123"` |
| `iss` | Issuer | `"ggid-auth"` |
| `aud` | Audience | `"ggid-gateway"` |
| `exp` | Expiration time | `1700000000` |
| `iat` | Issued at | `1699999100` |
| `jti` | Unique token ID (anti-replay) | `"jti_xyz789"` |
| `tenant_id` | Tenant UUID | `"00000000-..."` |
| `scope` | Space-delimited scopes | `"read:users write:roles"` |
| `roles` | Array of role names | `["admin", "viewer"]` |

### Anti-Replay Protection

JTI (JWT ID) tracking uses Redis SETNX:
1. On token issuance, generate unique JTI
2. On verification, `SETNX jti:<value> 1 EX <ttl>`
3. If key already exists → replay detected → reject
4. Key TTL matches token expiry (auto-cleanup)

---

## Tenant Isolation Architecture

### Three-Layer Isolation

```
┌──────────────────────────────────────────────────────┐
│ Layer 1: Application-Level Tenant Context             │
│                                                       │
│  JWT claim "tenant_id" is AUTHORITATIVE               │
│  X-Tenant-ID header is IGNORED when JWT is present    │
│  Prevents tenant spoofing via header injection         │
└───────────────────────┬──────────────────────────────┘
                        │
┌───────────────────────▼──────────────────────────────┐
│ Layer 2: Connection-Level Tenant Setting               │
│                                                       │
│  Every DB transaction:                                │
│    SET LOCAL app.tenant_id = '<uuid>'                 │
│  Set BEFORE any queries in the transaction             │
└───────────────────────┬──────────────────────────────┘
                        │
┌───────────────────────▼──────────────────────────────┐
│ Layer 3: Database-Level RLS Enforcement                │
│                                                       │
│  CREATE POLICY tenant_isolation ON <table>             │
│    USING (tenant_id = current_setting('app.tenant_id'))│
│                                                       │
│  Even if layers 1 and 2 fail, RLS blocks cross-tenant  │
└──────────────────────────────────────────────────────┘
```

### Tenant Context Flow

```
1. Client request arrives with JWT
2. Gateway extracts tenant_id from JWT claim (NOT from X-Tenant-ID header)
3. Gateway forwards request to backend with:
   - X-Tenant-ID header (for logging/audit)
   - Original JWT (authoritative)
4. Backend service:
   a. Verifies JWT signature
   b. Extracts tenant_id from JWT claim
   c. Opens DB connection
   d. SET LOCAL app.tenant_id = '<uuid>'
   e. Executes queries (RLS automatically filters)
```

### Default Tenant

- UUID: `00000000-0000-0000-0000-000000000001`
- Used for single-tenant deployments and development
- All API calls through Gateway need `X-Tenant-ID` header with this UUID

---

## Audit Pipeline Security

### Event Flow

```
┌────────┐    ┌───────────┐    ┌───────────┐    ┌────────────┐
│ Client │───▶│  Gateway  │───▶│   NATS    │───▶│ Audit Svc  │
│        │    │ generates │    │ JetStream │    │ stores to  │
│        │    │ AuditEvent│    │ (durable) │    │ PostgreSQL │
└────────┘    └───────────┘    └───────────┘    └────────────┘
                                     │
                                     │
                              ┌──────▼──────┐
                              │  Webhook    │
                              │  Delivery   │
                              │  (SIEM)     │
                              └─────────────┘
```

### Audit Event Fields

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | RFC3339 | Event time (UTC) |
| `method` | string | HTTP method |
| `path` | string | Request path |
| `status_code` | int | HTTP response code |
| `tenant_id` | UUID | Tenant context |
| `user_id` | string | Authenticated user (if any) |
| `client_ip` | string | Client IP address |
| `user_agent` | string | Client User-Agent |

### Tamper Evidence (Planned)

Current state: audit events are stored in PostgreSQL without cryptographic chaining.

Planned improvement:
1. Each audit event includes `previous_hash` (SHA-256 of prior event)
2. Forms a hash chain detectable if any event is modified
3. Periodic Merkle tree root published externally (tamper-evident log)

### SSRF Protection for Webhooks

Webhook delivery includes SSRF protections:
- Deny private IP ranges (RFC 1918) in webhook URLs
- Deny loopback (127.0.0.0/8) and link-local (169.254.0.0/16)
- DNS resolution check before connection
- Configurable allowlist of webhook destination domains

---

## Secrets Management

### Current State

| Secret Type | Storage | Rotation |
|-------------|---------|----------|
| JWT signing key | Environment variable | Manual |
| Database password | Environment variable | Manual |
| Redis password | Environment variable | Manual |
| NATS credentials | Environment variable | Manual |
| OAuth client secrets | Database (encrypted) | Per-client |
| WebAuthn relying party key | Environment variable | Manual |
| LDAP bind password | Environment variable | Manual |

### Production Recommendations

1. **HashiCorp Vault** or **AWS Secrets Manager** for centralized secret management
2. **Automatic rotation**: Database credentials rotated every 90 days
3. **JWT key rotation**: Support multiple signing keys simultaneously for zero-downtime rotation
4. **Never log secrets**: All logging middleware redacts sensitive fields
5. **Environment separation**: Different secrets per environment (dev/staging/prod)

### Password Storage

- Algorithm: bcrypt with cost factor 12
- Pepper: Optional server-side pepper (recommended for production)
- Migration: Old hashes upgraded on next login

---

## Transport Security

### External Traffic (HTTPS)

```
Client ──TLS 1.3──▶ Load Balancer ──HTTP──▶ Gateway
```

- TLS 1.3 required (TLS 1.2 minimum for legacy clients)
- HSTS header: `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- Certificate management: Let's Encrypt (automated) or commercial CA

### Internal Traffic (Currently Plaintext)

```
Gateway ──HTTP/gRPC (plaintext)──▶ Backend Services
Backend Services ──TCP (plaintext)──▶ PostgreSQL / Redis / NATS
```

**Warning**: Internal traffic is currently unencrypted. This is acceptable when:
- Services run on the same trusted host (Docker Compose)
- Network is isolated (Kubernetes pod network, VPC)

**Planned**: mTLS between all services via service mesh (Istio/Linkerd)

---

## Input Validation

### Request Validation Pipeline

```
1. Content-Type check (must be application/json for POST/PUT)
2. Body size limit (10MB default)
3. JSON schema validation (where applicable)
4. Field-level validation:
   - Email format (RFC 5322)
   - UUID format for IDs
   - String length limits
   - Numeric range checks
5. SQL injection prevention: parameterized queries only
6. XSS prevention: output encoding in responses
```

### File Upload Security

- MIME type validation (not just extension)
- File size limits
- Virus scanning (planned)
- Store outside web root
- Generate new filename (never trust user-provided filename)

---

## Incident Response Integration

### SIEM Integration via Webhooks

GGID can forward audit events to external SIEM platforms:

```
GGID Audit ──▶ Webhook ──▶ Splunk / ELK / Datadog / Sumo Logic
```

Configuration:
- Set webhook URL per tenant
- Events filtered by severity
- Retry with exponential backoff
- Dead letter queue for failed deliveries

### Anomaly Detection Hooks

Planned integration points for anomaly detection:

| Event | Trigger | Response |
|-------|---------|----------|
| Brute force login | >5 failed attempts in 60s | Account lockout + alert |
| Token reuse | JTI replay detected | Revoke session + alert |
| Cross-tenant access | RLS policy violation (error) | Alert security team |
| Privilege escalation | Admin API without admin scope | Block + alert |
| Unusual geo | Login from new country | Require MFA re-verification |

### Incident Response Runbook

1. **Detect**: SIEM alerts from webhook events, or internal monitoring
2. **Contain**: Revoke user sessions via Redis, disable account
3. **Investigate**: Query audit trail for affected user/tenant
4. **Eradicate**: Patch vulnerability, rotate compromised credentials
5. **Recover**: Restore from backup if data was modified
6. **Lessons Learned**: Update security architecture document

---

## Security Hardening Checklist

### Completed

- [x] CSRF protection: unpredictable state parameter
- [x] Rate limiter wired into production handler chain
- [x] Security headers middleware active
- [x] Tenant spoofing prevention: JWT claim takes priority over header
- [x] Admin API scope check: `hasAdminScope()` guards
- [x] OAuth state validation on token exchange
- [x] JWT secret empty → `log.Fatal` (no silent bypass)
- [x] Scope enforcement: `HasScope()` actually checks scopes
- [x] JTI anti-replay via Redis SETNX

### Outstanding (P0)

- [ ] OAuth introspection endpoint authentication
- [ ] Webhook SSRF protection enforcement
- [ ] Host header validation (DNS rebinding prevention)
- [ ] gRPC TLS/mTLS between services
- [ ] Audit hash chain for tamper evidence
- [ ] Database backup automation
- [ ] WebAuthn attestation format verification (5 of 6 unverified)
- [ ] Password pepper implementation

### STRIDE Assessment

| Category | Score (1-10) | Notes |
|----------|-------------|-------|
| Spoofing | 8 | JWT + rate limiting; mTLS planned |
| Tampering | 7 | RLS + input validation; hash chain planned |
| Repudiation | 7 | Audit trail complete; hash chain planned |
| Information Disclosure | 8 | RLS + tenant isolation strong |
| Denial of Service | 7 | Rate limiting + circuit breakers |
| Elevation of Privilege | 8 | Scope checks + RBAC + ABAC |
| **Overall** | **7.5** | Target: 8.0+ |

---

*Last updated: 2025-07-11*
