# Threat Modeling Guide

This guide covers threat modeling for GGID deployments — STRIDE analysis per service, attack surfaces, trust boundaries, data flow diagrams, and mitigation mapping.

## STRIDE Per Service

### Gateway

| Threat | Type | Attack Surface | Mitigation |
|--------|------|--------------|------------|
| JWT forgery | Spoofing | Authorization header | RS256 signature verification via JWKS |
| Tenant spoofing | Spoofing | X-Tenant-ID header | JWT claim takes priority over header |
| Rate limit bypass | DoS | High-frequency requests | Token bucket per IP + per user |
| SSRF via webhook | Information disclosure | Webhook URLs | Private IP blocking (RFC 1918, metadata) |
| Header injection | Tampering | HTTP headers | Input validation, no header reflection |

### Auth Service

| Threat | Type | Attack Surface | Mitigation |
|--------|------|--------------|------------|
| Credential stuffing | Spoofing | Login endpoint | Rate limit + account lockout |
| Password brute force | Spoofing | Login endpoint | Argon2id (memory-hard) + lockout |
| JWT secret leak | Elevation of privilege | JWT signing key | HSM/Vault storage + rotation |
| Password pepper loss | DoS | Pepper key | Redundant backups |
| Session fixation | Spoofing | Post-login session | Session regeneration on auth |
| MFA bypass | Elevation of privilege | MFA verify endpoint | TOTP window limiting + jti |

### Identity Service

| Threat | Type | Attack Surface | Mitigation |
|--------|------|--------------|------------|
| Unauthorized user creation | Elevation of privilege | POST /users | RBAC scope check (users:write) |
| User enumeration | Information disclosure | Search/login | Generic error messages |
| Mass user export | Information disclosure | GET /users/export | Admin scope + rate limit |
| SCIM injection | Tampering | POST /scim/v2/Users | Input validation + SCIM token auth |

### OAuth Service

| Threat | Type | Attack Surface | Mitigation |
|--------|------|--------------|------------|
| Authorization code interception | Spoofing | Redirect URI | PKCE mandatory (S256) |
| Redirect URI bypass | Spoofing | redirect_uri param | Exact string matching only |
| State CSRF | Spoofing | OAuth state | Redis-backed state validation |
| Implicit grant abuse | Elevation of privilege | response_type=token | Removed (OAuth 2.1) |
| Token theft (bearer) | Elevation of privilege | Stolen access token | jti anti-replay + short TTL |
| Client secret leak | Spoofing | Client config | Secrets manager + rotation |

### Policy Service

| Threat | Type | Attack Surface | Mitigation |
|--------|------|--------------|------------|
| Privilege escalation | Elevation of privilege | Role assignment | RBAC + SoD rules |
| Policy bypass | Tampering | Policy check | Defense in depth (RLS + scope) |
| Deny list tampering | Tampering | Policy CRUD | Admin scope + audit |

### Audit Service

| Threat | Type | Attack Surface | Mitigation |
|--------|------|--------------|------------|
| Audit log tampering | Tampering | Database | Hash chain (SHA-256) |
| Audit deletion | Repudiation | DELETE | Soft-delete only + immutability |
| SIEM forward block | DoS | SIEM endpoint | Async queue + retry |

## Trust Boundaries

```
┌────────────────────────────────────────────────────────┐
│ TRUST ZONE: Internet (untrusted)                        │
│  ┌──────────┐                                          │
│  │  Client   │ ← User browser, mobile app, API client   │
│  └─────┬─────┘                                          │
└────────┼───────────────────────────────────────────────┘
         │ TLS 1.2+ (boundary 1: external→gateway)
┌────────┼───────────────────────────────────────────────┐
│ TRUST ZONE: DMZ                                         │
│  ┌─────▼─────┐                                          │
│  │  Gateway   │ ← JWT verify, rate limit, tenant check   │
│  └─────┬─────┘                                          │
└────────┼───────────────────────────────────────────────┘
         │ mTLS (boundary 2: gateway→services)
┌────────┼───────────────────────────────────────────────┐
│ TRUST ZONE: Internal (trusted)                          │
│  ┌─────▼─────┐  ┌────────┐  ┌────────┐                  │
│  │ Identity   │  │  Auth  │  │ OAuth  │  ← 7 microservices│
│  └───────────┘  └────────┘  └────────┘                  │
│  ┌───────────┐  ┌────────┐  ┌────────┐                  │
│  │  Policy    │  │  Org   │  │ Audit  │                  │
│  └───────────┘  └────────┘  └────────┘                  │
└────────┼───────────────────────────────────────────────┘
         │ (boundary 3: services→data)
┌────────┼───────────────────────────────────────────────┐
│ TRUST ZONE: Data (restricted)                           │
│  ┌─────▼─────┐  ┌────────┐  ┌────────┐                  │
│  │ PostgreSQL │  │ Redis  │  │  NATS  │  ← Infrastructure │
│  │ (RLS)      │  │ (TLS)  │  │ (TLS)  │                  │
│  └───────────┘  └────────┘  └────────┘                  │
└────────────────────────────────────────────────────────┘
```

## Attack Surface Inventory

| Surface | Protocol | Exposed To | Protection |
|---------|----------|-----------|------------|
| Gateway HTTP | HTTPS:443 | Internet | TLS, WAF, rate limit |
| Gateway gRPC | — | Internal only | Not exposed |
| Service HTTP | HTTP:80xx | Internal only | Gateway proxy only |
| Service gRPC | gRPC:90xx | Internal only | mTLS |
| PostgreSQL | TCP:5432 | Internal only | TLS + RLS + password |
| Redis | TCP:6379 | Internal only | TLS + AUTH |
| NATS | TCP:4222 | Internal only | TLS |
| Console | HTTPS:3000 | Internet | TLS + auth |

## Data Flow Diagram

```
User → Gateway → Auth (login) → Redis (store session)
                          ↓
                 PostgreSQL (verify password)
                          ↓
                 JWT issued → Gateway → User

User → Gateway → Identity (GET user) → PostgreSQL (RLS query)
                          ↓
                 NATS (publish audit event) → Audit consumer
```

## Mitigation Priority Matrix

| Priority | Threat | Mitigation | Status |
|----------|--------|-----------|--------|
| P0 | JWT forgery | RS256 + JWKS | Done |
| P0 | Tenant spoofing | JWT > header | Done |
| P0 | Token theft | jti + short TTL | Done |
| P1 | Password brute force | Argon2id + lockout | Done |
| P1 | Auth code interception | PKCE mandatory | Done |
| P1 | Rate limit bypass | Token bucket | Done |
| P2 | Token binding (DPoP) | Sender-constrained | Planned |
| P2 | Device posture | Registry + checks | Planned |

## See Also

- [STRIDE Threat Analysis](../research/stride-analysis.md)
- [Security Audit Checklist](security-audit-checklist.md)
- [Zero Trust Architecture](../research/zero-trust-architecture.md)
- [ITDR Implementation](itdr-implementation.md)
