# RFC 6750: OAuth 2.0 Bearer Token Usage — Research & GGID Audit

## 1. Overview

RFC 6750 defines how clients use bearer tokens to access OAuth 2.0 protected
resources. Published in October 2012, it is the companion to RFC 6749 (the
authorization framework) and specifies the wire format for presenting access
tokens to resource servers.

A **bearer token** is a security token whose mere possession grants access —
any party holding the token ("the bearer") can use it without demonstrating
possession of a cryptographic key. This simplicity is the strength and weakness
of bearer tokens: they are trivial to implement but offer no protection against
theft or replay.

RFC 6750 defines **three methods** for presenting bearer tokens in HTTP
requests:

1. **Authorization Request Header** — `Authorization: Bearer <token>` (RECOMMENDED)
2. **Form-Encoded Body Parameter** — `access_token=<token>` in POST body
3. **URI Query Parameter** — `?access_token=<token>` in the URL

Only the first is recommended. OAuth 2.1 (draft) removes methods 2 and 3
entirely, keeping only the Authorization header.

---

## 2. Token Presentation Methods

### Method 1: Authorization Header (RECOMMENDED)

```
GET /api/v1/users HTTP/1.1
Host: api.ggid.dev
Authorization: Bearer mF_9.B5f-4.1JqM
```

**Pros:** standard HTTP header (works with all methods), token never in URL (no
log/history/Referer leakage), CORS-compatible with cacheable preflight.

**Use for:** all API calls. This is the only method GGID supports.

### Method 2: Form-Encoded Body Parameter

```
POST /api/v1/users HTTP/1.1
Host: api.ggid.dev
Content-Type: application/x-www-form-urlencoded

access_token=mF_9.B5f-4.1JqM&name=Alice
```

**Limitations:** only works with `application/x-www-form-urlencoded` bodies,
only for POST/PUT/PATCH (not GET/DELETE), mixes auth with request data,
incompatible with JSON/multipart bodies.

**Use only when:** Authorization header is impossible (rare legacy cases).

### Method 3: URI Query Parameter (DEPRECATED)

```
GET /api/v1/users?access_token=mF_9.B5f-4.1JqM HTTP/1.1
Host: api.ggid.dev
```

**Security problems:** token persists in server access logs, browser history,
Referer headers (leaks to third-party sites), proxy caches, and shared links.

**Use for:** browser-based fallback ONLY. OAuth 2.1 removes this method entirely.

### Comparison Table

| Method | Security | Universal | Logging Risk | CORS-Friendly | Recommendation |
|--------|----------|-----------|--------------|---------------|----------------|
| Authorization Header | High | All methods | Low (header not logged) | Yes (cacheable preflight) | **Always use** |
| Form Body Parameter | Medium | POST/PUT/PATCH only | Low (body not logged) | Requires preflight | Avoid except legacy |
| URI Query Parameter | **Low** | All methods | **High** (URL logged everywhere) | Yes | **Never use** |

---

## 3. WWW-Authenticate Error Response

When a resource server rejects a request due to a token problem, it MUST
return HTTP 401 with a `WWW-Authenticate` header using the `Bearer` scheme.

### Response Format

```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="ggid",
                              error="invalid_token",
                              error_description="The access token expired"
Content-Type: application/json
```

Attributes: `realm` (protection space), `error` (one of three codes below),
`error_description` (human-readable, no sensitive data), `scope` (for
`insufficient_scope` errors).

### Error Codes (RFC 6750 section 3.1)

| Error Code | HTTP Status | Meaning |
|------------|-------------|---------|
| `invalid_request` | 400 | Malformed request: missing required parameter, duplicate parameters, or multiple methods used simultaneously |
| `invalid_token` | 401 | Token is expired, revoked, malformed, or unknown to the resource server |
| `insufficient_scope` | 403 | Token is valid but lacks the scope required for the requested resource |

### Error Response Example (Expired Token)

```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="ggid",
                              error="invalid_token",
                              error_description="The access token expired"
```

### Scope Challenge Response (403 Forbidden)

```
HTTP/1.1 403 Forbidden
WWW-Authenticate: Bearer realm="ggid",
                              error="insufficient_scope",
                              scope="admin read:users",
                              error_description="Token missing required scope"
```

---

## 4. Scope Negotiation

### How Scopes Work

When a token is issued, the authorization server grants it specific scopes
(e.g., `"read:users write:users"`). The resource server defines the required
scope for each endpoint. Validation is straightforward:

```
token_scopes ⊇ required_scopes → allow
otherwise → 403 insufficient_scope
```

### Scope Models

**Flat** — each scope is independent (`read:users`, `write:users`).
**Hierarchical** — parent implies children (`admin` implies `read` + `write`).
GGID's policy service is the natural home for scope hierarchy.

### WWW-Authenticate Scope Challenge

On 403, include `scope` attribute so clients can re-authorize:
```
HTTP/1.1 403 Forbidden
WWW-Authenticate: Bearer realm="api", error="insufficient_scope", scope="write:users"
```

### Go: Scope Validation

```go
var ErrInsufficientScope = errors.New("insufficient scope")

// validateScopes checks that the token's scopes include all required scopes.
func validateScopes(tokenScopes, requiredScopes []string) error {
    granted := make(map[string]bool, len(tokenScopes))
    for _, s := range tokenScopes {
        granted[s] = true
    }
    for _, required := range requiredScopes {
        if !granted[required] {
            return fmt.Errorf("%w: missing %q", ErrInsufficientScope, required)
        }
    }
    return nil
}

// RequireScopes returns middleware that enforces scope requirements per route.
func RequireScopes(scopes ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := ExtractJWTClaims(r)
            if err := validateScopes(claims.Scopes, scopes); err != nil {
                w.Header().Set("WWW-Authenticate",
                    fmt.Sprintf(`Bearer realm="ggid", error="insufficient_scope", scope="%s"`,
                        strings.Join(scopes, " ")))
                w.WriteHeader(http.StatusForbidden)
                json.NewEncoder(w).Encode(map[string]string{
                    "error":   "insufficient_scope",
                    "detail":  err.Error(),
                })
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 5. Security Considerations

### TLS Is Mandatory

Bearer tokens MUST only be sent over TLS. GGID's gateway sets HSTS:
`Strict-Transport-Security: max-age=31536000; includeSubDomains`. HTTP-to-HTTPS
redirect should be enforced at the load balancer.

### Cookie CSRF Risks

Use the Authorization header (not cookies) for bearer tokens — GGID does this
correctly. For cookie-based sessions, GGID has `CSRFProtect` (double-submit
pattern) with `SameSite=Lax`.

### Token Leakage Vectors

| Vector | Risk | Mitigation |
|--------|------|------------|
| Server access logs | Medium | Never log Authorization header; GGID's `Logging` middleware logs only method/path/status — compliant |
| Browser history | High | Never use URI query parameter for tokens |
| Referer header | High | URI query param leaks via Referer to third-party sites |
| Error messages | Medium | Don't include token values in error response bodies |
| Reverse proxy logs | Medium | Configure proxies to strip Authorization header from access logs |

### Best Practices

- Short-lived tokens (15 min TTL) to limit exposure window
- Store tokens in memory (not `localStorage`) for browser SPAs
- Refresh token rotation with replay detection (revokes entire session)
- Revoke on logout via `RevokeAllForUser`
- Never share tokens across services without sender-constraining (DPoP/mTLS)

---

## 6. Bearer Token vs Sender-Constrained Tokens

### Bearer Token (Current — RFC 6750)

- Anyone possessing the token can use it
- Simplest to implement — just validate the signature and claims
- Vulnerable to theft and replay: a stolen token is fully functional until expiry

### DPoP — Demonstrating Proof-of-Possession (RFC 9449)

- Token is cryptographically bound to the client's public key
- Each request includes a `DPoP` header with a signed proof JWT
- A stolen token cannot be used without the client's private key
- See [dpop-rfc9449.md](./dpop-rfc9449.md) for detailed analysis

### mTLS — Mutual TLS (RFC 8705)

- Token is bound to the client's TLS certificate at issuance time
- The resource server validates the certificate on every request
- Strongest sender-constraining for service-to-service communication
- See [token-binding-and-dpop.md](./token-binding-and-dpop.md) for details

### Migration Path

| Phase | Mechanism | Use Case | Effort |
|-------|-----------|----------|--------|
| 1 (current) | Bearer | All clients | Complete |
| 2 | DPoP | High-security SPAs and mobile apps | ~1-2 weeks |
| 3 | mTLS | Service-to-service internal API calls | ~1 week |
| — | Bearer (retained) | Public/open APIs | Always |

Never remove bearer token support entirely — public APIs and simple clients
will always need it.

---

## 7. GGID Token Validation Audit

### Token Extraction (middleware.go, `JWTAuth`)

GGID extracts the token from the **Authorization header only**:

```go
authHeader := r.Header.Get("Authorization")
parts := strings.SplitN(authHeader, " ", 2)
if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
    // reject
}
```

No support for form body or query parameter methods. This is **fully RFC 6750
compliant** — only the recommended method is accepted.

### Error Responses (middleware.go, `writeUnauthorized`)

Current implementation:

```go
func writeUnauthorized(w http.ResponseWriter, msg string) {
    w.Header().Set("WWW-Authenticate", `Bearer realm="ggid"`)
    w.WriteHeader(http.StatusUnauthorized)
    // ... JSON body
}
```

**Gaps:**
- No `error` attribute (`invalid_token`, `invalid_request`)
- No `error_description` attribute in the WWW-Authenticate header
- No differentiation between missing token (should hint at authentication) vs.
  expired/invalid token (should say `invalid_token`)
- No `insufficient_scope` response (no scope enforcement at gateway level)

### Scope Validation

- `jwt_claims.go` extracts `scope`/`scopes` from the JWT payload and sets an
  `X-Scopes` header for downstream services
- However, `token_service.go` does **not** issue a `scope` claim in access
  tokens — only `tenant_id`, `iss`, `sub`, `aud`, `iat`, `exp`, `jti`
- No per-route scope enforcement middleware exists in the gateway

### TLS Enforcement

- `SecurityHeaders` middleware sets HSTS: compliant
- HTTP-to-HTTPS redirect is not implemented at the gateway level (assumed
  handled by the load balancer)

### Logging Safety

- The `Logging` middleware records only: method, path, status, size, duration,
  request ID — **does NOT log the Authorization header**. Compliant.

### Compliance Summary

| RFC 6750 Requirement | GGID Current Status | Compliant? | Action |
|----------------------|---------------------|------------|--------|
| Accept Authorization header (Bearer scheme) | Yes — `JWTAuth` middleware | Yes | None |
| Reject form body parameter | Not accepted (header-only) | Yes | None |
| Reject URI query parameter | Not accepted (header-only) | Yes | None |
| Return 401 with WWW-Authenticate on invalid token | Returns 401 + `Bearer realm="ggid"` | **Partial** | Add `error` and `error_description` |
| Include `invalid_token` error code on expired/malformed | Missing | **No** | Add error codes |
| Include `insufficient_scope` on 403 | No scope enforcement | **No** | Add `RequireScopes` middleware |
| Include `scope` attribute in 403 response | N/A (no scope enforcement) | **No** | Implement with scope challenge |
| Enforce TLS | HSTS header set | Yes | Verify LB redirects HTTP |
| Do not log tokens | Logging middleware omits Authorization | Yes | None |
| Issue scope claims in tokens | No `scope` claim in `IssueAccessToken` | **No** | Add scope claim |

---

## 8. Roadmap

### Phase 1: WWW-Authenticate Error Codes (P0, ~1 day)

Update `writeUnauthorized` to include RFC 6750 error attributes:

```go
func writeUnauthorizedDetailed(w http.ResponseWriter, errCode, desc string) {
    w.Header().Set("WWW-Authenticate",
        fmt.Sprintf(`Bearer realm="ggid", error="%s", error_description="%s"`,
            errCode, desc))
    w.WriteHeader(http.StatusUnauthorized)
}
```

Map validation failures to error codes:
- Missing/malformed Authorization header → `invalid_request`
- Expired/invalid signature/unknown token → `invalid_token`
- Valid token, insufficient scope → 403 + `insufficient_scope`

### Phase 2: Per-Route Scope Enforcement (P1, ~1-2 days)

1. Add `scope` claim to `IssueAccessToken` in `token_service.go`
2. Implement `RequireScopes(scopes ...string)` middleware
3. Apply per-route: `router.Handle("/admin/users", RequireScopes("admin")(handler))`
4. Return 403 with `WWW-Authenticate: Bearer ..., error="insufficient_scope", scope="..."`

### Phase 3: HSTS and TLS Hardening (P1, ~0.5 day)

- Add HTTP-to-HTTPS redirect at gateway level (not just LB)
- Consider `preload` directive for HSTS
- Add `X-Robots-Tag: noindex` to prevent token-bearing endpoints from being indexed

### Phase 4: Sender-Constrained Token Support (P2, ~1-2 weeks)

- Implement DPoP (RFC 9449) for high-security SPAs
- Implement mTLS (RFC 8705) for service-to-service calls
- Keep bearer tokens for public/open APIs

### Effort Estimate

- Phase 1-2: ~3-4 days (high impact, low effort)
- Phase 3: ~0.5 day
- Phase 4: ~1-2 weeks (separate project, see dpop-rfc9449.md)
