# JWT Lifecycle Management

This guide covers JWT token types, signing algorithms, key rotation, validation, refresh token rotation, revocation, and DPoP binding in GGID.

## Token Types

### Access Token

Short-lived token for API authorization.

```json
{
  "iss": "https://auth.ggid.example.com",
  "sub": "user-uuid",
  "aud": "ggid-api",
  "exp": 1700000600,
  "iat": 1700000000,
  "jti": "unique-token-id",
  "scope": "users:read users:write",
  "tenant_id": "tenant-uuid",
  "roles": ["admin"],
  "mfa_verified": true
}
```

| Property | Value |
|---|---|
| Lifetime | 15 minutes (default) |
| Signing | RS256 / ES256 / EdDSA |
| Storage | Client-side only (no server session) |
| Revocation | Via Redis blacklist (jti) |

### ID Token

OIDC identity token for client authentication.

```json
{
  "iss": "https://auth.ggid.example.com",
  "sub": "user-uuid",
  "aud": "client-id",
  "exp": 1700000600,
  "iat": 1700000000,
  "nonce": "client-nonce",
  "email": "user@example.com",
  "email_verified": true,
  "name": "John Doe"
}
```

| Property | Value |
|---|---|
| Lifetime | 15 minutes |
| Signing | RS256 / ES256 |
| Contains | User claims per granted scopes |
| Validation | Client validates signature + claims |

### Refresh Token

Long-lived token for obtaining new access tokens.

```json
{
  "iss": "https://auth.ggid.example.com",
  "sub": "user-uuid",
  "aud": "ggid-token-endpoint",
  "exp": 1700600000,
  "iat": 1700000000,
  "jti": "refresh-token-id",
  "tenant_id": "tenant-uuid"
}
```

| Property | Value |
|---|---|
| Lifetime | 7 days (default), configurable up to 30 days |
| Rotation | Every use (one-time refresh tokens) |
| Reuse Detection | If reused → revoke entire token family |
| Storage | Hashed in database + Redis for fast lookup |

### Agent Token

Special token for AI agent identity (MCP auth).

```json
{
  "iss": "https://auth.ggid.example.com",
  "sub": "agent-uuid",
  "aud": "mcp-server",
  "exp": 1700003600,
  "iat": 1700000000,
  "jti": "agent-token-id",
  "delegation_chain": ["user-uuid", "agent-uuid"],
  "mcp_servers": ["https://mcp.example.com"],
  "max_delegation_depth": 3
}
```

| Property | Value |
|---|---|
| Lifetime | 1 hour |
| Signing | RS256 |
| Delegation | Chain of trust from user → agent → sub-agent |

## Signing Algorithms

| Algorithm | Key Type | Key Size | Performance | Recommendation |
|---|---|---|---|---|
| HS256 | HMAC (shared secret) | 256-bit | Fast | Internal services only |
| RS256 | RSA | 2048-bit | Slower verify | Default, widely compatible |
| ES256 | ECDSA P-256 | 256-bit | Fast | Recommended for new deployments |
| EdDSA | Ed25519 | 256-bit | Very fast | Modern, excellent security |

### Algorithm Configuration

```yaml
jwt:
  signing_algorithm: "RS256"
  private_key: /etc/ggid/jwt/private.pem
  public_key: /etc/ggid/jwt/public.pem
  key_id: "2026-07-01"
```

### Algorithm Selection Guide

- **HS256**: Only for internal service-to-service where secret sharing is acceptable
- **RS256**: Default choice, broad ecosystem support
- **ES256**: Best performance-to-security ratio for new deployments
- **EdDSA**: Cutting edge, not all libraries support it yet

**Never use**: `none` algorithm, HS256 with weak keys, RS256 with <2048-bit keys.

## Key Rotation

### Dual-Key Period

GGID supports seamless key rotation using a dual-key overlap period:

```
Time:  ──────────────────────────────────────────▶
Keys:  [key-2026-01 active]     [key-2026-07 active]
                ↕ overlap (7 days) ↕
```

1. **Publish new key** to JWKS endpoint (key-2026-07)
2. **Dual signing period** (7 days): New tokens signed with new key, old key still validates
3. **Old key expires**: Remove from JWKS, reject tokens signed with old key

### Configuration

```yaml
jwt:
  key_rotation:
    enabled: true
    interval: 90d        # Rotate every 90 days
    overlap: 7d          # Dual-key period
    notification: 14d    # Notify admins 14 days before rotation
```

### JWKS Endpoint

```
GET /.well-known/jwks.json

{
  "keys": [
    {"kid": "2026-01", "kty": "RSA", "use": "sig", "alg": "RS256", "n": "...", "e": "AQAB"},
    {"kid": "2026-07", "kty": "RSA", "use": "sig", "alg": "RS256", "n": "...", "e": "AQAB"}
  ]
}
```

### Automatic Rotation Script

```bash
#!/bin/bash
# Generate new key pair
openssl genrsa -out /etc/ggid/jwt/new-private.pem 2048
openssl rsa -in /etc/ggid/jwt/new-private.pem -pubout -out /etc/ggid/jwt/new-public.pem
# Update key_id to current date
# Reload GGID to pick up new keys
systemctl reload ggid-auth
```

## Token Validation Pipeline

GGID validates every incoming JWT through a multi-stage pipeline:

```
1. Parse token → extract header + payload + signature
2. Verify signature → match kid from JWKS, verify with public key
3. Check exp → reject if expired
4. Check nbf → reject if not yet valid (clock skew tolerance)
5. Check iss → must match configured issuer
6. Check aud → must match expected audience
7. Check scope → required scopes present
8. Check jti → not in Redis revocation blacklist
9. Extract tenant_id → validate tenant is active
10. Optional: Check DPoP proof → verify token binding
```

### Validation Implementation

```go
func ValidateToken(tokenStr string, config *ValidationConfig) (*Claims, error) {
    // 1. Parse
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        kid := t.Header["kid"].(string)
        key := config.JWKS.GetKey(kid)
        return key, nil
    })
    if err != nil {
        return nil, fmt.Errorf("signature invalid: %w", err)
    }

    claims := token.Claims.(*Claims)

    // 2. Expiry
    if !claims.VerifyExpiresAt(time.Now().Add(config.ClockSkew), true) {
        return nil, ErrExpired
    }

    // 3. Not-before
    if !claims.VerifyNotBefore(time.Now().Add(-config.ClockSkew), true) {
        return nil, ErrNotYetValid
    }

    // 4. Issuer
    if claims.Issuer != config.Issuer {
        return nil, ErrInvalidIssuer
    }

    // 5. Audience
    if !claims.VerifyAudience(config.Audience, true) {
        return nil, ErrInvalidAudience
    }

    // 6. JTI revocation check
    if config.IsRevoked(claims.ID) {
        return nil, ErrRevoked
    }

    return claims, nil
}
```

## Refresh Token Rotation

### One-Time Use Refresh Tokens

Each refresh token is single-use. After use, a new refresh token is issued:

```
1. Client → POST /oauth/token (grant_type=refresh_token, refresh_token=RT1)
2. Server validates RT1 → issues AT2 + RT2
3. RT1 is marked used → added to reuse detection list
4. If RT1 is used again → revoke entire token family (RT2, RT3, ...)
```

### Reuse Detection

```go
func RotateRefreshToken(oldRT string) (*TokenPair, error) {
    // Check if token was already used
    if isUsed, _ := redis.Get(ctx, "rt:used:"+oldRT); isUsed == "1" {
        // COMPROMISED! Revoke entire family
        familyID := extractFamilyID(oldRT)
        revokeTokenFamily(familyID)
        log.Security("Refresh token reuse detected, family revoked: " + familyID)
        return nil, ErrTokenReuseDetected
    }

    // Mark old token as used
    redis.Set(ctx, "rt:used:"+oldRT, "1", 24*time.Hour)

    // Issue new token pair
    newRT := generateRefreshToken()
    newAT := generateAccessToken()
    return &TokenPair{Access: newAT, Refresh: newRT}, nil
}
```

### Token Family

All refresh tokens derived from the same initial login share a family ID. Reusing any member revokes the entire family.

## Token Revocation

### Redis Blacklist

```go
func RevokeToken(jti string, exp int64) error {
    ttl := time.Until(time.Unix(exp, 0))
    return redis.Set(ctx, "revoked:"+jti, "1", ttl)
}

func IsRevoked(jti string) bool {
    val, _ := redis.Get(ctx, "revoked:"+jti)
    return val == "1"
}
```

### Revocation Scenarios

| Scenario | Action |
|---|---|
| User logout | Revoke access token + refresh token |
| Admin force logout | Revoke all tokens for user (by family) |
| Password change | Revoke all tokens except current session |
| MFA reset | Revoke all tokens, require re-auth |
| Account suspension | Revoke all tokens immediately |
| Security incident | Revoke all tokens in tenant |

### Revocation Endpoint

```
POST /oauth/revoke
Content-Type: application/x-www-form-urlencoded
Authorization: Bearer <client_credentials>

token=<jwt>&token_type_hint=access_token
```

## Short-Lived vs Long-Lived Tradeoffs

| Factor | Short-Lived (15min) | Long-Lived (1h+) |
|---|---|---|
| Security | High (stolen tokens expire fast) | Lower (larger attack window) |
| User experience | Frequent refreshes | Fewer interruptions |
| Server load | High (more token issuance) | Lower |
| Revocation effectiveness | Fast (natural expiry) | Requires active blacklist |

### Recommended Lifetimes

| Token Type | Lifetime | Rationale |
|---|---|---|
| Access Token | 15 min | Balance security + performance |
| ID Token | 15 min | Matches access token |
| Refresh Token | 7 days | Convenience without excessive risk |
| Agent Token | 1 hour | Short for security, enough for MCP session |
| Service Token | 1 hour | For machine-to-machine |

## DPoP Binding

DPoP (Demonstration of Proof-of-Possession) binds tokens to a private key held by the client:

### Token Binding

```json
{
  "iss": "https://auth.ggid.example.com",
  "sub": "user-uuid",
  "cnf": {
    "jkt": "sha256-hash-of-public-key"
  }
}
```

### DPoP Proof Validation

```go
func ValidateDPoP(proof string, method string, url string, accessToken string) error {
    // 1. Parse DPoP JWT header
    header := parseJWTHeader(proof)
    if header["typ"] != "dpop+jwt" {
        return ErrInvalidDPoPType
    }

    // 2. Extract public key from header
    pubKey := extractPublicKey(header["jwk"])

    // 3. Verify signature
    if !verifySignature(proof, pubKey) {
        return ErrInvalidDPoPSignature
    }

    // 4. Verify htm (HTTP method) and htu (URL)
    claims := parseJWTClaims(proof)
    if claims["htm"] != method || claims["htu"] != url {
        return ErrDPoPMismatch
    }

    // 5. Verify jti (nonce) — prevent replay
    if isReplayed(claims["jti"]) {
        return ErrDPoPReplay
    }

    // 6. Verify ath (access token hash) matches
    if claims["ath"] != hashBase64URL(accessToken) {
        return ErrDPoPTokenMismatch
    }

    return nil
}
```

### DPoP Benefits

- Prevents token replay (attacker needs both token AND private key)
- No TLS client certificate required
- Works with existing OAuth/OIDC flows
- RFC 9449 compliant

## Best Practices

1. **Use asymmetric algorithms** (RS256/ES256/EdDSA) — no shared secret distribution
2. **Rotate keys regularly** — 90-day intervals with 7-day overlap
3. **Short-lived access tokens** — 15 minutes is the sweet spot
4. **Rotate refresh tokens** — one-time use with reuse detection
5. **Implement revocation** — Redis blacklist with TTL matching token expiry
6. **Validate all claims** — exp, nbf, iss, aud, scope, jti
7. **Use DPoP for high-security clients** — prevents token theft replay
8. **Audit token issuance** — log every token issued with metadata
9. **Monitor for anomalies** — unusual token issuance patterns, reuse attempts
10. **Clock skew tolerance** — 60 seconds is standard