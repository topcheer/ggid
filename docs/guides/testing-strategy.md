# Testing Strategy

## Overview

GGID uses a multi-layered testing strategy to ensure platform reliability, security, and correctness. This document describes the test pyramid, tooling, coverage targets, and CI integration.

## Test Pyramid

```
         /\
        /E2E\          33 tests — full gateway lifecycle
       /------\
      /Security\        52 tests — auth/authz boundary cases
     /----------\
    / Integration \     ~200 tests — service-level CRUD + API
   /--------------\
  /     Unit       \   ~4200 tests — business logic, repos, handlers
 /------------------\
```

## Test Categories

### 1. Unit Tests (~4200 functions)

**What:** Individual function/method correctness — business logic, repository queries, utility functions, type conversions.

**Location:** `services/*/internal/**/*_test.go`

**Tooling:** Go standard `testing` package, `httptest` for HTTP handler tests.

**Example:**
```go
func TestPasswordStrength_StrongPassword(t *testing.T) {
    result := EstimateStrength("MyV3ryStr0ng!Pass#2026")
    if result.Score < 3 {
        t.Errorf("expected score >= 3, got %d", result.Score)
    }
}
```

**Coverage target:** 60%+ per package.

### 2. Integration Tests (~200 functions)

**What:** Service-level API tests — CRUD operations, database schema validation, inter-service communication patterns.

**Location:** `services/*/internal/server/*_test.go`, `test/integration/`

**Pattern:** Each test creates a `Gateway` or `Handler` with mock dependencies and verifies the full HTTP request/response cycle.

**Example:**
```go
func TestConditionalAccess_Create(t *testing.T) {
    h := newTestHandler(t)
    req := httptest.NewRequest("POST", "/api/v1/auth/conditional-access/policies", body)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusCreated { t.Errorf("expected 201, got %d", rr.Code) }
}
```

### 3. API Security Tests (52 functions)

**What:** Authentication and authorization boundary cases across 25+ protected endpoints.

**Location:** `services/gateway/internal/router/api_security_test.go`

**Test matrix:**

| Category | Tests | What's Verified |
|----------|-------|-----------------|
| No token → 401 | 10 | All protected endpoints reject unauthenticated requests |
| Invalid token → 401 | 5 | Malformed, empty, garbage JWT, Basic auth, no Bearer prefix |
| Auth header fuzzing | 1 | 8 attack variations (null, undefined, zeros, CRLF, SQL injection) |
| Public path validation | 6 | Login, register, healthz, oauth/token, system/initialized |
| Admin endpoint → 401 | 3 | routes, stats, config, secrets, backup |
| Cross-tenant → 401 | 3 | No token, fake token, missing tenant header |
| Rate limiting | 2 | 50-request burst, public endpoint exemption |
| Invalid JSON → 400 | 5 | Malformed, empty, truncated, injection |
| Oversized body → 413 | 4 | MaxBodySize middleware, exact limit, login |
| Unknown path → 404 | 2 | Nonexistent resources, PATCH on protected path |
| Header injection | 3 | CRLF injection, SQL injection, 100KB token |
| Break-glass/CAE | 3 | Privileged operations require auth |

### 4. E2E Integration Tests (33 functions)

**What:** Full request lifecycle through the gateway — bootstrap, auth flows, resource access, OAuth, sessions, password management.

**Location:** `services/gateway/internal/router/e2e_integration_test.go`

**Flow coverage:**

```
Quickstart → Login → Register → Users/Roles/Policies (401) → 
OAuth (token/authorize public, clients protected) → 
Sessions (list/revoke → 401) → Password (forgot/reset public, change protected) → 
Swagger UI → Dashboard stats
```

## CI Pipeline

The CI pipeline runs 3 parallel jobs on every push:

### Job 1: Build & Test
```bash
go build ./...              # All packages must compile
make test                   # = go test -timeout 10m -cover ./...
```

### Job 2: Security Scan
```bash
govulncheck ./...           # Known vulnerability detection
gosec ./...                 # Static security analysis
```

### Job 3: Lint
```bash
golangci-lint run --timeout=5m   # CI uses v1.64.8
```

**Pre-commit local verification (mandatory):**
```bash
go build ./...
make test
golangci-lint run --timeout=5m
```

All 3 must pass before push.

## Coverage Targets

| Package Type | Target | Current |
|-------------|--------|---------|
| Core domain logic | 80%+ | 50-100% |
| HTTP handlers | 40%+ | 10-22% |
| Repository | 50%+ | 0-34% |
| Gateway middleware | 70%+ | 80-100% |
| Gateway router | 60%+ | 22% |
| Crypto/auth utils | 90%+ | 68-100% |

**Overall:** ~40% (target: 60% by v1.0)

## Test Conventions

### Naming
- Unit: `Test<Struct>_<Method>` (e.g., `TestPasswordStrength_StrongPassword`)
- Integration: `Test<Feature>_<Scenario>` (e.g., `TestConditionalAccess_Create`)
- Security: `TestAPISecurity_<Category>_<Detail>` (e.g., `TestAPISecurity_NoToken_Users`)
- E2E: `TestE2E_<Flow>_<Detail>` (e.g., `TestE2E_LoginFlow_EmptyBody_Rejected`)

### Table-Driven Tests
Prefer table-driven for multiple inputs:
```go
func TestComputeDiff_AllCases(t *testing.T) {
    tests := []struct{ name string; ... }{ ... }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) { ... })
    }
}
```

### Test Isolation
- Each test creates its own `Gateway`/`Handler` instance
- No shared state between tests (except read-only constants)
- Database-backed tests use nil pool guards to avoid panics

## Running Tests

```bash
# All tests
make test

# Specific service
go test ./services/auth/... -count=1

# Security tests only
go test ./services/gateway/internal/router/... -run TestAPISecurity -v

# E2E tests only
go test ./services/gateway/internal/router/... -run TestE2E -v

# With coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

# Race detector
go test ./... -race -count=1

# Benchmark
go test ./... -bench=. -benchmem
```

## Known Gaps

| Gap | Priority | Plan |
|-----|----------|------|
| Handler coverage 10-22% | P1 | Add CRUD tests per handler |
| Repository coverage 0-34% | P1 | Add DB integration tests with testcontainers |
| NHI engine in-memory | P2 | Migrate to PG-backed, add persistence tests |
| OpenAPI 27% coverage | P2 | Auto-generate from HandleFunc routes |
| Console E2E (Playwright) | P3 | Add browser-based UI testing |
| Load testing | P3 | k6 suite exists in deploy/k6/ |

## Test Stats (Current)

| Metric | Value |
|--------|-------|
| Total test functions | 4461 |
| Test files | 372 |
| API security tests | 52 |
| E2E integration tests | 33 |
| CI jobs | 3 (Build, Security, Lint) |
| Overall coverage | ~40% |
