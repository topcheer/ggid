# Token Lifecycle Design

This guide covers the full token lifecycle from issuance to destruction, covering access tokens, refresh tokens, ID tokens, agent tokens, storage, binding, and revocation.

## Token Types Summary

| Token | Lifetime | Purpose | Storage |
|---|---|---|---|
| Access Token | 15 min | API authorization | Client-side |
| Refresh Token | 7 days | Obtain new access tokens | Hashed in DB + Redis |
| ID Token | 15 min | User identity (OIDC) | Client-side |
| Agent Token | 1 hour | AI agent identity | Client-side |

## Full Lifecycle

```
Issuance → Distribution → Use → Validation → Rotation/Refresh → Revocation/Expiry
```

### 1. Issuance

```go
func issueAccessToken(user *User, client *Client, scopes []string) (string, error) {
    now := time.Now()
    claims := jwt.MapClaims{
        "iss": issuer, "sub": user.ID, "aud": client.ID,
        "exp": now.Add(15 * time.Minute).Unix(),
        "iat": now.Unix(), "jti": uuid.New().String(),
        "scope": strings.Join(scopes, " "),
        "tenant_id": user.TenantID, "roles": user.Roles,
    }
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = keyID
    return token.SignedString(privateKey)
}
```

### 2. Distribution

Tokens are returned via token endpoint response (never in URL fragment):

```json
{"access_token": "eyJ...", "refresh_token": "rt...", "token_type": "Bearer", "expires_in": 900}
```

### 3. Use

Client sends token in Authorization header:

```
Authorization: Bearer eyJ...
```

### 4. Validation Pipeline

1. Parse JWT → extract header + payload + signature
2. Verify signature via JWKS (kid match)
3. Check exp → reject if expired
4. Check nbf → reject if not yet valid
5. Check iss → must match configured issuer
6. Check aud → must match expected audience
7. Check scope → required scopes present
8. Check jti → not in Redis revocation blacklist
9. Optional: Verify DPoP proof

### 5. Refresh

```
Client → POST /token (grant_type=refresh_token, refresh_token=RT1)
Server → validates RT1, issues AT2 + RT2, invalidates RT1
If RT1 reused → revoke entire token family
```

### 6. Revocation

```go
func revokeToken(jti string, exp int64) {
    ttl := time.Until(time.Unix(exp, 0))
    redis.Set(ctx, "revoked:"+jti, "1", ttl)
    invalidateIntrospectionCache(jti)
    audit.Log("token_revoked", jti)
}
```

### Cascading Revocation

| Trigger | Revoke |
|---|---|
| User logout | Current access + refresh token |
| Admin force logout | All tokens for user (by family) |
| Password change | All tokens except current session |
| MFA reset | All tokens, require re-auth |
| Account suspension | All tokens immediately |
| Security incident | All tokens in tenant |

## Token Binding

### DPoP Binding

Token includes `cnf.jkt` claim. Resource server validates DPoP proof + cnf match.

### mTLS Binding

Token bound to client certificate fingerprint. Connection rejected if cert doesn't match.

## GGID Token Lifecycle

```yaml
token:
  access:
    lifetime: 15m
    signing: "RS256"
    key_rotation: 90d
  refresh:
    lifetime: 7d
    rotation: "required"
    reuse_detection: true
    family_revocation: true
  id_token:
    lifetime: 15m
    include_standard_claims: true
  agent:
    lifetime: 1h
    max_delegation_depth: 3
  revocation:
    method: "redis_blacklist"
    cascade: true
    audit: true
  binding:
    dpop: true
    mtls: true
```

## Best Practices

1. **Short access token lifetime** — 15 minutes
2. **Rotate refresh tokens** — One-time use with reuse detection
3. **Revoke on security events** — Password change, MFA reset, suspension
4. **Bind tokens with DPoP** — Prevent replay even if stolen
5. **Audit all token events** — Issuance, refresh, revocation
6. **Cache revocation checks** — Redis for fast jti lookup
7. **Cascade revocation** — Revoke entire family on reuse
8. **Key rotation with overlap** — 90 days with 7-day overlap
9. **Never store tokens in localStorage** — Use httpOnly cookies or Keychain
10. **Monitor for anomalies** — Unusual token issuance patterns