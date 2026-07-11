# Data Flow Architecture

> How requests flow through GGID services — registration, JWT verification, and audit event pipeline.

---

## Request Flow Overview

```
                          ┌─────────────┐
                          │   Client     │
                          │ (Browser/SDK)│
                          └──────┬──────┘
                                 │
                          ┌──────▼──────┐
                          │ API Gateway  │  :8080
                          │ (Reverse Proxy│
                          │  + JWT Verify)│
                          └──┬───┬───┬──┘
                             │   │   │
              ┌──────────────┘   │   └──────────────┐
              │                  │                  │
       ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐
       │  Identity    │   │    Auth     │   │   Policy    │
       │  (Users/SCIM)│   │  (Login/JWT)│   │  (RBAC/ABAC)│
       └──────┬──────┘   └──────┬──────┘   └──────┬──────┘
              │                  │                  │
              │                  │                  │
       ┌──────▼──────────────────▼──────────────────▼──────┐
       │                 PostgreSQL 16 (RLS)                 │
       │  users | credentials | roles | policies | orgs     │
       └─────────────────────────────────────────────────────┘
              │
       ┌──────▼──────┐   ┌──────────────┐
       │    Redis    │   │     NATS     │
       │ (Sessions/  │   │  JetStream   │
       │  Rate Limit)│   │  (Audit Bus) │
       └─────────────┘   └──────┬───────┘
                                │
                        ┌───────▼───────┐
                        │ Audit Service  │
                        │ (Consumer +    │
                        │  Hash Chain)   │
                        └───────────────┘
```

---

## 1. User Registration Flow

```
Client                    Gateway              Auth Service           Identity Service        PostgreSQL
  │                          │                      │                       │                      │
  │  POST /api/v1/auth/register                  │                       │                      │
  │  {username,email,password}                    │                       │                      │
  │ ────────────────────────▶│                      │                       │                      │
  │                          │  Forward to Auth     │                       │                      │
  │                          │ ────────────────────▶│                       │                      │
  │                          │                      │                       │                      │
  │                          │              Validate password strength     │                      │
  │                          │              Hash with Argon2id + pepper    │                      │
  │                          │                      │                       │                      │
  │                          │                      │  Create user + credential                   │
  │                          │                      │ ──────────────────────────────────────────▶│
  │                          │                      │                       │                      │
  │                          │                      │  Publish audit event (user.registered)      │
  │                          │                      │ ────────────────────────────┐               │
  │                          │                      │                              │               │
  │                          │                      │                       ┌──────▼──────┐         │
  │                          │                      │                       │ NATS Bus    │         │
  │                          │                      │                       └──────┬──────┘         │
  │                          │                      │                              │               │
  │                          │              201 Created + user_id                │               │
  │                          │ ◀────────────────────│                              │               │
  │  201 Created             │                      │                              │               │
  │ ◀────────────────────────│                      │                              │               │
```

**Services touched:** Gateway → Auth → PostgreSQL → NATS (audit)

---

## 2. Login & JWT Issuance Flow

```
Client               Gateway           Auth Service        AuthProvider Chain      Redis          PostgreSQL
  │                     │                   │                     │                  │                │
  │  POST /api/v1/auth/login                │                     │                  │                │
  │  {username, password}                   │                     │                  │                │
  │ ──────────────────▶│                    │                     │                  │                │
  │                     │  Forward           │                     │                  │                │
  │                     │ ─────────────────▶│                     │                  │                │
  │                     │                   │  Rate limit check   │                  │                │
  │                     │                   │ ──────────────────────────────────────▶│                │
  │                     │                   │                     │  Allow           │                │
  │                     │                   │                     │ ◀────────────────│                │
  │                     │                   │                     │                  │                │
  │                     │                   │  Lookup credential  │                  │                │
  │                     │                   │ ──────────────────────────────────────────────────────▶│
  │                     │                   │  {user, hash, roles}│                  │                │
  │                     │                   │ ◀─────────────────────────────────────────────────────│
  │                     │                   │                     │                  │                │
  │                     │                   │  Verify: Argon2id(hash, password + pepper)            │
  │                     │                   │                     │                  │                │
  │                     │                   │  Try LocalProvider  │                  │                │
  │                     │                   │ ─────────────────▶│                     │                │
  │                     │                   │                   │ If fail, try LDAP  │                │
  │                     │                   │ ◀─────────────────│                     │                │
  │                     │                   │                     │                  │                │
  │                     │                   │  Issue JWT (RS256)  │                  │                │
  │                     │                   │  + Refresh Token    │                  │                │
  │                     │                   │  Store session      │                  │                │
  │                     │                   │ ──────────────────────────────────────▶│                │
  │                     │                   │                     │                  │                │
  │                     │                   │  Publish audit event (user.login)      │                │
  │                     │                   │ ────▶ NATS          │                  │                │
  │                     │                   │                     │                  │                │
  │                     │  {access_token, refresh_token}          │                  │                │
  │                     │ ◀─────────────────│                     │                  │                │
  │  JWT + Refresh      │                   │                     │                  │                │
  │ ◀───────────────────│                   │                     │                  │                │
```

**Services touched:** Gateway → Auth → PostgreSQL → Redis → NATS

---

## 3. JWT Verification Path (Subsequent Requests)

```
Client                    Gateway                    Redis
  │                          │                         │
  │  GET /api/v1/users       │                         │
  │  Authorization: Bearer X │                         │
  │ ────────────────────────▶│                         │
  │                          │                         │
  │                          │ 1. Extract Bearer token │
  │                          │ 2. Parse JWT header     │
  │                          │ 3. Fetch JWKS key       │
  │                          │    (cached 15 min)      │
  │                          │                         │
  │                          │ 4. Verify RS256 sig     │
  │                          │ 5. Check exp, nbf, iss  │
  │                          │                         │
  │                          │ 6. Check jti anti-replay│
  │                          │ ───────────────────────▶│
  │                          │    SETNX jti (1x use)   │
  │                          │ ◀───────────────────────│
  │                          │                         │
  │                          │ 7. Extract claims:      │
  │                          │    sub, tenant_id,      │
  │                          │    roles, scope         │
  │                          │                         │
  │                          │ 8. Resolve tenant:      │
  │                          │    JWT claim > header   │
  │                          │                         │
  │                          │ 9. Forward to Identity  │
  │                          │    with X-Tenant-ID     │
  │                          ▼                         │
  │                    ┌──────────┐                    │
  │                    │ Identity │──▶ PostgreSQL (RLS)│
  │                    └──────────┘                    │
  │                          │                         │
  │  200 OK + users list     │                         │
  │ ◀────────────────────────│                         │
```

**Services touched:** Gateway → Redis (jti check) → Identity → PostgreSQL

---

## 4. Policy Check Flow (RBAC + ABAC)

```
Gateway              Policy Service          PostgreSQL
  │                       │                      │
  │  POST /policies/check │                      │
  │  {user_id, action,    │                      │
  │   resource, attrs}    │                      │
  │ ─────────────────────▶│                      │
  │                       │                      │
  │                       │ 1. Load user roles   │
  │                       │ ────────────────────▶│
  │                       │ ◀────────────────────│
  │                       │    {roles, perms}    │
  │                       │                      │
  │                       │ 2. Check RBAC:       │
  │                       │    role has action:  │
  │                       │    resource?         │
  │                       │                      │
  │                       │ 3. Check ABAC:       │
  │                       │    DENY policies     │
  │                       │    (by priority)     │
  │                       │ ────────────────────▶│
  │                       │ ◀────────────────────│
  │                       │    {deny_policies}   │
  │                       │                      │
  │                       │ 4. Algorithm:        │
  │                       │    DENY match → deny │
  │                       │    ALLOW match → ok  │
  │                       │    No match → default│
  │                       │                      │
  │  {allowed: true/false}│                      │
  │ ◀─────────────────────│                      │
```

---

## 5. Audit Event Pipeline

```
Any Service            NATS JetStream          Audit Consumer         PostgreSQL
  │                        │                       │                      │
  │  Publish event         │                       │                      │
  │  {type, actor, action, │                       │                      │
  │   resource, tenant_id} │                       │                      │
  │ ─────────────────────▶│                       │                      │
  │                        │                       │                      │
  │                        │  JetStream persists   │                      │
  │                        │  to disk (at-least-   │                      │
  │                        │  once delivery)       │                      │
  │                        │                       │                      │
  │                        │  Push to consumer     │                      │
  │                        │ ─────────────────────▶│                      │
  │                        │                       │                      │
  │                        │                       │  1. Compute hash     │
  │                        │                       │     prev_hash + data │
  │                        │                       │                      │
  │                        │                       │  2. Store event      │
  │                        │                       │     + hash chain     │
  │                        │                       │ ────────────────────▶│
  │                        │                       │                      │
  │                        │                       │  3. Ack to NATS      │
  │                        │ ◀─────────────────────│                      │
  │                        │                       │                      │
  │                        │                       │  Audit query API:   │
  │                        │                       │  GET /audit/events   │
  │                        │                       │ ◀────────────────────│
```

**Hash chain ensures tamper detection:**
- Each event's hash = SHA256(prev_hash + event_data)
- Deleting/inserting/modifying any event breaks the chain
- Verified via `verify_hash_chain()` function

---

## Service Port Reference

| Service | HTTP Port | gRPC Port | Description |
|---------|-----------|-----------|-------------|
| Gateway | 8080 | — | Public entry point |
| Identity | 8081 | 50051 | User/SCIM management |
| Auth | 9001 | 50052 | Login/JWT/MFA |
| OAuth | 9005 | — | OAuth/OIDC flows |
| Policy | 8070 | 50053 | RBAC/ABAC evaluation |
| Org | 8071 | 50054 | Organization management |
| Audit | 8072 | 50055 | Audit query API |

---

*See: [Architecture Overview](overview.md) | [Security Overview](security-overview.md) | [Policy Engine Design](../design/policy-engine.md)*

*Last updated: 2025-07-11*
