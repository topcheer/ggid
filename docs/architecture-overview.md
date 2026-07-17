# GGID Architecture Overview

> A guide for new developers joining the GGID project.

## System Map

GGID is a microservices-based Identity and Access Management (IAM) platform built in Go with a Next.js frontend.

```
                          ┌──────────────┐
                          │   Internet   │
                          └──────┬───────┘
                                 │
                          ┌──────▼───────┐
                          │   Ingress    │
                          │  (k3s nginx) │
                          └──────┬───────┘
                                 │
                    ┌────────────▼─────────────┐
                    │      Console (Next.js)    │
                    │   Port: 3000 (container)  │
                    └────────────┬─────────────┘
                                 │
                          ┌──────▼───────┐
                          │   Gateway    │
                          │  Port: 8080  │
                          └──┬──┬──┬──┬──┘
                             │  │  │  │
              ┌──────────────┘  │  │  └──────────────┐
              │        ┌────────┘  └────────┐        │
              ▼        ▼                     ▼        ▼
          ┌──────┐ ┌────────┐  ┌──────────┐ ┌──────┐ ┌──────┐
          │ Auth │ │Identity│  │  OAuth   │ │Policy│ │Audit │
          │9001  │ │ 8081   │  │  9005    │ │8070  │ │8072  │
          └──┬───┘ └───┬────┘  └────┬─────┘ └──┬───┘ └──┬───┘
             │         │            │          │        │
             └────┬────┴────┬───────┴──────────┴────────┘
                  │         │
              ┌───▼───┐ ┌──▼───┐
              │Postgres│ │Redis │
              │  :5432 │ │:6379 │
              └────────┘ └──────┘
```

## Services

### 1. Gateway (`services/gateway/`)
- **Port**: 8080
- **Role**: API gateway, reverse proxy, middleware chain.
- **Key middleware**: Auth token validation, API key auth, rate limiting, DLP egress scanning, audit logging, CORS, circuit breaker, request coalescing.
- **Routes**: Proxies `/api/v1/auth/*` → Auth, `/api/v1/users/*` → Identity, `/api/v1/oauth/*` → OAuth, etc.

### 2. Auth Service (`services/auth/`)
- **Port**: 9001 (HTTP) / 50052 (gRPC)
- **Role**: Authentication, session management, MFA (TOTP), passkeys (WebAuthn), biometric auth, password policies.
- **Database tables**: `users`, `sessions`, `mfa_factors`, `passkeys`, `device_bindings`.

### 3. Identity Service (`services/identity/`)
- **Port**: 8081 (HTTP) / 50051 (gRPC)
- **Role**: User lifecycle, groups, organizations, SCIM 2.0 provisioning, federation (SAML/OIDC), consent management, brand/theming.
- **Database tables**: `users_detail`, `groups`, `organizations`, `federation_entities`, `consent_records`.

### 4. OAuth Service (`services/oauth/`)
- **Port**: 9005
- **Role**: OAuth 2.1 authorization server, OIDC provider, token issuance (code, device, client credentials), DPoP (RFC 9449), SAML IdP/SP, consent screens, token introspection.
- **Database tables**: `oauth_clients`, `authorization_codes`, `access_tokens`, `refresh_tokens`, `consents`, `dpop_keys`.

### 5. Policy Service (`services/policy/`)
- **Port**: 8070 (HTTP) / 9070 (gRPC)
- **Role**: RBAC, ABAC, ReBAC (Zanzibar-style), policy evaluation engine, access certifications, risk scoring, JIT elevation.
- **Database tables**: `roles`, `permissions`, `policies`, `relation_tuples`, `risk_scores`.

### 6. Audit Service (`services/audit/`)
- **Port**: 8072 (HTTP) / 9072 (gRPC)
- **Role**: Audit event ingestion, ITDR (Identity Threat Detection), compliance reporting, anomaly detection, threat intelligence aggregation, audit log integrity (hash chain).
- **Database tables**: `audit_events`, `itdr_detections`, `compliance_assessments`, `threat_indicators`, `intel_sources`.

## Request Lifecycle

```
1. Client sends request → Gateway (8080)
2. Gateway middleware chain:
   a. CORS handling
   b. Rate limiting (Redis-backed token bucket)
   c. Authentication (JWT or API key validation)
   d. Tenant resolution (X-Tenant-ID header)
   e. Audit logging (async, to NATS/audit service)
   f. Request body validation
3. Gateway reverse-proxies to target service
4. Service processes request:
   a. Parse request
   b. Business logic (may call other services via gRPC)
   c. Database query (PostgreSQL via pgxpool)
   d. Build response
5. Gateway response middleware:
   a. DLP egress scan (PII redaction)
   b. Response caching (if applicable)
   c. Error page rendering (if error)
6. Response returned to client
```

## Security Layers

### Layer 1: Authentication (Auth Service)
- Username/password with Argon2id hashing
- Multi-factor authentication (TOTP, WebAuthn/passkey)
- Biometric authentication (device-bound)
- Session management with configurable timeout

### Layer 2: Authorization (Policy Service)
- **RBAC**: Role-based access control with role-permission mappings
- **ABAC**: Attribute-based access control with policy evaluation
- **ReBAC**: Relationship-based access control (Google Zanzibar model)
- **JIT**: Just-in-time privilege elevation with approval workflow
- **PDP**: Policy Decision Point pattern — all authorization decisions flow through the policy engine

### Layer 3: Token Security (OAuth Service)
- OAuth 2.1 compliance (PKCE mandatory, no implicit/ROPC)
- DPoP (RFC 9449): Sender-constrained tokens via proof-of-possession
- TrustChainValidator: Federation entity validation (SAML/OIDC)
- Token introspection and revocation (RFC 7662/7009)

### Layer 4: Threat Detection (Audit Service)
- **ITDR**: Identity Threat Detection & Response — detects credential stuffing, impossible travel, brute force
- **CAE**: Continuous Authorization Evaluation — real-time risk scoring per request
- **Threat Intel**: IOC aggregation from external sources (OTX, AbuseIPDB, HIBP)
- **Anomaly Detection**: ML-based pattern detection on audit events

### Layer 5: Data Protection (Gateway + Crypto)
- **DLP Egress**: Real-time PII detection and redaction in API responses
- **Envelope Encryption**: Per-tenant AES-256-GCM data keys
- **Audit Integrity**: Hash-chained audit log (tamper-evident)

## Tech Stack

| Component | Technology | Version |
|-----------|-----------|--------|
| Backend | Go | 1.26 |
| Frontend | Next.js (React) | 15.x |
| Database | PostgreSQL | 16 |
| Cache | Redis | 7 |
| Message Bus | NATS | 2.10 |
| Container | Docker | 27 |
| Orchestration | k3s | 1.31 |
| CI/CD | GitHub Actions | - |

### Key Go Libraries

- **pgx/v5**: PostgreSQL driver (connection pooling)
- **go-redis/v9**: Redis client
- **nats.go**: NATS messaging client
- **gorilla/mux**: HTTP routing (gateway)
- **net/http**: HTTP servers (per-service)
- **grpc-go**: gRPC for inter-service communication
- **prometheus/client_golang**: Metrics export
- **zerolog**: Structured logging

## Deployment

### Docker All-in-One

The simplest deployment is via Docker Compose:

```bash
cd deploy/all-in-one
docker compose up -d
```

This starts all services, PostgreSQL, and Redis in containers.

### k3s (Production)

Production deployment uses k3s with the following pods:

```bash
export KUBECONFIG=~/.kube/config.k3s
kubectl get pod -n ggid
```

**Pods:**
- `ggid-gateway` — API gateway (port 8080)
- `ggid-auth` — Auth service (port 9001)
- `ggid-identity` — Identity service (port 8081)
- `ggid-oauth` — OAuth service (port 9005)
- `ggid-policy` — Policy service (port 8070)
- `ggid-audit` — Audit service (port 8072)
- `ggid-console` — Next.js frontend (port 3000)
- `ggid-postgres` — PostgreSQL database
- `ggid-redis` — Redis cache

### Build & Deploy Individual Services

```bash
# Build a service image
docker build --platform linux/amd64 -f services/auth/Dockerfile -t registry.iot2.win/ggid/auth:latest .

# Push to registry
docker push registry.iot2.win/ggid/auth:latest

# Restart the pod
export KUBECONFIG=~/.kube/config.k3s
kubectl rollout restart deployment/ggid-auth -n ggid
```

## Inter-Service Communication

Services communicate via two patterns:

1. **HTTP/REST**: Synchronous calls through the gateway for client-facing APIs.
2. **gRPC**: Direct service-to-service calls for internal communication (e.g., Policy ← OAuth for token validation, Audit ← Auth for event ingestion).

## Database

Each service has its own schema in a shared PostgreSQL instance:

- **auth**: users, sessions, mfa_factors, passkeys
- **identity**: groups, organizations, federation_entities, consent_records
- **oauth**: oauth_clients, authorization_codes, tokens, consents
- **policy**: roles, permissions, policies, relation_tuples
- **audit**: audit_events, itdr_detections, compliance_assessments
- **org**: organizations, departments

Schema migrations are managed via SQL files in each service's migration directory.

## Development Workflow

```bash
# Clone
git clone https://github.com/topcheer/ggid.git
cd ggid

# Build all Go services
go build ./...

# Run tests
make test

# Start frontend dev server
cd console && npm install && npm run dev

# Build console for production
docker build --platform linux/amd64 -f console/Dockerfile -t registry.iot2.win/ggid/console:latest .
```

## Key Directories

| Path | Contents |
|------|----------|
| `services/auth/` | Auth service (login, MFA, sessions) |
| `services/identity/` | Identity service (users, groups, SCIM) |
| `services/oauth/` | OAuth service (tokens, clients, SAML) |
| `services/policy/` | Policy service (RBAC, ABAC, ReBAC) |
| `services/audit/` | Audit service (events, ITDR, compliance) |
| `services/gateway/` | API gateway + middleware |
| `services/org/` | Organization service |
| `console/` | Next.js frontend |
| `pkg/` | Shared Go packages (crypto, errors, etc.) |
| `deploy/` | Docker Compose, k8s manifests |
| `sdk/` | Client SDKs (Go, Rust, Python, C#) |
| `docs/` | Documentation, guides, research |
