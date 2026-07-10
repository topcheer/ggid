# GGID Testing Guide

This guide covers how to write, run, and maintain tests for the GGID IAM platform.

---

## Quick Start

```bash
# Run all tests with coverage
make test

# Run a specific package
go test -v ./pkg/crypto/

# Run a specific test
go test -v -run TestCrypto_AES ./pkg/crypto/

# Run with race detector
go test -race ./...

# Run integration tests (requires NATS, PostgreSQL)
go test -tags=integration -v ./test/integration/
```

---

## Test Commands

| Command | Description |
|---------|-------------|
| `make test` | Run all unit tests with coverage |
| `go test -race ./...` | Run with race detector |
| `go test -coverprofile=coverage.out ./...` | Generate coverage profile |
| `go tool cover -html=coverage.out` | View coverage in browser |
| `go test -tags=integration ./...` | Run integration tests |
| `bash deploy/e2e-docker-test.sh` | Run Docker E2E tests |

---

## Coverage Targets

| Package | Current | Target | Owner |
|---------|---------|--------|-------|
| pkg/errors | 100% | 100% | arch |
| pkg/tenant | 100% | 100% | arch |
| pkg/saml | 100% | 100% | dev |
| pkg/i18n | 100% | 100% | arch |
| pkg/authprovider | 97% | 95% | dev |
| pkg/config | 97% | 95% | arch |
| pkg/healthcheck | 97% | 95% | arch |
| pkg/pii | 96.6% | 95% | arch |
| pkg/social | 92.8% | 95% | dev |
| pkg/notification | 91.5% | 90% | arch |
| pkg/crypto | 89.4% | 90% | arch |
| pkg/audit | 82.4% | 85% | arch |
| pkg/email | 80.2% | 80% | arch |
| services/audit/service | 100% | 100% | dev3 |
| services/audit/server | 93.2% | 90% | dev3 |
| services/auth/domain | 98.5% | 95% | dev |
| services/auth/service | 86.0% | 85% | dev |
| services/auth/webauthn | 81.7% | 85% | dev |
| services/gateway/middleware | 86.7% | 90% | dev2 |
| services/gateway/router | 95.1% | 95% | dev2 |
| services/gateway/transport | 100% | 100% | dev2 |
| services/gateway/webhooks | 100% | 100% | dev2 |
| services/identity/service | 98.5% | 95% | dev |
| services/oauth/service | 95.7% | 95% | dev |
| services/org/service | 98.7% | 95% | dev3 |
| services/policy/service | 97.1% | 95% | dev3 |
| services/policy/server | 90.9% | 90% | dev3 |

---

## Test Naming Conventions

Follow Go community conventions:

```go
// Unit tests
func TestFunctionName_Scenario(t *testing.T) { }

// Table-driven tests
func TestVerifyPassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        hash     string
        want     bool
        wantErr  bool
    }{
        {"valid", "pw", hash, true, false},
        {"invalid", "wrong", hash, false, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := VerifyPassword(tt.password, tt.hash)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Naming Patterns

| Pattern | Example | When to use |
|---------|---------|-------------|
| `TestFunc_Success` | `TestHashPassword_Success` | Happy path |
| `TestFunc_ErrorCase` | `TestVerifyPassword_InvalidFormat` | Error path |
| `TestFunc_EdgeCase` | `TestGenerateRandomToken_ZeroLength` | Boundary |
| `TestFunc_RoundTrip` | `TestAES_RoundTrip` | Serialize/deserialize |
| `TestFunc_Concurrency` | `TestJWKS_ConcurrentAccess` | Race conditions |

---

## Mock Patterns

### Interface-based mocks

GGID uses interface-based mocking. Define interfaces for external dependencies:

```go
// Production code
type CredentialStore interface {
    GetCredential(id uuid.UUID) (*Credential, error)
    SaveCredential(cred *Credential) error
}

// Test code — implement the interface inline
type mockCredentialStore struct {
    credentials map[uuid.UUID]*Credential
    err         error
}

func (m *mockCredentialStore) GetCredential(id uuid.UUID) (*Credential, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.credentials[id], nil
}
```

### HTTP mock server

For OAuth/social connector tests, use `httptest.Server`:

```go
ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        json.NewEncoder(w).Encode(map[string]string{
            "access_token": "mock-token",
            "token_type":   "Bearer",
        })
        return
    }
    json.NewEncoder(w).Encode(map[string]any{
        "id":    "12345",
        "email": "user@example.com",
    })
}))
defer ts.Close()

// Override the connector's endpoint to point to mock server
connector.(*googleConnector).config.Endpoint = oauth2.Endpoint{
    AuthURL:  ts.URL + "/auth",
    TokenURL: ts.URL + "/token",
}
```

### Embedded NATS server

For audit/integration tests requiring NATS:

```go
func startTestServer(t *testing.T) (*server.Server, *nats.Conn) {
    t.Helper()
    opts := &server.Options{
        Port:      -1,
        JetStream: true,
        StoreDir:  t.TempDir(),
    }
    s := natsserver.RunServer(opts)
    t.Cleanup(s.Shutdown)
    // ...
    return s, nc
}
```

---

## Integration Tests

Integration tests live in `test/integration/` and use the `integration` build tag:

```bash
go test -tags=integration -v ./test/integration/
```

These tests require:
- PostgreSQL 16 (with RLS enabled)
- Redis 7
- NATS JetStream
- OpenLDAP (optional)

### Gateway E2E Tests

The Gateway E2E test suite verifies the full request flow:
register → login → JWT → CRUD → 401 → revoke.

```bash
bash deploy/e2e-docker-test.sh
```

---

## Common Pitfalls

1. **Duplicate test names** — Go does not allow two `func TestXxx` in the same package across files. Use suffixes like `_V2`, `_Sprint9`.

2. **API signature mismatches** — Always run `go build ./...` before `make test` after editing production code.

3. **Unsealed mocks** — If your mock implements an interface, ensure ALL methods are implemented. Missing methods cause compile errors.

4. **Goroutine leaks** — Always use `defer cancel()` for contexts, `defer ts.Close()` for test servers, and `t.Cleanup()` for resources.

5. **Time-sensitive tests** — Use `time.Sleep` sparingly. Prefer channels with timeouts for async assertions.

---

## CI/CD Integration

The GitHub Actions CI pipeline (`.github/workflows/ci.yml`) runs:

1. **Go Build & Test** — `go build ./... && go test -race -cover`
2. **Security Scan** — Trivy vulnerability scanner
3. **Helm Chart Lint** — `helm lint deploy/helm/ggid`
4. **Console Build** — `npm run build` for Next.js
5. **Docker E2E** — Full stack deployment and E2E test suite

Coverage results are uploaded as artifacts for each PR.

---

## Package Coverage Table

Current test coverage by package (updated Phase 10):

| Package | Coverage | Target | Notes |
|---------|----------|--------|-------|
| `pkg/errors` | 100% | 100% | Complete |
| `pkg/tenant` | 100% | 100% | Complete |
| `pkg/i18n` | 100% | 100% | Complete |
| `pkg/saml` | 100% | 100% | Complete |
| `services/audit/internal/service` | 100% | 95%+ | Complete |
| `services/gateway/internal/transport` | 100% | 100% | Complete |
| `services/gateway/internal/webhooks` | 100% | 100% | Complete |
| `pkg/authprovider` | 97% | 95%+ | |
| `services/gateway/internal/config` | 97% | 95%+ | |
| `services/gateway/internal/healthcheck` | 97% | 95%+ | |
| `services/identity/internal/service` | 98.5% | 95%+ | |
| `services/org/internal/service` | 98.7% | 95%+ | |
| `services/auth/internal/domain` | 98.5% | 95%+ | |
| `pkg/pii` | 96.6% | 95%+ | |
| `services/policy/internal/service` | 97.1% | 95%+ | |
| `services/oauth/internal/service` | 95.7% | 95%+ | |
| `services/gateway/internal/router` | 95.1% | 95%+ | |
| `pkg/social` | 93.5% | 95% | |
| `services/policy/internal/server` | 90.9% | 90%+ | |
| `pkg/notification` | 91.5% | 90%+ | |
| `services/auth/internal/webauthn` | 81.7% | 90%+ | Working towards 90% |
| `services/gateway/internal/http3` | 90.0% | 90%+ | |
| `pkg/crypto` | 89.4% | 90%+ | |
| `services/audit/internal/server` | 93.2% | 90%+ | |
| `services/auth/internal/service` | 86.0% | 85%+ | Target hit |
| `services/gateway/internal/middleware` | 86.7% | 85%+ | |
| `services/audit/internal/handler` | 83.3% | 85% | |
| `pkg/email` | 80.2% | 85% | SMTP tests need real server |
| `pkg/audit` | 82.4% | 85% | |

---

## How to Write Integration Tests

### Integration Test Structure

Integration tests live in `test/integration/` and use the `integration` build tag:

```go
//go:build integration

package integration

import (
    "context"
    "testing"
)

func TestGatewayE2E_RegisterLogin(t *testing.T) {
    // Full flow: register → login → JWT → API call → 401 check
}
```

### Running Integration Tests

```bash
# Start infrastructure
cd deploy && docker compose up -d postgres redis nats

# Run integration tests
go test -tags=integration -v ./test/integration/...

# Run a specific integration test
go test -tags=integration -run TestGatewayE2E_RegisterLogin -v ./test/integration/...
```

### Integration Test Patterns

#### Test via Gateway (E2E)

```go
func TestE2E_RegisterUser(t *testing.T) {
    gatewayURL := "http://localhost:8080"
    tenantID := "00000000-0000-0000-0000-000000000001"

    // Register
    resp, err := http.Post(gatewayURL+"/api/v1/auth/register",
        "application/json",
        strings.NewReader(`{
            "username": "testuser",
            "email": "test@example.com",
            "password": "TestPass123!"
        }`))
    require.NoError(t, err)
    assert.Equal(t, 201, resp.StatusCode)
}
```

#### Test Service Directly (Component)

```go
//go:build integration

package integration

import (
    "services/auth/internal/service"
    "testing"
)

func TestAuthService_Login_Integration(t *testing.T) {
    // Use real repository (connected to test database)
    repo := newTestRepo(t) // helper that sets up test DB
    svc := service.NewAuthService(repo, mockCache)

    // Test actual behavior
    token, err := svc.Login(ctx, "user", "pass")
    require.NoError(t, err)
    assert.NotEmpty(t, token.AccessToken)
}
```

---

## Mock Patterns

### Interface-Based Mocks

GGID uses interfaces for all external dependencies, making mocking straightforward:

```go
// Define interface at consumer
type UserRepository interface {
    GetByID(ctx context.Context, id uuid.UUID) (*User, error)
    Create(ctx context.Context, user *User) error
}

// In tests, create a mock implementation
type mockUserRepo struct {
    users map[uuid.UUID]*User
    err   error
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
    if m.err != nil {
        return nil, m.err
    }
    if u, ok := m.users[id]; ok {
        return u, nil
    }
    return nil, ErrNotFound
}
```

### Mock HTTP Server

For testing HTTP clients and webhook delivery:

```go
func setupMockServer(t *testing.T, status int, body string) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(status)
        w.Write([]byte(body))
    }))
}

func TestWebhookDelivery(t *testing.T) {
    srv := setupMockServer(t, 200, `{"ok":true}`)
    defer srv.Close()

    // Test webhook delivery against mock
    err := webhookSender.Send(srv.URL, event)
    require.NoError(t, err)
}
```

### Mock Redis

```go
func newMockRedis() *miniredis.Miniredis {
    mr, _ := miniredis.NewMiniRedis()
    mr.Start()
    return mr
}

// Use in tests
mr := newMockRedis(t)
defer mr.Close()
rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
```

---

## Test Naming Conventions

GGID follows a consistent naming convention:

| Pattern | Example | Use Case |
|---------|---------|----------|
| `Test<Type>_<Method>` | `TestAuthService_Login` | Basic test |
| `Test<Type>_<Method>_<Scenario>` | `TestAuthService_Login_InvalidPassword` | Specific scenario |
| `Test<Type>_<Method>_<Condition>` | `TestUserRepo_Create_DuplicateEmail` | Error condition |
| `Test<Type>_<Method>_Concurrent` | `TestTokenStore_Revoke_Concurrent` | Race condition test |
| `Benchmark<Function>` | `BenchmarkRBACCheck_10Roles` | Performance benchmark |

---

## Coverage Targets Per Package

| Package Category | Minimum | Target | Rationale |
|-----------------|---------|--------|-----------|
| `pkg/` shared utilities | 85% | 95%+ | Used by all services, high impact |
| `services/*/internal/service/` | 80% | 90%+ | Business logic, must be correct |
| `services/*/internal/server/` | 75% | 85%+ | HTTP handlers, thinner logic |
| `services/*/internal/handler/` | 75% | 85%+ | gRPC handlers |
| `services/*/internal/repository/` | 60% | 80%+ | Data access, harder to test |
| `services/*/cmd/` | N/A | — | Thin main(), not tested |
| `console/` (TypeScript) | 60% | 80%+ | UI components |

```bash
# Check coverage against thresholds
make test

# Generate detailed HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

---

*Last updated: Phase 10*
