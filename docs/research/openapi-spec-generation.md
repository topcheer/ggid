# OpenAPI 3.1 Spec Generation & API Documentation: Production Implementation Guide for GGID

> **Focus**: Generating and maintaining a complete OpenAPI 3.1 specification from GGID's 786+ API endpoints — annotation strategy, spec aggregation, Swagger UI hosting, SDK auto-generation, and contract testing. Builds on `openapi-audit.md` (846 lines, endpoint inventory).
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Related**: `openapi-audit.md` (gap analysis), `sdk-parity-developer-experience.md` (SDK generation).
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: OpenAPI Infrastructure](#2-ggid-current-state-openapi-infrastructure)
3. [Gap Analysis](#3-gap-analysis)
4. [Spec Generation Strategy](#4-spec-generation-strategy)
5. [Annotation Patterns](#5-annotation-patterns)
6. [Spec Aggregation Architecture](#6-spec-aggregation-architecture)
7. [Authentication Schemes](#7-authentication-schemes)
8. [Schema Generation](#8-schema-generation)
9. [Endpoint Precondition Check](#9-endpoint-precondition-check)
10. [API Design + Curl Commands](#10-api-design--curl-commands)
11. [SDK Auto-Generation](#11-sdk-auto-generation)
12. [Contract Testing](#12-contract-testing)
13. [Implementation Backlog with DoD](#13-implementation-backlog-with-dod)
14. [Competitive Differentiation](#14-competitive-differentiation)

---

## 1. Executive Summary

GGID has **786+ API endpoints** across 7 services but no unified OpenAPI specification. The existing `openapi-audit.md` (846 lines) inventoried every endpoint and identified massive gaps between code and spec.

**Existing OpenAPI infrastructure:**
- `OpenAPIAggregator` (`gateway/middleware/openapi_coverage_test.go:10`) — Aggregates per-service specs ✅
- `/openapi` endpoint at gateway ✅
- Mock backend test for spec aggregation ✅
- `openapi-audit.md` — 846-line gap analysis ✅

**What's missing:**
1. **No actual OpenAPI spec generated** — aggregator exists but no services produce specs
2. **No annotations** — Go handlers have no swag/kin-openapi annotations
3. **No Swagger UI** — No interactive documentation
4. **No schema generation** — Go structs not mapped to JSON Schema
5. **No contract tests** — Spec not validated against actual API
6. **No SDK generation from spec** — SDKs hand-written (see sdk-parity research)

**Recommendation**: Annotate all handlers with swaggo/swag v2 annotations, generate per-service OpenAPI 3.1 specs, aggregate at gateway, host Swagger UI/Redoc, and enable SDK auto-generation via openapi-generator.

**Estimated effort**: 3 sprints for MVP (annotations + spec generation + Swagger UI + contract tests).

---

## 2. GGID Current State: OpenAPI Infrastructure

### Existing Components

| Component | File:Line | Status | Notes |
|-----------|-----------|--------|-------|
| OpenAPIAggregator | `gateway/middleware/openapi_coverage_test.go:34` | ✅ Tested | Aggregates per-service specs |
| `/openapi` endpoint | gateway router | ✅ | Serves aggregated spec |
| OpenAPISpec struct | middleware package | ✅ | `{OpenAPI, Info, Paths}` |
| OpenAPI audit | `docs/research/openapi-audit.md` | ✅ 846 lines | Full endpoint inventory + gaps |
| 786+ endpoints | All services | ✅ | No annotations |

### What the Aggregator Does (Today)

```go
// gateway/middleware/openapi_coverage_test.go:34
agg := NewOpenAPIAggregator(map[string]string{
    "/api/v1/test": backend.URL,  // service → spec URL
})
// Fetches /openapi.json or /swagger/doc.json from each service
// Merges into unified spec
// Serves at gateway /openapi
```

The aggregator works but **no service actually produces a spec** — it's testing infrastructure only.

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | No spec generation | 786 endpoints undocumented |
| 2 | No handler annotations | Can't auto-generate |
| 3 | No Swagger UI | No interactive docs |
| 4 | No JSON Schema from structs | Response models undefined |
| 5 | No auth scheme documentation | OAuth/DPoP/API key not documented |
| 6 | No contract tests | Spec can drift from API |
| 7 | No SDK generation | Manual SDK sync |
| 8 | No version management | Spec not versioned with API |

---

## 4. Spec Generation Strategy

### Tool Selection: swaggo/swag v2

| Tool | Approach | Pros | Cons |
|------|----------|------|------|
| **swaggo/swag v2** | Annotation parsing | Popular, simple, Go-native | Requires annotations |
| **kin-openapi** | Code-first (programmatic) | Full control, no annotations | More boilerplate |
| **oapi-codegen** | Spec-first (reverse) | Spec is source of truth | Requires writing spec first |
| **protoc-gen-openapi** | From proto/gRPC | GGID has protobuf defs | Only covers gRPC endpoints |

**Recommendation**: **swaggo/swag v2** for HTTP handlers + **protoc-gen-openapi** for gRPC services. This matches GGID's hybrid HTTP+gRPC architecture.

### Generation Flow

```
Go source code (annotated handlers)
    │
    ├── swag init (per service)
    │   → docs/swagger.json
    │   → docs/swagger.yaml
    │
    ├── Gateway OpenAPIAggregator
    │   → Fetches each service's /swagger/doc.json
    │   → Merges into unified OpenAPI 3.1
    │   → Serves at gateway /openapi
    │
    ├── Swagger UI (gateway /docs)
    │   → Interactive documentation
    │
    └── openapi-generator
        → Go SDK, Python SDK, TypeScript SDK, Java SDK
```

---

## 5. Annotation Patterns

### Example: Auth Login Handler

```go
// Login authenticates a user and issues tokens.
//
// @Summary      User login
// @Description  Authenticate with username/password, receive OAuth 2.1 tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest   true  "Login credentials"
// @Success      200      {object}  LoginResponse  "Tokens issued"
// @Failure      401      {object}  ErrorResponse  "Invalid credentials"
// @Failure      429      {object}  ErrorResponse  "Rate limited"
// @Security     BearerAuth
// @Router       /api/v1/auth/login [post]
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
    // ... existing handler code ...
}
```

### Example: Policy Authorize

```go
// @Summary      Evaluate authorization decision
// @Description  Unified PDP — evaluates RBAC + ABAC + ReBAC + risk in one call
// @Tags         policy
// @Accept       json
// @Produce      json
// @Param        request  body      AuthorizeRequest   true  "Authorization request"
// @Success      200      {object}  AuthorizeResponse  "Decision: allow/deny/step_up"
// @Failure      400      {object}  ErrorResponse       "Invalid request"
// @Security     BearerAuth
// @Security     DPoPAuth
// @Router       /api/v1/policy/authorize [post]
```

### Annotation Strategy

| Element | Annotation | Example |
|---------|-----------|---------|
| Summary | `@Summary` | "User login" |
| Description | `@Description` | "Authenticate..." |
| Tags | `@Tags` | "auth", "identity", "oauth" |
| Request body | `@Param ... body` | `body LoginRequest true "..."` |
| Response | `@Success` / `@Failure` | `200 {object} LoginResponse` |
| Security | `@Security` | `BearerAuth`, `DPoPAuth`, `ApiKeyAuth` |
| Path | `@Router` | `/api/v1/auth/login [post]` |

### Per-Service swag.yaml

```yaml
# services/auth/swag.yaml
conf:
  dir: ./docs
  exclude: ./internal/test
  propertyStrategy: snake_case
output:
  - dir: ./docs
    format: openapi3.1
parse:
  depth: 3
```

---

## 6. Spec Aggregation Architecture

```
┌──────────────────────────────────────────────┐
│         Gateway (OpenAPIAggregator)           │
│                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │ Auth     │ │Identity  │ │ OAuth    │     │
│  │ spec     │ │ spec     │ │ spec     │     │
│  │ /auth/   │ │ /ident/  │ │ /oauth/  │     │
│  │ swagger  │ │ swagger  │ │ swagger  │     │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘     │
│       │            │            │            │
│  ┌────▼────────────▼────────────▼─────┐     │
│  │      Merged OpenAPI 3.1            │     │
│  │      /openapi                      │     │
│  └───────────────┬───────────────────┘     │
│                  │                          │
│  ┌───────────────▼───────────────────┐     │
│  │      Swagger UI                   │     │
│  │      /docs                        │     │
│  └───────────────────────────────────┘     │
└──────────────────────────────────────────────┘
```

### Existing Aggregator Enhancement

```go
// Gateway fetches per-service specs at startup + caches
type OpenAPIAggregator struct {
    serviceSpecs map[string]string  // service → /swagger/doc.json URL
    cachedSpec   *OpenAPISpec
    cacheExpiry  time.Duration
}

// Merge: prefix paths with service route prefix
// auth spec /login → /api/v1/auth/login
// identity spec /users → /api/v1/identity/users
```

---

## 7. Authentication Schemes

```yaml
# OpenAPI security schemes
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: "OAuth 2.1 access token"

    DPoPAuth:
      type: http
      scheme: DPoP
      description: "DPoP proof-of-possession token"

    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
      description: "API key for service-to-service"

    MutualTLS:
      type: mutualTLS
      description: "Client certificate (mTLS)"
```

---

## 8. Schema Generation

### Go Struct → JSON Schema (automatic via swag)

```go
// swag automatically generates schema from Go struct tags
type LoginRequest struct {
    Username string `json:"username" example:"alice@corp.com"`
    Password string `json:"password" example:"******"`
    OTP      string `json:"otp,omitempty" example:"123456"`
}

type LoginResponse struct {
    AccessToken  string `json:"access_token" example:"eyJhbG..."`
    RefreshToken string `json:"refresh_token" example:"rt_abc..."`
    ExpiresIn    int    `json:"expires_in" example:"3600"`
    TokenType    string `json:"token_type" example:"Bearer"`
}

type ErrorResponse struct {
    Error     string `json:"error" example:"invalid_credentials"`
    Message   string `json:"message" example:"Invalid username or password"`
    RequestID string `json:"request_id" example:"req_abc123"`
}
```

### Generated JSON Schema

```json
{
  "LoginRequest": {
    "type": "object",
    "required": ["username", "password"],
    "properties": {
      "username": { "type": "string", "example": "alice@corp.com" },
      "password": { "type": "string", "example": "******" },
      "otp": { "type": "string", "example": "123456" }
    }
  }
}
```

---

## 9. Endpoint Precondition Check

### Existing (Enhance)

| Component | File:Line | Status | Target |
|----------|-----------|--------|--------|
| OpenAPIAggregator | `openapi_coverage_test.go:34` | ✅ Tested | Use in production |
| `/openapi` endpoint | gateway router | ✅ | Serve merged spec |
| 786+ endpoints | All services | ✅ No annotations | Add swag annotations |
| OpenAPI audit | `openapi-audit.md` | ✅ 846 lines | Use as annotation checklist |

### New Components

| Component | Priority |
|-----------|----------|
| swag annotations on all handlers | P0 |
| Per-service `swag init` CI step | P0 |
| Swagger UI hosting | P0 |
| Contract test framework | P1 |
| openapi-generator for SDKs | P1 |

---

## 10. API Design + Curl Commands

### Fetch Unified Spec

```bash
curl https://ggid.corp.com/openapi

# Returns unified OpenAPI 3.1 spec for all 786+ endpoints
{
  "openapi": "3.1.0",
  "info": { "title": "GGID API", "version": "1.0.0" },
  "paths": {
    "/api/v1/auth/login": { "post": { ... } },
    "/api/v1/identity/users": { "get": { ... } },
    ...
  }
}
```

### Swagger UI

```bash
# Interactive documentation at gateway
open https://ggid.corp.com/docs
# Try-it-out: send live requests from browser
```

### Per-Service Spec

```bash
curl https://ggid.corp.com/api/v1/auth/swagger/doc.json
# Returns auth service OpenAPI spec only
```

---

## 11. SDK Auto-Generation

```yaml
# Generate SDKs from OpenAPI spec using openapi-generator
generators:
  - language: go
    output: sdk/go/generated/
    package: ggid
    
  - language: python
    output: sdk/python/generated/
    package: ggid
    
  - language: typescript
    output: sdk/node/generated/
    package: @ggid/sdk
    
  - language: java
    output: sdk/java/generated/
    package: com.ggid.sdk
```

### CI Pipeline

```yaml
# .github/workflows/sdk-generate.yml
- name: Generate SDKs from OpenAPI
  run: |
    # 1. Generate spec from annotations
    swag init --dir services/auth --output docs/
    swag init --dir services/identity --output docs/
    # ... per service

    # 2. Aggregate at gateway
    # (gateway aggregator fetches + merges)

    # 3. Generate SDKs
    openapi-generator generate -i openapi.json -g go -o sdk/go/generated/
    openapi-generator generate -i openapi.json -g python -o sdk/python/generated/
```

---

## 12. Contract Testing

```go
// Contract test: verify spec matches actual API
func TestContract_AuthLogin(t *testing.T) {
    spec := loadOpenAPISpec()

    // Verify endpoint exists in spec
    endpoint := spec.Paths["/api/v1/auth/login"].Post
    assert.NotNil(endpoint, "login endpoint must be in spec")

    // Verify request schema
    assert.Contains(endpoint.RequestBody.Content, "application/json")

    // Verify response schema
    assert.Contains(endpoint.Responses["200"].Content, "application/json")

    // Live test: send request, verify response matches schema
    resp := httpPost("/api/v1/auth/login", body)
    validateAgainstSchema(resp, endpoint.Responses["200"])
}
```

---

## 13. Implementation Backlog with DoD

### P0 — Annotations + Spec Generation (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | swag annotations on auth + identity (200+ endpoints) | ✅ All handlers annotated ✅ swag init generates valid spec ✅ ≥3 verified | 5d |
| 2 | swag annotations on oauth + policy + audit (300+ endpoints) | ✅ All annotated ✅ swag init ✅ ≥3 verified | 5d |
| 3 | Gateway aggregator wired to production | ✅ Fetches real specs ✅ Merges correctly ✅ ≥3 tests | 2d |
| 4 | Swagger UI hosting at /docs | ✅ Interactive docs ✅ Try-it-out works ✅ ≥3 tests | 1d |

### P1 — Schema + Contract Tests + SDK Gen (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | JSON Schema from Go structs | ✅ All response types documented ✅ Examples included ✅ ≥3 verified | 3d |
| 6 | Contract test framework | ✅ Spec ↔ API match ✅ CI integration ✅ ≥3 tests | 3d |
| 7 | SDK auto-generation pipeline | ✅ Go + Python + TS generated ✅ Published ✅ ≥3 tests | 3d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 8 | OpenAPI 3.1 validation (redocly lint) | CI-verified spec quality |
| 9 | Versioned specs (v1, v2) | Spec tied to API version |
| 10 | Changelog generation | Auto-generated from spec diff |
| 11 | Developer portal | Branded docs site |

---

## 14. Competitive Differentiation

| Feature | GGID (target) | Okta | Auth0 | Stripe | Keycloak |
|---------|---------------|------|-------|--------|----------|
| **OpenAPI spec** | 3.1 (target) | Custom | Custom | 3.1 ✅ | Partial |
| **Endpoint count** | 786+ | ~300 | ~200 | ~150 | ~100 |
| **Swagger UI** | Hosted (target) | Yes | Yes | Yes | Partial |
| **SDK generation** | openapi-generator | Hand-written | Hand-written | Hand-written | No |
| **Contract tests** | Target | Internal | Internal | Yes | No |
| **Schema examples** | Target | Yes | Yes | Yes | No |
| **Open source** | Yes | No | No | No | Yes |

**Key differentiator**: GGID would have the **largest open-source IAM API surface** (786+ endpoints) with complete OpenAPI 3.1 documentation, Swagger UI, and auto-generated SDKs — exceeding Auth0 and Okta in transparency.

---

## References

- [OpenAPI 3.1 Specification](https://spec.openapis.org/oas/v3.1.0) — Official spec
- [swaggo/swag v2](https://github.com/swaggo/swag) — Go annotation-based OpenAPI generation
- [kin-openapi](https://github.com/getkin/kin-openapi) — Programmatic OpenAPI for Go
- [openapi-generator](https://openapi-generator.org/) — Multi-language SDK generation
- [Swagger UI](https://swagger.io/tools/swagger-ui/) — Interactive documentation
- [Redoc](https://redocly.com/redoc/) — Alternative docs UI
- [Redocly CLI](https://redocly.com/docs/cli/) — OpenAPI linting
- [oasdiff](https://github.com/Tufin/oasdiff) — Breaking change detection
- [GGID OpenAPI Audit](./openapi-audit.md) — 846-line endpoint inventory
- [GGID OpenAPI Aggregator](../services/gateway/internal/middleware/openapi_coverage_test.go) — At line 34
- [GGID SDK Parity Research](./sdk-parity-developer-experience.md) — SDK auto-generation
- [GGID API Gateway Hardening](./api-gateway-hardening.md) — Gateway middleware stack
