# JWT Claim Validation Guide

Validation order, error handling, clock skew tolerance, and claim sources for iss, aud, exp, nbf, iat, sub, scope, and custom claims.

## Validation Order

Claims are validated in a specific order — fail fast on cheapest checks first:

```
1. Structure & signature (is it a valid JWT?)
2. iss (is it from the right issuer?)
3. aud (was it issued for this service?)
4. alg (was it signed with an allowed algorithm?)
5. exp (has it expired?)
6. nbf (is it valid yet?)
7. iat (was it issued in the past?)
8. sub (does the subject exist?)
9. scope (does it have required permissions?)
10. Custom claims (tenant_id, jti, etc.)
```

## Claim Reference

### `iss` (Issuer)

```go
if claims["iss"] != "https://auth.ggid.dev" {
    return ErrInvalidIssuer
}
```

| Rule | Enforcement |
|------|-------------|
| Must match GGID issuer URL exactly | Reject if mismatched |
| Case-sensitive | `HTTPS://` ≠ `https://` |
| No trailing slash | Configure consistently |

### `aud` (Audience)

```go
func validateAudience(claims jwt.MapClaims, expected string) error {
    switch v := claims["aud"].(type) {
    case string:
        if v != expected { return ErrAudienceMismatch }
    case []interface{}:
        found := false
        for _, a := range v {
            if a.(string) == expected { found = true; break }
        }
        if !found { return ErrAudienceMismatch }
    default:
        return ErrMissingAudience
    }
    return nil
}
```

| Service | Expected Audience |
|---------|------------------|
| Gateway | `ggid-gateway` |
| Identity | `identity-svc` |
| Policy | `policy-svc` |
| Audit | `audit-svc` |

A token issued for `identity-svc` is rejected by `policy-svc`.

### `exp` (Expiration)

```go
exp, ok := claims["exp"].(float64)
if !ok { return ErrMissingExp }
if time.Now().Add(clockSkew).Unix() > int64(exp) {
    return ErrTokenExpired
}
```

### `nbf` (Not Before)

```go
nbf, ok := claims["nbf"].(float64)
if !ok { return ErrMissingNbf }
if time.Now().Add(-clockSkew).Unix() < int64(nbf) {
    return ErrTokenNotYetValid
}
```

### `iat` (Issued At)

```go
iat, ok := claims["iat"].(float64)
if !ok { return ErrMissingIat }
if time.Now().Add(-clockSkew).Unix() < int64(iat) {
    return ErrTokenIssuedInFuture  // Clock skew or tampering
}
```

## Clock Skew Tolerance

```go
const DefaultClockSkew = 30 * time.Second

type Validator struct {
    clockSkew time.Duration
}

func NewValidator() *Validator {
    return &Validator{clockSkew: DefaultClockSkew}
}
```

| Claim | Skew Direction | Rationale |
|-------|---------------|-----------|
| `exp` | Grace period (+skew) | Allow tokens that expired seconds ago |
| `nbf` | Grace period (-skew) | Allow tokens valid in seconds |
| `iat` | Grace period (-skew) | Tolerate clock differences |

**Max skew: 60 seconds.** Larger values create security windows.

## `sub` (Subject)

```go
sub, ok := claims["sub"].(string)
if !ok || sub == "" { return ErrMissingSubject }

// Verify user exists and is active
user, err := userStore.Get(sub)
if err != nil { return ErrUnknownSubject }
if user.Status != "active" { return ErrSubjectInactive }
```

## `scope` (OAuth Scopes)

```go
func (v *Validator) RequireScope(claims jwt.MapClaims, required string) error {
    scopeStr, ok := claims["scope"].(string)
    if !ok { return ErrMissingScope }

    scopes := strings.Fields(scopeStr)
    for _, s := range scopes {
        if s == required || s == required+":*" || s == "*" {
            return nil
        }
    }
    return ErrInsufficientScope
}
```

### Scope Hierarchy Matching

```
users:admin  → satisfies → users:write → satisfies → users:read
*:admin      → satisfies → any admin scope
```

## Custom Claims

### `tenant_id`

```go
tenantID, ok := claims["tenant_id"].(string)
if !ok { return ErrMissingTenant }

// JWT claim takes priority over X-Tenant-ID header (prevent spoofing)
ctx = context.WithValue(ctx, "tenant_id", tenantID)
```

### `jti` (JWT ID — Anti-Replay)

```go
jti, ok := claims["jti"].(string)
if !ok { return ErrMissingJTI }

// Check Redis blacklist
if revoked := redis.SIsMember("jwt:blacklist", jti); revoked {
    return ErrTokenRevoked
}
```

### `act` (Actor — Delegation)

```go
if act, ok := claims["act"].(map[string]interface{}); ok {
    // Token was obtained via delegation
    delegatingService := act["sub"].(string)
    depth := getDelegationDepth(act)
    if depth > 3 { return ErrMaxDelegationDepth }
}
```

### `cnf` (Confirmation — Token Binding)

```go
if cnf, ok := claims["cnf"].(map[string]interface{}); ok {
    // DPoP binding
    if jkt, ok := cnf["jkt"].(string); ok {
        if requestDPoPThumbprint != jkt {
            return ErrTokenBindingMismatch
        }
    }
    // mTLS binding
    if x5t, ok := cnf["x5t"].(string); ok {
        if certThumbprint != x5t {
            return ErrCertBindingMismatch
        }
    }
}
```

## Claim Sources

| Source | Claims Available | Trust Level |
|--------|-----------------|-------------|
| Access token (JWT) | sub, scope, aud, exp, cnf | High (signed) |
| Introspection endpoint | All + active, client_id | High (server-verified) |
| UserInfo endpoint | sub, email, profile | Medium (may differ from token) |
| ID token | sub, email, profile, nonce | High (OIDC signed) |

### Introspection Fallback

```go
// If JWT validation fails (unknown kid), introspect
func validateWithFallback(token string) (Claims, error) {
    claims, err := localVerify(token)
    if err == ErrUnknownKey {
        return introspect(token)  // Server-side verification
    }
    return claims, err
}
```

## Error Handling

```go
type ValidationError struct {
    Claim   string
    Code    string
    Message string
}

var (
    ErrInvalidSignature  = ValidationError{"signature", "invalid_sig", "signature verification failed"}
    ErrTokenExpired      = ValidationError{"exp", "expired", "token has expired"}
    ErrAudienceMismatch  = ValidationError{"aud", "wrong_audience", "token audience mismatch"}
    ErrInsufficientScope = ValidationError{"scope", "insufficient_scope", "required scope missing"}
)
```

### HTTP Error Mapping

| Validation Error | HTTP Status | Error Code |
|-----------------|-------------|------------|
| Invalid signature | 401 | `invalid_token` |
| Expired | 401 | `invalid_token` (desc: "expired") |
| Wrong audience | 401 | `invalid_token` (desc: "wrong audience") |
| Missing scope | 403 | `insufficient_scope` |
| Unknown issuer | 401 | `invalid_token` (desc: "unknown issuer") |
| Revoked (jti) | 401 | `invalid_token` (desc: "revoked") |

## Monitoring

| Metric | Alert |
|--------|-------|
| Signature failures | >1% → possible attack or JWKS issue |
| Expired token rate | >5% → client not refreshing |
| Audience mismatches | Spike → misconfigured service |
| Clock skew rejections | Any → check NTP sync |

## See Also

- [JWT Security Best Practices](jwt-security-best-practices.md)
- [Token Exchange Patterns](token-exchange-patterns.md)
- [OAuth Refresh Token Rotation](oauth-refresh-token-rotation.md)
- [Gateway Architecture](gateway-architecture.md)
