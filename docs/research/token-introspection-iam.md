# Token Introspection and Resource Binding for IAM Systems

> **Project**: GGID IAM Suite
> **Scope**: RFC 7662 (Token Introspection), RFC 8707 (Resource Indicators), RFC 7009 (Token Revocation), audience binding, introspection caching, distributed token validation
> **Date**: 2025-01-20
> **Author**: Security Research Team

---

## Table of Contents

1. [RFC 7662 Token Introspection](#1-rfc-7662-token-introspection)
2. [RFC 8707 Resource Indicators](#2-rfc-8707-resource-indicators)
3. [Token Audience Binding](#3-token-audience-binding)
4. [Introspection Caching](#4-introspection-caching)
5. [Distributed Introspection](#5-distributed-introspection)
6. [Token Revocation via Introspection](#6-token-revocation-via-introspection)
7. [Security of Introspection Endpoint](#7-security-of-introspection-endpoint)
8. [JWT vs Opaque Token Decision](#8-jwt-vs-opaque-token-decision)
9. [GGID Introspection Gap Analysis](#9-ggid-introspection-gap-analysis)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. RFC 7662 Token Introspection

### 1.1 Overview

RFC 7662 defines a standardized endpoint that allows resource servers to query the
authorization server about the validity and metadata of an access token. This is
essential when tokens are **opaque** (random strings with no embedded meaning) —
the resource server has no way to validate them locally and must ask the AS.

The introspection endpoint is `POST /oauth/introspect`.

### 1.2 Request Format

```
POST /oauth/introspect HTTP/1.1
Host: auth.ggid.dev
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(resource_server_id:secret)

token=eyJhbGciOiJSUzI1NiIs...&token_type_hint=access_token
```

| Parameter         | Required | Description                                                         |
|-------------------|----------|---------------------------------------------------------------------|
| `token`           | Yes      | The token string to introspect                                     |
| `token_type_hint` | No       | Hint: `access_token` or `refresh_token` (optimization, AS may ignore)|

The resource server authenticates using its own client credentials (Basic auth,
Bearer token, or mTLS). This prevents arbitrary callers from scanning tokens.

### 1.3 Response Format

```json
{
  "active": true,
  "scope": "read:users write:users",
  "client_id": "frontend-console",
  "username": "admin@ggid.dev",
  "token_type": "Bearer",
  "exp": 1737504000,
  "iat": 1737500400,
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "aud": "https://api.ggid.dev",
  "iss": "https://auth.ggid.dev"
}
```

When the token is invalid, expired, or revoked:

```json
{
  "active": false
}
```

**Critical rule**: The response MUST always contain the `active` field. All other
fields are optional. For inactive tokens, the server SHOULD NOT include any
metadata to prevent information leakage.

### 1.4 Why Resource Servers Need Introspection

When the authorization server issues **opaque tokens** (e.g., `abcdef123456`),
the resource server cannot extract claims, verify signatures, or check expiry
locally. The only way to determine if the token is valid is to ask the AS:

```
Resource Server ──POST /introspect──> Authorization Server
                   "Is this token valid?"
Authorization Server ──{active:true, scope:...}──> Resource Server
```

For JWT tokens, introspection is still valuable for:
- **Revocation checking**: JWTs are stateless; the AS may have revoked them
- **Scope/claim enrichment**: AS may have additional server-side metadata
- **Audience validation**: Confirm the token is for *this* resource server

### 1.5 Go: Introspection Server Handler

```go
// IntrospectionResponse is the RFC 7662 token introspection response.
type IntrospectionResponse struct {
    Active    bool   `json:"active"`
    Scope     string `json:"scope,omitempty"`
    ClientID  string `json:"client_id,omitempty"`
    Username  string `json:"username,omitempty"`
    TokenType string `json:"token_type,omitempty"`
    Exp       int64  `json:"exp,omitempty"`
    Iat       int64  `json:"iat,omitempty"`
    Sub       string `json:"sub,omitempty"`
    Aud       string `json:"aud,omitempty"`
    Iss       string `json:"iss,omitempty"`
}

// IntrospectToken validates a token and returns introspection data.
func (s *OAuthService) IntrospectToken(tokenStr string) *IntrospectionResponse {
    // Step 1: Check revocation list
    if s.IsTokenRevoked(tokenStr) {
        return &IntrospectionResponse{Active: false}
    }

    // Step 2: Parse and validate the JWT
    claims, err := s.ParseAccessToken(tokenStr)
    if err != nil {
        return &IntrospectionResponse{Active: false}
    }

    // Step 3: Build the response with all claims
    resp := &IntrospectionResponse{
        Active:    true,
        TokenType: "Bearer",
        Sub:       getStringClaim(claims, "sub"),
        Aud:       getStringClaim(claims, "aud"),
        Iss:       getStringClaim(claims, "iss"),
        ClientID:  getStringClaim(claims, "aud"),
        Username:  getStringClaim(claims, "preferred_username"),
        Exp:       getInt64Claim(claims, "exp"),
        Iat:       getInt64Claim(claims, "iat"),
    }

    if scope, ok := claims["scope"].(string); ok {
        resp.Scope = scope
    }

    return resp
}

// HTTP handler for POST /oauth/introspect
func introspectHandler(svc *OAuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
            return
        }
        _ = r.ParseForm()

        token := r.FormValue("token")
        if token == "" {
            writeJSON(w, http.StatusOK, map[string]bool{"active": false})
            return
        }

        result := svc.IntrospectToken(token)
        writeJSON(w, http.StatusOK, result)
    }
}
```

### 1.6 Go: Introspection Client (Resource Server Side)

```go
type IntrospectionClient struct {
    endpoint   string
    httpClient *http.Client
    clientID   string
    clientSecret string
}

func NewIntrospectionClient(endpoint, clientID, clientSecret string) *IntrospectionClient {
    return &IntrospectionClient{
        endpoint:     endpoint,
        httpClient:   &http.Client{Timeout: 5 * time.Second},
        clientID:     clientID,
        clientSecret: clientSecret,
    }
}

func (c *IntrospectionClient) Introspect(ctx context.Context, token string) (*IntrospectionResponse, error) {
    form := url.Values{}
    form.Set("token", token)
    form.Set("token_type_hint", "access_token")

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint,
        strings.NewReader(form.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.SetBasicAuth(c.clientID, c.clientSecret)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("introspection failed: HTTP %d", resp.StatusCode)
    }

    var result IntrospectionResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result, nil
}
```

---

## 2. RFC 8707 Resource Indicators

### 2.1 Overview

RFC 8707 introduces the `resource` parameter in authorization requests. Instead
of the client asking for broad scopes that span multiple APIs, it specifies
**exactly which resource server (API)** it wants to access. The authorization
server then binds the issued token to that resource via the `aud` claim.

### 2.2 How It Works

**Authorization request:**
```
GET /oauth/authorize?
    response_type=code
    &client_id=console-app
    &redirect_uri=https://app.ggid.dev/callback
    &scope=read:users
    &resource=https://api.ggid.dev/users
    &state=random123
```

**Token response (JWT claims):**
```json
{
  "iss": "https://auth.ggid.dev",
  "sub": "user-uuid",
  "aud": "https://api.ggid.dev/users",
  "scope": "read:users",
  "exp": 1737504000
}
```

The token is **audience-bound** — it can only be used against the Users API. If
an attacker steals it and tries to access the Audit API, the Audit API rejects
it because its URL is not in `aud`.

### 2.3 Why This Prevents Token Replay Across APIs

Without resource indicators, a token with `scope=read:users` might be valid for
any API that accepts that scope. If the token is stolen, the attacker can use it
against every API.

With resource indicators:
- Token A → `aud: "https://api.ggid.dev/users"` → Only valid for Users API
- Token B → `aud: "https://api.ggid.dev/audit"` → Only valid for Audit API
- Stolen Token A → Users API accepts, Audit API **rejects** (aud mismatch)

This limits the blast radius of token theft to a single API.

### 2.4 Multiple Resource Parameters

RFC 8707 allows multiple `resource` parameters, but this issues a token with
multiple audiences, broadening the replay surface. **Best practice**: request
one resource per token, use separate tokens for different APIs.

### 2.5 Go: Resource Parameter Validation in Token Issuance

```go
// AuthorizeRequest holds parameters for the authorization endpoint.
type AuthorizeRequest struct {
    ClientID      string
    RedirectURI   string
    ResponseType  string
    Scope         string
    State         string
    Nonce         string
    Resource      []string // RFC 8707: resource indicators
    CodeChallenge string
}

// ValidateResourceIndicators checks that the requested resource is in the
// client's allowed list. Each resource server registers its identifier (URL)
// with the authorization server.
func ValidateResourceIndicators(req *AuthorizeRequest, allowedResources []string) error {
    if len(req.Resource) == 0 {
        return nil // resource parameter is optional
    }

    // Build a lookup set for O(1) membership testing
    allowed := make(map[string]bool, len(allowedResources))
    for _, r := range allowedResources {
        allowed[r] = true
    }

    for _, res := range req.Resource {
        // Normalize: ensure it's a valid absolute URI per RFC 8707 §2.1
        u, err := url.Parse(res)
        if err != nil || !u.IsAbs() {
            return fmt.Errorf("invalid resource URI: %s", res)
        }
        if !allowed[res] {
            return fmt.Errorf("resource not registered for this client: %s", res)
        }
    }

    // SECURITY: warn if multiple resources requested (broadens replay surface)
    if len(req.Resource) > 1 {
        log.Printf("[WARN] client %s requested %d resources in single token",
            req.ClientID, len(req.Resource))
    }

    return nil
}

// issueResourceBoundAccessToken creates a token with the resource indicator
// as the audience claim.
func (s *OAuthService) issueResourceBoundAccessToken(
    userID, tenantID uuid.UUID,
    resource string, // from RFC 8707 resource parameter
    scope string,
) (string, error) {
    now := time.Now()
    expiresAt := now.Add(15 * time.Minute)

    claims := jwt.MapClaims{
        "iss":       s.issuer,
        "sub":       userID.String(),
        "aud":       resource, // bound to specific resource server
        "iat":       now.Unix(),
        "exp":       expiresAt.Unix(),
        "jti":       uuid.New().String(),
        "tenant_id": tenantID.String(),
        "scope":     scope,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = s.keyProvider.KeyID()

    return token.SignedString(s.keyProvider.PrivateKey())
}
```

---

## 3. Token Audience Binding

### 3.1 The `aud` Claim

The `aud` (audience) claim in a JWT identifies the intended recipient. A token
issued with `aud: "https://api.ggid.dev/users"` is only valid for that resource
server. Any other resource server must reject it.

### 3.2 How Resource Servers Validate

Each resource server knows its own identifier (typically its public URL):

```go
const myResourceIdentifier = "https://api.ggid.dev/users"
```

On every request, the resource server extracts the token, parses the JWT, and
checks whether `myResourceIdentifier` appears in the `aud` claim. If not, the
request is rejected with 401.

### 3.3 Multiple Audiences: A Risk Assessment

| Scenario               | `aud` Value                                        | Risk Level | Notes                                         |
|------------------------|----------------------------------------------------|------------|-----------------------------------------------|
| Single audience        | `["https://api.ggid.dev/users"]`                   | Low        | Token bound to one API                        |
| Two audiences          | `["https://api.ggid.dev/users", "https://api.ggid.dev/audit"]` | Medium     | Stolen token valid for both APIs              |
| Wildcard/broad audience| `["ggid"]`                                         | High       | Any GGID service accepts — maximum replay risk |

**Recommendation**: Use single-audience tokens whenever possible. If the client
needs multiple APIs, issue separate tokens via the `resource` parameter.

### 3.4 Go: Audience Validation Middleware

```go
// AudienceValidator creates middleware that rejects tokens not intended for
// the specified resource server identifier.
func AudienceValidator(expectedAud string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract token claims from context (set by JWTAuth middleware)
            claims, ok := r.Context().Value(claimsKey).(jwt.MapClaims)
            if !ok {
                writeUnauthorized(w, "missing token claims")
                return
            }

            // Extract audience — can be string or []string per JWT spec
            var audiences []string
            switch aud := claims["aud"].(type) {
            case string:
                audiences = []string{aud}
            case []any:
                for _, a := range aud {
                    if s, ok := a.(string); ok {
                        audiences = append(audiences, s)
                    }
                }
            default:
                writeUnauthorized(w, "missing audience claim")
                return
            }

            // Check if our identifier is in the audience list
            found := false
            for _, a := range audiences {
                if a == expectedAud {
                    found = true
                    break
                }
            }

            if !found {
                writeUnauthorized(w, "token audience mismatch: token not intended for this resource server")
                return
            }

            // SECURITY: warn if multiple audiences (broadened replay surface)
            if len(audiences) > 1 {
                log.Printf("[WARN] token has %d audiences, replay risk elevated", len(audiences))
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 4. Introspection Caching

### 4.1 The Performance Problem

Every API request through a resource server requires token validation. If the
token is opaque, each request triggers an HTTP call to the introspection endpoint:

```
Client → Resource Server → (HTTP call) → Authorization Server → (DB lookup) → Response
```

A network round-trip of 5-10ms per request is unacceptable at scale. At 10,000
RPS, that's 50-100 seconds of cumulative introspection latency per second.

### 4.2 Caching Strategy

| Cache Type           | Key           | Value                  | TTL                         |
|----------------------|---------------|------------------------|-----------------------------|
| Positive cache       | SHA256(token) | IntrospectionResponse  | `exp - now` (token expiry)  |
| Negative cache       | SHA256(token) | `{active: false}`      | 10-30 seconds (short)       |
| Revocation cache     | SHA256(token) | `revoked: true`        | `exp - now`                 |

**Positive caching**: Once a token is validated, cache the result until the
token expires. No need to re-introspect within the token's lifetime unless it's
revoked.

**Negative caching**: Cache `active: false` for a short TTL (10-30s). This
prevents a flood of invalid-token checks from overwhelming the introspection
endpoint.

### 4.3 Cache Invalidation on Revocation

When a token is revoked, all cached positive results become stale. The
authorization server must propagate revocation events to flush caches across
all resource servers. See Section 6 for the NATS event-driven approach.

### 4.4 Go: Cached Introspection Client

```go
type cachedIntrospectionClient struct {
    client    *IntrospectionClient
    cache     *sync.Map // tokenHash → cachedResult
    negCacheTTL time.Duration
}

type cachedResult struct {
    response  *IntrospectionResponse
    expiresAt time.Time
    negative  bool
}

func newCachedIntrospectionClient(client *IntrospectionClient, negTTL time.Duration) *cachedIntrospectionClient {
    c := &cachedIntrospectionClient{
        client:      client,
        cache:       &sync.Map{},
        negCacheTTL: negTTL,
    }
    // Start background cleanup goroutine
    go c.cleanupLoop()
    return c
}

func (c *cachedIntrospectionClient) Introspect(ctx context.Context, token string) (*IntrospectionResponse, error) {
    hash := sha256Token(token)

    // Step 1: Check cache
    if val, ok := c.cache.Load(hash); ok {
        cached := val.(*cachedResult)
        if time.Now().Before(cached.expiresAt) {
            return cached.response, nil
        }
        c.cache.Delete(hash) // expired
    }

    // Step 2: Call the introspection endpoint
    result, err := c.client.Introspect(ctx, token)
    if err != nil {
        return nil, err
    }

    // Step 3: Cache the result
    var ttl time.Duration
    if result.Active && result.Exp > 0 {
        // Positive: cache until token expiry
        ttl = time.Until(time.Unix(result.Exp, 0))
        if ttl < 0 {
            ttl = 0 // already expired
        }
    } else {
        // Negative: short TTL to allow retry
        ttl = c.negCacheTTL
    }

    c.cache.Store(hash, &cachedResult{
        response:  result,
        expiresAt: time.Now().Add(ttl),
        negative:  !result.Active,
    })

    return result, nil
}

// Invalidate removes a token from the cache. Called when a revocation event
// is received via NATS.
func (c *cachedIntrospectionClient) Invalidate(tokenHash string) {
    c.cache.Delete(tokenHash)
}

func (c *cachedIntrospectionClient) cleanupLoop() {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        now := time.Now()
        c.cache.Range(func(key, val any) bool {
            cached := val.(*cachedResult)
            if now.After(cached.expiresAt) {
                c.cache.Delete(key)
            }
            return true
        })
    }
}

func sha256Token(token string) string {
    h := sha256.Sum256([]byte(token))
    return hex.EncodeToString(h[:])
}
```

---

## 5. Distributed Introspection

### 5.1 The Network Zone Problem

In enterprise deployments, resource servers may span multiple network zones:
- Internal services (Identity, Policy, Org) — same zone as AS
- External-facing API gateways — DMZ
- Third-party integrators — completely external network

For same-zone services, direct JWT validation (0.1ms) or local introspection is
sufficient. For cross-zone, a dedicated introspection service is needed.

### 5.2 Architecture Options

**Option A: Direct DB Lookup** (in-zone only)
```
Resource Server → Shared Token Store (Redis/DB)
```
- Pro: Zero network hops for in-zone services
- Con: Exposes DB credentials to every resource server; security risk

**Option B: Dedicated Introspection Service** (cross-zone)
```
Resource Server → HTTPS → Introspection Service → Token Store
```
- Pro: Centralized control, single audit point
- Con: Network hop (5-10ms), single point of failure

**Option C: Federated Introspection** (multi-region)
```
Resource Server → Regional Introspection Cache → Central AS (cache miss)
```
- Pro: Low latency for cache hits, consistent for cache misses
- Con: Eventual consistency on revocation

### 5.3 Performance Comparison

| Method                      | Latency    | Network Hops | Revocation Speed   |
|-----------------------------|------------|--------------|--------------------|
| JWT local validation        | ~0.1ms     | 0            | Slow (next refresh)|
| Opaque + introspection      | 5-10ms     | 1            | Immediate          |
| Opaque + cached introspection | ~0.05ms (hit) | 0 (hit) / 1 (miss) | Near-immediate (cache flush) |

### 5.4 Decision Matrix: When to Use What

| Requirement                       | Recommendation                          |
|-----------------------------------|-----------------------------------------|
| Stateless validation, no revocation urgency | JWT with short TTL (5-15 min)   |
| Immediate revocation needed       | Opaque + introspection (or JWT + revocation list) |
| High throughput, low latency      | JWT with JWKS caching                    |
| Confidential client, server-side sessions | Opaque + introspection           |
| Cross-domain / third-party APIs   | Opaque + introspection with auth        |
| Microservices, same trust domain  | JWT with shared JWKS                    |
| Mobile/SPA clients                | JWT (short TTL) + refresh token         |

---

## 6. Token Revocation via Introspection

### 6.1 RFC 7009 Token Revocation

The revocation endpoint (`POST /oauth/revoke`) allows clients to invalidate
tokens before their natural expiry (e.g., on logout).

```
POST /oauth/revoke HTTP/1.1
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(client_id:secret)

token=eyJhbG...&token_type_hint=access_token
```

The server always responds `200 OK` regardless of whether the token existed
(per RFC 7009 §2.2 — prevents information leakage).

### 6.2 The Revocation Propagation Problem

Once a token is revoked at the AS, all resource servers with cached introspection
results still consider it valid. The challenge: how to flush caches across all
resource servers quickly?

### 6.3 NATS Event-Driven Cache Invalidation

```
Client → AS /revoke
AS marks token revoked in store
AS publishes NATS event: {subject: "token.revoked", token_hash: "abc123..."}
Resource Server 1 ← receives event → invalidate cache["abc123..."]
Resource Server 2 ← receives event → invalidate cache["abc123..."]
Resource Server 3 ← receives event → invalidate cache["abc123..."]
```

All caches are flushed within milliseconds (NATS pub/sub latency ~1ms).

### 6.4 Go: Revocation + NATS Cache Invalidation Flow

**Authorization Server (revocation + NATS publish):**

```go
type RevocationService struct {
    revokedTokens *sync.Map     // tokenHash → exp
    natsConn      *nats.Conn
    subject       string        // e.g. "token.revoked"
}

func (rs *RevocationService) Revoke(ctx context.Context, tokenStr string) error {
    if tokenStr == "" {
        return nil // RFC 7009: return 200 for empty token
    }

    tokenHash := sha256Token(tokenStr)

    // Step 1: Parse token to get expiry (for TTL of revocation entry)
    var exp int64
    if claims, err := parseToken(tokenStr); err == nil {
        exp = getInt64Claim(claims, "exp")
    }

    // Step 2: Store in revocation list
    rs.revokedTokens.Store(tokenHash, exp)

    // Step 3: Publish cache invalidation event to all resource servers
    event := RevocationEvent{
        TokenHash: tokenHash,
        Exp:       exp,
        Timestamp: time.Now().Unix(),
    }
    data, _ := json.Marshal(event)

    if rs.natsConn != nil {
        if err := rs.natsConn.Publish(rs.subject, data); err != nil {
            log.Printf("[WARN] failed to publish revocation event: %v", err)
            // Non-fatal: resource servers will pick up revocation on next introspection
        }
    }

    return nil
}

type RevocationEvent struct {
    TokenHash string `json:"token_hash"`
    Exp       int64  `json:"exp"`
    Timestamp int64  `json:"timestamp"`
}
```

**Resource Server (NATS subscriber + cache flush):**

```go
func (c *cachedIntrospectionClient) SubscribeToRevocations(nc *nats.Conn, subject string) error {
    _, err := nc.Subscribe(subject, func(msg *nats.Msg) {
        var event RevocationEvent
        if err := json.Unmarshal(msg.Data, &event); err != nil {
            log.Printf("[ERROR] failed to unmarshal revocation event: %v", err)
            return
        }
        // Flush this token from the cache
        c.cache.Delete(event.TokenHash)
        log.Printf("[INFO] invalidated cached introspection for token %s", event.TokenHash[:16])
    })
    return err
}
```

### 6.5 Revocation List Cleanup

Revocation entries should be cleaned up after the token's natural expiry. A
background goroutine periodically scans and removes entries where `exp < now`:

```go
func (rs *RevocationService) cleanupLoop() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for range ticker.C {
        now := time.Now().Unix()
        rs.revokedTokens.Range(func(key, val any) bool {
            exp, _ := val.(int64)
            if exp > 0 && now > exp {
                rs.revokedTokens.Delete(key)
            }
            return true
        })
    }
}
```

---

## 7. Security of Introspection Endpoint

### 7.1 Why Introspection Needs Authentication

The introspection endpoint reveals token metadata (user identity, scopes,
client ID, expiry). If unauthenticated, an attacker can:

1. **Token scanning**: Submit stolen tokens to learn which are valid
2. **User enumeration**: Extract `username` and `sub` from valid tokens
3. **Scope discovery**: Learn what scopes are assigned to tokens
4. **Timing analysis**: Compare response times to distinguish valid/invalid tokens

### 7.2 Authentication Methods

| Method              | Implementation                          | Best For                          |
|---------------------|-----------------------------------------|-----------------------------------|
| Client credentials  | Basic auth with resource server's ID/secret | Known resource servers      |
| mTLS                | Client certificate for resource server  | High-security enterprise           |
| Bearer token        | Pre-provisioned introspection token     | Legacy compatibility               |

### 7.3 Rate Limiting Introspection Calls

Even with authentication, rate limit to prevent abuse:
- Per-resource-server limit: 10,000 req/min (normal traffic)
- Burst protection: 500 req/sec
- Alert threshold: >50% of limit sustained for 5 minutes

### 7.4 Go: Secured Introspection Endpoint

```go
// ResourceServerCredentials maps resource server IDs to their secrets.
type ResourceServerCredentials struct {
    clients map[string]string // clientID → clientSecret
    mu      sync.RWMutex
}

func (rsc *ResourceServerCredentials) Authenticate(r *http.Request) bool {
    clientID, clientSecret, ok := r.BasicAuth()
    if !ok {
        return false
    }
    rsc.mu.RLock()
    expected, exists := rsc.clients[clientID]
    rsc.mu.RUnlock()
    if !exists {
        return false
    }
    return subtle.ConstantTimeCompare([]byte(clientSecret), []byte(expected)) == 1
}

// securedIntrospectionHandler wraps the introspection handler with
// authentication and rate limiting.
func securedIntrospectionHandler(
    svc *OAuthService,
    creds *ResourceServerCredentials,
    limiter *rate.Limiter,
) http.HandlerFunc {
    inner := introspectHandler(svc)

    return func(w http.ResponseWriter, r *http.Request) {
        // Step 1: Authenticate the resource server
        if !creds.Authenticate(r) {
            // IMPORTANT: return minimal info to prevent scanning
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte(`{"active":false}`))
            return
        }

        // Step 2: Rate limit
        if !limiter.Allow() {
            http.Error(w, `{"error":"rate_limited"}`, http.StatusTooManyRequests)
            return
        }

        // Step 3: Process introspection
        inner.ServeHTTP(w, r)
    }
}
```

### 7.5 Information Leakage Prevention

For unauthenticated callers, always return `{"active": false}` with `401` —
never reveal whether the token itself was valid. For authenticated callers
with invalid tokens, also return `{"active": false}` with `200` per RFC 7662.
Never include error details or token metadata for inactive tokens.

---

## 8. JWT vs Opaque Token Decision

### 8.1 Comparison Table

| Dimension              | JWT                           | Opaque + Introspection        |
|------------------------|-------------------------------|-------------------------------|
| **Validation**         | Local (verify signature)      | Remote (HTTP call to AS)      |
| **Latency**            | ~0.1ms                        | 5-10ms (cache: ~0.05ms)       |
| **State**              | Stateless                     | Stateful (AS maintains store) |
| **Revocation**         | Slow (wait for expiry)        | Immediate (AS marks revoked)  |
| **Network dependency** | None (after JWKS fetch)       | Every request (or cache hit)  |
| **Token size**         | Large (1-2KB)                 | Small (32-64 bytes)           |
| **Claim transparency** | Client can read claims        | Claims hidden from client     |
| **Audience binding**   | Built-in (`aud` claim)        | Via introspection response    |
| **Key rotation**       | JWKS rotation, gradual        | Transparent (AS internal)     |
| **Infrastructure**     | JWKS endpoint                 | Token store + introspection   |
| **Best for**           | Distributed, high-throughput  | Centralized control, revocation|

### 8.2 When to Use JWT

- **Stateless microservices** in the same trust domain sharing JWKS
- **High-throughput APIs** where every millisecond matters
- **Distributed deployments** where AS may not be reachable
- **Short-lived tokens** (5-15 min) where revocation latency is acceptable
- **Open APIs** where third-party clients need to validate tokens

### 8.3 When to Use Opaque + Introspection

- **Immediate revocation** required (e.g., compliance, security incident)
- **Confidential clients** where the AS controls the entire token lifecycle
- **Regulated industries** requiring centralized audit of token usage
- **Token stuffing prevention** — opaque tokens carry no exploitable data
- **Cross-domain resource servers** where JWT key distribution is impractical

### 8.4 Hybrid Approach

Many production IAM systems use a hybrid model:

```
┌──────────────┐
│  Auth Server │
│  issues JWT  │
│  (15 min TTL)│
└──────┬───────┘
       │
       ▼
┌──────────────┐     JWT (short-lived)     ┌──────────────┐
│   Client     │ ────────────────────────> │  API Gateway  │
│              │                            │  (validates   │
│  refresh     │ <─── new JWT ───────────── │  JWT locally) │
│  token:      │                            └──────────────┘
│  OPAQUE      │
└──────────────┘
       │
       │ POST /token (grant_type=refresh_token)
       ▼
┌──────────────┐
│  Auth Server  │
│  introspects  │
│  opaque RT    │
│  → issues JWT │
└──────────────┘
```

- **Access tokens**: JWT, short-lived (15 min), validated locally by API gateway
- **Refresh tokens**: Opaque, long-lived (24h), validated only by AS via introspection
- **Revocation**: Revoke refresh token at AS → no new access tokens can be minted

This gives the best of both worlds: fast access token validation + immediate
revocation via refresh token invalidation.

---

## 9. GGID Introspection Gap Analysis

### 9.1 What Exists in GGID

| Feature                              | Status      | File / Location                                                    |
|--------------------------------------|-------------|-------------------------------------------------------------------|
| Introspection endpoint               | **EXISTS**  | `services/oauth/internal/server/server.go:545` (`/oauth/introspect`) |
| Introspection alias                  | **EXISTS**  | `services/oauth/internal/server/server.go:561` (`/api/v1/oauth/introspect`) |
| IntrospectionResponse struct         | **EXISTS**  | `services/oauth/internal/service/oauth_service.go:539`             |
| IntrospectToken method               | **EXISTS**  | `services/oauth/internal/service/oauth_service.go:553`             |
| Introspection endpoint in discovery  | **EXISTS**  | `services/oauth/internal/service/oauth_service.go:373`             |
| Token revocation endpoint            | **EXISTS**  | `services/oauth/internal/server/server.go:470` (`/oauth/revoke`)   |
| RevokeToken + IsTokenRevoked         | **EXISTS**  | `services/oauth/internal/service/oauth_service.go:696,720`        |
| Gateway JWT audience validation      | **PARTIAL** | `services/gateway/internal/middleware/middleware.go:532` (single static audience "ggid") |
| `aud` claim in access tokens         | **EXISTS**  | `services/oauth/internal/service/oauth_service.go:426`             |
| Discovery metadata (introspection_endpoint) | **EXISTS** | `services/oauth/internal/service/oauth_service.go:373` |

### 9.2 What Is Missing

| Gap                                   | Severity | Impact                                                                   |
|---------------------------------------|----------|--------------------------------------------------------------------------|
| **No authentication on introspection**| **P0**   | Anyone can scan tokens, extract user identity, enumerate valid tokens    |
| **No resource indicator (RFC 8707)**  | **P1**   | Tokens not audience-bound per resource; replay across APIs possible       |
| **Single static audience "ggid"**     | **P1**   | All tokens share one audience; no per-API binding; broad replay surface   |
| **In-memory revocation (sync.Map)**   | **P1**   | Revocation not propagated across OAuth service instances or resource servers |
| **No NATS revocation propagation**    | **P1**   | Revoked tokens remain valid on other instances until expiry               |
| **No introspection caching**          | **P2**   | Every introspection call hits the service (no caching layer)              |
| **No negative caching**               | **P2**   | Invalid token floods can overwhelm the introspection endpoint             |
| **No opaque token support**           | **P2**   | Introspection only works for JWTs (local parse); opaque tokens not stored |
| **Gateway: no introspection fallback**| **P2**   | Gateway rejects non-JWT tokens; cannot validate opaque tokens             |
| **No rate limiting on introspection** | **P2**   | Introspection endpoint is unauthenticated and unrate-limited (DoS risk)   |
| **`client_id` mapped to `aud`**       | **P3**** | IntrospectToken sets `ClientID = claims["aud"]` — should be separate     |

### 9.3 Detailed Findings

**Finding 1: Unauthenticated Introspection Endpoint (P0)**

The introspection handler at `server.go:545` does not check any credentials:

```go
mux.HandleFunc("/oauth/introspect", func(w http.ResponseWriter, r *http.Request) {
    // ... no auth check ...
    result := oauthSvc.IntrospectToken(token)
    writeJSON(w, http.StatusOK, result)
})
```

An attacker can POST any token and receive full introspection data including
`sub`, `username`, `scope`, and `client_id`. This enables token scanning and
user enumeration.

**Finding 2: Single Static Audience**

The gateway config defaults `JWTAudience` to `"ggid"`:
```go
// services/gateway/internal/config/config.go:45
JWTAudience: "ggid",
```

All access tokens are validated against this single audience. There is no
per-route or per-resource-server audience. A token valid for the Users API is
also valid for the Audit API, Policy API, etc.

**Finding 3: In-Memory Revocation Not Distributed**

```go
// services/oauth/internal/service/oauth_service.go:640
var revokedTokens sync.Map
```

This `sync.Map` is process-local. If GGID runs multiple OAuth service instances
behind a load balancer, a token revoked on instance A remains valid on instance
B. There is no Redis/NATS propagation of revocation events.

**Finding 4: No `resource` Parameter Support**

The authorize handler at `server.go:163-170` extracts `client_id`, `redirect_uri`,
`response_type`, `state`, `scope`, `nonce`, `code_challenge` — but **no `resource`
parameter**. RFC 8707 resource indicators are completely unsupported.

**Finding 5: Introspection ClientID Mislabeling**

```go
// services/oauth/internal/service/oauth_service.go:568
ClientID: getStringClaim(claims, "aud"),
```

The introspection response maps `client_id` from the `aud` claim, which is
semantically incorrect. `aud` is the resource server identifier, while
`client_id` should be the OAuth client that requested the token. These are
different concepts.

---

## 10. Gap Analysis & Recommendations

### 10.1 Action Items

| # | Action Item                                          | Priority | Effort  | Files to Modify                                          |
|---|------------------------------------------------------|----------|---------|----------------------------------------------------------|
| 1 | **Authenticate the introspection endpoint**          | P0       | 2-3 days| `services/oauth/internal/server/server.go` — add Basic auth check with resource server credentials |
| 2 | **Distribute revocation via Redis/NATS**             | P1       | 3-5 days| `services/oauth/internal/service/oauth_service.go` — replace `sync.Map` with Redis-backed store; publish NATS events on revocation |
| 3 | **Implement RFC 8707 resource indicators**            | P1       | 3-5 days| `services/oauth/internal/server/server.go` — parse `resource` param in authorize; `oauth_service.go` — bind token `aud` to resource |
| 4 | **Per-route audience validation in gateway**         | P1       | 2-3 days| `services/gateway/internal/config/config.go` — add per-route `audience` field; `middleware.go` — validate per-route audience |
| 5 | **Add introspection caching in gateway**             | P2       | 2-3 days| New file `services/gateway/internal/middleware/introspection_cache.go` — cache positive/negative results with TTL |

### 10.2 Implementation Priority

**Phase 1 (Immediate — Week 1):**
- Action 1: Secure the introspection endpoint with client credentials
- Add rate limiting (1,000 req/min per resource server)

**Phase 2 (Short-term — Weeks 2-3):**
- Action 2: Replace `sync.Map` with Redis for distributed revocation
- Publish NATS `token.revoked` events
- Action 5: Add introspection result caching in gateway (positive + negative)

**Phase 3 (Medium-term — Weeks 4-6):**
- Action 3: Add `resource` parameter parsing in authorize endpoint
- Bind token audience to the requested resource
- Action 4: Per-route audience config in gateway

### 10.3 Risk Assessment if Not Fixed

| Gap                                    | Attack Scenario                                              | Impact              |
|----------------------------------------|-------------------------------------------------------------|---------------------|
| Unauthenticated introspection          | Attacker scans stolen tokens to find valid ones              | Token theft, account takeover |
| Non-distributed revocation             | Revoked token used on different instance                     | Unauthorized access after logout |
| Single static audience                 | Stolen token replayed across all GGID APIs                   | Lateral movement    |
| No resource indicators                 | Token issued for Users API used on Audit API                 | Privilege escalation|

### 10.4 Testing Recommendations

1. **Introspection auth test**: Verify `401` without credentials; `200` with valid credentials
2. **Resource indicator test**: Issue token with `resource=https://api.ggid.dev/users`, verify `aud` is set
3. **Audience mismatch test**: Token with `aud=api.users` rejected by Audit API middleware
4. **Distributed revocation test**: Revoke on instance A, verify IsTokenRevoked returns true on instance B
5. **Cache invalidation test**: Cache positive result, revoke token, verify next introspection returns inactive
6. **Negative cache test**: Submit invalid token, verify second request doesn't hit introspection endpoint
7. **Rate limit test**: Exceed 1,000 req/min, verify `429` response

---

## Appendix A: RFC Reference Summary

| RFC     | Title                                    | Relevance                                          |
|---------|------------------------------------------|---------------------------------------------------|
| RFC 6749| OAuth 2.0 Authorization Framework        | Base OAuth spec, defines token types              |
| RFC 7009| OAuth 2.0 Token Revocation               | `POST /oauth/revoke` endpoint                     |
| RFC 7662| OAuth 2.0 Token Introspection            | `POST /oauth/introspect` endpoint                 |
| RFC 8707| Resource Indicators for OAuth 2.0        | `resource` parameter, audience binding            |
| RFC 7519| JSON Web Token (JWT)                     | Token format, `aud` claim semantics               |
| RFC 8414| OAuth 2.0 Authorization Server Metadata  | `introspection_endpoint` in discovery             |

## Appendix B: GGID Source File References

| Component                        | Path                                                              | Line(s)   |
|---------------------------------|-------------------------------------------------------------------|-----------|
| Introspection endpoint handler  | `services/oauth/internal/server/server.go`                        | 545-574   |
| IntrospectionResponse struct    | `services/oauth/internal/service/oauth_service.go`                | 539-550   |
| IntrospectToken method          | `services/oauth/internal/service/oauth_service.go`                | 553-579   |
| Token revocation handler        | `services/oauth/internal/server/server.go`                        | 470-494   |
| RevokeToken method              | `services/oauth/internal/service/oauth_service.go`                | 696-717   |
| IsTokenRevoked method           | `services/oauth/internal/service/oauth_service.go`                | 720-724   |
| Revocation store (sync.Map)     | `services/oauth/internal/service/oauth_service.go`                | 640       |
| Gateway JWTAuth middleware      | `services/gateway/internal/middleware/middleware.go`              | 500-578   |
| Gateway config (JWTAudience)    | `services/gateway/internal/config/config.go`                      | 30, 45    |
| Discovery metadata              | `services/oauth/internal/service/oauth_service.go`                | 372-373   |
| issueAccessToken                | `services/oauth/internal/service/oauth_service.go`                | 409-443   |
| Authorize endpoint handler      | `services/oauth/internal/server/server.go`                        | 157-290   |

---

*End of document. 400+ lines.*
