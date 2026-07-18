# Contributing to GGID

Thank you for your interest in contributing to GGID! This document covers everything you need to get started.

## Development Setup

### Prerequisites

- **Go 1.25+**
- **Node.js 20+** (for Console UI)
- **Docker** (for PostgreSQL, Redis, NATS)
- **Make**

### Quick Start

```bash
# Clone the repository
git clone https://github.com/topcheer/ggid.git
cd ggid

# Start infrastructure (PostgreSQL, Redis, NATS)
make docker-run

# Apply database migrations
make migrate-up

# Build all services
make build

# Run tests
make test

# Start Console UI development server
cd console && npm install && npm run dev
```

## Project Structure

```
ggid/
├── api/               # Protobuf definitions + generated code
├── console/           # Next.js admin console
├── deploy/            # Docker, K8s, migrations
├── docs/              # Research, guides, kanban
├── pkg/               # Shared Go packages (crypto, tenant, auth)
├── sdk/               # 11 language SDKs
└── services/          # Microservices
    ├── gateway/       # API gateway (routing, middleware, plugins)
    ├── auth/          # Authentication (login, MFA, WebAuthn)
    ├── identity/      # User/group/device management
    ├── oauth/         # OAuth 2.1 / OIDC provider
    ├── policy/        # RBAC/ABAC/ReBAC authorization
    ├── audit/         # Audit log + ITDR + detection
    ├── org/           # Organization management
    └── mcp/           # MCP server integration
```

## Code Style

### Go

- Run `go fmt` before committing
- Run `make lint` (golangci-lint) to catch issues
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use meaningful variable names (no single letters except in short loops)
- Add doc comments to all exported functions

### TypeScript / React

- Use TypeScript strict mode
- Use functional components with hooks
- Run `npx tsc --noEmit` to verify compilation
- Follow existing patterns in `console/src/`

## Testing

### Running Tests

```bash
# Full test suite
make test

# Short tests only (skips integration tests)
make test-short

# Race detector
make test-race

# Coverage report
make coverage

# Single package
go test ./services/auth/internal/server/ -v
```

### Test Conventions

- Every new feature must have **≥3 tests**
- No in-memory maps as DB substitutes (use real PG or test containers)
- No `log.Printf` placeholders in production code
- No hardcoded mock data in handlers (DB-backed)
- Reference: [Team Acceptance Checklist](docs/team-acceptance-checklist.md)

## Commit Conventions

We follow [Conventional Commits](https://conventionalcommits.org/):

```
<type>(<scope>): <subject>
```

### Types

| Type | Use |
|------|-----|
| `feat` | New feature |
| `fix` | Bug fix |
| `security` | Security fix |
| `perf` | Performance improvement |
| `docs` | Documentation |
| `refactor` | Code refactoring |
| `test` | Test improvements |
| `chore` | Maintenance |
| `ci` | CI/CD changes |

### Examples

```
feat(auth): add passkey self-enrollment
fix(oauth): token exchange delegation chain order
security(audit): fix hash chain verification bypass
docs(research): add multi-region active-active doc
```

## Pull Request Process

1. **Fork** the repository and create a feature branch
2. **Write tests** for your changes (≥3 tests per feature)
3. **Run** `make test` to ensure all tests pass
4. **Run** `make lint` to ensure code quality
5. **Create a PR** with a clear description of changes
6. **Link** related issues/backlog items
7. **Wait for CI** — all checks must pass
8. **Address review feedback**

### PR Checklist

- [ ] Tests added (≥3 per feature)
- [ ] `make test` passes
- [ ] No hardcoded mock data
- [ ] No `log.Printf` in request handlers
- [ ] No in-memory maps replacing DB
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventional commits

## Architecture Decisions

Major architectural decisions are documented in [docs/research/](docs/research/). Browse the 48+ research documents to understand design rationale.

## Getting Help

- Check existing [Issues](https://github.com/topcheer/ggid/issues)
- Review the [Kanban](docs/kanban.md) for roadmap and priorities
- Read the [Research Library](docs/research/) for architectural context

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
