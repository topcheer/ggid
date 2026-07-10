# JWT Claim Validation for IAM Systems

> Research document examining JWT claim validation requirements for identity and access management (IAM) systems. Covers temporal claims (exp, nbf, iat), identity claims (iss, aud, sub), and custom IAM claims (tenant_id, scope, amr, acr). Includes a security audit of GGID's gateway JWT validation middleware.

> **Scope note:** Algorithm confusion attacks and key ID manipulation are covered in `jwt-algorithm-confusion.md` (353 lines). This document focuses exclusively on **claim validation** — verifying that token claims match expected values and that temporal constraints are enforced.

---

## 1. exp (Expiration Time) Validation

### Why exp Matters

The `exp` (expiration time) claim is the single most critical temporal control in a JWT. It defines the moment after which the token MUST NOT be accepted for processing. Without exp validation, a leaked or stolen token remains valid indefinitely — an attacker who captures a token once gains permanent access.

**Attack scenario:** An attacker gains access to application logs containing a JWT (e.g., through a log injection or SIEM misconfiguration). If the gateway does not check `exp`, the stolen token never expires. The attacker replays it weeks or months later.

### Clock Skew Tolerance

JWT `exp` is a Unix timestamp. In distributed systems, clock drift between the token issuer and the validating server is common. A strict equality check (`now > exp`) causes false rejections when the clocks differ by even a few seconds.

Industry recommendation: **tolerate ±30 seconds** of clock skew. This is short enough to prevent meaningful replay windows, but long enough to absorb NTP drift and network latency.

> RFC 7519 Section 4.1.4: "Implementers MAY provide for some small leeway, usually no more than a few minutes, to account for clock skew."

### What Happens When the Gateway Forgets to Check exp

| Component | Behavior Without exp Check |
|---|---|
| Gateway | Accepts expired tokens indefinitely |
| Backend services | Trust the gateway, no additional check |
| Attacker | Can replay any stolen token forever |
| Audit log | Cannot correlate token lifetime to access events |

In GGID, the `golang-jwt/jwt/v5` library validates `exp` by default when the claim is present. However, this is library-dependent behavior — if a developer switches to `ParseUnverified` or a custom parser, exp checking can silently disappear.

### Go Code: exp Validation with Skew

```go
// validateExp checks that the token has not expired, with a configurable
// clock skew tolerance. Returns an error if the token is expired.
func validateExp(claims jwt.MapClaims, skew time.Duration) error {
    expRaw, ok := claims["exp"]
    if !ok {
        // exp is REQUIRED for access tokens per RFC 9068.
        return fmt.Errorf("missing required claim: exp")
    }

    // golang-jwt stores exp as a json.Number when UseNumber is enabled
    expFloat, ok := expRaw.(float64)
    if !ok {
        return fmt.Errorf("invalid exp claim type: %T", expRaw)
    }

    expTime := time.Unix(int64(expFloat), 0)
    if time.Now().After(expTime.Add(skew)) {
        return fmt.Errorf("token expired at %s (skew: %s)", expTime, skew)
    }
    return nil
}

// Usage with 30-second skew:
// err := validateExp(claims, 30*time.Second)
```

### Token Replay After Expiry

Even with exp validation, there is a replay window between token theft and expiry. For short-lived access tokens (15 minutes), this window is small. For long-lived tokens (24 hours), it is significant. Mitigations:

1. **Short token lifetimes** — access tokens: 5–15 minutes.
2. **Refresh token rotation** — refresh tokens are single-use; rotation detects theft.
3. **JTI tracking** — maintain a replay cache keyed by `jti` (GGID implements this in `jti_replay.go`).
4. **Token revocation** — support RFC 7009 token revocation for incident response.

---

## 2. nbf (Not Before) Validation

### When nbf Matters

The `nbf` (not before) claim defines the time before which the token MUST NOT be accepted. It is useful for:

- **Scheduled activation** — tokens issued for future use (e.g., time-limited admin elevation).
- **Clock coordination** — tokens minted slightly ahead of the consumer's clock.
- **Pre-authorized access** — OAuth grants issued before the resource is ready.

### Clock Skew Handling

Like `exp`, `nbf` should allow clock skew. The same ±30 second tolerance applies. Without skew, a token with `nbf = now()` can be rejected if the consumer's clock is 1 second behind the issuer.

### Why Skipping nbf Is Dangerous

If the gateway ignores `nbf`, a token intended for future activation can be used immediately. Consider:

- An IAM admin schedules a JIT (just-in-time) elevation token valid at 2:00 PM.
- The token is issued at 1:45 PM with `nbf = 14:00`.
- An attacker intercepts the token at 1:50 PM.
- Without nbf validation, the attacker uses the elevated access 10 minutes early.

### Go Code: nbf Validation

```go
// validateNbf checks that the current time is after the token's nbf,
// with clock skew tolerance. Returns nil if nbf is absent (optional claim).
func validateNbf(claims jwt.MapClaims, skew time.Duration) error {
    nbfRaw, ok := claims["nbf"]
    if !ok {
        // nbf is optional — absence is not an error
        return nil
    }

    nbfFloat, ok := nbfRaw.(float64)
    if !ok {
        return fmt.Errorf("invalid nbf claim type: %T", nbfRaw)
    }

    nbfTime := time.Unix(int64(nbfFloat), 0)
    // Allow skew: the token is valid if now + skew >= nbf
    if time.Now().Add(skew).Before(nbfTime) {
        return fmt.Errorf("token not valid before %s (skew: %s)", nbfTime, skew)
    }
    return nil
}
```

---

## 3. iat (Issued At) Validation

### Using iat to Detect Token Age Anomalies

The `iat` (issued at) claim records when the token was minted. While it does not directly gate access, it enables:

1. **Maximum token age enforcement** — reject tokens older than a policy limit (e.g., 1 hour), even if `exp` has not passed. This catches tokens with abnormally long lifetimes.
2. **Anomaly detection** — a token with `iat` 10 years ago is suspicious. It may indicate a compromised signing key or a misconfigured issuer.
3. **auth_time correlation** — in OIDC, `auth_time` records when the user actually authenticated. Comparing `iat` and `auth_time` detects token refresh after a stale session.

### iat in the Future

An `iat` value in the future is a red flag. It indicates either:
- Significant clock skew between issuer and consumer (benign).
- Token forgery with incorrect timestamp (malicious).

Rejecting future `iat` values (with skew tolerance) prevents both.

### Go Code: iat Validation

```go
// validateIat checks that the token's iat is not in the future (beyond
// clock skew) and, optionally, not older than maxAge.
func validateIat(claims jwt.MapClaims, skew time.Duration, maxAge time.Duration) error {
    iatRaw, ok := claims["iat"]
    if !ok {
        // iat is RECOMMENDED for access tokens — log a warning if absent
        return fmt.Errorf("missing recommended claim: iat")
    }

    iatFloat, ok := iatRaw.(float64)
    if !ok {
        return fmt.Errorf("invalid iat claim type: %T", iatRaw)
    }

    iatTime := time.Unix(int64(iatFloat), 0)
    now := time.Now()

    // Reject tokens issued in the future (beyond skew)
    if iatTime.After(now.Add(skew)) {
        return fmt.Errorf("token iat is in the future: %s (now: %s)", iatTime, now)
    }

    // Reject tokens older than maxAge, even if exp hasn't passed
    if maxAge > 0 && now.Sub(iatTime) > maxAge {
        return fmt.Errorf("token too old: issued %s ago (max: %s)", now.Sub(iatTime), maxAge)
    }

    return nil
}

// Usage: reject tokens older than 1 hour, allow 30s clock skew
// err := validateIat(claims, 30*time.Second, time.Hour)
```

---

## 4. aud (Audience) Binding

### Why Audience Validation Prevents Cross-Service Replay

The `aud` (audience) claim identifies the intended recipient of the token. Without audience validation:

1. A token issued for Service A can be replayed against Service B.
2. An attacker with access to one service can pivot to all services sharing the same issuer.

**Attack scenario:** A microservice "orders-api" accepts any JWT signed by the IdP. An attacker obtains a token intended for "inventory-api" and replays it against "orders-api" to access order data. If both services validated `aud`, the token would be rejected because it was not minted for "orders-api".

### Rejecting Tokens with Wrong Audience

Audience must be validated strictly. The token's `aud` must contain the resource server's expected audience value. Any mismatch = reject.

### Multiple Audiences

JWT supports multiple audiences. The `aud` claim can be:
- A single string: `"aud": "orders-api"`
- An array: `"aud": ["orders-api", "inventory-api"]`

A resource server should accept a token if its expected audience appears in the array.

### Resource Indicator (RFC 8707)

RFC 8707 defines the `resource` parameter in authorization requests. The resulting token's `aud` is set to the exact resource URL requested, not a generic audience. This provides precise per-resource audience binding:

```json
{
  "aud": "https://api.ggid.dev/orders"
}
```

### Go Code: Strict Audience Validation

```go
// validateAudience checks that the token's aud claim contains the expected
// audience. Uses constant-time comparison to prevent timing attacks.
func validateAudience(claims jwt.MapClaims, expectedAud string) error {
    audRaw, ok := claims["aud"]
    if !ok {
        return fmt.Errorf("missing required claim: aud")
    }

    switch aud := audRaw.(type) {
    case string:
        if subtle.ConstantTimeCompare([]byte(aud), []byte(expectedAud)) != 1 {
            return fmt.Errorf("audience mismatch: got %q, want %q", aud, expectedAud)
        }
    case []any:
        found := false
        for _, a := range aud {
            if s, ok := a.(string); ok {
                if subtle.ConstantTimeCompare([]byte(s), []byte(expectedAud)) == 1 {
                    found = true
                    break
                }
            }
        }
        if !found {
            return fmt.Errorf("audience %q not found in token audiences", expectedAud)
        }
    default:
        return fmt.Errorf("invalid aud claim type: %T", audRaw)
    }
    return nil
}
```

> **Note:** golang-jwt/v5 handles audience validation natively when `jwt.WithAudience(expected)` is passed to the parser. The manual implementation above shows the logic for custom validators.

---

## 5. iss (Issuer) Pinning

### Validating iss Matches Expected Issuer

The `iss` (issuer) claim identifies the principal that issued the token. In an IAM system with multiple identity providers (IdPs), issuer pinning ensures the token came from the expected IdP.

### Preventing Token Substitution

**Attack scenario:** An organization uses both an internal IdP (GGID Auth) and an external IdP (Auth0) for different applications. Both use RS256 with keys published via JWKS. An attacker who has a valid Auth0 token (with lower privileges) attempts to use it against an internal service expecting GGID tokens.

Without issuer validation, the service accepts any token signed by any key in its JWKS cache — but if both IdPs' keys are cached (unlikely but possible in federated setups), the token passes signature verification.

With issuer pinning, the service rejects the Auth0 token because `iss != "https://auth.ggid.dev"`.

### iss in OIDC Discovery Metadata

OIDC discovery (`/.well-known/openid-configuration`) publishes the issuer URL:

```json
{
  "issuer": "https://auth.ggid.dev",
  "jwks_uri": "https://auth.ggid.dev/.well-known/jwks.json"
}
```

The validated issuer must match the `issuer` field in the discovery document. This prevents an attacker from standing up a rogue IdP with the same JWKS keys.

### Go Code: Issuer Pinning

```go
// validateIssuer checks that the token's iss claim matches the expected
// issuer exactly (case-sensitive, including trailing slashes).
func validateIssuer(claims jwt.MapClaims, expectedIssuer string) error {
    iss, ok := claims["iss"].(string)
    if !ok {
        return fmt.Errorf("missing required claim: iss")
    }

    if iss != expectedIssuer {
        return fmt.Errorf("issuer mismatch: got %q, want %q", iss, expectedIssuer)
    }
    return nil
}

// In OIDC, the expected issuer comes from discovery metadata:
// config, _ := provider.Discover("https://auth.ggid.dev/.well-known/openid-configuration")
// expectedIssuer := config.Issuer
```

---

## 6. sub (Subject) Validation

### Ensuring sub Matches Authenticated User

The `sub` (subject) claim identifies the principal that is the subject of the token. In IAM systems, this is typically the user ID. Validation ensures:

1. `sub` is present and non-empty (unless the token is intentionally anonymous).
2. `sub` matches the authenticated session user (for session-bound operations).
3. `sub` is not manipulated between issuance and consumption.

### Preventing Subject Manipulation

An attacker cannot forge `sub` without the signing key, but subject manipulation can occur at the application layer:

- A gateway extracts `sub` from the token and passes it as a header (`X-User-ID`) to backend services.
- If the backend trusts `X-User-ID` without verifying it came from the gateway (not the client), an attacker can inject `X-User-ID` directly.

**Mitigation:** Backend services must strip incoming identity headers and only accept them from the trusted gateway.

### Pairwise vs Public Subject Identifiers (OIDC)

OIDC defines two subject identifier types:

| Type | Description | Example |
|---|---|---|
| **Public** | Same `sub` across all clients | `sub: "user-123"` |
| **Pairwise** | Different `sub` per client (sector-specific) | `sub: "user-abc-for-client-X"` |

Pairwise identifiers prevent cross-client user correlation. The `sub` claim should be validated against the expected type and, for pairwise, the expected sector identifier.

### Anonymous Tokens

Some tokens intentionally omit `sub` (e.g., client-credentials flow, machine-to-machine). The validator must distinguish between "intentionally anonymous" and "maliciously stripped":

```go
if sub == "" && grantType != "client_credentials" {
    return fmt.Errorf("missing sub for non-client-credentials token")
}
```

### Go Code: Subject Validation

```go
// validateSubject checks that sub is present and valid.
// allowEmpty controls whether anonymous tokens (no sub) are accepted.
func validateSubject(claims jwt.MapClaims, allowEmpty bool) (string, error) {
    sub, ok := claims["sub"].(string)
    if !ok || sub == "" {
        if allowEmpty {
            return "", nil // intentionally anonymous token
        }
        return "", fmt.Errorf("missing required claim: sub")
    }

    // Validate sub is a non-empty, printable string
    if strings.TrimSpace(sub) == "" {
        return "", fmt.Errorf("sub must not be whitespace-only")
    }

    // Optional: validate sub format (e.g., must be a UUID for user tokens)
    // if _, err := uuid.Parse(sub); err != nil {
    //     return "", fmt.Errorf("sub must be a valid UUID: %w", err)
    // }

    return sub, nil
}
```

---

## 7. Custom Claim Validation for IAM

IAM systems extend JWT with domain-specific claims. These are the most security-critical custom claims in GGID.

### tenant_id Claim Binding

GGID is a multi-tenant system. Every request must be scoped to a tenant. The `tenant_id` claim must:

1. Be present in the token (for authenticated requests).
2. Match the request's tenant context (from subdomain, header, or path).

**Attack scenario:** Without tenant binding, a user from Tenant A presents a token to access Tenant B's data. If the gateway does not verify `tenant_id` in the token matches the resolved tenant, cross-tenant data access is possible.

### scope Claim Enforcement

Scopes define what operations the token authorizes. GGID uses space-delimited scopes (OAuth 2.1 convention):

```json
{ "scope": "users:read roles:write audit:read" }
```

Scope enforcement must happen at both the gateway (coarse) and the backend service (fine-grained). The gateway can do path-based scope checks; services do resource-level authorization.

### role Claim Validation

Roles group permissions. The `roles` claim (or `realm_access.roles` in Keycloak-style tokens) lists the user's assigned roles. Validation:

1. Roles must be from the expected tenant's role store.
2. Role names must not contain injection characters.
3. Role hierarchy must be enforced (admin implies user).

### amr (Authentication Methods References)

The `amr` claim is an array of strings identifying how the user authenticated:

```json
{ "amr": ["pwd", "otp"] }
```

Common values (per RFC 8176):
- `pwd` — password
- `otp` — one-time password
- `swk` — software key (WebAuthn)
- `hwk` — hardware key
- `mfa` — multi-factor authentication

Services can enforce step-up authentication by requiring specific `amr` values (e.g., admin endpoints require `amr` containing `mfa`).

### acr (Authentication Context Class Reference)

The `acr` claim indicates the assurance level of authentication:

```json
{ "acr": "urn:mace:incommon:iap:silver" }
```

Or using levels: `"acr": "2"` (LoA2) or `"acr": "3"` (LoA3).

### Go Code: IAM-Specific Claim Validation

```go
// IAMClaimValidator validates IAM-specific custom claims.
type IAMClaimValidator struct {
    requiredScopes []string
    requiredAMR    []string
    requiredACR    string
}

// Validate checks tenant_id binding, scopes, amr, and acr.
func (v *IAMClaimValidator) Validate(claims jwt.MapClaims, requestTenantID string) error {
    // 1. tenant_id binding
    tokenTenantID, _ := claims["tenant_id"].(string)
    if tokenTenantID == "" {
        return fmt.Errorf("missing tenant_id claim")
    }
    if requestTenantID != "" && tokenTenantID != requestTenantID {
        return fmt.Errorf("tenant_id mismatch: token=%s request=%s",
            tokenTenantID, requestTenantID)
    }

    // 2. scope enforcement
    if len(v.requiredScopes) > 0 {
        tokenScopes := extractScopes(claims)
        for _, required := range v.requiredScopes {
            if !contains(tokenScopes, required) {
                return fmt.Errorf("missing required scope: %s", required)
            }
        }
    }

    // 3. amr enforcement (step-up auth)
    if len(v.requiredAMR) > 0 {
        tokenAMR := extractStringArray(claims, "amr")
        for _, required := range v.requiredAMR {
            if !contains(tokenAMR, required) {
                return fmt.Errorf("missing required amr: %s", required)
            }
        }
    }

    // 4. acr enforcement
    if v.requiredACR != "" {
        tokenACR, _ := claims["acr"].(string)
        if tokenACR != v.requiredACR {
            return fmt.Errorf("acr mismatch: got %q, want %q", tokenACR, v.requiredACR)
        }
    }

    return nil
}

func extractScopes(claims jwt.MapClaims) []string {
    switch v := claims["scope"].(type) {
    case string:
        return strings.Fields(v)
    case []any:
        var scopes []string
        for _, s := range v {
            if str, ok := s.(string); ok {
                scopes = append(scopes, str)
            }
        }
        return scopes
    }
    return nil
}

func extractStringArray(claims jwt.MapClaims, key string) []string {
    arr, ok := claims[key].([]any)
    if !ok {
        return nil
    }
    var result []string
    for _, v := range arr {
        if s, ok := v.(string); ok {
            result = append(result, s)
        }
    }
    return result
}

func contains(slice []string, val string) bool {
    for _, s := range slice {
        if s == val {
            return true
        }
    }
    return false
}
```

---

## 8. Complete Validation Pipeline

### Order of Validation

The validation pipeline should execute checks in order of cost (cheapest first) to fail fast and minimize resource consumption:

```
1. Parse token structure (cheap — string splitting)
2. Verify signature (expensive — cryptographic operation)
3. Check alg is in allowlist (cheap — string comparison)
4. Validate exp (cheap — integer comparison)
5. Validate nbf (cheap — integer comparison)
6. Validate iat (cheap — integer comparison)
7. Validate iss (cheap — string comparison)
8. Validate aud (cheap — string comparison)
9. Validate custom claims (medium — depends on logic)
```

**Why order matters:**

- Signature verification is the most expensive operation. However, it must happen BEFORE claim validation — claims are only trustworthy after the signature is verified.
- Temporal checks (exp, nbf, iat) are cheap and catch the most common attack (expired/replayed tokens).
- Issuer and audience checks prevent cross-service replay, a common lateral movement technique.
- Custom claim checks are last because they may involve database lookups (e.g., validating tenant_id exists).

### Go Code: Complete JWT Validation Middleware

```go
// JWTValidatorConfig holds all configuration for JWT claim validation.
type JWTValidatorConfig struct {
    Issuer           string        // Expected iss claim
    Audience         string        // Expected aud claim
    AllowedAlgs      []string      // Allowed signing algorithms (e.g. ["RS256"])
    ClockSkew        time.Duration // Tolerance for clock drift (recommended: 30s)
    MaxTokenAge      time.Duration // Maximum token age (recommended: 1h for access tokens)
    RequiredScopes   []string      // Scopes required for this endpoint
    RequireTenantID  bool          // Whether tenant_id must be present
    RequiredAMR      []string      // Required amr values (step-up auth)
}

// DefaultJWTValidatorConfig returns secure defaults.
func DefaultJWTValidatorConfig() JWTValidatorConfig {
    return JWTValidatorConfig{
        AllowedAlgs:     []string{"RS256"},
        ClockSkew:       30 * time.Second,
        MaxTokenAge:     time.Hour,
        RequireTenantID: true,
    }
}

// JWTValidationMiddleware is a complete JWT validation middleware with
// full claim validation pipeline.
func JWTValidationMiddleware(jwks *JWKSClient, cfg JWTValidatorConfig, required bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Step 1: Extract and parse token
            tokenStr, err := extractBearerToken(r)
            if err != nil {
                if required {
                    writeUnauthorized(w, err.Error())
                    return
                }
                next.ServeHTTP(w, r)
                return
            }

            // Step 2-3: Verify signature and enforce algorithm allowlist
            parseOpts := []jwt.ParserOption{
                jwt.WithValidMethods(cfg.AllowedAlgs),
                jwt.WithLeeway(cfg.ClockSkew),
            }
            if cfg.Issuer != "" {
                parseOpts = append(parseOpts, jwt.WithIssuer(cfg.Issuer))
            }
            if cfg.Audience != "" {
                parseOpts = append(parseOpts, jwt.WithAudience(cfg.Audience))
            }

            token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
                // Reject non-RSA algorithms defensively
                if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
                    return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
                }
                keyID, _ := token.Header["kid"].(string)
                if keyID == "" {
                    keyID = jwks.KeyID()
                }
                return jwks.GetKey(keyID)
            }, parseOpts...)
            if err != nil || !token.Valid {
                if required {
                    writeUnauthorized(w, "invalid or expired token")
                    return
                }
                next.ServeHTTP(w, r)
                return
            }

            claims, ok := token.Claims.(jwt.MapClaims)
            if !ok {
                if required {
                    writeUnauthorized(w, "invalid claims type")
                    return
                }
                next.ServeHTTP(w, r)
                return
            }

            // Step 4-6: Temporal validation (exp, nbf, iat handled by jwt/v5
            // with WithLeeway, but iat max-age needs manual check)
            if err := validateIatMaxAge(claims, cfg.ClockSkew, cfg.MaxTokenAge); err != nil {
                if required {
                    writeUnauthorized(w, err.Error())
                    return
                }
                next.ServeHTTP(w, r)
                return
            }

            // Step 7-8: iss and aud validated by parser via parseOpts

            // Step 9: Custom IAM claim validation
            requestTenantID := r.Header.Get("X-Tenant-ID") // from TenantResolver
            if err := validateIAMClaims(claims, requestTenantID, cfg); err != nil {
                if required {
                    writeForbidden(w, err.Error())
                    return
                }
                next.ServeHTTP(w, r)
                return
            }

            // Inject validated identity into context
            ctx := r.Context()
            if sub, _ := claims["sub"].(string); sub != "" {
                ctx = context.WithValue(ctx, UserIDKey, sub)
            }
            if tid, _ := claims["tenant_id"].(string); tid != "" {
                ctx = context.WithValue(ctx, TenantIDKey, tid)
            }

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// validateIatMaxAge rejects tokens older than maxAge, even if exp hasn't passed.
func validateIatMaxAge(claims jwt.MapClaims, skew, maxAge time.Duration) error {
    iatRaw, ok := claims["iat"]
    if !ok {
        return nil // iat is optional
    }
    iatFloat, ok := iatRaw.(float64)
    if !ok {
        return nil
    }
    iatTime := time.Unix(int64(iatFloat), 0)
    if time.Now().Sub(iatTime) > maxAge+skew {
        return fmt.Errorf("token exceeds maximum age")
    }
    return nil
}

// validateIAMClaims performs IAM-specific validation (tenant, scope, amr, acr).
func validateIAMClaims(claims jwt.MapClaims, requestTenantID string, cfg JWTValidatorConfig) error {
    // Tenant binding
    if cfg.RequireTenantID {
        tokenTenantID, _ := claims["tenant_id"].(string)
        if tokenTenantID == "" {
            return fmt.Errorf("missing tenant_id claim")
        }
        if requestTenantID != "" && tokenTenantID != requestTenantID {
            return fmt.Errorf("tenant_id mismatch")
        }
    }

    // Scope enforcement
    if len(cfg.RequiredScopes) > 0 {
        tokenScopes := extractScopes(claims)
        for _, req := range cfg.RequiredScopes {
            if !contains(tokenScopes, req) {
                return fmt.Errorf("insufficient scope: need %s", req)
            }
        }
    }

    // AMR enforcement (step-up authentication)
    if len(cfg.RequiredAMR) > 0 {
        tokenAMR := extractStringArray(claims, "amr")
        for _, req := range cfg.RequiredAMR {
            if !contains(tokenAMR, req) {
                return fmt.Errorf("insufficient authentication: need %s", req)
            }
        }
    }

    return nil
}

func extractBearerToken(r *http.Request) (string, error) {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        return "", fmt.Errorf("missing Authorization header")
    }
    parts := strings.SplitN(authHeader, " ", 2)
    if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
        return "", fmt.Errorf("invalid Authorization header format")
    }
    return strings.TrimSpace(parts[1]), nil
}
```

---

## 9. GGID JWT Validation Audit

### Files Reviewed

| File | Purpose |
|---|---|
| `services/gateway/internal/middleware/middleware.go` | `JWTAuth()` middleware — primary validation entry point |
| `services/gateway/internal/middleware/jwt_claims.go` | `ExtractJWTClaims()` — claim extraction (no verification) |
| `services/gateway/internal/middleware/jti_replay.go` | JTI replay tracking (in-memory) |
| `services/auth/internal/service/token_service.go` | Token issuance — defines claim structure |

### What's Validated

| Claim/Check | Status | Implementation Detail |
|---|---|---|
| **Signature** | VALIDATED | RS256 signature verified via JWKS (`middleware.go:535-544`) |
| **Algorithm** | VALIDATED | `jwt.WithValidMethods(["RS256"])` enforces RS256 only (`middleware.go:527`) |
| **exp** | VALIDATED | `golang-jwt/v5` validates exp by default; `TestJWT_ExpiredToken` confirms |
| **nbf** | VALIDATED | `golang-jwt/v5` validates nbf by default; `TestJWT_NotBeforeInFuture` confirms |
| **iss** | VALIDATED* | `jwt.WithIssuer(issuer)` — but only when issuer config is non-empty (`middleware.go:529-531`) |
| **aud** | VALIDATED* | `jwt.WithAudience(audience)` — but only when audience config is non-empty (`middleware.go:532-534`) |
| **sub** | EXTRACTED | Extracted and injected into context (`middleware.go:569-571`), but not validated against format or session |
| **iat** | NOT EXPLICITLY CHECKED | Relies on jwt/v5 default; no max-age enforcement |
| **jti** | TRACKED | In-memory replay tracker exists but is not wired into `JWTAuth()` middleware |
| **tenant_id** | EXTRACTED | Extracted into context; TenantResolver gives JWT priority over header (P0 fix applied) |
| **scope** | EXTRACTED | Extracted in `jwt_claims.go` but NOT enforced in middleware |
| **amr** | NOT VALIDATED | No amr checking anywhere in gateway |
| **acr** | NOT VALIDATED | No acr checking anywhere in gateway |
| **roles** | NOT VALIDATED | No role validation in gateway |

### Key Findings

**Finding 1: Clock skew is not explicitly configured.**
The `JWTAuth` middleware does not pass `jwt.WithLeeway()` to the parser. The `golang-jwt/v5` library defaults to 0 seconds leeway, meaning any clock drift causes token rejection. This is a reliability issue, not a security vulnerability (fail-closed).

**Finding 2: iss and aud validation are conditional.**
If the gateway is started without `issuer` or `audience` configuration, these checks are silently skipped (`middleware.go:529-534`). An empty string is falsy, so the parser option is not added. This means a misconfigured deployment has no issuer or audience binding.

**Finding 3: No maximum token age enforcement.**
A token with a very long `exp` (e.g., 1 year) would be accepted. There is no `iat`-based max-age check. The token's lifetime is entirely controlled by the issuer, not by a resource-server policy.

**Finding 4: JTI replay tracker exists but is not integrated.**
`JTIReplayTracker` (in `jti_replay.go`) provides replay detection, but it is not called from `JWTAuth()`. The tracker exists as infrastructure but is not wired into the validation pipeline. Tokens can be replayed without detection.

**Finding 5: No scope enforcement at middleware level.**
Scopes are extracted in `jwt_claims.go` and forwarded as `X-Scopes` header, but the middleware does not enforce required scopes. Any token with a valid signature and temporal claims can access any route regardless of its scope claims. Scope enforcement is delegated entirely to backend services.

**Finding 6: Claim extraction without verification.**
`ExtractJWTClaims()` in `jwt_claims.go` decodes the JWT payload without signature verification. This is documented as safe because "JWTAuth will verify the token later." However, if middleware ordering changes or `JWTClaimExtraction` runs before `JWTAuth`, unverified claims would be trusted. The X-Tenant-ID header set by this middleware could be spoofed by a client if JWTAuth is not in the chain.

**Finding 7: Tenant ID from JWT takes priority over header (GOOD).**
The TenantResolver (`middleware.go:246-251`) correctly gives JWT `tenant_id` priority over the `X-Tenant-ID` header. This was a P0 fix — previously, the unauthenticated header could override the authenticated JWT claim, enabling tenant spoofing.

---

## 10. Gap Analysis & Recommendations

### Gap 1: Wire JTI Replay Tracker into JWTAuth Middleware

**Problem:** The `JTIReplayTracker` exists but is not called during JWT validation. Replay attacks go undetected.

**Fix:** Pass the tracker to `JWTAuth()` and check `IsReplayed(jti, exp)` after signature verification. Transition from in-memory map to Redis SETNX for multi-instance deployments.

**Effort:** 2–4 hours (code change + test + Redis integration).

### Gap 2: Make iss and aud Validation Non-Optional

**Problem:** If `issuer` or `audience` config is empty, validation is silently skipped. A misconfigured deployment has no issuer or audience binding.

**Fix:** Treat empty issuer/audience as a fatal configuration error at startup. Log a warning and refuse to start, or enforce a default based on the deployment URL.

**Effort:** 1 hour.

### Gap 3: Add Clock Skew Configuration

**Problem:** No explicit `jwt.WithLeeway()` — defaults to 0 seconds. Clock drift causes false rejections.

**Fix:** Add a `clockSkew` parameter to `JWTAuth()` (default: 30 seconds) and pass it to the parser via `jwt.WithLeeway(clockSkew)`.

**Effort:** 30 minutes.

### Gap 4: Enforce Maximum Token Age

**Problem:** No `iat`-based max-age check. A token with a 1-year `exp` is accepted indefinitely.

**Fix:** Add `validateIatMaxAge()` call in `JWTAuth()` after temporal validation. Default max age: 1 hour for access tokens.

**Effort:** 1 hour.

### Gap 5: Add Scope and AMR Enforcement at Gateway Level

**Problem:** Scopes are extracted but not enforced. Any valid token accesses any route. AMR (step-up auth) is not checked.

**Fix:** Add per-route scope requirements in the router configuration. For admin endpoints, require `amr` containing `mfa`. Use a route map (`path → requiredScopes`) checked in middleware.

**Effort:** 4–8 hours (route configuration + middleware integration + tests).

### Summary Table

| Gap | Severity | Effort | Impact |
|---|---|---|---|
| JTI replay tracker not wired | High | 2–4h | Replay attack detection |
| iss/aud validation optional | High | 1h | Prevents misconfiguration bypass |
| Clock skew not configured | Medium | 30m | Reliability fix |
| No max token age | Medium | 1h | Limits stolen token lifetime |
| No scope/AMR enforcement | High | 4–8h | Authorization gap |

### Conclusion

GGID's JWT validation covers the essentials: signature verification, algorithm pinning, and temporal claim checks (exp, nbf) via library defaults. However, several critical gaps exist:

1. **Replay detection** infrastructure exists but is not activated.
2. **Issuer and audience** validation is conditional on configuration, not enforced.
3. **Scope enforcement** is delegated entirely to backend services with no gateway-level check.
4. **No step-up authentication** (amr/acr) enforcement.
5. **No maximum token age** policy.

Addressing these gaps would raise GGID's JWT security posture from "signature-verified" to "fully claim-validated," which is the standard for production IAM systems.

---

## References

- RFC 7519 — JSON Web Token (JWT)
- RFC 8417 — JWT Claims for OpenID Connect 1.0 Identity Tokens
- RFC 8707 — Resource Indicators for OAuth 2.0
- RFC 9068 — JSON Web Token (JWT) Profile for OAuth 2.0 Access Tokens
- RFC 9701 — OAuth 2.0 Token Introspection
- OIDC Core 1.0 — Section 2 (ID Token), Section 3.1.3.7 (ID Token Validation)
- OWASP JWT Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html
- golang-jwt/jwt/v5 documentation — https://pkg.go.dev/github.com/golang-jwt/jwt/v5
