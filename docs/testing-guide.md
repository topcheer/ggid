# GGID Testing Guide

Comprehensive testing strategy for the GGID IAM Platform.

## Test Pyramid

```
                    /\
                   /E2E\          deploy/e2e-docker-test.sh
                  /------\        (requires Docker stack)
                 /Integration\    //go:build integration
                /--------------\  (requires running services)
               /  Unit Tests   \  service/internal/*_test.go
              /------------------\ (mock-based, no external deps)
```

## Unit Tests

The primary test type — fast, isolated, mock-based.

### Running

```bash
make test                    # all packages
go test -v ./services/auth/internal/service/...
go test -race -cover ./...   # with race detector + coverage
```

### Mock Strategy

Services depend on **repo interfaces**, enabling mock substitution:

```go
// Interface in service package
type CredentialRepo interface {
    GetByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*domain.Credential, error)
}

// Mock in test file
type mockCredentialRepo struct {
    creds map[uuid.UUID]*domain.Credential
    err   error
}

func (m *mockCredentialRepo) GetByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*domain.Credential, error) {
    if m.err != nil { return nil, m.err }
    return m.creds[userID], nil
}

// Test uses mock
func TestLogin(t *testing.T) {
    repo := &mockCredentialRepo{creds: map[uuid.UUID]*domain.Credential{...}}
    svc := NewAuthService(repo)
    _, err := svc.Login(ctx, "user", "pass")
    assert.NoError(t, err)
}
```

### Coverage Targets

| Package | Current | Target |
|---------|---------|--------|
| pkg/errors | 100% | Maintain |
| pkg/tenant | 100% | Maintain |
| audit/service | 100% | Maintain |
| policy/service | 93.9% | >90% |
| auth/domain | 92.9% | >90% |
| authprovider | 88.1% | >85% |
| auth/service | 72.2% | >75% |
| identity/service | 72.3% | >75% |
| policy | 54.6% | >60% |

## Integration Tests

Tagged with `//go:build integration` — require running services.

```bash
# Start Docker stack
cd deploy && docker compose up -d
sleep 30

# Run integration tests
go test -tags=integration -v ./test/integration/...
```

Tests gracefully skip if services are unavailable.

## E2E Tests (Docker)

Full end-to-end through the Gateway:

```bash
bash deploy/e2e-docker-test.sh
```

**11 tests:**

| # | Test | Expected |
|---|------|----------|
| 1 | Gateway healthz | 200 |
| 2 | Register user | 201 |
| 3 | Login + JWT | 693+ chars |
| 4 | 401 without JWT | 401 |
| 5 | List users | 200 |
| 6 | Create role | 201 |
| 7 | List roles | 200 |
| 8 | Create org | 201 |
| 9 | Audit query | 200 |
| 10 | Wrong password | 401 |
| 11 | Duplicate register | 409 |

## k6 Performance Tests

```bash
k6 run deploy/k6/login-bench.js      # login benchmark
k6 run deploy/k6/api-bench.js        # full API benchmark
k6 run deploy/k6/mixed-workload.js   # mixed read/write
```

Key thresholds: p95 < 100ms, error rate < 1%.

## Test Conventions

1. **Run `go build ./...` before `go test`** — catch compilation errors first
2. **Use interface mocks** — never connect to real DB in unit tests
3. **Table-driven tests** for multi-scenario logic
4. **`t.Helper()`** in assertion helpers
5. **No `time.Sleep`** — use channels or `eventually` patterns
6. **Test file naming:** `*_test.go` in same package as code under test

## Debugging Test Failures

```bash
# Verbose output
go test -v ./services/auth/internal/service/... -run TestLogin

# No cache
go test -count=1 ./...

# Race detector
go test -race ./services/gateway/...

# Specific test with timeout
go test -v -timeout 30s -run TestCreateUser ./services/identity/...
```
