# GGID — Production-Grade Identity & Access Management Suite

> Apache 2.0 open-source IAM platform. Go 1.25 microservices + Next.js 15 console.

## Quick Reference

```bash
make build          # Build all 7 services
make test           # Run all tests (go test -race -cover ./...)
go test -tags=integration -v ./test/integration/  # E2E tests (requires Docker infra)
docker compose -f deploy/docker-compose.yaml up -d  # Start full stack
make proto          # Regenerate protobuf/gRPC code
```

## Architecture

7 microservices, each following `cmd/ → internal/{config,data,domain,repository,service,handler,server}` structure:

| Service | HTTP | gRPC | Responsibility |
|---------|------|------|---------------|
| Gateway | :8080 | — | JWT verification + reverse proxy to all services |
| Identity | :8081 | :50051 | User CRUD, lock/unlock, SCIM 2.0 |
| Auth | :9001 | — | Register/login, JWT, refresh, MFA (TOTP), WebAuthn, LDAP, password reset |
| OAuth | :9005 | — | OAuth2/OIDC, JWKS, SAML 2.0, discovery |
| Policy | :8070 | :9070 | RBAC + ABAC engine, roles, permissions, policy check |
| Org | :8071 | :9071 | Tenant CRUD, org tree (LTREE), departments, teams, memberships |
| Audit | :8072 | :9072 | Audit event query, NATS JetStream consumer |

Infrastructure: PostgreSQL 16 (RLS) · Redis 7 · NATS JetStream · OpenLDAP

## Shared Packages (`pkg/`)

- `errors` — GGIDError type with codes (NotFound, InvalidArgument, etc.) + `AsGGIDError()` helper
- `tenant` — Multi-tenant context propagation (`X-Tenant-ID` header)
- `crypto` — Argon2id password hashing, AES-256-GCM encryption
- `authprovider` — Auth provider chain: Local + LDAP
- `audit` — NATS JetStream event publisher (`Publisher.Publish/PublishAsync`)

## Service Layer Conventions

1. **Interfaces for testability** — Services depend on repo interfaces (e.g. `TenantRepo`, `AuditRepo`), not concrete `*repository.XxxRepository` types
2. **Mock repos in tests** — Each service test file defines mock implementations of its repo interface
3. **Domain models in `internal/domain/models.go`** — Plain structs with `uuid.UUID` IDs
4. **HTTP servers in `internal/server/http.go`** — REST endpoints registered alongside `/healthz`
5. **gRPC handlers in `internal/handler/`** — Convert between proto types and domain models
6. **`writeServiceError()`** — Maps `GGIDError` codes to HTTP status codes

## Testing

- `make test` = 15 packages, 250+ test cases, 0 FAIL
- Integration tests use `//go:build integration` tag, skip gracefully if services not running
- Coverage: errors 100% · tenant 100% · audit/service 93.8% · auth/domain 92.9% · authprovider 88.1% · audit/handler 83.3% · oauth 77.1% · auth/service 74.9% · identity 73.4% · org 68.4% · policy 54.6%

## Key Technical Notes

- **Go 1.25** module: `github.com/ggid/ggid`
- **All dependencies must use `@latest`** — never pin outdated versions
- **pgx v5** for PostgreSQL; SET LOCAL doesn't support `$1` parameters — use `fmt.Sprintf`
- **Proto generation**: `make proto` runs `buf generate` for all 6 service protos
- **Generated code**: `api/gen/{policy,org,audit}/v1/` — do not hand-edit
- **Optional proto fields** are `*string` in Go — use `strPtr()` helper in tests
- **NATS publisher** is best-effort in main.go — services run without NATS (audit events just get skipped)
- **RLS**: Docker uses superuser which bypasses RLS; production needs non-superuser role

## Docker

`deploy/docker-compose.yaml` defines 13 services:
- Infrastructure: postgres, redis, nats, ldap, ldap-seed
- Microservices: identity, auth, gateway, policy, org, audit, oauth
- Frontend: console (Next.js :3000)

Volumes: `ggid-pgdata`, `ggid-ldapdata`, `ggid-ldapconfig`, `ggid-configs`

## Console (`console/`)

Next.js 15 + Tailwind CSS. Pages: Dashboard, Users (CRUD+lock), Roles (tabs), Organizations (tree), Audit (events table), Login.
API proxy through Gateway at `:8080`.

## SDK (`sdk/`)

- **Go** (`sdk/go/`) — JWT verification middleware + permission checking
- **Node.js** (`sdk/node/`) — Express/Hono middleware with JWKS
- **Java** (`sdk/java/`) — Servlet Filter (`GGIDAuthFilter`)

## Team Conventions

- Don't modify other teammates' service files without coordination
- Run `go build ./...` before `go test`
- Use interface mocks, not real DB connections in unit tests
- Commit messages include `Co-Authored-By: ggcode <noreply@ggcode.dev>`
