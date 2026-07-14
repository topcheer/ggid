# Gateway Architecture Guide

Reverse proxy, JWT verification, rate limiting, circuit breaker, CORS, request routing, health checks, and graceful shutdown.

## Architecture

```
Client (Browser/Mobile/SDK)
    │  HTTPS
    ▼
┌─────────────────────────────────────────────┐
│              API Gateway                     │
│                                             │
│  TLS Termination                             │
│       │                                     │
│       ▼                                     │
│  CORS Preflight                              │
│       │                                     │
│       ▼                                     │
│  JWT Verification (middleware chain)         │
│       │                                     │
│       ▼                                     │
│  Rate Limiter (token bucket per IP/user)     │
│       │                                     │
│       ▼                                     │
│  Circuit Breaker (per backend service)       │
│       │                                     │
│       ▼                                     │
│  Request Router (path → backend service)     │
│       │                                     │
│  ┌─────┬───────┬───────┬───────┬───────┐    │
│  │Auth │Ident  │Policy │  Org  │ Audit │    │
│  └─────┴───────┴───────┴───────┴───────┘    │
└─────────────────────────────────────────────┘
```

## Reverse Proxy

GGID gateway uses `httputil.ReverseProxy` to forward requests to backend services:

```go
func NewRouter(services map[string]string) http.Handler {
    proxy := &httputil.ReverseProxy{
        Director: func(req *http.Request) {
            backend := services[matchService(req.URL.Path)]
            req.URL.Scheme = "http"
            req.URL.Host = backend

            // Inject tenant context
            if tenantID := req.Context().Value("tenant_id"); tenantID != nil {
                req.Header.Set("X-Tenant-ID", tenantID.(string))
            }
            // Inject request ID for tracing
            req.Header.Set("X-Request-ID", uuid.New().String())
        },
        ErrorHandler: handleProxyError,
    }
    return proxy
}
```

## Request Routing

### Route Table

```go
var Routes = []Route{
    {Prefix: "/api/v1/auth",      Backend: "auth:9001"},
    {Prefix: "/api/v1/identity",  Backend: "identity:8081"},
    {Prefix: "/api/v1/oauth",     Backend: "oauth:9005"},
    {Prefix: "/api/v1/policy",    Backend: "policy:8070"},
    {Prefix: "/api/v1/orgs",      Backend: "org:8071"},
    {Prefix: "/api/v1/audit",     Backend: "audit:8072"},
    {Prefix: "/api/v1/users",     Backend: "identity:8081"},
    {Prefix: "/api/v1/roles",     Backend: "policy:8070"},
    {Prefix: "/api/v1/agents",    Backend: "oauth:9005"},
    {Prefix: "/api/v1/admin",     Backend: "identity:8081", AdminOnly: true},
}
```

### Path Matching

Exact prefix match → longest match wins. Unknown paths return 404.

## JWT Verification

### Middleware Chain

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip auth for health and public endpoints
        if isPublicPath(r.URL.Path) {
            next.ServeHTTP(w, r)
            return
        }

        token := extractBearerToken(r)
        if token == "" {
            http.Error(w, "missing token", 401)
            return
        }

        claims, err := jwtVerifier.Verify(token)
        if err != nil {
            http.Error(w, "invalid token", 401)
            return
        }

        // Tenant from JWT claim (priority over header)
        tenantID := claims["tenant_id"].(string)

        ctx := context.WithValue(r.Context(), "user_id", claims["sub"])
        ctx = context.WithValue(ctx, "tenant_id", tenantID)
        ctx = context.WithValue(ctx, "scopes", claims["scope"])

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### JWKS Caching

```go
jwksCache := cache.New(15 * time.Minute)

func (v *JWTVerifier) Verify(tokenStr string) (jwt.MapClaims, error) {
    token, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
        kid := t.Header["kid"].(string)
        if key := jwksCache.Get(kid); key != nil {
            return key, nil
        }
        // Re-fetch JWKS on unknown kid
        jwks := fetchJWKS()
        for _, k := range jwks.Keys { jwksCache.Set(k.Kid, k) }
        return jwksCache.Get(kid), nil
    })
    return token.Claims.(jwt.MapClaims), nil
}
```

## Rate Limiting

### Token Bucket (per IP)

```go
limiter := ratelimit.NewTokenBucket(
    ratelimit.WithRate(100),       // 100 requests per minute
    ratelimit.WithBurst(200),      // Burst of 200
    ratelimit.WithKey("client_ip"),
    ratelimit.WithStorage(redis),  // Distributed via Redis
)
```

### Per-Endpoint Limits

| Endpoint Pattern | Rate Limit | Scope |
|-----------------|-----------|-------|
| `/api/v1/auth/login` | 10/min | Per IP |
| `/api/v1/auth/register` | 5/min | Per IP |
| `/api/v1/oauth/token` | 30/min | Per client |
| Default | 100/min | Per user |
| Admin API | 60/min | Per admin |

Response headers:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1700000600
Retry-After: 12  // When 429
```

## Circuit Breaker

Protects backend services from cascading failures:

```go
breaker := circuit.New(
    circuit.WithThreshold(5),           // 5 failures → open
    circuit.WithTimeout(30*time.Second), // Half-open after 30s
    circuit.WithHalfOpenRequests(3),     // 3 test requests in half-open
)

func proxyWithCircuitBreaker(service string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !breaker.Allow(service) {
            http.Error(w, "service unavailable", 503)
            return
        }

        rec := &statusRecorder{ResponseWriter: w}
        next.ServeHTTP(rec, r)

        if rec.status >= 500 {
            breaker.RecordFailure(service)
        } else {
            breaker.RecordSuccess(service)
        }
    })
}
```

### Circuit States

```
Closed (normal) → 5 failures → Open (reject all)
  Open → 30s timeout → Half-Open (test 3 requests)
    Half-Open → all pass → Closed
    Half-Open → any fail → Open (reset timeout)
```

## CORS

```go
cors := cors.New(cors.Options{
    AllowedOrigins:   []string{"https://console.ggid.dev", "https://*.corp.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
    ExposedHeaders:   []string{"X-RateLimit-Remaining", "X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           600, // 10 min preflight cache
})
```

## Health Checks

### Gateway Health

```bash
GET /healthz
# → 200 {"status":"ok","uptime":"72h","version":"2.0.0"}

GET /healthz/ready
# → 200 if all backend services reachable
# → 503 if any backend is down
```

### Backend Health (per service)

```go
type HealthChecker struct {
    services map[string]string // service → health endpoint
}

func (h *HealthChecker) CheckAll() map[string]bool {
    results := make(map[string]bool)
    for name, url := range h.services {
        resp, err := http.Get(url + "/healthz")
        results[name] = err == nil && resp.StatusCode == 200
    }
    return results
}
```

## Graceful Shutdown

```go
func main() {
    srv := &http.Server{Addr: ":8080", Handler: router}

    go func() {
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    // Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    // Drain: stop accepting new connections, finish in-flight
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    srv.Shutdown(ctx)

    log.Println("Gateway shut down gracefully")
}
```

### Shutdown Sequence

1. Stop accepting new connections
2. Wait for in-flight requests (max 30s)
3. Close upstream service connections
4. Flush metrics and logs
5. Exit

## Monitoring

| Metric | Alert |
|--------|-------|
| Request latency p99 | >500ms |
| Error rate (5xx) | >1% |
| Circuit breaker open | Any |
| Rate limit hits (429) | >10% of traffic |
| JWKS fetch failures | Any |
| Backend health check failures | >3 consecutive |

## See Also

- [gRPC vs REST](grpc-vs-rest.md)
- [Rate Limiting Strategy](rate-limiting-strategy.md)
- [Session Security](session-security.md)
- [Authentication Flows](authentication-flows.md)
