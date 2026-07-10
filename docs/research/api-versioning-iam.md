# API Versioning and Lifecycle Management for IAM Systems

> **Research Document** — GGID IAM Platform
> Topic: API versioning strategies, backward compatibility rules, deprecation policy, and version-specific security enforcement for identity and access management APIs.
> Status: Active — Last updated 2025

---

## Table of Contents

1. [Versioning Strategies](#1-versioning-strategies)
2. [Backward Compatibility Rules](#2-backward-compatibility-rules)
3. [Deprecation Policy](#3-deprecation-policy)
4. [Breaking Change Management](#4-breaking-change-management)
5. [Version-Specific Security Policies](#5-version-specific-security-policies)
6. [API Gateway Version Routing](#6-api-gateway-version-routing)
7. [GGID API Versioning Audit](#7-ggid-api-versioning-audit)
8. [Version Documentation and Discovery](#8-version-documentation-and-discovery)
9. [Gap Analysis and Recommendations](#9-gap-analysis-and-recommendations)

---

## 1. Versioning Strategies

API versioning determines how clients select which version of an API they interact
with. For IAM systems, the choice has direct security implications: vulnerable
versions must be identifiable and rejectable at the routing layer.

### 1.1 Strategy Comparison

| Strategy | Example | Pros | Cons | IAM Suitability |
|---|---|---|---|---|
| **URL Path** | `/v1/users` | Explicit, cacheable, easy to route, visible in logs | Pollutes URL namespace | **Best** — clear audit trail |
| **Header (Accept)** | `Accept: application/vnd.ggid.v1+json` | RESTful, clean URLs, content negotiation native | Invisible in browser, hard to test with curl | Moderate — hidden version complicates debugging |
| **Query Parameter** | `?version=1` | Simple, easy to add | Easy to omit (defaults silently), breaks caching | Poor — silent fallback is dangerous for auth |
| **Custom Header** | `X-API-Version: 1` | Non-invasive | Non-standard, easy to forget, invisible to intermediaries | Poor — intermediaries strip headers |
| **Content Negotiation** | `Accept: application/json; version=1` | RFC-compliant | Complex parsing, subtle bugs | Moderate |

### 1.2 Why URL Path Versioning Is Best for IAM APIs

URL path versioning is the dominant choice for IAM systems for four reasons:

1. **Auditability**: Every request log entry contains the API version in the URL.
   For IAM, where every request is security-relevant, this is critical. With
   header-based versioning, the version may be stripped by load balancers or
   proxies before logging.

2. **Routing simplicity**: API gateways can route `/v1/*` and `/v2/*` to different
   backend deployments without parsing headers. This enables running vulnerable
   v1 and patched v2 simultaneously during migration.

3. **Client simplicity**: SDKs embed the version in the base URL. No complex
   header management is needed. This reduces integration errors in the
   authentication flow, where a missing version header could cause silent
   fallback to a deprecated (insecure) auth flow.

4. **Cache correctness**: CDN and proxy caches key on URL path. Header-based
   versioning can cause cache poisoning if the version header is not part of
   the cache key. For token endpoints (which should not be cached) this is less
   relevant, but for JWKS and discovery endpoints, correct cache invalidation
   across versions matters.

### 1.3 GGID's Current Approach

GGID uses URL path versioning: all gateway routes are prefixed with `/api/v1/`:

```go
// From services/gateway/internal/config/config.go
Routes: map[string]string{
    "/api/v1/auth":         "http://localhost:9001",
    "/api/v1/users":        "http://localhost:8081",
    "/api/v1/roles":        "http://localhost:8070",
    "/api/v1/permissions":  "http://localhost:8070",
    "/api/v1/policies":     "http://localhost:8070",
    "/api/v1/orgs":         "http://localhost:8081",
    "/api/v1/audit":        "http://localhost:8082",
    "/oauth":               "http://localhost:9005",
    "/saml":                "http://localhost:9005",
},
```

The gRPC services use proto package versioning (`api/gen/policy/v1`,
`api/gen/org/v1`, `api/gen/audit/v1`), which is the gRPC standard.

---

## 2. Backward Compatibility Rules

### 2.1 What Is Backward Compatible?

Changes that do not break existing clients:

| Change Type | Example | Safe? |
|---|---|---|
| Adding a new response field | `{"id":1}` → `{"id":1,"created_at":"..."}` | Yes |
| Adding a new endpoint | New `POST /v1/users/export` | Yes |
| Adding an optional request parameter | New `?include_deleted=true` (defaults false) | Yes |
| Adding new enum values | Role type gains `"service"` | Yes (clients should handle unknown enums) |
| Making a required parameter optional | `email` was required, now optional if `username` given | Yes |
| Changing field documentation (not semantics) | Clarify that `expires_in` is in seconds | Yes |
| Changing internal implementation | New DB schema, same API contract | Yes |

### 2.2 What Is a Breaking Change?

Changes that break existing clients and require a version bump:

| Change Type | Example | Impact |
|---|---|---|
| Removing a response field | Drop `legacy_id` | Client crashes |
| Changing a field type | `id` from `int` to `string` | Deserialization failure |
| Changing field semantics | `expires_in` was seconds, now milliseconds | Silent data corruption |
| Making an optional parameter required | `tenant_id` now mandatory | 400 errors |
| Removing an endpoint | Delete `POST /v1/auth/legacy-login` | Integration breaks |
| Changing error response format | `{"error":"..."}` → `{"message":"..."}` | Error handling breaks |
| Changing default behavior | Default page size 100 → 10 | Unexpected pagination |
| Changing authentication requirements | Endpoint that was public now requires JWT | 401 errors |

### 2.3 Semantic Versioning for APIs

For IAM APIs, we recommend a simplified semantic versioning scheme:

- **Major version** (v1, v2): Breaking changes. Requires URL path bump.
- **Minor version** (v1.1, v1.2): Additive changes (new fields, new endpoints).
  No URL change — tracked in response headers: `X-API-Minor-Version: 2`.
- **Patch version** (v1.1.3): Bug fixes only. No header change needed.

```go
// APIVersion represents the version of an API response.
type APIVersion struct {
    Major int    `json:"major"`           // URL path version: /v1, /v2
    Minor int    `json:"minor"`           // Additive changes within major
    Patch int    `json:"patch"`           // Bug fixes
    Label string `json:"label,omitempty"` // "stable", "beta", "deprecated"
}

// CurrentVersion is the current API version for v1 endpoints.
var CurrentV1 = APIVersion{
    Major: 1, Minor: 3, Patch: 0, Label: "stable",
}

// VersionHeader sets the API version in the response.
func VersionHeader(v APIVersion) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("X-API-Version", fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch))
            w.Header().Set("X-API-Status", v.Label)
            next.ServeHTTP(w, r)
        })
    }
}
```

### 2.4 Version-Aware Request Handler

```go
// versionContextKey is the context key for API version.
type versionContextKey struct{}

// VersionedHandler routes requests to version-specific handlers.
type VersionedHandler struct {
    handlers map[int]http.Handler // map[version]handler
    fallback http.Handler         // served when version not found
}

func NewVersionedHandler() *VersionedHandler {
    return &VersionedHandler{
        handlers: make(map[int]http.Handler),
    }
}

func (vh *VersionedHandler) Register(version int, h http.Handler) {
    vh.handlers[version] = h
}

// ServeHTTP extracts the API version from the URL path and dispatches.
// Path format: /api/v{version}/...
func (vh *VersionedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    version, remaining, ok := extractVersion(r.URL.Path)
    if !ok {
        http.Error(w, `{"error":"invalid API version in path"}`, http.StatusBadRequest)
        return
    }

    handler, exists := vh.handlers[version]
    if !exists {
        if vh.fallback != nil {
            handler = vh.fallback
        } else {
            http.Error(w, fmt.Sprintf(`{"error":"API version v%d is not supported"}`, version),
                http.StatusNotFound)
            return
        }
    }

    // Strip version prefix from path for downstream handlers
    r2 := r.Clone(r.Context())
    r2.URL.Path = remaining
    ctx := context.WithValue(r2.Context(), versionContextKey{}, version)
    r2 = r2.WithContext(ctx)

    handler.ServeHTTP(w, r2)
}

// extractVersion parses "/api/v1/users" → (1, "/users", true).
func extractVersion(path string) (int, string, bool) {
    // Expected format: /api/v{N}/{rest}
    if !strings.HasPrefix(path, "/api/v") {
        return 0, "", false
    }
    rest := path[len("/api/v"):]
    slash := strings.Index(rest, "/")
    var versionStr, remaining string
    if slash == -1 {
        versionStr = rest
        remaining = "/"
    } else {
        versionStr = rest[:slash]
        remaining = rest[slash:]
    }
    version, err := strconv.Atoi(versionStr)
    if err != nil || version < 1 {
        return 0, "", false
    }
    return version, remaining, true
}

// VersionFromContext extracts the API version from the request context.
func VersionFromContext(ctx context.Context) (int, bool) {
    v, ok := ctx.Value(versionContextKey{}).(int)
    return v, ok
}
```

---

## 3. Deprecation Policy

### 3.1 Sunset Header (RFC 8594)

RFC 8594 defines the `Sunset` HTTP header for communicating the removal date of
an API endpoint. GGID should emit it on every deprecated endpoint.

```
Deprecation: @1735689600   <!-- Unix timestamp: 2025-01-01 -->
Sunset: @1767225600        <!-- Unix timestamp: 2026-01-01 -->
Link: <https://docs.ggid.dev/migration/v1-to-v2>; rel="deprecation"
```

### 3.2 Deprecation Timeline

For enterprise IAM systems, the minimum deprecation window is **12 months**:

```
Phase 1 (Month 0):   Announce deprecation in changelog and developer portal.
                     No code changes — API still works normally.

Phase 2 (Month 1):   Add Deprecation + Sunset headers to all deprecated responses.
                     Log deprecation warnings server-side for analytics.

Phase 3 (Month 6):   Return 299 (Miscellaneous Persistent Warning) in addition
                     to normal response. Include migration deadline in body.
                     Begin proactively notifying clients via email.

Phase 4 (Month 11):  Return 410 (Gone) for new clients. Existing clients with
                     registered migration tokens get a 30-day grace extension.

Phase 5 (Month 12):  Remove endpoint entirely. Return 410 (Gone) with migration
                     documentation link for all requests.
```

### 3.3 Communication Channels

| Channel | Audience | Timing |
|---|---|---|
| API changelog | Developers reading docs | Phase 1 onward |
| Email notification | Registered API clients | Phase 1, 3, 4 |
| Developer portal banner | All portal visitors | Phase 1 onward |
| `Deprecation` header | All API consumers | Phase 2 onward |
| `Sunset` header | All API consumers | Phase 2 onward |
| Response body warning field | Programmatic clients | Phase 3 onward |
| Status page | Operations teams | Phase 1 onward |

### 3.4 Deprecation Middleware

```go
// DeprecationInfo describes a deprecated endpoint.
type DeprecationInfo struct {
    SunsetDate   time.Time // When the endpoint will be removed
    MigrationURL string    // Link to migration documentation
    ReplacedBy   string    // New endpoint that replaces this one
}

// DeprecationMiddleware adds Deprecation and Sunset headers to responses
// for endpoints that have been deprecated.
func DeprecationMiddleware(deprecated map[string]DeprecationInfo) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            info, isDeprecated := deprecated[r.URL.Path]
            if isDeprecated {
                // RFC 8594: Sunset header
                w.Header().Set("Sunset", formatHTTPDate(info.SunsetDate))

                // Deprecation header (RFC draft): Unix timestamp
                w.Header().Set("Deprecation", fmt.Sprintf("@%d", info.SunsetDate.Unix()))

                // Link header with migration guidance
                if info.MigrationURL != "" {
                    w.Header().Set("Link",
                        fmt.Sprintf(`<%s>; rel="deprecation"; type="text/html"`, info.MigrationURL))
                }
                if info.ReplacedBy != "" {
                    w.Header().Set("Link",
                        fmt.Sprintf(`<%s>; rel="successor-version"`, info.ReplacedBy))
                }

                // Log deprecation usage for analytics
                log.Printf("[DEPRECATION] %s %s — sunset: %s",
                    r.Method, r.URL.Path, info.SunsetDate.Format(time.RFC3339))
            }
            next.ServeHTTP(w, r)
        })
    }
}

func formatHTTPDate(t time.Time) string {
    // RFC 7231 IMF-fixdate format: "Sun, 06 Nov 1994 08:49:37 GMT"
    return t.UTC().Format(http.TimeFormat)
}
```

### 3.5 Usage in Gateway

```go
deprecatedEndpoints := map[string]DeprecationInfo{
    "/api/v1/auth/legacy-login": {
        SunsetDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
        MigrationURL: "https://docs.ggid.dev/migration/auth-v2",
        ReplacedBy:   "/api/v2/auth/login",
    },
}

handler := DeprecationMiddleware(deprecatedEndpoints)(gateway.Handler())
```

---

## 4. Breaking Change Management

### 4.1 Version Bump Process (v1 to v2)

A breaking change triggers the following process:

1. **Create v2 namespace**: New route prefix `/api/v2/*` in gateway config.
   New proto package `api/gen/{service}/v2`. New handler implementations.

2. **Copy and modify**: Start from v1 handler code. Make the breaking change.
   Do not modify v1 — it stays frozen.

3. **Dual-run period**: Both v1 and v2 run simultaneously for the full
   deprecation window (12 months minimum).

4. **Migration documentation**: Publish a migration guide listing every
   breaking change with before/after examples.

5. **SDK update**: Release new SDK version targeting v2. Deprecate old SDK
   that targets v1.

6. **Announce**: Email all registered API clients. Update developer portal.

### 4.2 Running Multiple Versions Simultaneously

```go
// MultiVersionMux routes requests to different API versions.
// Each version has its own complete set of handlers.
type MultiVersionMux struct {
    versions map[int]*http.ServeMux
}

func NewMultiVersionMux() *MultiVersionMux {
    return &MultiVersionMux{
        versions: make(map[int]*http.ServeMux),
    }
}

func (m *MultiVersionMux) AddVersion(v int, mux *http.ServeMux) {
    m.versions[v] = mux
}

func (m *MultiVersionMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    version, remaining, ok := extractVersion(r.URL.Path)
    if !ok {
        // No version in path — serve latest stable
        version = m.latestStable()
        remaining = r.URL.Path
    }

    mux, exists := m.versions[version]
    if !exists {
        writeJSON(w, http.StatusNotFound, map[string]any{
            "error":           "unsupported_api_version",
            "requested":       fmt.Sprintf("v%d", version),
            "supported":       m.supportedVersions(),
            "latest_stable":   fmt.Sprintf("v%d", m.latestStable()),
        })
        return
    }

    // Rewrite path to strip version prefix
    r2 := r.Clone(r.Context())
    r2.URL.Path = remaining
    mux.ServeHTTP(w, r2)
}

func (m *MultiVersionMux) latestStable() int {
    max := 1
    for v := range m.versions {
        if v > max {
            max = v
        }
    }
    return max
}

func (m *MultiVersionMux) supportedVersions() []string {
    versions := make([]string, 0, len(m.versions))
    for v := range m.versions {
        versions = append(versions, fmt.Sprintf("v%d", v))
    }
    return versions
}
```

### 4.3 Security Implications of Old Versions

Running old API versions indefinitely creates security debt:

- **Known vulnerabilities**: v1 may have auth bypass bugs that are fixed in v2
  but still exploitable via v1 endpoints.
- **Weak crypto**: v1 may accept SHA-1 token signatures; v2 mandates SHA-256.
- **Deprecated flows**: v1 may still accept OAuth implicit grant; v2 removed it.
  Attackers can use the weaker v1 flow.
- **Missing security headers**: v1 may not emit required security headers.

**Recommendation**: Never run more than 2 major versions simultaneously.
When v3 launches, v1 must be removed (v2 enters deprecation).

---

## 5. Version-Specific Security Policies

### 5.1 Forcing Upgrade from Vulnerable Versions

When a critical vulnerability is discovered in an API version, the gateway must
be able to immediately restrict or block that version without waiting for the
full deprecation window.

```go
// VersionSecurityPolicy defines per-version security rules.
type VersionSecurityPolicy struct {
    Version          int
    Blocked          bool     // If true, reject all requests to this version
    BlockReason      string   // Human-readable reason for blocking
    BlockedFlows     []string // Specific auth flows to reject (e.g., "implicit")
    MaxTokenLifetime time.Duration // Maximum token TTL this version can issue
    RequiredHeaders  []string // Headers that must be present
    ForbiddenHeaders []string // Headers that must not be present
}

// VersionPolicyEnforcer enforces security policies per API version.
type VersionPolicyEnforcer struct {
    policies map[int]*VersionSecurityPolicy
}

func NewVersionPolicyEnforcer() *VersionPolicyEnforcer {
    return &VersionPolicyEnforcer{
        policies: make(map[int]*VersionSecurityPolicy),
    }
}

func (e *VersionPolicyEnforcer) SetPolicy(p *VersionSecurityPolicy) {
    e.policies[p.Version] = p
}

func (e *VersionPolicyEnforcer) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        version, _, ok := extractVersion(r.URL.Path)
        if !ok {
            // Non-versioned path — skip policy check
            next.ServeHTTP(w, r)
            return
        }

        policy, exists := e.policies[version]
        if !exists {
            // No policy for this version — allow
            next.ServeHTTP(w, r)
            return
        }

        // Check if entire version is blocked
        if policy.Blocked {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusGone)
            json.NewEncoder(w).Encode(map[string]string{
                "error":  "api_version_blocked",
                "detail": policy.BlockReason,
            })
            return
        }

        // Check forbidden auth flows
        if flow := extractAuthFlow(r); flow != "" {
            for _, blocked := range policy.BlockedFlows {
                if flow == blocked {
                    w.Header().Set("Content-Type", "application/json")
                    w.WriteHeader(http.StatusForbidden)
                    json.NewEncoder(w).Encode(map[string]string{
                        "error":  "auth_flow_blocked",
                        "detail": fmt.Sprintf("The '%s' flow is not available in API v%d", flow, version),
                    })
                    return
                }
            }
        }

        // Check required headers
        for _, hdr := range policy.RequiredHeaders {
            if r.Header.Get(hdr) == "" {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]string{
                    "error":  "missing_required_header",
                    "detail": fmt.Sprintf("Header '%s' is required for API v%d", hdr, version),
                })
                return
            }
        }

        next.ServeHTTP(w, r)
    })
}

// extractAuthFlow identifies the OAuth flow from request parameters.
func extractAuthFlow(r *http.Request) string {
    responseType := r.URL.Query().Get("response_type")
    switch responseType {
    case "token":
        return "implicit"
    case "code":
        return "authorization_code"
    default:
        return ""
    }
}
```

### 5.2 Version-Aware Rate Limiting

Older API versions may receive stricter rate limits to incentivize migration:

```go
// VersionRateLimits maps API versions to rate limit multipliers.
// v1 gets 0.5x the normal rate limit; v2 gets 1.0x.
var VersionRateLimits = map[int]float64{
    1: 0.5,  // v1: half the normal rate limit
    2: 1.0,  // v2: full rate limit
}

// ApplyVersionRateLimit adjusts the rate limit based on API version.
func ApplyVersionRateLimit(baseLimit int, version int) int {
    multiplier, ok := VersionRateLimits[version]
    if !ok {
        multiplier = 1.0
    }
    return int(float64(baseLimit) * multiplier)
}
```

### 5.3 Version-Aware Scope Enforcement

```go
// VersionScopePolicy defines which scopes are valid per API version.
type VersionScopePolicy struct {
    // Scopes that were removed in this version (rejected if present)
    RemovedScopes map[string]bool
    // Scopes that are newly required in this version
    RequiredScopes map[string]bool
}

// EnforceVersionScopes validates that JWT scopes comply with version policies.
func EnforceVersionScopes(policies map[int]*VersionScopePolicy) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            version, _, ok := extractVersion(r.URL.Path)
            if !ok {
                next.ServeHTTP(w, r)
                return
            }

            policy, exists := policies[version]
            if !exists {
                next.ServeHTTP(w, r)
                return
            }

            scopes := extractScopesFromJWT(r)

            // Reject removed scopes
            for _, s := range scopes {
                if policy.RemovedScopes[s] {
                    writeJSON(w, http.StatusForbidden, map[string]string{
                        "error":  "scope_not_available",
                        "detail": fmt.Sprintf("Scope '%s' is not available in API v%d", s, version),
                    })
                    return
                }
            }

            // Check required scopes
            for reqScope := range policy.RequiredScopes {
                found := false
                for _, s := range scopes {
                    if s == reqScope {
                        found = true
                        break
                    }
                }
                if !found {
                    writeJSON(w, http.StatusForbidden, map[string]string{
                        "error":  "missing_required_scope",
                        "detail": fmt.Sprintf("Scope '%s' is required for API v%d", reqScope, version),
                    })
                    return
                }
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 6. API Gateway Version Routing

### 6.1 Version-Based Reverse Proxy

The gateway must route `/v1/*` and `/v2/*` to potentially different backend
deployments, or to the same backend with version-aware handlers.

```go
// VersionRouter routes requests based on API version prefix.
type VersionRouter struct {
    // backends maps "service:version" to backend URL.
    // Example: "auth:1" -> "http://auth-v1:9001", "auth:2" -> "http://auth-v2:9001"
    backends map[string]string

    // proxies caches reverse proxies per backend.
    proxies map[string]*httputil.ReverseProxy
}

func NewVersionRouter(routes map[string]string) *VersionRouter {
    vr := &VersionRouter{
        backends: routes,
        proxies:  make(map[string]*httputil.ReverseProxy),
    }
    for key, url := range routes {
        parsed, _ := url_.Parse(url)
        vr.proxies[key] = httputil.NewSingleHostReverseProxy(parsed)
    }
    return vr
}

// Route resolves the backend for a given path.
// Path format: /api/v{version}/{service}/{resource}
func (vr *VersionRouter) Route(path string) (*httputil.ReverseProxy, bool) {
    version, remaining, ok := extractVersion(path)
    if !ok {
        return nil, false
    }

    // Extract service name from remaining path: "/auth/login" -> "auth"
    parts := strings.SplitN(strings.TrimPrefix(remaining, "/"), "/", 2)
    if len(parts) == 0 {
        return nil, false
    }
    service := parts[0]

    key := fmt.Sprintf("%s:%d", service, version)
    proxy, exists := vr.proxies[key]
    return proxy, exists
}

// VersionProxyHandler is an http.Handler that routes by version.
func (vr *VersionRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    proxy, ok := vr.Route(r.URL.Path)
    if !ok {
        // Try fallback: same service, latest version
        version, remaining, _ := extractVersion(r.URL.Path)
        parts := strings.SplitN(strings.TrimPrefix(remaining, "/"), "/", 2)
        if len(parts) > 0 {
            // Find latest version for this service
            for v := version + 1; v <= version+5; v++ {
                key := fmt.Sprintf("%s:%d", parts[0], v)
                if p, exists := vr.proxies[key]; exists {
                    p.ServeHTTP(w, r)
                    return
                }
            }
        }
        writeJSON(w, http.StatusNotFound, map[string]string{
            "error": "no_backend_for_version",
        })
        return
    }
    proxy.ServeHTTP(w, r)
}
```

### 6.2 Deprecation Tracking at Gateway Level

The gateway is the ideal place to track deprecation usage because all traffic
flows through it. Per-version metrics should include:

- Request count per version per endpoint
- Unique client count per version (from JWT `azp` claim or API key)
- Top clients still using deprecated endpoints (for targeted outreach)

```go
// VersionMetrics tracks per-version request statistics.
type VersionMetrics struct {
    mu       sync.Mutex
    counters map[string]*versionCounter // key: "v1:/api/v1/auth/login"
}

type versionCounter struct {
    requests     int64
    lastAccessed time.Time
    topClients   map[string]int64 // client_id -> request count
}

func (vm *VersionMetrics) Record(version int, path, clientID string) {
    key := fmt.Sprintf("v%d:%s", version, path)
    vm.mu.Lock()
    defer vm.mu.Unlock()

    counter, exists := vm.counters[key]
    if !exists {
        counter = &versionCounter{topClients: make(map[string]int64)}
        vm.counters[key] = counter
    }
    counter.requests++
    counter.lastAccessed = time.Now()
    if clientID != "" {
        counter.topClients[clientID]++
    }
}
```

---

## 7. GGID API Versioning Audit

### 7.1 Gateway Router Analysis

The GGID gateway (`services/gateway/internal/router/router.go`) uses **hardcoded
`/api/v1/` route prefixes** configured in `services/gateway/internal/config/config.go`:

```go
Routes: map[string]string{
    "/api/v1/auth":         "http://localhost:9001",
    "/api/v1/users":        "http://localhost:8081",
    "/api/v1/roles":        "http://localhost:8070",
    "/api/v1/permissions":  "http://localhost:8070",
    "/api/v1/policies":     "http://localhost:8070",
    "/api/v1/orgs":         "http://localhost:8081",
    "/api/v1/audit":        "http://localhost:8082",
    "/oauth":               "http://localhost:9005",
    "/saml":                "http://localhost:9005",
},
```

**Findings:**

1. **All REST routes are hardcoded to v1.** There is no mechanism to add v2 routes.
   The `publicPaths` list in `router.go` also hardcodes v1-specific paths:
   ```go
   var publicPaths = []string{
       "/api/v1/auth/login",
       "/api/v1/auth/register",
       "/api/v1/auth/refresh",
       "/api/v1/auth/password/forgot",
       "/api/v1/auth/password/reset",
       "/api/v1/auth/social/",
       // ...
   }
   ```
   Adding v2 auth endpoints would require updating this list.

2. **No version extraction logic.** The `matchBackend` function does longest-prefix
   matching against route prefixes. It does not parse or extract the API version
   from the path. If `/api/v2/auth` were added, it would need a separate route
   entry, and the `buildHealthChecker` method would need updating since it
   strips `/api/v1/` prefix:
   ```go
   name := strings.TrimPrefix(prefix, "/api/v1/")
   ```

3. **OAuth and SAML endpoints are unversioned.** Paths like `/oauth/token`,
   `/oauth/authorize`, and `/saml/acs` have no version prefix. OAuth 2.1 and
   OIDC spec changes are handled by RFC compliance, not API versioning.

4. **Gateway management APIs are v1-only.** Paths like `/api/v1/gateway/routes`
   and `/api/v1/admin/stats` are gateway-internal and would need their own
   versioning if the gateway API changes.

### 7.2 Service-Level Versioning

| Service | Versioning Mechanism | Versioned? | Notes |
|---|---|---|---|
| Gateway | URL path `/api/v1/` | Yes (hardcoded v1) | No v2 support mechanism |
| Identity | gRPC proto `api/gen/identity/v1` + HTTP `/api/v1/users` | Yes | Proto package versioned |
| Auth | URL path `/api/v1/auth/` | Yes | No version negotiation |
| OAuth | Mixed: `/oauth/*` (unversioned) + `/api/v1/oauth/*` (v1) | **Inconsistent** | OAuth spec endpoints unversioned; REST API v1 |
| Policy | gRPC proto `api/gen/policy/v1` + HTTP `/api/v1/roles` | Yes | gRPC versioned |
| Org | gRPC proto `api/gen/org/v1` + HTTP `/api/v1/orgs` | Yes | gRPC versioned |
| Audit | gRPC proto `api/gen/audit/v1` + HTTP `/api/v1/audit` | Yes | gRPC versioned |

### 7.3 Versioning Gaps Identified

1. **No version extraction or dispatch logic.** The gateway does not parse the
   version from the URL. All routes are string-prefix matched. There is no
   `extractVersion()` function. Adding v2 requires manually duplicating all
   route entries with v2 prefixes.

2. **`publicPaths` is version-coupled.** If v2 auth endpoints are added, the
   public paths list must be manually updated. This is error-prone — a missed
   entry would cause v2 login to return 401.

3. **No deprecation or sunset header support.** No middleware emits
   `Deprecation`, `Sunset`, or `Link: rel="deprecation"` headers. There is no
   mechanism to mark endpoints as deprecated.

4. **No version discovery endpoint.** There is no `GET /api/versions` or similar
   endpoint that lists supported API versions and their status (stable, beta,
   deprecated, sunset date).

5. **No per-version metrics.** The `StatsCollector` and `BackendStats` track
   per-prefix statistics but do not aggregate by API version.

6. **Health checker strips only v1 prefix.** The `buildHealthChecker` method
   uses `strings.TrimPrefix(prefix, "/api/v1/")`, which would fail for v2
   routes.

7. **OAuth dual-path inconsistency.** The OAuth service registers both
   unversioned paths (`/oauth/token`) and versioned aliases
   (`/api/v1/oauth/introspect`). This is inconsistent — some endpoints have v1
   aliases and others do not.

8. **No version-aware security policy.** There is no middleware that enforces
   different security rules (blocked flows, required headers, scope validation)
   based on API version.

---

## 8. Version Documentation and Discovery

### 8.1 Version Discovery Endpoint

Clients need a machine-readable way to discover which API versions are supported
and their lifecycle status.

```go
// VersionInfo describes a single API version.
type VersionInfo struct {
    Version     string   `json:"version"`      // "v1", "v2"
    Status      string   `json:"status"`       // "stable", "beta", "deprecated", "sunset"
    ReleasedAt  string   `json:"released_at"`  // ISO 8601
    SunsetAt    string   `json:"sunset_at,omitempty"` // ISO 8601, if deprecated
    ChangesURL  string   `json:"changes_url"`  // Link to changelog
    MigrationURL string  `json:"migration_url,omitempty"` // Link to migration guide
}

// VersionDiscoveryResponse is the response for GET /api/versions.
type VersionDiscoveryResponse struct {
    Versions       []VersionInfo `json:"versions"`
    LatestStable   string        `json:"latest_stable"`
    ServerVersion  string        `json:"server_version"`
}

// HandleVersionDiscovery serves the version discovery endpoint.
func HandleVersionDiscovery(versions []VersionInfo, serverVersion string) http.HandlerFunc {
    latestStable := "v1"
    for _, v := range versions {
        if v.Status == "stable" {
            latestStable = v.Version
        }
    }
    return func(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, http.StatusOK, VersionDiscoveryResponse{
            Versions:      versions,
            LatestStable:  latestStable,
            ServerVersion: serverVersion,
        })
    }
}

// Registration in gateway:
// mux.HandleFunc("/api/versions", HandleVersionDiscovery(allVersions, "1.0.0"))
```

Example response:

```json
{
  "versions": [
    {
      "version": "v1",
      "status": "deprecated",
      "released_at": "2024-01-15T00:00:00Z",
      "sunset_at": "2026-01-01T00:00:00Z",
      "changes_url": "https://docs.ggid.dev/changelog/v1",
      "migration_url": "https://docs.ggid.dev/migration/v1-to-v2"
    },
    {
      "version": "v2",
      "status": "stable",
      "released_at": "2025-06-01T00:00:00Z",
      "changes_url": "https://docs.ggid.dev/changelog/v2"
    }
  ],
  "latest_stable": "v2",
  "server_version": "1.0.0"
}
```

### 8.2 OpenAPI Spec Per Version

Each API version should have its own OpenAPI specification. The gateway already
serves a single OpenAPI spec at `/api-docs`. This should be extended to serve
version-specific specs:

```go
// HandleVersionedOpenAPI serves OpenAPI specs per version.
func HandleVersionedOpenAPI(specs map[int]string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        version, _, ok := extractVersion(r.URL.Path)
        if !ok {
            // Serve aggregated spec (all versions)
            w.Header().Set("Content-Type", "application/json")
            _, _ = w.Write([]byte(specs[1])) // default to v1
            return
        }
        spec, exists := specs[version]
        if !exists {
            http.NotFound(w, r)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(spec))
    }
}
```

### 8.3 Changelog Automation

Changelogs should be auto-generated from conventional commit messages:

```
feat(auth): add v2 passwordless login endpoint      → Minor version bump
break(api): remove deprecated v1 role assignment     → Major version bump
fix(auth): correct token expiration calculation       → Patch version bump
```

A CI job parses commit messages since the last release tag, categorizes them,
and generates a markdown changelog. The changelog is published to the developer
portal and linked from the version discovery endpoint.

---

## 9. Gap Analysis and Recommendations

### 9.1 Current State Summary

GGID uses URL path versioning (`/api/v1/`) consistently for REST APIs and proto
package versioning for gRPC. However, the versioning is **static and hardcoded**.
There is no infrastructure for version lifecycle management: no deprecation
headers, no version discovery endpoint, no version-aware security policies, and
no mechanism to add v2 routes without manual code changes.

### 9.2 Action Items

| # | Action | Priority | Effort | Description |
|---|---|---|---|---|
| 1 | **Add version extraction utility** | P1 | 2h | Create `extractVersion()` function in gateway router. Replace hardcoded `/api/v1/` string matching with version-parsed routing. Update `publicPaths` to be version-agnostic (match `/api/v*/auth/login`). |
| 2 | **Implement deprecation middleware** | P1 | 4h | Add `DeprecationMiddleware` that emits `Deprecation`, `Sunset`, and `Link` headers on deprecated endpoints. Create a deprecation registry (config file or DB table) listing deprecated paths with sunset dates. Wire into gateway middleware chain. |
| 3 | **Add version discovery endpoint** | P2 | 2h | Implement `GET /api/versions` returning all supported versions with status, release date, and sunset date. Register in gateway router. Link from developer portal. |
| 4 | **Add version-aware security policy** | P2 | 6h | Implement `VersionPolicyEnforcer` middleware that can block specific auth flows per version (e.g., reject implicit grant in v2), enforce required headers, and apply version-specific rate limits. |
| 5 | **Fix health checker version stripping** | P3 | 1h | Update `buildHealthChecker` to use `extractVersion()` instead of hardcoded `strings.TrimPrefix(prefix, "/api/v1/")`. This ensures v2 routes get correct health check service names. |

### 9.3 Security Recommendations

1. **Never silently drop a version.** When removing v1, return `410 Gone` with
   a `Link` header pointing to migration documentation for at least 30 days
   after removal.

2. **Block deprecated auth flows immediately in new versions.** When v2 launches,
   the gateway should reject OAuth implicit grant (`response_type=token`) at the
   v2 level even if the backend still supports it. Use
   `VersionPolicyEnforcer.BlockedFlows`.

3. **Track version usage for security audit.** Every IAM request should log the
   API version. During a security incident, this enables identifying which
   clients are still using a vulnerable version.

4. **Enforce minimum version for admin APIs.** Admin endpoints
   (`/api/v1/admin/*`) should require the latest stable version. If v2 exists,
   admin access via v1 should be blocked (admin APIs change most frequently and
   have the highest security impact).

5. **Version the JWKS and discovery endpoints.** While OAuth/OIDC endpoints are
   traditionally unversioned (they follow the spec), the JWKS endpoint should
   support a version query parameter to allow key rotation announcements: the
   response should include the key version alongside the key ID.

---

## References

- RFC 8594: The Sunset HTTP Header Field
- RFC 7231: Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content (IMF-fixdate)
- OAuth 2.0 Authorization Framework (RFC 6749)
- OpenID Connect Discovery 1.0
- Google API Design Guide: Versioning
- Stripe API Versioning Guide
- AWS API Lifecycle Management
