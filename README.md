# GGID

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**GGID** is a production-grade Identity & Access Management (IAM) suite, built with Go and React.

## Features

- **Authentication** — Password, MFA (TOTP/WebAuthn), Social Login (OAuth2/OIDC)
- **Multi-Backend Auth** — Local users, LDAP/Active Directory, external IdP integration
- **Authorization** — RBAC + ABAC policy engine
- **SSO** — SAML 2.0, OIDC Provider, OAuth2 Authorization Server
- **Multi-Tenancy** — Three isolation levels (shared RLS / schema / database)
- **Organization Management** — Tenant, org tree, departments, teams
- **Audit** — Full audit logging, compliance reporting
- **Developer Platform** — REST API, gRPC, SDKs (Go/Node.js/Java)

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.23+ / go-kratos v2 |
| Frontend | React 19 / Next.js 15 / TypeScript |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Message Queue | NATS JetStream |

## Quick Start

```bash
# Start infrastructure
make docker-run

# Run migrations
make migrate-up

# Build and run services
make build
./services/gateway/bin/gateway
```

## Project Structure

```
ggid/
├── api/proto/         # Protobuf definitions
├── pkg/               # Shared libraries
├── services/          # Microservices
│   ├── gateway/       # API Gateway
│   ├── identity/      # Identity Service
│   ├── auth/          # Authentication Service
│   ├── oauth/         # OAuth/OIDC Service
│   ├── policy/        # Policy Engine
│   ├── org/           # Organization Service
│   └── audit/         # Audit Service
├── console/           # Admin Console (Next.js)
├── sdk/               # SDK (Go, Node.js, Java)
└── deploy/            # Docker Compose + Helm
```

## License

Apache License 2.0
