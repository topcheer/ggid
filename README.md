# GGID — Production-Grade Identity & Access Management Suite

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](#)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8)](#)
[![Tests](https://img.shields.io/badge/tests-19%20packages%2C%200%20FAIL-brightgreen)](#)
[![Docker](https://img.shields.io/badge/Docker-13%20containers-blue)](#)
[![Coverage](https://img.shields.io/badge/coverage-75%25%2B-green)](#)

**GGID** is a full-stack IAM platform: authentication, authorization, SSO, multi-tenancy, audit logging, and admin console. Built with Go microservices and React.

## Quick Start

### Option A: Docker Compose (Recommended)

```bash
# Start all services (PostgreSQL, Redis, NATS, LDAP, 7 microservices, Console)
cd deploy && docker compose up -d

# Wait for healthchecks
sleep 30

# Run E2E tests
bash deploy/e2e-docker-test.sh
```

Access points:
| Service | URL |
|---------|-----|
| Admin Console | http://127.0.0.1:3000 |
| Hosted Login | http://127.0.0.1:8080/login |
| API Gateway | http://127.0.0.1:8080 |
| Swagger UI | http://127.0.0.1:8080/docs |

Default credentials: `admin / Admin@123456`

### Option B: From Source

### 1. Start Infrastructure

```bash
docker compose -f deploy/docker-compose.yaml up -d postgres redis nats ldap
```

### 2. Run Database Migrations

```bash
# Create database (first time only)
docker exec ggid-postgres psql -U ggid -d postgres -c "CREATE DATABASE ggid"

# Run migrations
deploy/migrate.sh
```

### 3. Generate RSA Keys (for JWT)

```bash
mkdir -p configs
openssl genpkey -algorithm RSA -out configs/rsa_private.pem -pkeyopt rsa_keygen_bits:2048
openssl rsa -pubout -in configs/rsa_private.pem -out configs/rsa_public.pem
```

### 4. Build & Start Services

```bash
make build

# Terminal 1: Identity Service
DATABASE_URL="postgres://ggid:ggid@127.0.0.1:5432/ggid?sslmode=disable" \
  ./bin/identity --http-addr=:8081 --grpc-addr=:50051

# Terminal 2: Auth Service
DATABASE_URL="postgres://ggid:ggid@127.0.0.1:5432/ggid?sslmode=disable" \
REDIS_ADDR="127.0.0.1:6379" AUTH_HTTP_ADDR=":9001" \
JWT_PRIVATE_KEY_PATH="configs/rsa_private.pem" \
JWT_PUBLIC_KEY_PATH="configs/rsa_public.pem" \
  ./bin/auth

# Terminal 3: API Gateway
GATEWAY_ADDR=":8080" ./bin/gateway
```

### 5. Test the Auth Flow

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","email":"admin@test.local","password":"AdminPassw0rd123!"}'

# Login → get JWT
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"AdminPassw0rd123!"}'

# Access protected API
curl http://localhost:8080/api/v1/users \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Authorization: Bearer <your-jwt>"
```

### 6. Start Admin Console

```bash
cd console
npm install
npm run dev
# Open http://localhost:3000
```

## Features

- **Authentication** — Password (Argon2id), MFA (TOTP + WebAuthn/Passkey + Email OTP), Magic Link, LDAP/AD
- **Social Login** — Google, GitHub, Microsoft, Discord, LinkedIn, Slack, GitLab
- **Enterprise SSO** — OAuth2/OIDC, SAML 2.0, Generic OIDC federation
- **Authorization** — RBAC + ABAC hybrid policy engine with wildcard matching and role hierarchy
- **Multi-Tenancy** — PostgreSQL Row-Level Security (defense in depth)
- **API Gateway** — JWT verification (RS256+JWKS), rate limiting, CORS, circuit breaker, compression, OTel tracing
- **Audit** — NATS JetStream event pipeline + queryable log + CSV export + SSE streaming + anomaly detection
- **Admin Console** — Next.js 15 + Tailwind CSS (10 pages)
- **SDK** — Go / Node.js / Java / Python
- **SCIM 2.0** — Standard user provisioning protocol
- **Webhooks** — Pre/post auth hooks, HMAC-signed payloads
- **Auth Hooks** — Extensible plugin system (pre-registration, post-login, pre-token-issue)

## Why GGID?

| Feature | GGID | Auth0 | Keycloak | Ory |
|---------|------|-------|----------|-----|
| **License** | Apache 2.0 | Proprietary | Apache 2.0 | Apache 2.0 |
| **Self-hosted** | Yes | No | Yes | Yes |
| **Multi-tenancy** | RLS (defense in depth) | Built-in | Realms | Partial |
| **Language** | Go (fast, low memory) | Node.js | Java | Go |
| **Image size** | 18-35 MB per service | N/A | 600MB+ | 50MB+ |
| **OAuth 2.1 + PKCE** | Yes | Yes | Yes | Yes |
| **SAML 2.0** | Yes | Paid tier | Yes | No |
| **WebAuthn/Passkeys** | Yes | Yes | Yes | Yes |
| **SCIM 2.0** | Yes | Paid tier | Yes | No |
| **Audit pipeline** | NATS JetStream | Logs | DB | DB |
| **Admin Console** | Next.js 15 | Hosted | React | No |
| **AI Agent Identity (MCP)** | Planned | In dev | No | No |

## Documentation

| Document | Description |
|----------|-------------|
| [Quick Start](docs/quick-start.md) | 5-min guide: Docker → register → login → JWT |
| [Integration Guide](docs/integration-guide.md) | Third-party developer integration (SDK + JWT + middleware) |
| [OpenAPI Spec](docs/openapi.yaml) | Complete REST API reference (Swagger/OpenAPI 3.1) |
| [API Examples](docs/api-examples.md) | curl examples for every endpoint |
| [SDK Guide](docs/sdk-guide.md) | Go / Node.js / Java / Python side-by-side comparison |
| [Deployment Guide](docs/deployment.md) | Production deployment (Docker, K8s, TLS, backup) |
| [Security Hardening](docs/security-hardening.md) | Production security checklist |
| [Security Audit](docs/security-audit-checklist.md) | OWASP Top 10 alignment |
| [Performance Tuning](docs/performance.md) | DB indexing, connection pools, pprof |
| [Migration Guide](docs/migration-guide.md) | Auth0 / Keycloak → GGID |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and fixes |
| [Plugin Development](docs/plugin-development.md) | Auth hooks and webhooks |
| [Console Guide](docs/console-guide.md) | Admin Console user manual |
| [Developer Guide](docs/developer-guide.md) | Code structure, testing, PR workflow |
| [Contributing](docs/contributing-quickstart.md) | 5-min contributor quick start |
| [Testing Guide](docs/testing-guide.md) | Unit / integration / E2E / k6 |
| [FAQ](docs/faq.md) | Frequently asked questions |
| [Changelog](docs/CHANGELOG.md) | v1.0 release notes |
| [ADRs](docs/adr/) | Architecture Decision Records |
| [Design Docs](docs/design/) | RLS, NATS audit, policy engine designs |

## Architecture

```
┌──────────────┐    ┌──────────────────────────────────────────────┐
│  Admin Console │    │              API Gateway (:8080)              │
│  (Next.js)     │───▶│  JWT Verification · Routing · Rate Limit     │
└──────────────┘    └──────┬──────┬──────┬──────┬──────┬──────┬──────┘
                           │      │      │      │      │      │
                    ┌──────▼──┐┌──▼───┐┌─▼────┐│┌─────▼──┐┌─▼────┐
                    │Identity ││ Auth ││OAuth ││ Policy  ││ Audit│
                    │ (:8081) ││(:9001)││(:9005)││ (:8070) ││(:8072)│
                    └─────────┘└──────┘└──────┘└─────────┘└──────┘
                                         ┌──────────┐
                                         │ Org Svc  │
                                         │ (:8071)  │
                                         └──────────┘
                    ┌───────────────────────────────────────────────┐
                    │ PostgreSQL 16  ·  Redis 7  ·  NATS  ·  LDAP  │
                    └───────────────────────────────────────────────┘
```

## API Endpoints

| Service | Endpoints |
|---------|-----------|
| Auth | `/api/v1/auth/register`, `/login`, `/refresh`, `/mfa/*` |
| Identity | `/api/v1/users` (CRUD + lock/unlock) |
| Policy | `/api/v1/roles`, `/permissions`, `/policies`, `/policies/check` |
| Org | `/api/v1/orgs`, `/departments`, `/teams`, `/memberships` |
| Audit | `/api/v1/audit/events` |
| OAuth | `/oauth/authorize`, `/oauth/token`, `/oauth/jwks`, `/.well-known/openid-configuration` |
| SAML | `/saml/metadata`, `/saml/acs`, `/saml/sso` |
| SCIM | `/scim/v2/Users` |

## Development

```bash
# Run tests
make test                    # 15 packages, 200+ test cases

# Integration tests (requires Docker)
go test -tags=integration -v ./test/integration/

# Build all services
make build
```

## Project Structure

```
ggid/
├── api/proto/          # Protobuf definitions
├── api/gen/            # Generated gRPC code
├── pkg/                # Shared libraries (crypto, tenant, errors, authprovider, audit)
├── services/           # 7 microservices
│   ├── gateway/        # API Gateway (:8080)
│   ├── identity/       # Identity Service (:8081)
│   ├── auth/           # Auth Service (:9001)
│   ├── oauth/          # OAuth/OIDC Service (:9005)
│   ├── policy/         # Policy Engine (:8070)
│   ├── org/            # Organization Service (:8071)
│   └── audit/          # Audit Service (:8072)
├── console/            # Admin Console (Next.js :3000)
├── sdk/                # SDK (go, node, java)
├── deploy/             # Docker Compose + Helm Chart
└── test/integration/   # E2E integration tests
```

## License

Apache License 2.0
