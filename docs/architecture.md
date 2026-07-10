# GGID Architecture

> C4 Model — System Context, Container, and Component views.

---

## 1. System Context

```
                    ┌──────────────────────────────────────────┐
                    │              End Users                    │
                    │  (Browser, Mobile App, CLI, API Client)  │
                    └──────────────┬───────────────────────────┘
                                   │
                          HTTPS / gRPC
                                   │
                    ┌──────────────▼───────────────────────────┐
                    │            GGID Platform                  │
                    │  (IAM Suite — 7 microservices + Console)  │
                    └──────┬───────┬───────┬───────┬───────────┘
                           │       │       │       │
                    ┌──────▼──┐ ┌──▼──┐ ┌──▼──┐ ┌─▼────────┐
                    │PostgreSQL│ │Redis│ │NATS │ │OpenLDAP  │
                    │  (RLS)  │ │     │ │Stream│ │          │
                    └─────────┘ └─────┘ └─────┘ └──────────┘
```

### External Integrations

| System | Protocol | Purpose |
|--------|----------|---------|
| Google / GitHub / Microsoft | OAuth2 | Social login connectors |
| Enterprise IdP (Okta, ADFS) | SAML 2.0 / OIDC | Enterprise SSO |
| SMTP Server | SMTP | Email verification, password reset |
| SMS Gateway (Twilio) | REST API | Phone OTP, MFA |
| LDAP / Active Directory | LDAPv3 | Enterprise directory sync |

---

## 2. Container Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        GGID Platform                                │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    API Gateway (:8080)                       │   │
│  │  JWT Verification · Rate Limiting · CORS · Tracing · Proxy  │   │
│  └──────┬──────────┬──────────┬──────────┬──────────┬──────────┘   │
│         │          │          │          │          │               │
│  ┌──────▼────┐ ┌───▼────┐ ┌──▼─────┐ ┌─▼──────┐ ┌─▼──────┐       │
│  │ Identity  │ │  Auth  │ │ OAuth  │ │ Policy │ │  Org   │       │
│  │  (:8081)  │ │(:9001) │ │(:9005) │ │(:8070) │ │(:8071) │       │
│  │  gRPC     │ │        │ │        │ │ gRPC   │ │ gRPC   │       │
│  │  :50051   │ │        │ │        │ │ :9070  │ │ :9071  │       │
│  └───────────┘ └────────┘ └────────┘ └────────┘ └────────┘       │
│         │          │          │          │          │               │
│  ┌──────▼──────────────────────────────────────────────────────┐  │
│  │                    Audit (:8072)                            │  │
│  │         gRPC :9072 · NATS JetStream Consumer               │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              Admin Console (:3000)                          │   │
│  │          Next.js 15 · React 19 · TailwindCSS               │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### Service Responsibilities

| Service | Port (HTTP/gRPC) | Responsibility |
|---------|------------------|----------------|
| **Gateway** | 8080 / — | Entry point. JWT verification, rate limiting, request routing, CORS, tracing, API key auth |
| **Identity** | 8081 / 50051 | User CRUD, credential storage, SCIM 2.0, user profile, user import/export |
| **Auth** | 9001 / — | Login, register, password reset, MFA (TOTP/WebAuthn), LDAP, session management |
| **OAuth** | 9005 / — | OAuth2 authorization code, PKCE, client credentials, OIDC, token revocation, SAML SP |
| **Policy** | 8070 / 9070 | RBAC + ABAC engine, role/permission management, policy evaluation, role hierarchy |
| **Org** | 8071 / 9071 | Organization tree, department management, multi-tenant RLS isolation |
| **Audit** | 8072 / 9072 | Event logging via NATS JetStream, audit query, retention policy, anomaly detection |
| **Console** | 3000 / — | Admin dashboard, user/role/org management, audit viewer, settings |

---

## 3. Component Diagram (Gateway)

```
┌───────────────────────────────────────────────┐
│                 API Gateway                   │
│                                               │
│  ┌─────────┐  ┌──────────┐  ┌──────────────┐ │
│  │  CORS   │→ │ Rate     │→ │ JWT Verify   │ │
│  │Middleware│  │ Limiter  │  │ Middleware   │ │
│  └─────────┘  └──────────┘  └──────┬───────┘ │
│                                      │         │
│  ┌──────────────────────────────────▼───────┐ │
│  │          Request Router                  │ │
│  │  /api/v1/auth/*    → Auth Service       │ │
│  │  /api/v1/users/*   → Identity Service   │ │
│  │  /api/v1/roles/*   → Policy Service     │ │
│  │  /api/v1/orgs/*    → Org Service        │ │
│  │  /api/v1/audit/*   → Audit Service      │ │
│  │  /oauth/*          → OAuth Service      │ │
│  │  /login, /register → Hosted Login Pages │ │
│  └──────────────────────────────────────────┘ │
│                                               │
│  ┌─────────┐  ┌──────────┐  ┌──────────────┐ │
│  │ Tracing │  │ Metrics  │  │  Webhooks    │ │
│  │ (W3C)   │  │(Prometh.)│  │  Delivery    │ │
│  └─────────┘  └──────────┘  └──────────────┘ │
└───────────────────────────────────────────────┘
```

### Middleware Chain (execution order)

1. **Request ID** — Generate/propagate `X-Request-ID`
2. **CORS** — Cross-origin preflight handling
3. **Bot Detection** — User-Agent + behavior analysis
4. **Rate Limiting** — Per-tenant configurable limits
5. **Compression** — gzip/brotli negotiation
6. **Body Size Limit** — Per-route configurable
7. **JWT Verification** — RS256/HS256, JWKS refresh
8. **API Key Auth** — Alternative M2M authentication
9. **Tenant Resolution** — `X-Tenant-ID` header → context injection
10. **Proxy** — Reverse proxy to backend service
11. **Metrics** — Prometheus histogram per route
12. **Tracing** — W3C traceparent span export

---

## 4. Data Flow: Login

```
User                Gateway             Auth           Identity          Redis
 │                     │                 │                │                │
 │  POST /auth/login   │                 │                │                │
 │  {username,password}│                 │                │                │
 │────────────────────>│                 │                │                │
 │                     │  Rate limit     │                │                │
 │                     │  check ──────────────────────────────────────────>│
 │                     │                 │                │                │
 │                     │  Proxy to Auth  │                │                │
 │                     │────────────────>│                │                │
 │                     │                 │  Verify creds  │                │
 │                     │                 │───────────────>│                │
 │                     │                 │  <- user record │                │
 │                     │                 │                │                │
 │                     │                 │  Password hash verify            │
 │                     │                 │  MFA check (if enabled)          │
 │                     │                 │  Generate JWT (RS256)            │
 │                     │                 │  Store session ─────────────────>│
 │                     │                 │                │                │
 │                     │  <- JWT + refresh_token             │                │
 │  <- 200 OK + tokens  │                 │                │                │
 │<────────────────────│                 │                │                │
```

## 5. Data Flow: Policy Evaluation

```
Client              Gateway            Policy             PostgreSQL
 │                     │                  │                    │
 │  GET /api/v1/data   │                  │                    │
 │  Bearer <JWT>       │                  │                    │
 │────────────────────>│                  │                    │
 │                     │  Verify JWT      │                    │
 │                     │  Extract: user_id, tenant_id, roles  │
 │                     │                  │                    │
 │                     │  Proxy + tenant  │                    │
 │                     │  context ───────>│                    │
 │                     │                  │  Evaluate RBAC     │
 │                     │                  │  + ABAC conditions │
 │                     │                  │───────────────────>│
 │                     │                  │  <- roles+perms     │
 │                     │                  │                    │
 │                     │                  │  Decision: allow  │
 │                     │                  │<─── deny ─────────│
 │                     │                  │                    │
 │                     │  <- response      │                    │
 │  <- data or 403      │                  │                    │
 │<────────────────────│                  │                    │
```

---

## 6. Multi-Tenancy Model

GGID uses **shared database with Row-Level Security (RLS)**:

```sql
-- Every table has tenant_id + RLS policy
CREATE POLICY tenant_isolation ON users
  USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Gateway sets tenant context per request
SET LOCAL app.tenant_id = 'xxxx-xxxx-xxxx';
```

- Tenant isolation enforced at PostgreSQL level (not just application)
- Each request carries `X-Tenant-ID` header → Gateway injects into DB session
- Cross-tenant queries are impossible even with SQL injection

---

## 7. Technology Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.25 |
| Protocol | gRPC (internal) + REST (external) |
| Web Framework | Standard net/http + Chi router |
| Database | PostgreSQL 16 (RLS, JSONB, pgcrypto) |
| Cache | Redis 7 (session, rate limit, JWKS cache) |
| Message Queue | NATS JetStream (audit events, webhooks) |
| Directory | OpenLDAP (LDAPv3 enterprise sync) |
| Frontend | Next.js 15, React 19, TailwindCSS, Recharts |
| Container | Docker, Docker Compose, Kubernetes (Helm) |
| Observability | OpenTelemetry, Prometheus, structured logging |
| Auth | JWT (RS256), TOTP (RFC 6238), WebAuthn, SAML 2.0 |
| SDK | Go, Node.js, Java, Python |
