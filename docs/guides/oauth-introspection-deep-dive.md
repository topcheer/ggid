# OAuth Introspection Deep Dive

This guide covers RFC 7662 full spec, introspection endpoint security, response fields, caching strategy, DPoP-bound introspection, resource server patterns, rate limiting, and GGID's implementation.

## RFC 7662 Overview

Token introspection allows resource servers to query the authorization server about the validity and metadata of an access token. This is essential for opaque (non-JWT) tokens or when the resource server needs real-time token status.

### Endpoint

```
POST /oauth/introspect
Content-Type: application/x-www-form-urlencoded
Authorization: Bearer <resource_server_credentials>

token=eyJ...&token_type_hint=access_token
```

### Response

```json
{
  "active": true,
  "scope": "users:read users:write",
  "client_id": "web-app-001",
  "username": "jdoe@example.com",
  "token_type": "Bearer",
  "exp": 1700000600,
  "iat": 1700000000,
  "sub": "user-uuid-1234",
  "aud": "ggid-api",
  "iss": "https://auth.ggid.example.com",
  "jti": "token-uuid-5678",
  "cnf": {
    "jkt": "sha256-hash-of-dpop-key"
  }
}
```

## Endpoint Security

### Authentication Required

The introspection endpoint MUST be authenticated. Only authorized resource servers should query it.

| Method | Description | Recommendation |
|---|---|---|
| Bearer token | Resource server uses its own token | Minimum |
| Mutual TLS (mTLS) | Client cert + bearer | Recommended |
| HTTP Basic | client_id + client_secret | Acceptable |
| Private key JWT | Signed JWT assertion | High security |

### mTLS Authentication (RFC 8705)

```yaml
introspection:
  auth:
    method: "tls_client_auth"
    required_cert: true
    cert_subject_cn: "api.ggid.example.com"
    reject_without_cert: true
```

```go
func introspectionAuth(r *http.Request) error {
    // Verify mTLS
    if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
        return ErrClientCertRequired
    }
    cert := r.TLS.PeerCertificates[0]
    if cert.Subject.CommonName != "api.ggid.example.com" {
        return ErrUnauthorizedClient
    }
    // Verify bearer token
    token := extractBearer(r)
    if !isValidServiceToken(token) {
        return ErrInvalidToken
    }
    return nil
}
```

### Rate Limiting

```yaml
introspection:
  rate_limit:
    per_client: 1000/minute
    per_ip: 100/minute
    burst: 50
```

Resource servers should cache introspection results (see below) to reduce calls.

## Response Fields

### Core Fields (RFC 7662)

| Field | Type | Description |
|---|---|---|
| `active` | boolean | Token is currently active |
| `scope` | string | Space-delimited scopes |
| `client_id` | string | Client that requested token |
| `username` | string | Human-readable user identifier |
| `token_type` | string | Token type (Bearer, DPoP) |
| `exp` | integer | Expiration timestamp |
| `iat` | integer | Issued-at timestamp |
| `sub` | string | Subject (user ID) |
| `aud` | string | Intended audience |
| `iss` | string | Token issuer |
| `jti` | string | Token unique identifier |
| `nbf` | integer | Not-before timestamp |

### Extension Fields

| Field | Type | Description |
|---|---|---|
| `tenant_id` | string | Tenant identifier |
| `roles` | array | User roles |
| `permissions` | array | User permissions |
| `mfa_verified` | boolean | MFA completion |
| `acr` | string | Authentication context class |
| `amr` | array | Authentication methods |
| `cnf` | object | Confirmation (DPoP binding) |
| `step_up_at` | integer | Step-up timestamp |

### Active Flag Semantics

`active: true` when ALL of:
- Token signature valid
- Not expired (exp > now)
- Not yet valid (nbf <= now, if present)
- Not revoked (jti not in blacklist)
- Issuer matches
- Audience matches (if checked)

`active: false` for:
- Expired tokens
- Revoked tokens
- Invalid signatures
- Unknown tokens

## Caching Strategy

### Positive Caching (Active Tokens)

Cache `active: true` responses with short TTL:

```yaml
introspection:
  caching:
    positive:
      enabled: true
      ttl: 60s  # Short — don't cache too long
      max_entries: 100000
      key: "jti"  # Cache by token JTI
```

### Negative Caching (Inactive Tokens)

Cache `active: false` responses with longer TTL:

```yaml
introspection:
  caching:
    negative:
      enabled: true
      ttl: 300s  # Longer — expired tokens won't become active
      max_entries: 50000
```

### Cache Invalidation

```go
func invalidateIntrospectionCache(jti string) {
    cache.Delete("introspect:" + jti)
}

// On token revocation
func RevokeToken(jti string) {
    redis.Set(ctx, "revoked:"+jti, "1", ttl)
    invalidateIntrospectionCache(jti)  // Clear cached "active" result
}
```

### Cache Implementation

```go
type IntrospectionCache struct {
    cache Cache
    config CacheConfig
}

func (c *IntrospectionCache) Get(token string) (*IntrospectionResponse, bool) {
    key := "introspect:" + hashToken(token)
    if val, ok := c.cache.Get(key); ok {
        resp := val.(*IntrospectionResponse)
        // Check if cached response is still valid
        if resp.Active && time.Now().Unix() > resp.Exp {
            c.cache.Delete(key)
            return nil, false
        }
        return resp, true
    }
    return nil, false
}

func (c *IntrospectionCache) Set(token string, resp *IntrospectionResponse) {
    key := "introspect:" + hashToken(token)
    ttl := c.config.PositiveTTL
    if !resp.Active {
        ttl = c.config.NegativeTTL
    }
    // Don't cache beyond token expiry
    if resp.Exp > 0 {
        maxTTL := time.Until(time.Unix(resp.Exp, 0))
        if ttl > maxTTL {
            ttl = maxTTL
        }
    }
    c.cache.Set(key, resp, ttl)
}
```

## DPoP-Bound Introspection

### DPoP Token Introspection

For DPoP-bound tokens, the introspection response includes `cnf`:

```json
{
  "active": true,
  "cnf": {
    "jkt": "sha256-hash-of-dpop-public-key"
  }
}
```

### Resource Server Validation

```go
func validateDPoPToken(token string, dpopProof string, r *http.Request) error {
    // 1. Introspect token
    introspection := introspect(token)

    // 2. Check DPoP binding
    if introspection.Cnf == nil || introspection.Cnf.JKT == "" {
        return ErrTokenNotDPoPBound
    }

    // 3. Validate DPoP proof
    dpopKey := extractDPoPPublicKey(dpopProof)
    jkt := sha256Hash(dpopKey)

    if jkt != introspection.Cnf.JKT {
        return ErrDPoPKeyMismatch
    }

    // 4. Validate DPoP proof signature + htm + htu
    return validateDPoPProof(dpopProof, r.Method, r.URL.String(), token)
}
```

## Resource Server Patterns

### Pattern 1: JWT Validation (No Introspection)

For JWT tokens, resource servers can validate locally:
- Verify signature using JWKS
- Check exp, nbf, iss, aud
- Check revocation via Redis (if implemented)

### Pattern 2: Introspection (Opaque Tokens)

For opaque tokens, introspect on every request (with caching):
- Call introspection endpoint
- Cache result
- Use cached result for subsequent requests

### Pattern 3: Hybrid

- Parse JWT locally for basic validation
- Introspect for real-time revocation status
- Cache introspection result for revocation check only

```go
func validateToken(token string) (*Claims, error) {
    // Try JWT validation first
    claims, err := parseJWT(token)
    if err == nil {
        // Check revocation via cached introspection
        if cached, ok := introspectionCache.Get(token); ok {
            if !cached.Active {
                return nil, ErrRevoked
            }
            return claims, nil
        }
        // Cache miss — introspect
        resp, err := introspect(token)
        if err != nil {
            return nil, err
        }
        introspectionCache.Set(token, resp)
        if !resp.Active {
            return nil, ErrRevoked
        }
        return claims, nil
    }

    // Not a JWT — full introspection
    resp, err := introspect(token)
    if err != nil {
        return nil, err
    }
    if !resp.Active {
        return nil, ErrInvalidToken
    }
    return introspectionToClaims(resp), nil
}
```

## GGID Implementation

### Endpoint

```bash
POST /oauth/introspect
Authorization: Bearer <resource_server_token>
Content-Type: application/x-www-form-urlencoded

token=eyJ...&token_type_hint=access_token
```

### Handler

```go
func (s *Server) HandleIntrospect(w http.ResponseWriter, r *http.Request) {
    // Authenticate caller
    if err := introspectionAuth(r); err != nil {
        writeError(w, 401, "unauthorized")
        return
    }

    token := r.FormValue("token")
    if token == "" {
        writeError(w, 400, "missing_token")
        return
    }

    // Check cache
    if cached, ok := s.cache.Get(token); ok {
        writeJSON(w, cached)
        return
    }

    // Introspect
    response := s.introspectToken(token)
    
    // Cache result
    s.cache.Set(token, response)
    
    writeJSON(w, response)
}

func (s *Server) introspectToken(token string) *IntrospectionResponse {
    // Parse token
    claims, err := s.parseToken(token)
    if err != nil {
        return &IntrospectionResponse{Active: false}
    }

    // Check revocation
    if s.isRevoked(claims.ID) {
        return &IntrospectionResponse{Active: false}
    }

    // Check expiry
    if time.Now().Unix() > claims.ExpiresAt {
        return &IntrospectionResponse{Active: false}
    }

    return &IntrospectionResponse{
        Active:   true,
        Scope:    claims.Scope,
        ClientID: claims.Audience,
        Subject:  claims.Subject,
        Exp:      claims.ExpiresAt,
        Iat:      claims.IssuedAt,
        Iss:      claims.Issuer,
        Jti:      claims.ID,
        Audience: claims.Audience,
        TenantID: claims.TenantID,
        Roles:    claims.Roles,
        Cnf:      claims.Cnf,
    }
}
```

### Configuration

```yaml
introspection:
  enabled: true
  auth:
    method: "tls_client_auth"
    require_bearer: true
  caching:
    positive:
      enabled: true
      ttl: 60s
    negative:
      enabled: true
      ttl: 300s
  rate_limit:
    per_client: 1000/minute
    per_ip: 100/minute
  dpop:
    support: true
```

## Best Practices

1. **Always authenticate the caller** — Introspection reveals token metadata
2. **Use mTLS** — Stronger than bearer token alone
3. **Cache aggressively** — Reduce introspection endpoint load
4. **Negative cache** — Cache `active: false` longer (expired tokens stay expired)
5. **Invalidate on revocation** — Clear cache when tokens are revoked
6. **Rate limit** — Prevent introspection endpoint abuse
7. **Support DPoP** — Include cnf in response for DPoP-bound tokens
8. **Don't leak sensitive data** — Only return fields the caller is authorized to see
9. **Monitor introspection calls** — Track volume, cache hit rate, response times
10. **Consider JWT for local validation** — Reduce introspection calls with self-contained tokens