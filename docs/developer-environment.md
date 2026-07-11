# Developer Environment Setup

> Complete guide to setting up a local development environment for the GGID IAM platform.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Go Backend Setup](#go-backend-setup)
4. [Next.js Console Setup](#nextjs-console-setup)
5. [Infrastructure Setup](#infrastructure-setup)
6. [Docker Compose Full Stack](#docker-compose-full-stack)
7. [Make Commands Reference](#make-commands-reference)
8. [VS Code Configuration](#vs-code-configuration)
9. [Debugging Tips](#debugging-tips)
10. [Common Errors and Solutions](#common-errors-and-solutions)
11. [Contribution Workflow](#contribution-workflow)
12. [Testing Guide](#testing-guide)

---

## Prerequisites

### Required Software

| Software | Minimum Version | Recommended | Check |
|----------|----------------|-------------|-------|
| **Go** | 1.25 | latest | `go version` |
| **Node.js** | 20 LTS | 22 LTS | `node --version` |
| **pnpm** | 9 | latest | `pnpm --version` |
| **Docker** | 24.0 | latest | `docker --version` |
| **Docker Compose** | v2 | latest | `docker compose version` |
| **PostgreSQL** | 16 | 16 | `psql --version` |
| **Redis** | 7 | 7 | `redis-cli --version` |
| **Git** | 2.40 | latest | `git --version` |

### Optional Software

| Software | Purpose |
|----------|---------|
| **Make** | Build automation (`make` commands) |
| **Protocol Buffers compiler** | Regenerate gRPC stubs (`protoc`) |
| **OpenLDAP** | Local LDAP testing |
| **NATS CLI** | Inspect NATS subjects (`nats` command) |
| **TablePlus/DBeaver** | Database GUI |
| **Postman/Insomnia** | API testing |

### OS-Specific Notes

#### macOS (Apple Silicon)

```bash
# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install go node pnpm docker git make postgresql@16 redis nats-server

# Protocol Buffers
brew install protobuf protoc-gen-go protoc-gen-go-grpc
```

#### Linux (Ubuntu/Debian)

```bash
# Go
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Node.js via NodeSource
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt-get install -y nodejs
sudo npm install -g pnpm

# Docker
curl -fsSL https://get.docker.com | sh

# Protocol Buffers
sudo apt install -y protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

---

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/ggid/ggid.git
cd ggid

# 2. Start infrastructure (PostgreSQL, Redis, NATS, OpenLDAP)
cd deploy && docker compose up -d postgres redis nats ldap
cd ..

# 3. Run database migrations
go run ./cmd/migrate up

# 4. Build and test
make build
make test

# 5. Start all services (in separate terminals or use Docker Compose)
make run-gateway
make run-auth
make run-identity
# ... or
cd deploy && docker compose up -d
```

---

## Go Backend Setup

### GOPATH and Module

The project uses Go modules. No specific GOPATH configuration is needed.

```bash
# Verify Go installation
go version
# Output: go version go1.25.0 darwin/arm64

# Download dependencies
go mod download

# Verify build
go build ./...
```

### Environment Variables

Create a `.env` file in the project root (not committed):

```bash
# Database
DATABASE_URL=postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable
DB_HOST=localhost
DB_PORT=5432
DB_USER=ggid
DB_PASSWORD=ggid
DB_NAME=ggid
DB_SSLMODE=disable

# Redis
REDIS_URL=redis://localhost:6379

# NATS
NATS_URL=nats://localhost:4222

# Auth Service
JWT_SECRET=dev-secret-change-in-production
AUTH_LISTEN=:9001

# OAuth Service
OAUTH_LISTEN=:9005

# Identity Service
IDENTITY_GRPC=:50051
IDENTITY_HTTP=:8081

# LDAP (optional)
LDAP_URL=ldap://localhost:389
LDAP_BIND_DN=cn=admin,dc=ggid,dc=io
LDAP_BIND_PASSWORD=admin
LDAP_BASE_DN=dc=ggid,dc=io
LDAP_USER_FILTER=(uid=%s)
LDAP_START_TLS=false
LDAP_AUTO_PROVISION=true

# Logging
LOG_LEVEL=debug
```

### Protocol Buffers

```bash
# Regenerate gRPC stubs (only needed when proto files change)
make proto

# Or manually
protoc --go_out=. --go-grpc_out=. proto/identity/v1/*.proto
```

---

## Next.js Console Setup

```bash
cd console

# Install dependencies
pnpm install

# Run development server
pnpm dev
# Console available at http://localhost:3000

# Build for production
pnpm build

# Type check
pnpm type-check

# Lint
pnpm lint
```

### Console Environment Variables

Create `console/.env.local`:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_DEFAULT_TENANT_ID=00000000-0000-0000-0000-000000000001
```

---

## Infrastructure Setup

### Option A: Docker Compose (Recommended for Development)

```bash
cd deploy

# Start ALL services + infrastructure
docker compose up -d

# Start only infrastructure (run services manually)
docker compose up -d postgres redis nats ldap

# Check status
docker compose ps

# View logs
docker compose logs -f gateway
docker compose logs -f auth
docker compose logs -f nats

# Stop everything
docker compose down

# Stop and remove volumes (DESTRUCTIVE)
docker compose down -v
```

### Option B: Local Install (macOS)

```bash
# Start PostgreSQL
brew services start postgresql@16
createdb ggid
psql -d ggid -c "CREATE USER ggid WITH PASSWORD 'ggid';"
psql -d ggid -c "GRANT ALL ON DATABASE ggid TO ggid;"

# Start Redis
brew services start redis

# Start NATS
nats-server -m 8222 &

# Start OpenLDAP (optional)
docker run -d --name ldap -p 389:389 \
  osixia/openldap:1.5.0
```

### Database Migration

```bash
# Run all migrations up
go run ./cmd/migrate up

# Check current migration status
go run ./cmd/migrate status

# Rollback last migration
go run ./cmd/migrate down

# Create new migration
go run ./cmd/migrate create add_new_table
```

---

## Docker Compose Full Stack

### Container Architecture

| Container | Port(s) | Purpose |
|-----------|---------|---------|
| **gateway** | 8080 | API Gateway |
| **identity** | 8081 | Identity service (HTTP + gRPC) |
| **auth** | 9001 | Auth service |
| **oauth** | 9005 | OAuth/OIDC service |
| **policy** | 8070, 9070 | Policy service (HTTP + gRPC) |
| **org** | 8071, 9071 | Organization service |
| **audit** | 8072, 9072 | Audit service |
| **console** | 3000 | Admin console |
| **postgres** | 5432 | PostgreSQL database |
| **redis** | 6379 | Redis cache |
| **nats** | 4222, 8222 | NATS messaging |
| **ldap** | 389 | OpenLDAP directory |

### E2E Test

```bash
# Start full stack
cd deploy && docker compose up -d

# Wait for healthchecks (30s)
sleep 30

# Run E2E test suite
bash deploy/e2e-docker-test.sh

# Expected: 11/11 PASS
```

---

## Make Commands Reference

| Command | Description |
|---------|-------------|
| `make build` | Build all Go binaries |
| `make test` | Run all unit tests |
| `make test-race` | Run tests with race detector |
| `make test-cover` | Run tests with coverage report |
| `make lint` | Run golangci-lint |
| `make proto` | Regenerate Protocol Buffer stubs |
| `make run-gateway` | Start gateway service |
| `make run-auth` | Start auth service |
| `make run-identity` | Start identity service |
| `make run-oauth` | Start OAuth service |
| `make run-policy` | Start policy service |
| `make run-org` | Start org service |
| `make run-audit` | Start audit service |
| `make docker-up` | Start full Docker Compose stack |
| `make docker-down` | Stop Docker Compose stack |
| `make clean` | Clean build artifacts |
| `make fmt` | Format all Go code |
| `make vet` | Run go vet |

---

## VS Code Configuration

### Recommended Extensions

```json
{
  "recommendations": [
    "golang.go",
    "ms-azuretools.vscode-docker",
    "dbaeumer.vscode-eslint",
    "esbenp.prettier-vscode",
    "bradlc.vscode-tailwindcss",
    "ms-kubernetes-tools.vscode-kubernetes-tools",
    "42crunch.vscode-openapi",
    "redhat.vscode-yaml"
  ]
}
```

### Settings (`.vscode/settings.json`)

```json
{
  "go.useLanguageServer": true,
  "gopls": {
    "ui.semanticTokens": true,
    "formatting.gofumpt": true
  },
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "go.testFlags": ["-v", "-race"],
  "go.coverOnSingleTest": true,
  "editor.formatOnSave": true,
  "[go]": {
    "editor.defaultFormatter": "golang.go",
    "editor.codeActionsOnSave": {
      "source.organizeImports": "explicit"
    }
  },
  "[typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  },
  "[typescriptreact]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  }
}
```

### Launch Configuration (`.vscode/launch.json`)

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Gateway",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/services/gateway/cmd",
      "envFile": "${workspaceFolder}/.env",
      "cwd": "${workspaceFolder}"
    },
    {
      "name": "Debug Auth",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/services/auth/cmd",
      "envFile": "${workspaceFolder}/.env",
      "cwd": "${workspaceFolder}"
    },
    {
      "name": "Debug Tests (current file)",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${fileDirname}"
    }
  ]
}
```

---

## Debugging Tips

### Go Debugging

```bash
# Debug a specific test
dlv test ./services/auth/internal/service/ -run TestLogin

# Attach to running process
dlv attach $(pgrep ggid-gateway)

# CPU profile
go test -cpuprofile=cpu.prof -bench=. ./services/gateway/...
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof -bench=. ./services/gateway/...
go tool pprof mem.prof
```

### NATS Debugging

```bash
# List all subjects with messages
nats sub "audit.>" --count 5

# Publish test message
nats pub audit.events.test '{"method":"GET","path":"/test"}'

# Check JetStream streams
nats stream ls

# Inspect specific stream
nats stream info AUDIT
```

### PostgreSQL Debugging

```sql
-- Check RLS policies
SELECT * FROM pg_policies WHERE schemaname = 'public';

-- Check active connections
SELECT pid, usename, datname, state, query FROM pg_stat_activity;

-- Check tenant context
SHOW app.tenant_id;

-- Explain query plan
EXPLAIN ANALYZE SELECT * FROM users WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
```

### Redis Debugging

```bash
# Monitor all commands
redis-cli monitor

# Check rate limiter keys
redis-cli keys "rate_limit:*"

# Check JTI anti-replay keys
redis-cli keys "jti:*"

# Check session keys
redis-cli keys "session:*"
```

### Gateway Request Tracing

```bash
# Enable debug logging
LOG_LEVEL=debug make run-gateway

# Test with curl + verbose output
curl -v http://localhost:8080/healthz
curl -v -H "Authorization: Bearer <jwt>" http://localhost:8080/api/v1/users
```

---

## Common Errors and Solutions

### 1. `too many errors` in Compilation

```
coverage_sprint24_test.go:459:24: too many errors
```

**Cause**: Stale build cache from concurrent teammate commits.

**Fix**:
```bash
go clean -testcache
go build ./...
make test
```

### 2. `SET LOCAL doesn't support $1 parameters`

```
ERROR: syntax error at or near "$1"
```

**Cause**: PostgreSQL `SET LOCAL` cannot use parameterized values.

**Fix**: Use `fmt.Sprintf` instead:
```go
// Wrong
conn.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)

// Right
conn.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
```

### 3. `connection refused` on Service Startup

**Cause**: Infrastructure (PostgreSQL/Redis/NATS) not started.

**Fix**:
```bash
cd deploy && docker compose up -d postgres redis nats
sleep 5
make run-gateway
```

### 4. `no go files listed`

**Cause**: Running `go build` without specifying packages.

**Fix**:
```bash
# Always specify packages
go build ./...
go test ./services/gateway/...
```

### 5. Auth Rate Limiting (429 Too Many Requests)

**Cause**: More than 5 failed login attempts triggers rate limit.

**Fix**:
```bash
# Restart auth container to clear rate limit
docker compose restart auth

# Or wait 60 seconds for the window to reset
```

### 6. Register Returns 409 Conflict

**Cause**: Auth handler reads `username` field (not `email`) as the credential identifier.

**Fix**: Always include `username` in registration payload:
```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "SecurePass123!"
}
```

### 7. Create Role Returns 500

**Cause**: Roles table has `UNIQUE(tenant_id, key)`. Empty key conflicts.

**Fix**: Always provide a unique `key` field:
```json
{
  "name": "Editor",
  "key": "editor",
  "description": "Content editor role"
}
```

### 8. NATS Healthcheck Fails

**Cause**: NATS monitoring port not enabled.

**Fix**: Ensure NATS starts with `-m 8222` flag:
```yaml
# docker-compose.yml
nats:
  command: ["-m", "8222"]
```

### 9. `unknown field` in Test Compilation

**Cause**: Referencing a struct field that doesn't exist (often from concurrent edits).

**Fix**: Check the struct definition:
```bash
grep -rn "type.*struct" services/gateway/internal/middleware/*.go
```

---

## Contribution Workflow

### 1. Create a Branch

```bash
git checkout main
git pull origin main
git checkout -b feature/my-new-feature
```

### 2. Write Code

- Follow existing patterns and conventions
- Run `go fmt` and `go vet` before committing
- Add tests for new functionality
- Update documentation

### 3. Run Pre-Commit Checks

```bash
# Format
make fmt

# Vet
make vet

# Build
make build

# Test
make test
```

### 4. Commit

```bash
git add -A
git commit -m "feat: add user impersonation feature

- Add impersonation endpoint in auth service
- Add audit logging for impersonation events
- Add scope check for admin impersonation

Closes #123"
```

**Commit message format**:
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation
- `refactor:` code refactoring
- `test:` test additions
- `chore:` maintenance tasks

### 5. Push and Create PR

```bash
git push origin feature/my-new-feature
```

Create a Pull Request with:
- Description of changes
- Link to related issue
- Test results (`make test` output)
- Breaking changes (if any)

### 6. Code Review

- All PRs require at least one approval
- CI must pass (build, test, lint)
- No decrease in test coverage
- Documentation updated

---

## Testing Guide

### Unit Tests

```bash
# Run all tests
make test

# Run specific package
go test -v ./services/auth/internal/service/...

# Run with race detector
go test -race -v ./services/gateway/...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run single test
go test -v -run TestLogin ./services/auth/internal/service/
```

### Integration Tests

```bash
# Run integration tests (requires infrastructure running)
go test -tags=integration -v ./test/integration/...
```

### Docker E2E Tests

```bash
cd deploy && docker compose up -d
sleep 30
bash deploy/e2e-docker-test.sh
```

### Test Coverage Targets

| Package | Current Coverage | Target |
|---------|-----------------|--------|
| pkg/errors | 100% | 100% |
| pkg/tenant | 100% | 100% |
| pkg/crypto | 89% | 90%+ |
| services/gateway/middleware | 90% | 92%+ |
| services/auth/service | 87% | 90%+ |
| services/oauth/service | 87% | 90%+ |
| services/policy/service | 97% | 95%+ |
| services/org/service | 99% | 95%+ |

---

*Last updated: 2025-07-11*
