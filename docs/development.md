# Development Guide

Local development setup, codebase structure, coding conventions, testing
guide, PR workflow, and release process for GGID contributors.

---

## Table of Contents

- [Local Development Setup](#local-development-setup)
- [Codebase Structure](#codebase-structure)
- [Coding Conventions](#coding-conventions)
- [Testing Guide](#testing-guide)
- [PR Workflow](#pr-workflow)
- [Release Process](#release-process)

---

## Local Development Setup

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25+ | Build and run services |
| PostgreSQL | 16+ | Primary database |
| Redis | 7+ | Session cache, rate limiting |
| NATS | 2.10+ | Audit event streaming |
| Docker | 24+ | Containerized development |
| `protoc` | 25+ | gRPC code generation |
| Make | Any | Build automation |

### Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/ggid/ggid.git
cd ggid

# 2. Install Go tooling
make install-tools
# Installs: golangci-lint, protoc-gen-go, protoc-gen-go-grpc, mockgen

# 3. Start infrastructure (PostgreSQL, Redis, NATS, LDAP)
docker compose -f deploy/docker-compose.dev.yaml up -d

# 4. Run database migrations
make migrate-up

# 5. Build all services
make build

# 6. Run all services (each in a terminal tab)
make run-gateway    # :8080
make run-identity   # :8081
make run-auth       # :9001
make run-oauth      # :9005
make run-policy     # :8070
make run-org        # :8071
make run-audit      # :8072
```

### Development Database

```bash
# Connect to development database
docker exec -it ggid-postgres psql -U ggid -d ggid

# Seed test data
make seed-data

# Reset database
make migrate-down
make migrate-up
make seed-data
```

### Hot Reload (air)

```bash
# Install air
go install github.com/air-verse/air@latest

# Run a service with hot reload
cd services/auth && air
```

### VS Code Setup

```json
// .vscode/settings.json
{
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports",
  "gopls": {
    "ui.semanticTokens": true
  }
}
```

---

## Codebase Structure

```
ggid/
├── services/              # Microservices (one dir per service)
│   ├── gateway/           # API Gateway — HTTP entry point
│   │   ├── cmd/           # main.go
│   │   └── internal/
│   │       ├── config/    # Configuration loading
│   │       ├── middleware/ # HTTP middleware (JWT, CORS, rate limit)
│   │       ├── router/    # Route definitions, reverse proxy
│   │       └── transport/ # HTTP transport layer
│   │
│   ├── identity/          # User & Group management
│   │   ├── cmd/
│   │   └── internal/
│   │       ├── domain/    # Domain entities, interfaces
│   │       ├── service/   # Business logic
│   │       └── scim/      # SCIM 2.0 handler
│   │
│   ├── auth/              # Authentication (login, register, MFA)
│   │   ├── cmd/
│   │   └── internal/
│   │       ├── domain/    # Credential, session entities
│   │       ├── service/   # Auth logic, password, JWT
│   │       └── webauthn/  # WebAuthn ceremony handlers
│   │
│   ├── oauth/             # OAuth 2.1 / OIDC
│   │   ├── cmd/
│   │   └── internal/
│   │       └── service/   # Token issuance, PKCE, refresh rotation
│   │
│   ├── policy/            # RBAC + ABAC policy engine
│   │   ├── cmd/
│   │   └── internal/
│   │       ├── server/    # gRPC server
│   │       └── service/   # Policy evaluation, caching
│   │
│   ├── org/               # Organization management
│   │   ├── cmd/
│   │   └── internal/
│   │       └── service/
│   │
│   └── audit/             # Audit logging
│       ├── cmd/
│       └── internal/
│           ├── handler/   # REST query handler
│           ├── server/    # gRPC server
│           └── service/   # Event storage, NATS consumer
│
├── pkg/                   # Shared packages
│   ├── audit/             # Audit publisher (NATS)
│   ├── authprovider/      # Auth provider chain (Local, LDAP)
│   ├── crypto/            # AES, bcrypt, JWT helpers
│   ├── email/             # SMTP email sender
│   ├── errors/            # Standardized error types
│   ├── i18n/              # Internationalization
│   ├── notification/      # Multi-channel notification
│   ├── pii/               # PII encryption / masking
│   ├── saml/              # SAML 2.0 signing/verification
│   ├── social/            # Social login connectors (9 providers)
│   ├── tenant/            # Tenant context extraction
│   └── transport/         # Shared HTTP/gRPC transport helpers
│
├── proto/                 # Protocol Buffer definitions
│   ├── identity/v1/
│   ├── auth/v1/
│   ├── policy/v1/
│   ├── org/v1/
│   └── audit/v1/
│
├── sdk/                   # Client SDKs
│   ├── go/                # Go SDK
│   ├── node/              # Node.js / TypeScript SDK
│   └── java/              # Java SDK
│
├── console/               # Admin Console (Next.js 15)
│   ├── src/
│   │   ├── app/           # App router pages
│   │   ├── components/    # React components
│   │   └── lib/           # API client, auth helpers
│   └── package.json
│
├── deploy/                # Deployment configurations
│   ├── docker-compose.yaml
│   ├── docker-compose.dev.yaml
│   ├── Dockerfile.*       # Per-service Dockerfiles
│   └── helm/              # Kubernetes Helm charts
│
├── docs/                  # Documentation
├── test/                  # Integration and E2E tests
│   └── integration/
│
├── Makefile               # Build automation
└── go.mod
```

### Package Dependency Rules

```
services/* → pkg/*
services/* → proto/*
pkg/*      → no dependency on services/
sdk/*      → no dependency on services/ or pkg/ (standalone)
console/*  → independent (TypeScript)
```

> `pkg/` packages must never import from `services/`. This prevents circular
> dependencies.

---

## Coding Conventions

### Go Style

Follow [Effective Go](https://go.dev/doc/effective_go) and the
[Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

#### Naming

| Item | Convention | Example |
|------|-----------|---------|
| Packages | lowercase, single word | `crypto`, `tenant` |
| Exported | PascalCase | `HashPassword`, `AuthProvider` |
| Unexported | camelCase | `hashToken`, `validatePKCE` |
| Interfaces | `-er` suffix | `Authorizer`, `Publisher` |
| Constants | PascalCase | `MaxRetries`, `DefaultTimeout` |
| Acronyms | All caps | `userID`, `HTTPClient`, `JWTSecret` |

#### Error Handling

```go
// Define sentinel errors at package level
var (
    ErrUserNotFound    = errors.New("user not found")
    ErrInvalidPassword = errors.New("invalid password")
)

// Wrap errors with context
if err := s.repo.Create(ctx, user); err != nil {
    return fmt.Errorf("create user %s: %w", user.ID, err)
}

// Check with errors.Is
if errors.Is(err, ErrUserNotFound) {
    return nil, ErrNotFound
}
```

#### Context Propagation

```go
// Always pass context as first parameter
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
    // Extract tenant from context
    tenantID, ok := tenant.FromContext(ctx)
    if !ok {
        return nil, ErrMissingTenant
    }
    return s.repo.Get(ctx, tenantID, id)
}
```

#### Constructor Pattern

```go
// Config struct for optional parameters
type ServiceConfig struct {
    Timeout    time.Duration
    MaxRetries int
    Cache      Cache
}

// New constructor with sensible defaults
func NewService(repo Repository, cfg ServiceConfig) *Service {
    if cfg.Timeout == 0 {
        cfg.Timeout = 30 * time.Second
    }
    return &Service{repo: repo, cfg: cfg}
}
```

### Proto / gRPC Conventions

```protobuf
// Package versioning
package ggid.identity.v1;

// Request/response naming
message GetUserRequest {
    string user_id = 1;
}

message GetUserResponse {
    User user = 1;
}

// Service definition
service IdentityService {
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
}
```

### Testing Conventions

```go
// Test file: service_test.go (same package)
// Test function: TestSubject_Action_Condition
func TestUserService_Create_DuplicateEmail(t *testing.T) {
    // Arrange
    svc := NewService(mockRepo, testConfig())

    // Act
    _, err := svc.Create(ctx, validUser)

    // Assert
    require.ErrorIs(t, err, ErrDuplicateEmail)
}
```

---

## Testing Guide

### Test Commands

```bash
# Run all tests
make test

# Run tests for a specific package
go test -v ./services/auth/internal/service/...

# Run a specific test
go test -v -run TestUserService_Create ./services/auth/internal/service/

# Run with race detector
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run integration tests
go test -tags=integration -v ./test/integration/...
```

### Coverage Targets

| Package | Current | Target |
|---------|---------|--------|
| pkg/errors | 100% | 100% |
| pkg/tenant | 100% | 100% |
| pkg/crypto | 89% | 90% |
| pkg/authprovider | 97% | 95% |
| pkg/saml | 91% | 90% |
| services/auth/internal/service | 87% | 90% |
| services/oauth/internal/service | 96% | 95% |
| services/identity/internal/service | 99% | 95% |

### Mocking

Use `mockgen` for interface mocks:

```bash
# Generate mocks
go generate ./...
```

```go
//go:generate mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks

type Repository interface {
    Get(ctx context.Context, id uuid.UUID) (*User, error)
    Create(ctx context.Context, user *User) error
}

// In test:
mockRepo := mocks.NewMockRepository(gomock.NewController(t))
mockRepo.EXPECT().Get(gomock.Any(), userID).Return(nil, ErrNotFound)
```

### Table-Driven Tests

```go
func TestValidatePassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"too short", "Ab1!", true},
        {"no uppercase", "abcdef1!", true},
        {"valid", "Abcdefg1!", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.password)
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Integration Tests

```bash
# Start infrastructure
docker compose -f deploy/docker-compose.dev.yaml up -d

# Run integration tests (requires running services)
go test -tags=integration -v ./test/integration/...
```

---

## PR Workflow

### Branch Naming

```
feature/add-webauthn-recovery
fix/oauth-pkce-validation
docs/api-reference-update
refactor/auth-service-cleanup
```

### Commit Messages

```
type(scope): short description

Longer description explaining the change.

Fixes #123
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`

### PR Checklist

Before requesting review:

- [ ] `make build` passes
- [ ] `make test` passes (no new failures)
- [ ] `golangci-lint run` passes
- [ ] Coverage doesn't decrease by more than 2%
- [ ] New code has tests
- [ ] Public API changes documented
- [ ] Breaking changes noted in PR description
- [ ] `go mod tidy` run (no stray dependencies)

### Review Process

1. Automated CI checks: lint, test, build
2. Code review: at least 1 reviewer
3. For `pkg/` or `proto/` changes: at least 2 reviewers
4. Squash merge on approval
5. Auto-delete branch after merge

### Force Push

Never force-push to `main`, `release/*`, or after review approval.

---

## Release Process

### Versioning

GGID follows [Semantic Versioning](https://semver.org/):

```
MAJOR.MINOR.PATCH
  1     .2    .3
```

| Change | Version Bump |
|--------|-------------|
| Breaking API change | MAJOR |
| New feature, backward compatible | MINOR |
| Bug fix, backward compatible | PATCH |

### Release Steps

```bash
# 1. Create release branch
git checkout -b release/v1.2.0

# 2. Run full test suite
make test

# 3. Update version files
# - CHANGELOG.md
# - go.mod (if SDK version bumped)
# - console/package.json (console version)

# 4. Commit release prep
git commit -m "chore: prepare release v1.2.0"

# 5. Tag
git tag -a v1.2.0 -m "Release v1.2.0"

# 6. Push
git push origin release/v1.2.0
git push origin v1.2.0

# 7. CI builds Docker images and publishes:
#    - ggid/gateway:v1.2.0
#    - ggid/auth:v1.2.0
#    - ... (all services)

# 8. Deploy to staging
# 9. Smoke test
# 10. Promote to production
```

### CHANGELOG Format

```markdown
## [1.2.0] - 2024-01-15

### Added
- WebAuthn passkey authentication
- OAuth 2.1 device flow (RFC 8628)
- SCIM 2.0 bulk operations

### Changed
- JWT access token lifetime reduced to 15 minutes
- PKCE now required for all OAuth clients

### Fixed
- LDAP group mapping not applying for nested groups
- Token refresh race condition causing session loss

### Deprecated
- Implicit grant flow (removed in v2.0)

### Security
- Fixed CSRF token predictability (CVE-2024-GGID-001)
- Added rate limiting to auth endpoints
```

### Docker Image Tags

```
ggid/gateway:latest        # Latest release
ggid/gateway:v1.2.0        # Semantic version
ggid/gateway:v1.2          # Minor version (latest patch)
ggid/gateway:sha-abc123    # Git commit SHA
```
