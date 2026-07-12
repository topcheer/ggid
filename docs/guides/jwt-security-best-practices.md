# JWT Security Best Practices

## Algorithm Confusion Prevention

GGID only accepts RS256 and ES256. Rejects `alg: none` and HS256-with-RSA-key attacks.

```go
// Only allow these algorithms
var allowedAlgs = []string{"RS256", "ES256"}
token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
    if !contains(allowedAlgs, t.Header["alg"]) {
        return nil, fmt.Errorf("unexpected algorithm")
    }
    return jwks.Get(t.Header["kid"])
})
```

## Key Rotation

See [Key Rotation Procedure](key-rotation-procedure.md). Publish old + new keys in JWKS during grace period.

## Claim Validation Checklist

| Claim | Validation |
|-------|-----------|
| `exp` | Must be in future |
| `nbf` | Must be before now |
| `iat` | Must be in past |
| `iss` | Must match GGID issuer |
| `aud` | Must match this service |
| `sub` | Must be valid user UUID |
| `tenant_id` | Must match request tenant |
| `scope` | Must include required scope |
| `jti` | Must not be in Redis blacklist |

## Short-Lived Tokens

| Token Type | TTL | Rationale |
|-----------|-----|-----------|
| Access token | 15 min | Minimize theft window |
| Refresh token | 7 days | Single-use rotation |
| MFA token | 5 min | Step-up only |
| Agent token | 15 min | Delegation |

## Refresh Rotation

```
Refresh A → (used) → New access + New refresh B → A invalidated
If A reused after B → Token family revoked (theft detected)
```

## JWKS Caching

```go
// Cache JWKS 15 minutes, re-fetch on unknown kid
jwksCache := cache.New(15*time.Minute)
func getKey(kid string) (crypto.PublicKey, error) {
    if key := jwksCache.Get(kid); key != nil { return key, nil }
    jwks := fetchJWKS() // HTTP GET /.well-known/jwks.json
    for _, k := range jwks.Keys { jwksCache.Set(k.Kid, k) }
    return jwksCache.Get(kid)
}
```

## See Also

- [Token Revocation Strategy](token-revocation-strategy.md)
- [Key Rotation Procedure](key-rotation-procedure.md)
- [OAuth API](../api/oauth.md)
