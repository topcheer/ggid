# GGID — Identity & Access Management Suite

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Build](https://img.shields.io/badge/build-passing-green)](Makefile)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8)](https://go.dev)

**GGID** is a production-grade Identity & Access Management platform built with Go and React.

## Features

- **Authentication** — Password, MFA (TOTP), Social Login (OAuth2/OIDC), LDAP/AD
- **Authorization** — RBAC + ABAC policy engine with role inheritance
- **SSO** — SAML 2.0 IdP, OIDC Provider, OAuth2 Authorization Server
- **Multi-Tenancy** — PostgreSQL RLS row-level isolation
- **Audit** — Full audit logging via NATS JetStream
- **Admin Console** — Next.js 15 + Tailwind CSS web UI
- **SDK** — Go, Node.js, Java client libraries
- **SCIM 2.0** — Enterprise user provisioning (skeleton)
- **Passkey/WebAuthn** — Passwordless auth (skeleton)

## Quick Start

### 1. Start Infrastructure

```bash
docker compose -f deploy/docker-compose.yaml up -d postgres redis nats ldap
```

### 2. Run Database Migrations

```bash
bash deploy/migrate.sh
```

### 3. Build All Services

```bash
make build
```

### 4. Start Services

```bash
# Terminal 1: Identity Service
DATABASE_URL="postgres://ggid:ggid@127.0.0.1:5432/ggid?sslmode=disable" \
./services/identity/bin/identity --http-addr=:8081 --grpc-addr=:50052

# Terminal 2: Auth Service
DATABASE_URL="postgres://ggid:ggid@127.0.0.1:5432/ggid?sslmode=disable" \
REDIS_ADDR="127.0.0.1:6379" AUTH_HTTP_ADDR=":9001" \
JWT_PRIVATE_KEY_PATH="configs/rsa_private.pem" \
JWT_PUBLIC_KEY_PATH="configs/rsa_public.pem" \
./services/auth/bin/auth

# Terminal 3: API Gateway
GATEWAY_ADDR=":8080" ./services/gateway/bin/gateway
```

### 5. Test the Auth Flow

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","email":"admin@test.local","password":"AdminPassw0rd123!"}'

# Login → Get JWT
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"AdminPassw0rd123!"}'

# List Users (with JWT)
curl http://localhost:8080/api/v1/users \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Authorization: Bearer <your-jwt-token>"
```

### 6. Start Admin Console

```bash
cd console
npm install && npm run dev
# Open http://localhost:3000
```

## Architecture

```
┌─────────────────────────────────────────────────────┐
│              Admin Console (Next.js)                 │
├─────────────────────────────────────────────────────┤
│              API Gateway (:8080)                     │
│         JWT Verification · Routing · CORS           │
├──────────┬──────────┬──────────┬──────────┬─────────┤
│ Identity │   Auth   │  Policy  │   Org    │  Audit  │
│  (:8081) │ (:9001)  │  (:8070) │ (:8071)  │ (:8072) │
├──────────┴──────────┴──────────┴──────────┴─────────┤
│  PostgreSQL 16 · Redis 7 · NATS JetStream · LDAP    │
└─────────────────────────────────────────────────────┘
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.25 / go-kratos v2 |
| Frontend | React 19 / Next.js 15 / Tailwind CSS v4 |
| Database | PostgreSQL 16 (RLS, LTREE, partitioning) |
| Cache | Redis 7 |
| Message Queue | NATS JetStream |
| Auth | JWT RS256, Argon2id, TOTP |
| Deployment | Docker Compose / Helm / Kubernetes |

## Testing

```bash
# Unit tests (200+ test cases)
make test

# Integration tests (requires Docker)
go test -tags=integration -v ./test/integration/
```

## Project Structure

```
ggid/
├── api/proto/          # Protobuf definitions
├── api/gen/            # Generated gRPC code
├── pkg/                # Shared libraries (crypto, tenant, errors, audit)
├── services/           # 7 microservices
│   ├── gateway/        # API Gateway (:8080)
│   ├── identity/       # User management (:8081)
│   ├── auth/           # Authentication (:9001)
│   ├── oauth/          # OAuth/OIDC/SAML (:9005)
│   ├── policy/         # RBAC/ABAC engine (:8070)
│   ├── org/            # Organization management (:8071)
│   └── audit/          # Audit logging (:8072)
├── console/            # Admin Console (Next.js)
├── sdk/                # SDK (Go, Node.js, Java)
├── deploy/             # Docker Compose + Helm + Migrations
└── test/integration/   # E2E integration tests
```

## License

Apache License 2.0
