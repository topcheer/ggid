# Architecture Overview

> For architects and engineering leads evaluating GGID. Understand the system design, service boundaries, data flow, and security model.

---

## System Architecture

```
                    ┌─────────────────────────────────────────────────┐
                    │                   Internet                       │
                    └────────────────────┬────────────────────────────┘
                                         │
                    ┌────────────────────▼────────────────────────────┐
                    │              API Gateway (:8080)                 │
                    │  • JWT Verification (HMAC-SHA256)               │
                    │  • Rate Limiting (Token Bucket)                  │
                    │  • CORS + Security Headers                       │
                    │  • Circuit Breaker                               │
                    │  • Request Routing                               │
                    │  • Tenant Context Extraction                     │
                    └──┬──────┬──────┬──────┬──────┬──────┬─────────┘
                       │      │      │      │      │      │
          ┌────────────▼┐ ┌──▼───┐ ┌▼─────┐ ┌▼─────┐ ┌▼─────┐ ┌──────▼──┐
          │  Identity    │ │ Auth │ │OAuth │ │Policy│ │ Org  │ │ Audit   │
          │  (:8081)     │ │(:9001)│ │(:9005)│ │(:8070)│ │(:8071)│ │ (:8072) │
          │              │ │      │ │      │ │      │ │      │ │         │
          │  • User CRUD │ │• Login│ │• Auth │ │• RBAC│ │• Tree│ │• NATS   │
          │  • SCIM 2.0  │ │• MFA  │ │  Code │ │• ABAC│ │• Dept│ │• Query  │
          │  • Search    │ │• LDAP │ │• PKCE │ │• Chk │ │• Team│ │• Export │
          └──────┬───────┘ └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘ └────┬────┘
                 │            │        │        │        │          │
          ┌──────┴────────────┴────────┴────────┴────────┴──────────┴────┐
          │                    Infrastructure Layer                      │
          │  ┌────────────┐  ┌─────────┐  ┌──────┐  ┌──────┐            │
          │  │ PostgreSQL │  │  Redis  │  │ NATS │  │ LDAP │            │
          │  │    16      │  │    7    │  │  2   │  │ 1.5  │            │
          │  │  + RLS     │  │ Sessions│  │Audit │  │(opt) │            │
          │  └────────────┘  └─────────┘  └──────┘  └──────┘            │
          └─────────────────────────────────────────────────────────────┘
```

---

## Service Responsibilities

| Service | Port(s) | Responsibility |
|---------|---------|----------------|
| **Gateway** | 8080 | Entry point. JWT verification, rate limiting, routing, circuit breaker |
| **Identity** | 8081 / 50051 | User lifecycle: CRUD, search, lock/unlock, SCIM 2.0 provisioning |
| **Auth** | 9001 | Authentication: login, MFA, LDAP bind, WebAuthn, password reset |
| **OAuth** | 9005 | OAuth 2.1: authorization code, PKCE, client credentials, device flow, token exchange |
| **Policy** | 8070 / 9070 | Authorization: RBAC roles, ABAC rules, permission checks |
| **Org** | 8071 / 9071 | Organization hierarchy: tree, departments, teams, memberships |
| **Audit** | 8072 / 9072 | Audit pipeline: NATS consumer, event storage, query API, webhook delivery |

---

## Data Flow: Login Request

```
1. Client POST /api/v1/auth/login (username + password)
   ↓
2. Gateway receives request
   • Rate limit check (token bucket per IP)
   • Route to Auth service
   ↓
3. Auth Service
   • Provider chain: Local DB → LDAP (if configured)
   • Verify password (bcrypt cost 12)
   • Check MFA enrollment
   • Generate JWT (HMAC-SHA256, 15min TTL)
   • Store refresh token in Redis (7d TTL)
   • Store JTI in Redis for anti-replay
   ↓
4. Gateway returns response
   • { access_token, refresh_token, user }
   ↓
5. Audit event published to NATS
   • auth.login event
   • Consumed by Audit service → PostgreSQL
   • Consumed by Webhook delivery (if registered)
```

---

## Multi-Tenancy Model

### Three-Layer Isolation

```
Layer 1: Application — JWT claim `tenant_id` is authoritative
         (X-Tenant-ID header ignored when JWT present)
         ↓
Layer 2: Connection — SET LOCAL app.tenant_id per transaction
         (every DB transaction sets this before queries)
         ↓
Layer 3: Database — PostgreSQL RLS policy enforces row-level filter
         (even if layers 1+2 fail, RLS blocks cross-tenant access)
```

### Tenant Lifecycle

1. Super admin creates tenant (`POST /api/v1/tenants`)
2. First user registered becomes tenant admin
3. Roles, orgs, policies created within tenant
4. All data isolated via RLS
5. Tenant can be suspended (`POST /api/v1/tenants/{id}/suspend`)

---

## Security Model

### Trust Zones

| Zone | Boundary | Controls |
|------|----------|----------|
| **Untrusted** | Internet → Gateway | Rate limiting, CORS, security headers, body size limits |
| **Semi-Trusted** | Gateway → Services | JWT verified, tenant context extracted, scope checked |
| **Trusted** | Services → Database | RLS policies, parameterized queries, least-privilege DB roles |

### Key Security Controls

1. **JWT Verification**: Signature, expiry, issuer, audience, JTI anti-replay
2. **Tenant Isolation**: Three-layer (JWT → SET LOCAL → RLS)
3. **Rate Limiting**: Token bucket per IP (10 req/min unauthenticated, 1000/min authenticated)
4. **Circuit Breaker**: Prevents cascade failures (CLOSED → OPEN → HALF-OPEN)
5. **Audit Trail**: Every API call logged via NATS JetStream pipeline
6. **Password Security**: bcrypt cost 12, optional pepper, breach check (planned)
7. **MFA**: TOTP, WebAuthn/Passkeys, Email OTP

---

## Technology Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| Language | Go 1.25 | Performance, concurrency, small binaries |
| Database | PostgreSQL 16 | RLS, JSONB, mature ecosystem |
| Cache | Redis 7 | Sessions, rate limiting, JTI anti-replay |
| Message Bus | NATS JetStream | Lightweight, durable, audit pipeline |
| Frontend | Next.js 15 + React | SSR, type-safe, fast DX |
| Protocol | gRPC + REST | Internal efficiency + external compatibility |
| Deployment | Docker / K8s / Bare Metal | Flexibility |

---

## Deployment Options

| Option | Best For | Guide |
|--------|----------|-------|
| Docker Compose | Development, small teams | [deploy/docker.md](../deploy/docker.md) |
| Kubernetes / Helm | Production, scalable | [deploy/kubernetes.md](../deploy/kubernetes.md) |
| K3s | Edge, lightweight K8s | [deploy/k3s.md](../deploy/k3s.md) |
| Bare Metal | VMs, on-prem | [deploy/bare-metal.md](../deploy/bare-metal.md) |

---

*Last updated: 2025-07-11*