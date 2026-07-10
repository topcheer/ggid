# GGID Developer Guide

Guide for contributors to the GGID IAM Platform. Covers code structure,
development setup, testing, coding standards, and PR workflow.

---

## Table of Contents

- [Project Structure](#project-structure)
- [Development Environment Setup](#development-environment-setup)
- [Building & Running](#building--running)
- [Testing Strategy](#testing-strategy)
- [Coding Standards](#coding-standards)
- [Service Layer Conventions](#service-layer-conventions)
- [Adding a New API Endpoint](#adding-a-new-api-endpoint)
- [Proto / gRPC Code Generation](#proto--grpc-code-generation)
- [File Ownership](#file-ownership)
- [PR Workflow](#pr-workflow)
- [Debugging Tips](#debugging-tips)
- [Adding a New Service](#adding-a-new-service)

---

## Project Structure

```
ggid/
├── services/                # 7 microservices
│   ├── gateway/             # API Gateway (JWT verify, reverse proxy)
│   │   ├── cmd/             # main.go entrypoint
│   │   └── internal/
│   │       ├── config/      # Configuration loading
│   │       ├── middleware/   # JWT, CORS, rate limit, metrics
│   │       ├── router/      # Reverse proxy + route table
│   │       └── healthcheck/ # Backend health aggregation
│   ├── identity/            # User CRUD, SCIM 2.0
│   │   ├── cmd/
│   │   └── internal/
│   │       ├── domain/      # User model, CreateUserInput
│   │       ├── service/     # Business logic
│   │       ├── repository/  # pgx queries
│   │       ├── server/      # HTTP handlers (REST)
│   │       ├── handler/     # gRPC handlers
│   │       └── scim/        # SCIM 2.0 protocol
│   ├── auth/                # Auth: register, login, JWT, MFA
│   │   ├── cmd/
│   │   └── internal/
│   │       ├── conf/        # Config + password policy
│   │       ├── domain/      # Credential, Token models
│   │       ├── service/     # AuthService, TokenService, etc.
│   │       ├── server/      # HTTP handlers
│   │       └── webauthn/    # WebAuthn/Passkey
│   ├── oauth/               # OAuth2/OIDC, SAML
│   ├── policy/              # RBAC + ABAC engine
│   ├── org/                 # Org tree, departments, teams
│   └── audit/               # Audit event query, NATS consumer
├── pkg/                     # Shared packages
│   ├── errors/              # GGIDError type + code mapping
│   ├── tenant/              # Multi-tenant context propagation
│   ├── crypto/              # Argon2id, AES-256-GCM, JWT signing
│   ├── authprovider/        # Auth chain: Local + LDAP
│   └── audit/               # NATS JetStream publisher
├── sdk/                     # Client SDKs
│   ├── go/                  # Go SDK (client + middleware)
│   ├── node/                # Node.js SDK
│   └── java/                # Java SDK
├── console/                 # Next.js 15 admin console
├── api/                     # Protobuf definitions
│   ├── proto/               # .proto source files
│   └── gen/                 # Generated Go code (DO NOT EDIT)
├── deploy/                  # Docker Compose, migrations, Helm
│   ├── docker-compose.yaml
│   ├── migrations/          # SQL migration files
│   └── e2e-docker-test.sh
├── docs/                    # Documentation
├── test/                    # Integration tests
├── Makefile
├── go.mod / go.sum
└── GGCODE.md                # Project conventions
```

### Service Internal Layout

Each microservice follows the same internal structure:

```
services/<name>/
├── cmd/
│   └── main.go              # Entrypoint: wire deps, start HTTP/gRPC
└── internal/                # Private to this service
    ├── conf/ or config/     # Environment-based configuration
    ├── domain/              # Domain models (plain structs)
    ├── repository/          # Database layer (pgx queries)
    ├── service/             # Business logic (interfaces + impls)
    ├── server/ or handler/  # HTTP REST handlers
    └── handler/             # gRPC handlers (if applicable)
```

---

## Development Environment Setup

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25+ | Build services |
| Docker | 24+ | Run infrastructure |
| Docker Compose | v2+ | Orchestrate containers |
| Node.js | 18+ | Console development |
| buf | 1.35+ | Proto code generation |
| PostgreSQL client | 16+ | Direct DB access (optional) |

### Setup Steps

```bash
# 1. Clone the repo
git clone https://github.com/ggid/ggid.git
cd ggid

# 2. Install Go dependencies
go mod download

# 3. Start infrastructure (PostgreSQL, Redis, NATS, LDAP)
cd deploy
docker compose up -d postgres redis nats ldap ldap-seed

# 4. Run database migrations
docker compose up migrate

# 5. Return to project root
cd ..

# 6. Build all services
make build
# or: go build ./...

# 7. Run tests
make test
```

### Console (Frontend) Setup

```bash
cd console
npm install
npm run dev    # Starts at localhost:3000
```

### Generate Proto Code

```bash
# Requires buf CLI
make proto
# or: buf generate
```

---

## Building & Running

### Build Commands

```bash
make build              # Build all service binaries
go build ./...          # Compile-check everything
make proto              # Regenerate protobuf code
```

### Running Services Locally (without Docker for the services)

Start infrastructure via Docker, then run service binaries directly:

```bash
# Terminal 1: Identity
DATABASE_URL="postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable" \
  go run ./services/identity/cmd

# Terminal 2: Auth
DATABASE_URL="postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable" \
REDIS_ADDR="localhost:6379" \
JWT_PRIVATE_KEY_PATH="configs/rsa_private.pem" \
JWT_PUBLIC_KEY_PATH="configs/rsa_public.pem" \
  go run ./services/auth/cmd

# Terminal 3: Policy
DB_HOST=localhost DB_PORT=5432 DB_USER=ggid DB_PASSWORD=ggid DB_DATABASE=ggid \
NATS_URL="nats://localhost:4222" \
  go run ./services/policy/cmd

# Terminal 4: Gateway
JWT_PUBLIC_KEY_PATH="configs/rsa_public.pem" \
  go run ./services/gateway/cmd
```

### Full Docker Stack

```bash
cd deploy && docker compose up -d
```

---

## Testing Strategy

### Test Pyramid

```
                    /\
                   /E2E\         <- test/integration/ (Docker required)
                  /------\
                 /  Integration  <- //go:build integration
                /----------------\
               /     Unit Tests    <- service/internal/*_test.go
              /--------------------\
```

### Unit Tests

Unit tests are the primary test type. They use **mock interfaces** (not real DB).

```bash
# Run all unit tests
make test

# Run a specific package
go test -v ./services/auth/internal/service/...

# Run with coverage
go test -cover ./services/identity/...

# Run with race detector
go test -race ./...
```

**Mock pattern example:**

```go
// In service_test.go
type mockUserRepo struct {
    users map[uuid.UUID]*domain.User
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
    m.users[user.ID] = user
    return nil
}

func TestCreateUser(t *testing.T) {
    repo := &mockUserRepo{users: make(map[uuid.UUID]*domain.User)}
    svc := service.NewIdentityService(repo)

    user, err := svc.CreateUser(ctx, &domain.CreateUserInput{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "password",
    })

    assert.NoError(t, err)
    assert.Equal(t, "testuser", user.Username)
}
```

### Integration Tests

Integration tests require running services. They use the `integration` build tag:

```bash
# Start the full Docker stack first
cd deploy && docker compose up -d

# Run integration tests
go test -tags=integration -v ./test/integration/...
```

Integration tests gracefully skip if services are not running.

### E2E Tests (Docker)

```bash
# Full end-to-end test through the Gateway
bash deploy/e2e-docker-test.sh
```

Tests: healthz, register, login, JWT verification, user CRUD, role CRUD,
org CRUD, audit query, error handling.

### Coverage Targets

| Package | Current Coverage | Target |
|---------|-----------------|--------|
| pkg/errors | 100% | Maintain |
| pkg/tenant | 100% | Maintain |
| audit/service | 93.8% | >90% |
| auth/domain | 92.9% | >90% |
| authprovider | 88.1% | >85% |
| audit/handler | 83.3% | >80% |
| oauth | 77.1% | >75% |
| auth/service | 74.9% | >70% |
| identity | 73.4% | >70% |
| org | 68.4% | >70% |
| policy | 54.6% | >60% |

---

## Coding Standards

### Go Conventions

1. **Follow `gofmt` and `golangci-lint`**
   ```bash
   gofmt -w .
   go vet ./...
   ```

2. **Use interfaces for dependencies** — Services depend on repo interfaces,
   not concrete types. This enables mock-based testing.

3. **Error handling** — Use `pkg/errors.GGIDError` for domain errors:
   ```go
   if user == nil {
       return ggiderrors.NewError(ggiderrors.NotFound, "user not found")
   }
   ```

4. **Context propagation** — Pass `context.Context` as the first parameter:
   ```go
   func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error)
   ```

5. **Use `uuid.UUID`** for all entity IDs.

6. **Dependencies must use `@latest`** — never pin outdated versions:
   ```bash
   go get github.com/some/package@latest
   ```

7. **Comments on exported identifiers** — Every exported function, type, and
   constant must have a doc comment:
   ```go
   // CreateUser creates a new user in the identity store.
   func (s *IdentityService) CreateUser(ctx context.Context, ...) (*User, error)
   ```

### Database Conventions

1. **Use pgx v5** for PostgreSQL queries (not `database/sql`).
2. **SET LOCAL** for RLS tenant context — does NOT support `$1` params;
   use `fmt.Sprintf` with validated UUIDs.
3. **Every multi-tenant table** has a `tenant_id UUID NOT NULL` column.
4. **Unique constraints** include `tenant_id` (e.g., `UNIQUE(tenant_id, key)`).

### HTTP Handler Conventions

1. **Register routes** in `internal/server/http.go`:
   ```go
   mux.HandleFunc("/api/v1/myresource", h.handleMyResource)
   ```

2. **Use `writeServiceError()`** to map `GGIDError` to HTTP status codes.
3. **Validate input** before calling the service layer.
4. **Return JSON** with `writeJSON(w, status, data)`.
5. **Support X-Tenant-ID header** via `tenant.FromContext()`.

### Console (TypeScript/React) Conventions

1. **Next.js App Router** — pages in `app/<route>/page.tsx`.
2. **Tailwind CSS** for styling.
3. **Server Components** by default; use `"use client"` only when needed.
4. **API calls** through the Gateway proxy at `/api/*`.

---

## Service Layer Conventions

### Interface-Based Design

```go
// Define the interface in the service package
type CredentialRepo interface {
    GetByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*domain.Credential, error)
    UpdateSecret(ctx context.Context, credID uuid.UUID, secret string) error
}

// Service depends on the interface, not the concrete repo
type AuthService struct {
    credentialRepo CredentialRepo  // interface, not *repository.CredentialRepository
}

// Tests use mock implementations
type mockCredentialRepo struct { ... }
```

### Error Mapping

```go
// In internal/server/http.go
func writeServiceError(w http.ResponseWriter, err error) {
    var ggidErr *ggiderrors.GGIDError
    if errors.As(err, &ggidErr) {
        switch ggidErr.Code {
        case ggiderrors.NotFound:
            writeError(w, http.StatusNotFound, ggidErr.Message)
        case ggiderrors.InvalidArgument:
            writeError(w, http.StatusBadRequest, ggidErr.Message)
        case ggiderrors.AlreadyExists:
            writeError(w, http.StatusConflict, ggidErr.Message)
        default:
            writeError(w, http.StatusInternalServerError, ggidErr.Message)
        }
        return
    }
    writeError(w, http.StatusInternalServerError, "internal server error")
}
```

---

## Adding a New API Endpoint

### Step-by-step Example: Adding `GET /api/v1/users/{id}/roles`

1. **Add domain model** (if needed) in `internal/domain/models.go`:
   ```go
   type UserRole struct {
       UserID uuid.UUID
       RoleID uuid.UUID
       RoleKey string
   }
   ```

2. **Add repo method** to the interface:
   ```go
   // In internal/service/identity_service.go
   type IdentityService struct {
       repo IdentityRepo  // add method to this interface
   }

   type IdentityRepo interface {
       // ... existing methods
       GetUserRoles(ctx context.Context, tenantID, userID uuid.UUID) ([]domain.UserRole, error)
   }
   ```

3. **Implement the repo method** in `internal/repository/`:
   ```go
   func (r *IdentityRepository) GetUserRoles(ctx context.Context, tenantID, userID uuid.UUID) ([]domain.UserRole, error) {
       rows, err := r.db.Query(ctx,
           "SELECT user_id, role_id, role_key FROM user_roles WHERE tenant_id = $1 AND user_id = $2",
           tenantID, userID)
       // ...
   }
   ```

4. **Add service method:**
   ```go
   func (s *IdentityService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.UserRole, error) {
       tc, _ := tenant.FromContext(ctx)
       return s.repo.GetUserRoles(ctx, tc.TenantID, userID)
   }
   ```

5. **Add HTTP handler** in `internal/server/http.go`:
   ```go
   // In registerRoutes()
   // Already handled by /api/v1/users/ path prefix

   // In handleUserByID():
   case action == "roles" && r.Method == http.MethodGet:
       h.getUserRoles(ctx, userID, w, r)

   // New handler:
   func (h *HTTPHandler) getUserRoles(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
       roles, err := h.svc.GetUserRoles(ctx, userID)
       if err != nil {
           writeServiceError(w, err)
           return
       }
       writeJSON(w, http.StatusOK, map[string]any{"roles": roles})
   }
   ```

6. **Write tests:**
   ```go
   func TestGetUserRoles(t *testing.T) {
       repo := &mockIdentityRepo{
           userRoles: []domain.UserRole{{RoleKey: "admin"}},
       }
       svc := NewIdentityService(repo)

       roles, err := svc.GetUserRoles(ctx, uuid.New())
       assert.NoError(t, err)
       assert.Len(t, roles, 1)
   }
   ```

7. **Build and verify:**
   ```bash
   go build ./services/identity/...
   go test ./services/identity/...
   ```

---

## Proto / gRPC Code Generation

### When to Modify Protos

Only when adding/changing gRPC methods. The REST API uses `net/http` directly
(not gRPC-Gateway), so REST endpoints don't need proto changes.

### Generating Code

```bash
make proto
# Equivalent to: buf generate
```

Generated code goes to `api/gen/{service}/v1/`. **Never hand-edit generated files.**

### Adding a New Proto Method

1. Edit the `.proto` file in `api/proto/<service>/v1/<service>.proto`
2. Run `make proto`
3. Implement the new method in `internal/handler/`
4. Update the gRPC server registration in `cmd/main.go`

---

## File Ownership

The GGID team follows a clear ownership model to avoid merge conflicts:

| Owner | Owns |
|-------|------|
| **dev** (ggcxf_dev) | Identity service, Auth Provider, OAuth/OIDC, MFA, SCIM, WebAuthn, Java SDK |
| **dev2** (ggcxf_dev2) | Auth Service (login/JWT/MFA flows), API Gateway, Auth interface refactoring |
| **dev3** (ggcxf_dev3) | Policy, Org, Audit services, REST APIs, NATS publisher, E2E tests |
| **arch** (ggcxf_arch) | Shared `pkg/` packages, SDK (Go/Node), Infrastructure, Console, Architecture |
| **doc** (ggcxf_doc) | All `docs/` files |

### Rules

1. **Don't modify another teammate's files** without coordination.
2. If you need a change in someone else's service, file an issue or message them.
3. Shared packages (`pkg/`) changes require arch approval.
4. Console changes require coordination with arch.
5. When in doubt, ask in the team chat.

---

## PR Workflow

### Before Submitting a PR

1. **Build passes:**
   ```bash
   go build ./...
   ```

2. **Tests pass:**
   ```bash
   make test
   ```

3. **Vet passes:**
   ```bash
   go vet ./...
   ```

4. **Formatted:**
   ```bash
   gofmt -l .  # should return no output
   ```

5. **Commit messages** follow conventional format:
   ```
   <type>: <description>

   Optional body explaining the change.

   Co-Authored-By: ggcode <noreply@ggcode.dev>
   ```

   Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`

### Review Criteria

- [ ] Code follows the service layer conventions (interfaces, mock-friendly)
- [ ] Unit tests added for new logic
- [ ] No new dependencies without justification
- [ ] Dependencies use `@latest`
- [ ] Error handling uses `GGIDError` for domain errors
- [ ] No hardcoded secrets or credentials
- [ ] Multi-tenant queries include `tenant_id`
- [ ] Breaking changes are documented

---

## Debugging Tips

### Enable Debug Logging

```bash
# Set log level via environment variable
LOG_LEVEL=debug go run ./services/auth/cmd

# Or in Docker
docker compose exec auth sh -c 'LOG_LEVEL=debug'
```

### Inspect JWT Contents

```bash
# Decode JWT header
echo $TOKEN | cut -d. -f1 | base64 -d 2>/dev/null | jq .

# Decode JWT payload
echo $TOKEN | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

### Debug Gateway Routing

```bash
# Check which routes are configured
docker compose logs gateway | grep -i "route\|proxy"

# Test a specific backend directly (bypass gateway)
curl http://localhost:8081/healthz   # identity
curl http://localhost:9001/healthz   # auth
curl http://localhost:8070/healthz   # policy
```

### Debug Database Queries

```bash
# Connect to PostgreSQL
docker exec -it ggid-postgres psql -U ggid -d ggid

# Check if RLS is enabled
SELECT relname, relrowsecurity FROM pg_class WHERE relname LIKE '%';

# Check tenant data
SELECT id, tenant_id, username FROM users LIMIT 10;

# Enable query logging
ALTER SYSTEM SET log_statement = 'all';
SELECT pg_reload_conf();
```

### Debug NATS

```bash
# Check NATS health
curl http://localhost:8222/healthz

# List JetStream streams
curl http://localhost:8222/jsz?streams=true | jq .

# Check consumer info
curl http://localhost:8222/jsz?consumers=true | jq .
```

### Use Delve (Go Debugger)

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a service
dlv debug ./services/auth/cmd --headless --listen=:2345 --api-version=2

# Connect from VS Code or another IDE
```

### Profile Performance

```bash
# Enable pprof in the service
import _ "net/http/pprof"
go http.ListenAndServe(":6060", nil)

# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Heap profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

## Adding a New Service

If you need to add an entirely new microservice:

1. **Create the directory structure:**
   ```
   services/<name>/
   ├── cmd/main.go
   ├── Dockerfile
   └── internal/
       ├── config/config.go
       ├── domain/models.go
       ├── service/service.go
       └── server/http.go
   ```

2. **Follow the existing service pattern** (look at `services/policy/` as a template).

3. **Register Gateway routes** in `services/gateway/internal/config/config.go`:
   ```go
   Routes: map[string]string{
       "/api/v1/<name>": "http://localhost:<port>",
   },
   ```

4. **Add to Docker Compose** in `deploy/docker-compose.yaml`.

5. **Add env vars** to the Gateway's `LoadFromEnv()`.

6. **Write tests** following the mock interface pattern.

7. **Add health check** at `/healthz`.

8. **Update `GGCODE.md`** with the new service in the architecture table.
