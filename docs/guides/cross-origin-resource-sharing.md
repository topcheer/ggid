# CORS Configuration Guide

Preflight, allowed origins per tenant, credential mode, header policy, error handling, security headers, and CORS vs reverse proxy.

## Overview

Cross-Origin Resource Sharing (CORS) controls which browser origins can access GGID APIs. Misconfigured CORS is a common security vulnerability — this guide ensures correct, secure configuration.

## How CORS Works

### Simple Request (No Preflight)

```
Browser (from https://app.example.com)
  → GET /api/v1/users
  → Origin: https://app.example.com

Server responds:
  → Access-Control-Allow-Origin: https://app.example.com
  → Access-Control-Allow-Credentials: true
```

### Preflight Request (OPTIONS)

For non-simple requests (POST, custom headers, PUT/DELETE):

```
Browser sends preflight:
  → OPTIONS /api/v1/users
  → Origin: https://app.example.com
  → Access-Control-Request-Method: POST
  → Access-Control-Request-Headers: Content-Type, Authorization

Server responds:
  → Access-Control-Allow-Origin: https://app.example.com
  → Access-Control-Allow-Methods: GET, POST, PUT, DELETE
  → Access-Control-Allow-Headers: Content-Type, Authorization
  → Access-Control-Max-Age: 600
```

Browser caches preflight response for `Max-Age` seconds (default 600).

## GGID CORS Configuration

### Per-Tenant Origins

```bash
# Configure allowed origins per tenant
PUT /api/v1/admin/tenants/{tenant_id}/cors
{
  "allowed_origins": [
    "https://app.acme.com",
    "https://console.acme.com",
    "https://localhost:3000"
  ],
  "allowed_methods": ["GET", "POST", "PUT", "PATCH", "DELETE"],
  "allowed_headers": ["Authorization", "Content-Type", "X-Request-ID"],
  "exposed_headers": ["X-RateLimit-Remaining", "X-Request-ID"],
  "allow_credentials": true,
  "max_age": 600
}
```

### Gateway Middleware

```go
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")

            // Validate origin against allowlist
            if isAllowed(origin, allowedOrigins) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Credentials", "true")
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-Tenant-ID")
                w.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Remaining, X-Request-ID")
                w.Header().Set("Access-Control-Max-Age", "600")
            }

            // Handle preflight
            if r.Method == "OPTIONS" {
                w.WriteHeader(204)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

## Origin Validation

### Exact Match (Required)

```go
func isAllowed(origin string, allowed []string) bool {
    for _, a := range allowed {
        if origin == a {  // Exact match, no wildcards
            return true
        }
    }
    return false
}
```

**Never use wildcards** (`*`) with credentials. Browsers reject `Access-Control-Allow-Origin: *` when `Allow-Credentials: true`.

### Subdomain Pattern Matching

```go
// Allow *.acme.com for a tenant
func isAllowedPattern(origin string, patterns []string) bool {
    for _, pattern := range patterns {
        if strings.HasPrefix(pattern, "*.") {
            suffix := pattern[1:] // ".acme.com"
            if strings.HasSuffix(origin, suffix) {
                // Verify it's a subdomain, not a spoof
                host := strings.TrimSuffix(origin, suffix)
                if host != "" && !strings.Contains(host, ".") {
                    return true
                }
            }
        } else if origin == pattern {
            return true
        }
    }
    return false
}
```

## Credential Mode

```http
Access-Control-Allow-Credentials: true
```

When enabled, browsers send cookies and Authorization headers cross-origin. Required for GGID because auth tokens are sent via cookies or headers.

### With Credentials

| Implication | Detail |
|-------------|--------|
| Origin must be specific | `*` is rejected by browsers |
| Cookies sent cross-origin | Ensure SameSite=Lax or None+Secure |
| Auth headers sent | Bearer tokens cross-origin |

## Header Policy

### Allowed Request Headers

| Header | Required | Purpose |
|--------|----------|---------|
| `Authorization` | Yes | Bearer token |
| `Content-Type` | Yes | JSON requests |
| `X-Request-ID` | Yes | Request tracing |
| `X-Tenant-ID` | Conditional | Tenant override (admin only) |
| `X-CSRF-Token` | Conditional | CSRF protection |

### Exposed Response Headers

| Header | Purpose |
|--------|---------|
| `X-RateLimit-Remaining` | Client-side rate limit awareness |
| `X-Request-ID` | Request correlation |
| `X-Total-Count` | Pagination total |

## Security Headers

CORS is one layer — combine with:

```http
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'; connect-src 'self' https://api.ggid.dev
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
```

## Common Pitfalls

### 1. Wildcard with Credentials

```http
# WRONG — browser rejects this combination
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true
```

### 2. Origin Reflection (Vulnerable)

```go
// WRONG — reflects any origin (allows all sites)
w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
// Never do this! Validate against allowlist.
```

### 3. Null Origin

```http
Origin: null
```

Some browsers send `null` for sandboxed iframes or file:// origins. Always reject `null`.

### 4. Missing Preflight Handler

```go
// If OPTIONS is not handled, it reaches the actual handler
// which may reject it (401 auth required) — breaking CORS
if r.Method == "OPTIONS" {
    w.WriteHeader(204) // Must short-circuit preflight
    return
}
```

## CORS vs Reverse Proxy

### Option 1: CORS (Cross-Origin)

```
Browser (app.acme.com) → CORS → Gateway (api.ggid.dev)
```

- Pro: Clean separation, scalable
- Con: CORS preflight overhead, cookie complexity

### Option 2: Reverse Proxy (Same-Origin)

```
Browser (app.acme.com) → Next.js API Route (/api/*) → Gateway (internal)
```

- Pro: No CORS needed (same origin), cookies just work
- Con: Extra hop, all traffic through frontend

GGID Console uses Option 2 (reverse proxy via Next.js API routes). External SDKs use Option 1 (CORS).

## Error Handling

CORS errors are browser-side — the server can't "fix" them after the response. Common errors:

| Browser Error | Cause | Fix |
|--------------|-------|-----|
| `No 'Access-Control-Allow-Origin' header` | Origin not in allowlist | Add origin to tenant CORS config |
| `Credentials flag is true, but header is *` | Wildcard with credentials | Use specific origin |
| `Method not allowed` | Preflight rejected method | Add method to allowed_methods |
| `Header not allowed` | Preflight rejected header | Add header to allowed_headers |

## Monitoring

| Metric | Alert |
|--------|-------|
| CORS preflight failures | >1% → check allowed origins |
| Unknown origins (rejected) | Spike → possible misconfigured client |
| Preflight cache misses | High rate → check Max-Age |

## See Also

- [Gateway Architecture](gateway-architecture.md)
- [Session Security](session-security.md)
- [Console Development](console-development.md)
- [SDK Integration Guide](sdk-integration-guide.md)
