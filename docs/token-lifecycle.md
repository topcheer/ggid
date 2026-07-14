# Token Lifecycle

Complete reference for the lifecycle of OAuth 2.1 / OIDC tokens in GGID:
access token issuance, refresh token rotation with reuse detection, token
introspection (RFC 7662), token revocation (RFC 7009), and shortest-lived
token strategies.

---

## Table of Contents

- [Token Types](#token-types)
- [Access Token Lifecycle](#access-token-lifecycle)
- [Refresh Token Rotation](#refresh-token-rotation)
- [Token Reuse Detection](#token-reuse-detection)
- [Token Introspection (RFC 7662)](#token-introspection-rfc-7662)
- [Token Revocation (RFC 7009)](#token-revocation-rfc-7009)
- [Shortest-Lived Token Strategy](#shortest-lived-token-strategy)
- [Token Storage](#token-storage)
- [Monitoring](#monitoring)

---

## Token Types

| Token | Purpose | Lifetime | Contains User Data | Stored Server-Side |
|-------|---------|----------|:------------------:|:------------------:|
| Access Token | API authorization | 5-15 min | Yes (JWT claims) | No (stateless) |
| Refresh Token | Obtain new access tokens | 8-24 hours | No (opaque) | Yes (hashed) |
| ID Token | User identity (OIDC) | 5-15 min | Yes (OIDC claims) | No (stateless) |
| Authorization Code | One-time exchange | 30 sec - 5 min | No (opaque) | Yes (hashed) |
| Device Code | Device flow polling | 15-30 min | No (opaque) | Yes |

### JWT Structure

Access tokens are JWTs with three parts:

```
eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleS0xIn0  ← Header (alg, kid)
.eyJzdWIiOiJ1c2VyMTIzIiwiaXNz...         ← Payload (claims)
.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c  ← Signature
```

#### Standard Claims

| Claim | Description | Example |
|-------|-------------|---------|
| `iss` | Issuer URL | `https://iam.example.com` |
| `sub` | Subject (user ID) | `550e8400-e29b-...` |
| `aud` | Audience | `api.example.com` |
| `exp` | Expiration timestamp | `1705312800` |
| `iat` | Issued at timestamp | `1705311900` |
| `jti` | Unique token ID | `token-uuid` |
| `scope` | Granted scopes | `openid profile email` |
| `tenant_id` | Tenant context | `00000000-0000-...` |
| `amr` | Authentication methods | `["pwd", "mfa"]` |
| `cnf` | Confirmation key (DPoP) | `{"jkt":"thumbprint"}` |

---

## Access Token Lifecycle

```
Issuance          Validation              Expiration          Refresh
    │                 │                       │                  │
    ▼                 ▼                       ▼                  ▼
┌─────────┐    ┌───────────┐          ┌───────────┐     ┌───────────────┐
│ Client  │    │ Resource  │          │ Token     │     │ Client uses   │
│ requests│───►│ Server    │─────────►│ expired   │────►│ refresh_token │
│ token   │    │ validates │          │ (401)     │     │ to get new AT │
└─────────┘    │ signature │          └───────────┘     └───────┬───────┘
               │ + exp     │                                    │
               │ + scope   │                              ┌─────▼─────┐
               │ + aud     │                              │ New AT +  │
               └───────────┘                              │ New RT    │
                  │                                       └───────────┘
                  ▼
            ┌───────────┐
            │ API       │
            │ response  │
            └───────────┘
```

### Issuance

```go
func (s *OAuthService) issueAccessToken(tenantID, userID uuid.UUID, clientID string, scopes []string) (string, int, error) {
    now := time.Now()
    expiresIn := s.config.AccessTokenLifetimeSeconds // Default: 900 (15 min)

    claims := jwt.MapClaims{
        "iss":       s.config.Issuer,
        "sub":       userID.String(),
        "aud":       clientID,
        "iat":       now.Unix(),
        "exp":       now.Add(time.Duration(expiresIn) * time.Second).Unix(),
        "jti":       uuid.New().String(),
        "scope":     strings.Join(scopes, " "),
        "tenant_id": tenantID.String(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = s.config.SigningKeyID

    signed, err := token.SignedString(s.config.SigningPrivateKey)
    if err != nil {
        return "", 0, err
    }

    return signed, expiresIn, nil
}
```

### Validation (Resource Server Side)

Resource servers validate access tokens on every request:

```
1. Parse JWT header → extract `kid`
2. Fetch public key by `kid` (cached, JWKS endpoint)
3. Verify signature using public key
4. Check `exp` → reject if expired
5. Check `iss` → must match expected issuer
6. Check `aud` → must include this resource server
7. Check `scope` → must include required scope
8. Check revocation list (if introspection-based)
9. Extract `sub`, `tenant_id` for request context
```

### Lifetime Configuration

```yaml
oauth:
  token:
    access_token_lifetime: "15m"     # Recommended: 5-15 min
    id_token_lifetime: "15m"
    auth_code_lifetime: "5m"
```

| Token Type | Minimum | Recommended | Maximum |
|------------|---------|-------------|---------|
| Access token | 5 min | 15 min | 1 hour |
| ID token | 5 min | 15 min | 1 hour |
| Auth code | 30 sec | 5 min | 10 min |

---

## Refresh Token Rotation

Each use of a refresh token issues a new access token AND a new refresh token.
The old refresh token is immediately invalidated.

### Rotation Flow

```
Step 1: Initial Token Issuance
  Client receives: AT-1 + RT-A

Step 2: Access Token Expires
  Client sends: RT-A
  Server validates RT-A → issues AT-2 + RT-B
  RT-A is marked "used"

Step 3: AT-2 Expires
  Client sends: RT-B
  Server validates RT-B → issues AT-3 + RT-C
  RT-B is marked "used"

Token Family: { RT-A → RT-B → RT-C }
```

### Implementation

```go
func (s *OAuthService) RefreshToken(req *RefreshTokenRequest) (*TokenResponse, error) {
    // 1. Validate the refresh token
    stored, err := s.validateRefreshToken(req.RefreshToken)
    if err != nil {
        return nil, errors.InvalidGrant("invalid refresh token")
    }

    // 2. Check for reuse
    if stored.Used {
        // REUSE DETECTED — revoke entire family
        s.revokeTokenFamily(stored.FamilyID)
        return nil, errors.InvalidGrant("refresh token reuse detected — all tokens revoked")
    }

    // 3. Mark current token as used
    stored.Used = true
    s.refreshTokenRepo.Update(stored)

    // 4. Issue new access token
    accessToken, expiresIn, err := s.issueAccessToken(stored.TenantID, stored.UserID, stored.ClientID, stored.Scopes)
    if err != nil {
        return nil, err
    }

    // 5. Issue new refresh token (same family)
    newRefresh, err := s.issueRefreshToken(stored.TenantID, stored.UserID, stored.ClientID, stored.FamilyID, stored.Scopes)
    if err != nil {
        return nil, err
    }

    return &TokenResponse{
        AccessToken:  accessToken,
        RefreshToken: newRefresh,
        ExpiresIn:    expiresIn,
        TokenType:    "Bearer",
    }, nil
}
```

### Configuration

```yaml
oauth:
  refresh_token:
    rotation: true               # Rotate on each use
    lifetime: "24h"              # Max lifetime from issuance
    idle_timeout: "8h"           # Revoked if unused for 8h
    family_tracking: true        # Track token families for reuse detection
```

---

## Token Reuse Detection

When a used refresh token is presented again, GGID detects potential token
theft and revokes the entire token family.

### Attack Scenario

```
1. Legitimate client has: RT-A
2. Attacker steals: RT-A (copy)
3. Legitimate client refreshes: RT-A → RT-B (RT-A marked used)
4. Attacker uses stolen: RT-A
5. Server detects: RT-A is already used
6. Action: Revoke RT-A, RT-B, RT-C, ... (entire family)
7. Result: Both legitimate client and attacker lose access
8. Legitimate client must re-authenticate
```

### Family Tracking

All refresh tokens derived from the same authentication session share a
`family_id`:

```
family_id: "fam-abc-123"
  ├── RT-A (used, revoked)
  ├── RT-B (used, revoked)
  └── RT-C (active) → revoked on reuse detection
```

### Implementation

```go
func (s *OAuthService) revokeTokenFamily(familyID string) {
    tokens, _ := s.refreshTokenRepo.GetByFamilyID(familyID)
    for _, t := range tokens {
        // Revoke each token in the family
        s.refreshTokenRepo.Revoke(t.ID)

        // Also revoke access tokens by storing their jti in revocation list
        if t.AccessTokenJTI != "" {
            revokedTokens.Store(t.AccessTokenJTI, t.ExpiresAt.Unix())
        }
    }

    // Emit security event
    s.eventBus.Publish("security.token_reuse_detected", map[string]interface{}{
        "family_id": familyID,
        "revoked_count": len(tokens),
    })
}
```

### Grace Period (Optional)

For environments where network retries are common, a short grace period allows
the previous refresh token to be used once more:

```yaml
oauth:
  refresh_token:
    grace_period: "30s"    # Old token valid for 30s after rotation
```

> Grace period reduces security. Use only when false-positive reuse detection
> is a significant problem.

---

## Token Introspection (RFC 7662)

Resource servers can query the authorization server to check token validity
and metadata.

### Request

```bash
curl -X POST https://iam.example.com/oauth/introspect \
  -d "token=eyJhbG..." \
  -d "token_type_hint=access_token"
```

> The introspection endpoint requires client authentication (same as token
> endpoint).

### Active Token Response

```json
{
  "active": true,
  "scope": "openid profile email",
  "client_id": "web-app",
  "username": "jane.doe@example.com",
  "token_type": "Bearer",
  "exp": 1705312800,
  "iat": 1705311900,
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "aud": "api.example.com",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "amr": ["pwd", "mfa"]
}
```

### Inactive Token Response

```json
{
  "active": false
}
```

### When to Use Introspection

| Scenario | Use Introspection? |
|----------|:------------------:|
| JWT validation (offline) | No — verify signature locally |
| Opaque token validation | **Yes** — only way to validate |
| Check if token is revoked | **Yes** — checks revocation list |
| Get token metadata (scope, sub) | Yes (or decode JWT locally) |

### Caching

Introspection responses should be cached to reduce server load:

```yaml
oauth:
  introspection:
    cache_ttl: "30s"       # Cache active status for 30s
    cache_max_size: 10000  # Max tokens in cache
```

---

## Token Revocation (RFC 7009)

Revoke tokens before their natural expiration.

### Request

```bash
curl -X POST https://iam.example.com/oauth/revoke \
  -d "token=eyJhbG..." \
  -d "token_type_hint=access_token"
```

### Response

Always `200 OK` (RFC 7009 compliance — don't leak whether token was valid):

```
HTTP/1.1 200 OK
Content-Length: 0
```

### Revocation Behavior

| Token Type | Revocation Action |
|------------|-------------------|
| Access token (JWT) | Store JTI in revocation list until `exp` |
| Refresh token | Mark as revoked in database |
| Entire session | Revoke all tokens for session |

### Revoking All Tokens for a User

```bash
curl -X POST https://iam.example.com/api/v1/admin/users/{user_id}/revoke-all-tokens \
  -H "Authorization: Bearer <admin-jwt>"
```

This:
1. Revokes all active refresh tokens
2. Adds all active access token JTIs to revocation list
3. Revokes all active sessions
4. Emits `session.revoked` audit event

### Revocation List Storage

```go
// In-memory revocation list (for JWT access tokens)
// Key: SHA256(token), Value: expiration timestamp
var revokedTokens sync.Map

func (s *OAuthService) RevokeToken(tokenStr string, tokenTypeHint ...string) error {
    if tokenStr == "" {
        return nil // RFC 7009: always 200
    }

    tokenHash := hashTokenSHA256(tokenStr)
    claims, err := s.ParseAccessToken(tokenStr)
    if err != nil {
        // Store hash even for unparseable tokens
        revokedTokens.Store(tokenHash, int64(0))
        return nil
    }

    exp := getInt64Claim(claims, "exp")
    revokedTokens.Store(tokenHash, exp)
    return nil
}
```

### Cleanup

Expired entries are garbage-collected:

```go
func cleanupRevokedTokens() {
    now := time.Now().Unix()
    revokedTokens.Range(func(key, value any) bool {
        if value.(int64) > 0 && value.(int64) < now {
            revokedTokens.Delete(key)
        }
        return true
    })
}
```

---

## Shortest-Lived Token Strategy

### Principle

Use the shortest practical token lifetime to minimize the window of
opportunity for token theft.

### Recommended Lifetimes

```
┌──────────────────────────────────────────────────────────────────┐
│                  Token Lifetime Strategy                         │
│                                                                  │
│  ┌──────────┐  Access Token (15 min max)                        │
│  │    AT    │  ─────────────────────────                        │
│  └────┬─────┘  Short-lived, stateless, high-frequency rotation  │
│       │                                                         │
│       ▼                                                         │
│  ┌──────────┐  Refresh Token (8-24h)                           │
│  │    RT    │  ─────────────────────────                        │
│  └────┬─────┘  Rotated on each use, reuse detection enabled     │
│       │                                                         │
│       ▼                                                         │
│  ┌──────────┐  Re-authentication (session-based)               │
│  │   Auth   │  ─────────────────────────                        │
│  └──────────┘  User must re-enter credentials / biometric       │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Configuration by Security Tier

| Tier | Access Token | Refresh Token | Re-auth |
|------|-------------|---------------|---------|
| Standard | 15 min | 24 hours | 7 days |
| Sensitive | 10 min | 8 hours | 24 hours |
| High-security | 5 min | 4 hours | 8 hours |
| Banking/Finance | 5 min | 2 hours | 4 hours |

```yaml
oauth:
  token:
    access_token_lifetime: "15m"
    refresh_token_lifetime: "24h"
    reauth_interval: "168h"       # 7 days
```

### Per-Client Overrides

```yaml
oauth:
  clients:
    - client_id: "banking-app"
      token:
        access_token_lifetime: "5m"
        refresh_token_lifetime: "2h"
        reauth_interval: "4h"

    - client_id: "internal-dashboard"
      token:
        access_token_lifetime: "1h"
        refresh_token_lifetime: "72h"
```

### Step-Up Authentication

For sensitive operations, require a fresh authentication regardless of token
validity:

```bash
# Step-up: requires recent MFA (within 5 minutes)
curl -X POST https://api.example.com/transfer \
  -H "Authorization: Bearer <access-token>" \
  -H "X-Require-Step-Up: mfa" \
  -d '{ "amount": 10000, "to": "account-xxx" }'
```

If the token's `amr` claim doesn't include recent MFA, the API returns:

```json
{
  "error": "insufficient_authentication",
  "required_amr": ["mfa"],
  "max_age": 300
}
```

The client then triggers an MFA prompt and obtains a new token with `amr: ["mfa"]`.

---

## Token Storage

### Client-Side Storage

| Client Type | Access Token | Refresh Token | ID Token |
|-------------|-------------|---------------|----------|
| Web app (server) | Server session | DB / encrypted | Server session |
| SPA | Memory (never localStorage) | HttpOnly cookie | Memory |
| Mobile | Secure Enclave / Keychain | Secure storage | Secure storage |
| Desktop | Keychain (macOS) / DPAPI (Windows) | Credential Manager | Keychain |

### Server-Side Storage

| Token Type | Storage | Hashed |
|------------|---------|:------:|
| Refresh token | PostgreSQL | Yes (SHA-256) |
| Authorization code | PostgreSQL | Yes (SHA-256) |
| Revocation list | In-memory (`sync.Map`) | Yes (SHA-256) |

```sql
CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    client_id       VARCHAR(128) NOT NULL,
    token_hash      VARCHAR(64) NOT NULL,      -- SHA-256 of token
    family_id       UUID NOT NULL,              -- For reuse detection
    scopes          TEXT[],
    used            BOOLEAN DEFAULT FALSE,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ,
    UNIQUE(token_hash)
);

CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
```

---

## Monitoring

### Token Metrics

```
# Token issuance rate
ggid_oauth_tokens_issued_total{grant_type="authorization_code"} 1523
ggid_oauth_tokens_issued_total{grant_type="client_credentials"} 4521
ggid_oauth_tokens_issued_total{grant_type="refresh_token"} 8932

# Refresh token rotation
ggid_oauth_refresh_token_rotations_total 8234
ggid_oauth_refresh_token_reuse_detected_total 3

# Active tokens
ggid_oauth_active_access_tokens 4523
ggid_oauth_active_refresh_tokens 2104

# Revocations
ggid_oauth_tokens_revoked_total{source="user"} 89
ggid_oauth_tokens_revoked_total{source="admin"} 12
ggid_oauth_tokens_revoked_total{source="reuse_detection"} 3

# Introspection calls
ggid_oauth_introspection_total 15234
ggid_oauth_introspection_active 14890
ggid_oauth_introspection_inactive 344
```

### Alerting

```yaml
groups:
  - name: token-security
    rules:
      - alert: RefreshTokenReuseDetected
        expr: rate(ggid_oauth_refresh_token_reuse_detected_total[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Refresh token reuse detected — possible token theft"

      - alert: HighTokenRevocationRate
        expr: rate(ggid_oauth_tokens_revoked_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Unusual token revocation rate"

```
