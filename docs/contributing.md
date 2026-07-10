# Contributing to GGID

Thank you for your interest in contributing to GGID! This guide covers
everything you need to get started — from setting up your development
environment to submitting pull requests.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Development Setup](#development-setup)
- [Code Style](#code-style)
- [Testing Requirements](#testing-requirements)
- [Pull Request Workflow](#pull-request-workflow)
- [Commit Message Format](#commit-message-format)
- [Branch Naming](#branch-naming)
- [Issue Templates](#issue-templates)
- [Code Review Checklist](#code-review-checklist)

---

## Code of Conduct

GGID follows the [Contributor Covenant 2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).
Be respectful, inclusive, and constructive in all interactions.

---

## Development Setup

### Prerequisites

```bash
# Required tools
go version    # 1.25+
node --version  # 20+ (for Console development)
docker --version  # 24+
make --version  # GNU Make

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### Clone and Build

```bash
git clone https://github.com/ggid/ggid.git
cd ggid

# Download dependencies
go mod download

# Verify build
go build ./...

# Run tests
make test

# Start the full stack locally
cd deploy && docker compose up -d
```

### Project Structure

```
ggid/
├── services/           # Microservices
│   ├── gateway/        # API Gateway (HTTP reverse proxy + JWT verify)
│   ├── identity/       # User & credential management
│   ├── auth/           # Authentication (password, MFA, LDAP, WebAuthn)
│   ├── oauth/          # OAuth 2.0 / OIDC provider
│   ├── policy/         # RBAC + ABAC policy engine
│   ├── org/            # Organization & group management
│   └── audit/          # Audit event pipeline (NATS JetStream)
├── pkg/                # Shared packages
│   ├── crypto/         # Cryptographic utilities
│   ├── tenant/         # Multi-tenancy context
│   ├── errors/         # Standardized error types
│   ├── social/         # Social login connectors
│   ├── saml/           # SAML 2.0 utilities
│   ├── webauthn/       # WebAuthn helpers
│   └── ...
├── console/            # Admin Console (Next.js 15)
├── sdk/                # Client SDKs (Go, Node, Java)
├── deploy/             # Docker Compose, Helm, scripts
├── docs/               # Documentation
└── test/               # Integration & E2E tests
```

---

## Code Style

### Go Conventions

GGID follows the standard Go conventions with additional project-specific rules:

1. **Run `gofmt` and `goimports`** before every commit:
   ```bash
   gofmt -w .
   goimports -w .
   ```

2. **Follow Effective Go** — https://go.dev/doc/effective_go

3. **Error handling** — Always wrap errors with context:
   ```go
   // Good
   if err := db.QueryRow(ctx, query, id).Scan(&name); err != nil {
       return fmt.Errorf("scanning user %s: %w", id, err)
   }

   // Bad — bare error return
   if err != nil {
       return err
   }
   ```

4. **Context propagation** — All I/O functions must accept `context.Context` as
   the first parameter:
   ```go
   func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error)
   ```

5. **Interface segregation** — Define interfaces at the consumer, not the producer:
   ```go
   // In the service that uses it
   type UserRepository interface {
       GetByID(ctx context.Context, id uuid.UUID) (*User, error)
       Create(ctx context.Context, user *User) error
   }

   // In the data layer — just implement it, don't declare the interface
   type pgUserRepo struct { pool *pgxpool.Pool }
   ```

6. **Naming** — Exported identifiers need doc comments:
   ```go
   // TokenValidator verifies JWT access tokens and extracts claims.
   type TokenValidator interface { ... }
   ```

7. **No panics in business logic** — Use error returns. Panics are only for
   truly unrecoverable situations (e.g., init-time config errors).

8. **Dependency injection** — Pass dependencies via constructors, not globals:
   ```go
   // Good
   func NewAuthService(repo UserRepository, cache Cache) *AuthService { ... }

   // Bad — global state
   var globalRepo UserRepository
   ```

### Linting

```bash
# Run linter
make lint
# or: golangci-lint run ./...

# Run vulnerability scanner
govulncheck ./...
```

#### Key Linter Rules

| Rule | Setting | Rationale |
|------|---------|-----------|
| `errcheck` | Enabled | Don't ignore returned errors |
| `govet` | Enabled | Catch common mistakes |
| `ineffassign` | Enabled | Detect ineffectual assignments |
| `staticcheck` | Enabled | Comprehensive static analysis |
| `unused` | Enabled | Remove dead code |
| `gocyclo` | max-complexity: 15 | Keep functions simple |
| `lll` | max-line-length: 120 | Readable line length |
| `gosec` | Enabled | Security-focused checks |

### Console (TypeScript/React)

```bash
cd console

# Lint
npm run lint

# Format
npm run format

# Type check
npx tsc --noEmit
```

---

## Testing Requirements

### Coverage Thresholds

| Package Type | Minimum Coverage | Target |
|-------------|-----------------|--------|
| `pkg/` (shared utilities) | 85% | 95%+ |
| `services/*/internal/service/` | 80% | 90%+ |
| `services/*/internal/server/` | 75% | 85%+ |
| `services/*/cmd/` | N/A (thin main) | — |

```bash
# Run all tests with coverage
make test

# Run single package
go test -v -race -cover ./services/auth/internal/service/...

# Run with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Test Conventions

1. **Table-driven tests** — Prefer table-driven for clarity:
   ```go
   func TestValidatePassword(t *testing.T) {
       tests := []struct {
           name    string
           input   string
           wantErr bool
       }{
           {"too short", "Ab1!", true},
           {"valid strong", "Str0ng!Pass123", false},
           {"no digits", "StrongPassword!", true},
       }
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               err := ValidatePassword(tt.input)
               if (err != nil) != tt.wantErr {
                   t.Errorf("ValidatePassword(%q) error = %v, wantErr %v",
                       tt.input, err, tt.wantErr)
               }
           })
       }
   }
   ```

2. **Use mocks, not real databases** — All tests use interface mocks:
   ```go
   // Use mockery to generate mocks
   mockery --name=UserRepository --output=./mocks
   ```

3. **Race detector** — Always run with `-race`:
   ```bash
   go test -race ./...
   ```

4. **No `time.Sleep` in tests** — Use channels or `eventually` patterns:
   ```go
   // Good
   done := make(chan struct{})
   go func() {
       defer close(done)
       // do work
   }()
   <-done

   // Bad — flaky
   time.Sleep(100 * time.Millisecond)
   ```

5. **Test naming** — `Test<Type>_<Method>_<Scenario>`:
   ```go
   func TestAuthService_Login_InvalidPassword(t *testing.T)
   func TestUserRepo_Create_DuplicateEmail(t *testing.T)
   ```

### Integration Tests

```bash
# Integration tests require running infrastructure
cd deploy && docker compose up -d postgres redis nats

# Run integration tests
go test -tags=integration -v ./test/integration/...

# E2E tests (full stack)
bash deploy/e2e-docker-test.sh
```

---

## Pull Request Workflow

### 1. Create a Branch

```bash
git checkout -b feat/add-password-history
```

### 2. Make Changes

Write code, add tests, update documentation.

### 3. Verify Locally

```bash
# Build
go build ./...

# Lint
golangci-lint run ./...

# Test
go test -race -cover ./...

# Console (if modified)
cd console && npm run lint && npm run build && cd ..
```

### 4. Commit

See [Commit Message Format](#commit-message-format) below.

### 5. Push and Create PR

```bash
git push origin feat/add-password-history
```

Create a PR using the template. Fill in all sections.

### PR Template

```markdown
## Summary

Brief description of what this PR changes and why.

## Changes

- [ ] Change 1
- [ ] Change 2

## Testing

- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes
- [ ] Coverage maintained or improved
- [ ] `golangci-lint run` passes
- [ ] Updated relevant documentation

## Breaking Changes

- [ ] None
- [ ] Yes (describe migration path)

## Related Issues

Fixes #123
```

### PR Review Process

1. **Automated checks** must pass (CI: build, test, lint, coverage)
2. **At least one reviewer** must approve
3. **Breaking changes** require two reviewers and a migration guide
4. **Squash merge** — All PRs are squash-merged to keep history clean

---

## Commit Message Format

GGID follows [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf` | Performance improvement |
| `test` | Adding or correcting tests |
| `chore` | Build process, tooling, dependencies |
| `ci` | CI/CD changes |
| `revert` | Reverting a previous commit |

### Scope (optional)

Service or package name: `auth`, `gateway`, `policy`, `pkg/crypto`, `console`, `docs`, `deploy`.

### Examples

```
feat(auth): add password history enforcement

Checks last 5 passwords and rejects reuse. Configurable via
PASSWORD_HISTORY_COUNT env var.

Closes #234

feat(gateway): add request ID middleware

Generates X-Request-ID for each request if not present.
Propagates to all downstream services.

fix(policy): correct RBAC check for nested role inheritance

Roles inherited through 3+ levels were not evaluated correctly
due to a missing recursive call in CheckPermission.

docs: expand deployment guide with Helm examples

chore(deps): bump pgx to v5.5.1

perf(crypto): use sync.Pool for bcrypt hash buffers

test(auth): add table-driven tests for password policy validation
```

### Rules

- **Subject line** — Imperative mood, lowercase, no period, max 72 chars
- **Body** — Wrap at 80 chars, explain *why* not just *what*
- **Footer** — Reference issues (`Closes #123`, `Fixes #456`) or add co-authors

---

## Branch Naming

| Pattern | Example | Use Case |
|---------|---------|----------|
| `feat/<description>` | `feat/password-history` | New feature |
| `fix/<description>` | `fix/rbac-nested-roles` | Bug fix |
| `docs/<description>` | `docs/deployment-guide` | Documentation |
| `refactor/<description>` | `refactor/auth-interfaces` | Code refactoring |
| `chore/<description>` | `chore/update-deps` | Maintenance |
| `hotfix/<description>` | `hotfix/jwt-validation` | Urgent production fix |

### Rules

- Use `kebab-case` (lowercase, hyphens)
- Keep branch names under 40 characters
- One feature/fix per branch

---

## Issue Templates

### Bug Report

```markdown
**Describe the bug**
Clear description of the unexpected behavior.

**To Reproduce**
1. Go to '...'
2. Click on '...'
3. See error

**Expected behavior**
What you expected to happen.

**Environment:**
- GGID version: [e.g., v1.2.0]
- Deployment: [Docker / Kubernetes / Source]
- Go version: [e.g., 1.25]

**Logs**
```
Paste relevant logs here.
```
```

### Feature Request

```markdown
**Is your feature request related to a problem?**
Description of the problem.

**Proposed solution**
Clear description of what you want.

**Alternatives considered**
Other solutions you've considered.

**Additional context**
Any other context, screenshots, or references.
```

---

## Code Review Checklist

### For Authors

Before requesting review:

- [ ] Code compiles (`go build ./...`)
- [ ] Tests pass (`go test -race ./...`)
- [ ] Linter passes (`golangci-lint run ./...`)
- [ ] Coverage maintained or improved
- [ ] No hardcoded secrets or credentials
- [ ] No `fmt.Println` debug statements left in code
- [ ] Doc comments on exported functions
- [ ] Relevant documentation updated
- [ ] Commit messages follow conventional format
- [ ] PR description is complete

### For Reviewers

When reviewing:

- [ ] **Correctness** — Does the code do what it claims?
- [ ] **Security** — No SQL injection, XSS, or credential leaks
- [ ] **Error handling** — Errors wrapped with context, not silently ignored
- [ ] **Context propagation** — `context.Context` passed to all I/O operations
- [ ] **Tenant isolation** — All queries scoped by `tenant_id`
- [ ] **No globals** — Dependencies injected, not global
- [ ] **Test quality** — Tests cover edge cases, not just happy path
- [ ] **No data races** — Concurrent code is safe (`-race` passes)
- [ ] **Performance** — No N+1 queries, unnecessary allocations, or blocking calls
- [ ] **Naming** — Clear, consistent, idiomatic Go naming
- [ ] **Documentation** — Public APIs have doc comments

---

## Release Process

Releases follow semantic versioning (`vMAJOR.MINOR.PATCH`):

1. **Patch** (`v1.2.3`) — Bug fixes only, backward compatible
2. **Minor** (`v1.3.0`) — New features, backward compatible
3. **Major** (`v2.0.0`) — Breaking changes, migration required

### Release Steps

1. Create release branch: `git checkout -b release/v1.3.0`
2. Update version in `version.go`
3. Update `CHANGELOG.md`
4. Tag: `git tag v1.3.0`
5. Push tag: `git push origin v1.3.0`
6. GitHub Actions builds and publishes Docker images + release artifacts

---

## Getting Help

- **GitHub Issues** — Bug reports and feature requests
- **GitHub Discussions** — Questions and general discussion
- **Documentation** — [docs/](./) directory
- **Code of Conduct** — conduct@ggid.dev

---

## References

- [Testing Guide](./testing-guide.md) — Detailed testing strategies
- [Architecture](./architecture.md) — System architecture overview
- [Developer Guide](./developer-guide.md) — Development deep-dive
- [Security Checklist](./security-checklist.md) — Security best practices
