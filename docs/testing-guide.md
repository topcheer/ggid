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

*Last updated: Phase 10*
