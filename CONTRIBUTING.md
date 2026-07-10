# Contributing to GGID

Thank you for your interest in contributing to GGID! This guide covers the development workflow.

---

## Development Setup

### Prerequisites

- **Go** 1.25+
- **Node.js** 20+ (for Console)
- **Docker** + Docker Compose
- **PostgreSQL** 16 (or use Docker)
- **Redis** 7 (or use Docker)
- **NATS** 2.x (or use Docker)

### Quick Start

```bash
# Clone
git clone https://github.com/ggid/ggid.git
cd ggid

# Start infrastructure
cd deploy && docker compose up -d postgres redis nats ldap
sleep 10

# Run migrations
deploy/migrate.sh

# Build all services
make build

# Run tests
make test

# Start Console
cd console && npm install && npm run dev
```

---

## Project Structure

```
ggid/
├── api/proto/          # Protobuf definitions
├── api/gen/            # Generated gRPC code
├── pkg/                # Shared libraries
│   ├── crypto/         # Encryption, hashing
│   ├── tenant/         # Multi-tenant context
│   ├── errors/         # Error types
│   ├── authprovider/   # Auth provider chain
│   ├── social/         # Social login connectors
│   └── audit/          # Audit event publisher
├── services/           # 7 microservices
│   ├── gateway/        # API Gateway
│   ├── identity/       # Identity Service
│   ├── auth/           # Auth Service
│   ├── oauth/          # OAuth/OIDC Service
│   ├── policy/         # Policy Engine
│   ├── org/            # Organization Service
│   └── audit/          # Audit Service
├── console/            # Admin Console (Next.js)
├── sdk/                # SDKs (Go, Node.js, Java, Python)
├── deploy/             # Docker Compose + Helm + k6
├── docs/               # Documentation
└── test/integration/   # E2E integration tests
```

---

## Code Style

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` / `goimports` (enforced in CI)
- Every public function needs a comment
- Error handling: always check errors, wrap with context
- Testing: table-driven tests, interface mocks (no real DB)

### TypeScript (Console)

- Strict mode enabled
- Functional components with hooks
- TailwindCSS for styling
- Recharts for data visualization

### Git Commits

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(auth): add WebAuthn passwordless login
fix(gateway): resolve tenant_id forwarding for POST requests
test(policy): coverage 91% → 93% with edge cases
docs(arch): security whitepaper and C4 architecture
ci: add Trivy security scanning
chore: upgrade Go dependencies
```

---

## Testing

### Run Tests

```bash
# All Go tests
make test

# Specific package
go test -v ./services/auth/...

# With coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Integration tests (requires Docker)
go test -tags=integration -v ./test/integration/

# Console
cd console && npm run build

# Docker E2E
bash deploy/e2e-docker-test.sh
```

### Coverage Targets

| Package | Target | Current |
|---------|--------|---------|
| pkg/crypto | 80%+ | 81.8% |
| pkg/tenant | 100% | 100% |
| pkg/errors | 100% | 100% |
| services/auth | 85%+ | 80.4% |
| services/policy | 90%+ | 93.3% |
| services/audit | 95%+ | 93.8% |
| services/gateway | 80%+ | 81.0% |

---

## Adding a New Feature

1. **Create a branch**: `git checkout -b feat/your-feature`
2. **Write code**: Follow the existing patterns
3. **Write tests**: Aim for 80%+ coverage on new code
4. **Run locally**: `make build && make test`
5. **Update docs**: If user-facing, update relevant docs
6. **Submit PR**: Include description, test results, screenshots (if UI)

---

## Adding a New Social Connector

1. Implement the `Connector` interface in `pkg/social/`:
   ```go
   type Connector interface {
       Name() string
       GetAuthURL(state string) string
       ExchangeCode(ctx context.Context, code string) (*UserInfo, error)
   }
   ```
2. Register in `pkg/social/registry.go`
3. Add tests in `pkg/social/<provider>_test.go`
4. Add connector button to Console login page

---

## Adding a New API Endpoint

1. **Define the route** in the service's HTTP handler
2. **Add gateway route** in `services/gateway/internal/router/router.go`
3. **Write handler tests** (mock repository)
4. **Add to OpenAPI spec** if external
5. **Update E2E test** in `deploy/e2e-docker-test.sh`

---

## Release Process

1. Update version in `deploy/helm/ggid/Chart.yaml`
2. Update `CHANGELOG.md`
3. Tag: `git tag v0.x.0`
4. Build Docker images: `cd deploy && docker compose build`
5. Push images to registry
6. Create GitHub Release with release notes

---

## Questions?

- Open an issue on GitHub
- Read the [architecture docs](docs/architecture.md)
- Check the [feature matrix](docs/feature-matrix.md)
