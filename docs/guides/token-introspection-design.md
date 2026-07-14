# Token Introspection Design (RFC 7662)

Endpoint design, caching strategy, scope filtering, rate limiting, per-resource-server auth, and privacy considerations.

## Overview

Token introspection allows resource servers to verify an access token's validity and metadata by querying the authorization server. Used when JWT local verification isn't possible (opaque tokens, revocation checks).

## Endpoint

```bash
POST /api/v1/oauth/introspect
Authorization: Basic base64(resource_server:secret)
Content-Type: application/x-www-form-urlencoded

token=opaque_or_jwt_token
&token_type_hint=access_token
```

### Response (Active)

```json
{
  "active": true,
  "scope": "openid profile users:read",
  "client_id": "client-123",
  "username": "jane@corp.com",
  "token_type": "Bearer",
  "exp": 1700000900,
  "iat": 1700000000,
  "sub": "user-uuid",
  "aud": "identity-svc",
  "tenant_id": "uuid",
  "jti": "jwt-id-123"
}
```

### Response (Inactive)

```json
{
  "active": false
}
```

## Per-Resource-Server Authentication

Each resource server authenticates with its own credentials:

```go
func IntrospectHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Authenticate the resource server (not the token holder)
    rsID, rsSecret, ok := r.BasicAuth()
    if !ok || !validateResourceServer(rsID, rsSecret) {
        http.Error(w, "unauthorized resource server", 401)
        return
    }

    // 2. Extract token to introspect
    token := r.PostFormValue("token")

    // 3. Introspect
    result := introspectToken(token)

    // 4. Filter response based on resource server's permissions
    filtered := filterForResourceServer(result, rsID)

    json.NewEncoder(w).Encode(filtered)
}
```

### Scope Filtering

```go
// Resource server only sees scopes relevant to it
func filterForResourceServer(claims Claims, rsID string) Claims {
    allowedScopes := rsScopes[rsID] // e.g., identity-svc can see users:* scopes
    filtered := []string{}
    for _, scope := range claims.Scopes {
        if contains(allowedScopes, scope) || matchesPattern(allowedScopes, scope) {
            filtered = append(filtered, scope)
        }
    }
    claims.Scopes = filtered
    return claims
}
```

## Caching Strategy

Introspection is expensive — cache results:

```go
type IntrospectionCache struct {
    cache *ristretto.Cache
    ttl   time.Duration
}

func (ic *IntrospectionCache) Introspect(token string) (*IntrospectionResult, error) {
    // Check cache first
    key := hashToken(token)
    if cached, ok := ic.cache.Get(key); ok {
        result := cached.(*IntrospectionResult)
        // Don't serve expired tokens from cache
        if time.Now().Before(result.Expiry) {
            return result, nil
        }
    }

    // Cache miss → introspect at server
    result, err := introspectAtServer(token)
    if err != nil { return nil, err }

    // Cache until min(token_exp, max_cache_ttl)
    cacheTTL := min(
        time.Until(result.Expiry),
        60*time.Second, // Max 60s cache
    )
    ic.cache.Set(key, result, cacheTTL)

    return result, nil
}
```

### Cache Invalidation

| Event | Action |
|-------|--------|
| Token revoked | Remove from cache immediately |
| User suspended | Remove all user's tokens from cache |
| Scope changed | Remove affected tokens |
| TTL expiry | Auto-evicted by ristretto |

### Cache TTL Guidelines

| Token TTL | Cache TTL | Rationale |
|-----------|-----------|-----------|
| 15 min (access) | 60s | Balance freshness vs load |
| 7 days (refresh) | Don't cache | Too long-lived |
| Opaque token | 30s | Can't verify locally |

## Rate Limiting

```bash
# Per resource server
POST /api/v1/oauth/introspect
# Headers:
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 942
```

| Limiter | Rate | Burst |
|---------|------|-------|
| Per resource server | 1000/min | 2000 |
| Per token (dedup) | 30/min | 60 |
| Global | 10000/min | 20000 |

## Opaque vs JWT Introspection

| Token Type | Local Verify | Introspection Needed |
|-----------|-------------|---------------------|
| JWT | ✅ (JWKS) | Only for revocation check |
| Opaque | ❌ | ✅ Always required |

### Hybrid (Recommended)

```go
func verifyToken(token string) (*Claims, error) {
    // Try local JWT verification first
    if claims, err := jwtVerify(token); err == nil {
        // Check revocation via cache (Redis jti blacklist)
        if !isRevoked(claims.JTI) {
            return claims, nil
        }
    }

    // Fallback to introspection (opaque or revoked check)
    return introspect(token)
}
```

## Privacy Considerations

### Minimal Response

Only return claims the resource server needs:

```yaml
introspection_response_policy:
  identity-svc:
    allow: [active, scope, sub, tenant_id, exp]
    deny: [username, email, client_id]

  audit-svc:
    allow: [active, scope, sub, exp]
    deny: [everything_else]
```

### Logging

```go
// NEVER log the token being introspected
// Log only: resource server ID + result
audit.Log("token_introspected", map[string]interface{}{
    "resource_server": rsID,
    "active": result.Active,
    "scopes": result.Scopes,
    // NO token value, NO user PII
})
```

## Security

### Authentication Required

```http
# Introspection endpoint requires resource server credentials
POST /api/v1/oauth/introspect
Authorization: Basic base64(identity-svc:rs-secret)
```

Without authentication, anyone could check arbitrary tokens (token scanning attack).

### DPoP Bound Tokens

```go
// If token is DPoP-bound, verify proof at introspection
if result.Cnf != nil && result.Cnf["jkt"] != "" {
    dpopProof := r.Header.Get("DPoP")
    if err := verifyDPoP(dpopProof, result.Cnf["jkt"]); err != nil {
        return inactiveResult // Treat as inactive
    }
}
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Introspection latency | <50ms | >200ms → optimize |
| Cache hit rate | >90% | <80% → increase TTL |
| Auth failures (RS) | <0.1% | Spike → cert/secret issue |
| Introspection volume | Track | Spike → possible scanning |

## See Also

- [JWT Claim Validation](jwt-claim-validation.md)
- [OAuth PAR/JAR/DPoP](oauth-par-jar-dpop.md)
- [OAuth Scope Design](oauth-scope-design.md)
- [Gateway Architecture](gateway-architecture.md)
