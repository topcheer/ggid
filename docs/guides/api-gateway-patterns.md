# API Gateway Patterns

Gateway responsibilities, pattern comparison, request lifecycle, plugin architecture, circuit breaker, protocol bridging, and GGID implementation.

## Pattern Comparison

| Pattern | Scope | When to Use |
|---------|-------|------------|
| Reverse proxy | Routing only | Simple passthrough |
| API gateway | Auth, rate limit, transform | Multi-service APIs |
| BFF (Backend for Frontend) | Per-client optimization | Mobile vs web differ |
| Service mesh | East-west mTLS, policy | Service-to-service |

## Gateway Responsibilities

```
Request → JWT verify → Rate limit → Route → Transform → Backend
              │            │          │          │
              ▼            ▼          ▼          ▼
           Authz       Throttle   Path map  REST↔gRPC
```

| Responsibility | Implementation |
|---------------|---------------|
| Routing | Path-based → service mapping |
| Authentication | JWT verify + scope check |
| Rate limiting | Token bucket per client/tenant |
| Request transformation | REST → gRPC, add headers |
| Response aggregation | Combine multiple service calls |
| Circuit breaking | Fail fast on backend errors |
| Logging | Request/response audit trail |
| CORS | Per-tenant origin policy |

## Request Lifecycle

```go
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Recover from panics
    defer recoverMiddleware(w, r)

    // 2. Verify JWT
    claims, err := g.auth.Verify(r)
    if err != nil { writeError(w, 401, "unauthorized"); return }

    // 3. Rate limit
    if !g.limiter.Allow(claims.TenantID) {
        writeError(w, 429, "rate limited"); return
    }

    // 4. Route + proxy
    backend := g.router.Match(r.URL.Path)
    if backend == nil { writeError(w, 404, "not found"); return }

    // 5. Circuit breaker
    if g.breaker.IsOpen(backend) {
        writeError(w, 503, "backend unavailable"); return
    }

    // 6. Transform + proxy
    g.proxy.ServeHTTP(w, r, backend, claims)
}
```

## Protocol Bridging (REST ↔ gRPC)

```yaml
routes:
  - path: /api/v1/users
    method: GET
    backend: identity-svc:9080
    grpc_method: identitypb.IdentityService/ListUsers
    transform:
      request: RESTParams → gRPC ListUsersRequest
      response: gRPC ListUsersResponse → REST JSON
```

## Circuit Breaker at Gateway

| State | Behavior | Transition |
|-------|---------|-----------|
| Closed | All requests pass | 5 failures → Open |
| Open | All requests fail fast (503) | After 30s → Half-Open |
| Half-Open | 1 test request | Success → Closed, Fail → Open |

## GGID Gateway Implementation

| Feature | Implementation |
|---------|---------------|
| Reverse proxy | httputil.ReverseProxy |
| JWT verification | Local JWKS + Redis blacklist |
| Rate limiting | Redis token bucket |
| Tenant injection | JWT claim → X-Tenant-ID header |
| mTLS to backends | Service mesh (Istio) |
| Health checks | /healthz on each backend |
| Graceful shutdown | Drain connections over 30s |

## Monitoring

| Metric | Alert |
|--------|-------|
| Gateway latency p99 | <100ms | >500ms → investigate |
| Backend error rate | <1% | >5% → circuit breaker |
| Rate limit hits | <2% | >10% → scale or tune |
| Circuit open events | 0 | Any → backend issue |

## See Also

- [Gateway Architecture](gateway-architecture.md)
- [Service Mesh Integration](service-mesh-integration.md)
- [API Rate Limit Tuning](api-rate-limit-tuning.md)
- [gRPC Interceptor Patterns](grpc-interceptor-patterns.md)
