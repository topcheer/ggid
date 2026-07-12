# Grant Type Security

This guide covers OAuth/OIDC grant type security requirements, attack vectors per grant type, and GGID's grant type enforcement.

## Grant Types Overview

| Grant Type | RFC | Status in 2.1 | Use Case |
|---|---|---|---|
| authorization_code | 6749 | Required (with PKCE) | Web/mobile apps |
| client_credentials | 6749 | Supported | Server-to-server |
| refresh_token | 6749 | Supported (rotation mandatory) | Token renewal |
| device_code | 8628 | Supported | Input-constrained devices |
| implicit | 6749 | **Deprecated** | Was for SPAs |
| password | 6749 | **Deprecated** | Was for direct login |
| JWT assertion | 7521 | Supported | Enterprise federation |

## Authorization Code (with PKCE)

### Flow

```
1. Client generates code_verifier (random 43-128 chars)
2. Client computes code_challenge = BASE64URL(SHA256(code_verifier))
3. Client → GET /authorize?response_type=code&client_id=...&code_challenge=...&code_challenge_method=S256
4. Server returns authorization code
5. Client → POST /token (grant_type=authorization_code, code, code_verifier)
6. Server verifies code_verifier matches code_challenge
7. Server issues access_token + refresh_token
```

### Security Requirements

| Requirement | Enforcement |
|---|---|
| PKCE mandatory | Reject requests without code_challenge |
| code_challenge_method = S256 | Reject "plain" method |
| Authorization code single-use | Code consumed after exchange |
| Short code lifetime | 10 minutes max |
| Redirect URI exact match | Must match registered URI exactly |
| State parameter required | CSRF protection |

### Attack Vectors

| Attack | Description | Mitigation |
|---|---|---|
| Authorization code interception | Attacker steals code from redirect | PKCE (code_verifier not sent in redirect) |
| CSRF | Attacker injects their code | State parameter validation |
| Redirect URI manipulation | Attacker modifies redirect | Exact match enforcement |
| Code replay | Code used multiple times | Single-use + short TTL |
| Mix-up attack | Code from one IdP used at another | Issuer validation in token |

### GGID Enforcement

```yaml
authorization_code:
  pkce:
    required: true
    method: "S256"
    verifier_length: 43  # min
  code:
    lifetime: 10m
    single_use: true
  redirect_uri:
    exact_match: true
    allow_loopback: true  # For native apps (http://localhost)
  state:
    required: true
    length: 32
```

## Client Credentials

### Flow

```
1. Client → POST /token (grant_type=client_credentials, client_id, client_secret, scope)
2. Server validates client credentials
3. Server issues access_token (no refresh_token, no user context)
```

### Security Requirements

| Requirement | Enforcement |
|---|---|
| Client authentication | client_secret or private_key_jwt |
| No user context | Token has no sub claim |
| Scope limitation | Only service scopes, not user scopes |
| Short lifetime | 1 hour max |
| No refresh token | Must re-authenticate |

### Attack Vectors

| Attack | Description | Mitigation |
|---|---|---|
| Credential theft | Secret leaked | mTLS or private_key_jwt |
| Scope escalation | Client requests more than allowed | Pre-registered scopes only |
| Token replay | Stolen token reused | Short lifetime + DPoP |

### GGID Enforcement

```yaml
client_credentials:
  auth_method:
    - "client_secret_basic"
    - "client_secret_post"
    - "private_key_jwt"
    - "none"  # Only for public clients (not recommended)
  token:
    lifetime: 1h
    include_refresh: false
  scope:
    restrict_to_client_scopes: true
```

## Refresh Token

### Flow

```
1. Client → POST /token (grant_type=refresh_token, refresh_token=RT1)
2. Server validates RT1
3. Server issues new access_token + new refresh_token RT2
4. RT1 is invalidated (one-time use)
5. If RT1 is used again → revoke entire token family
```

### Security Requirements

| Requirement | Enforcement |
|---|---|
| One-time use | Each refresh token used only once |
| Rotation | New refresh token on every exchange |
| Reuse detection | Using old token revokes family |
| Family revocation | All tokens from same login revoked on reuse |
| Bounded lifetime | Max 30 days |
| Secure storage | Hashed at rest |

### Attack Vectors

| Attack | Description | Mitigation |
|---|---|---|
| Token theft | Attacker steals refresh token | Rotation + reuse detection |
| Token replay | Old token used after rotation | Family revocation |
| Token fixation | Attacker injects token | Bind to client + session |
| Long-lived theft | Stolen token valid for long time | Bounded lifetime |

### GGID Enforcement

```yaml
refresh_token:
  rotation: "required"
  reuse_detection: true
  family_revocation: true
  lifetime: 7d
  max_lifetime: 30d  # Absolute max from initial login
  storage: "hashed"  # Argon2id hash
  bind_to_client: true
```

### Reuse Detection Implementation

```go
func RotateRefreshToken(oldToken string) (*TokenPair, error) {
    // Check if already used
    used, _ := redis.Get(ctx, "rt:used:"+hashToken(oldToken))
    if used == "1" {
        // COMPROMISED — revoke entire family
        familyID := extractFamilyID(oldToken)
        revokeFamily(familyID)
        securityAlert("refresh_token_reuse", familyID)
        return nil, ErrTokenReuse
    }

    // Mark as used
    redis.Set(ctx, "rt:used:"+hashToken(oldToken), "1", 24*time.Hour)

    // Issue new tokens
    newRefresh := generateRefreshToken()
    setFamilyID(newRefresh, extractFamilyID(oldToken))

    return &TokenPair{
        Access:  generateAccessToken(),
        Refresh: newRefresh,
    }, nil
}
```

## Device Code (RFC 8628)

### Flow

```
1. Device → POST /device_authorization (client_id, scope)
2. Server returns device_code, user_code, verification_uri
3. Device displays: "Go to https://ggid.example.com/device and enter code: ABC-123"
4. User opens browser → enters code → authenticates
5. Device polls: POST /token (grant_type=device_code, device_code)
6. Server returns: authorization_pending → authorization_pending → access_token
```

### Security Requirements

| Requirement | Enforcement |
|---|---|
| Slow polling | 5 second interval minimum |
| Rate limit on poll | Reject faster than interval (429) |
| Short device_code lifetime | 15 minutes |
| User must authorize | User logs in on separate device |
| Device code single-use | Consumed after token issued |

### Attack Vectors

| Attack | Description | Mitigation |
|---|---|---|
| Device code phishing | Attacker tricks user into authorizing | User code format (ABC-123) + confirmation |
| Polling abuse | Client polls too fast | Rate limit + backoff |
| Code interception | Attacker steals device code | Short lifetime + single use |
| Session fixation | Attacker hijacks user session | Bind device code to user session |

### GGID Enforcement

```yaml
device_code:
  device_code_lifetime: 15m
  user_code:
    length: 8
    charset: "BCDFGHJKLMNPQRSTVWXZ"  # No ambiguous chars
    format: "XXX-XXX"  # e.g., "RNQ-KPM"
  polling:
    interval: 5s
    max_attempts: 180  # 15 min / 5s
  verification_uri: "https://auth.ggid.example.com/device"
```

## Deprecated Grant Types

### Implicit Grant (Deprecated)

**Status**: Removed in OAuth 2.1. GGID supports deprecation mode.

```yaml
implicit:
  enabled: false  # Disabled by default
  deprecation_warning: true  # Log if attempted
  sunset_date: "2026-12-31"
```

**Migration**: Use authorization_code + PKCE.

### Resource Owner Password (Deprecated)

**Status**: Removed in OAuth 2.1. GGID never implemented this.

**Migration**: Use authorization_code or device_code.

## Per-Grant-Type Security Requirements

### Summary Matrix

| Requirement | auth_code | client_creds | refresh | device_code |
|---|---|---|---|---|
| PKCE | Required | N/A | N/A | N/A |
| Client auth | Optional (public) | Required | Required | Optional |
| User auth | Required | N/A | Via original | Required (separate) |
| Token lifetime | 15 min | 1 hour | 7 days | 15 min |
| Refresh token | Yes | No | Yes (new) | Yes |
| Single use | Code | N/A | Token | Device code |
| Reuse detection | N/A | N/A | Yes | N/A |

## GGID Grant Type Enforcement

### Configuration

```yaml
oauth:
  grants:
    authorization_code:
      enabled: true
      pkce: required
    client_credentials:
      enabled: true
    refresh_token:
      enabled: true
      rotation: required
    device_code:
      enabled: true
    implicit:
      enabled: false
    password:
      enabled: false
```

### Per-Client Grant Type Restriction

```bash
# Register client with specific grants
POST /oauth/register
{
  "client_name": "Web App",
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"]
}
```

### Validation

```go
func ValidateGrantType(client *Client, grantType string) error {
    if !contains(client.GrantTypes, grantType) {
        return ErrGrantTypeNotAllowed
    }

    switch grantType {
    case "authorization_code":
        if !client.PKCERequired {
            return ErrPKCERequired
        }
    case "implicit":
        return ErrImplicitDeprecated
    case "password":
        return ErrPasswordDeprecated
    }

    return nil
}
```

## Best Practices

1. **Always use PKCE** — Even for confidential clients
2. **Rotate refresh tokens** — One-time use with reuse detection
3. **Short token lifetimes** — 15 min for access, 7 days for refresh
4. **Restrict grants per client** — Only allow what each client needs
5. **Disable deprecated grants** — No implicit, no password
6. **Use mTLS for service-to-service** — Stronger than client_secret
7. **Audit grant usage** — Log which grant types are used
8. **Monitor for abuse** — Track unusual grant type patterns
9. **Document migration paths** — Help clients move off deprecated grants
10. **Test all grant flows** — Automated tests for each grant type