# Contributing to GGID

Thank you for your interest in contributing to GGID!

## Development Setup

```bash
# Clone the repo
git clone https://github.com/ggid/ggid.git
cd ggid

# Start infrastructure
make docker-run

# Run migrations
make migrate-up

# Build all services
make build

# Run tests
make test
```

## Code Style

- **Go**: Follow [Effective Go](https://go.dev/doc/effective_go) and `gofmt`
- **Tests**: All new code must include unit tests. Aim for >60% coverage on service packages.
- **Proto**: Changes to `.proto` files require running `make proto`
- **Dependencies**: Always use the latest versions. Run `go get -u ./...` before submitting.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add user activation endpoint
fix: correct JWT expiry calculation
test: add policy evaluator edge cases
docs: update deployment guide
```

## Pull Request Process

1. Create a feature branch from `main`
2. Ensure `make test` passes with your changes
3. Ensure `go build ./...` compiles cleanly
4. Include tests for new functionality
5. Update documentation as needed

## Project Structure

Each microservice is self-contained under `services/`. Shared code lives in `pkg/`.
Do not modify another contributor's service without coordinating first.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
