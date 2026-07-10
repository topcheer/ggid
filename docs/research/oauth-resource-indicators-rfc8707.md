# RFC 8707: Resource Indicators for OAuth 2.0 — GGID Integration Research

> **Status**: Research / Design Document  
> **RFC**: [RFC 8707](https://www.rfc-editor.org/rfc/rfc8707.html) — Resource Indicators for OAuth 2.0  
> **Priority**: P1 (critical for microservices token isolation)

---

## 1. Overview

RFC 8707 defines the `resource` request parameter, enabling an OAuth 2.0 client
to explicitly tell the Authorization Server (AS) **which resource server (API)**
it needs a token for.

**The problem.** In standard OAuth 2.0 (RFC 6749), access tokens carry an
ambiguous or absent audience. A token issued by the AS can be replayed at any
resource server that trusts that AS. In a microservices architecture like GGID,
where one AS (the OAuth service) issues tokens consumed by identity, policy,
org, and audit services, this creates a significant cross-service replay surface.

**The solution.** The client passes `resource=https://api.example.com/identity`
when requesting a token. The AS binds the resulting token's `aud` claim to that
identifier. Each resource server validates that `aud` matches its own identifier
and rejects tokens minted for a different service.

**Key outcomes:**
- Tokens are audience-bound — a token for `identity` fails at `policy`.
- Reduced blast radius — compromise of one token does not expose other services.
- Clearer audit trail — `aud` reveals which service the token was scoped to.

---

## 2. The resource Parameter

### 2.1 Authorization Request

The client includes the `resource` parameter in the authorization request.
The parameter **can be repeated** to request a token valid for multiple resources:

```
GET /oauth/authorize?
    response_type=code
    &client_id=s6BhdRkqt3
    &redirect_uri=https%3A%2F%2Fclient.example.org%2Fcb
    &scope=openid profile
    &resource=https%3A%2F%2Fggid.example.com%2Fidentity
    &resource=https%3A%2F%2Fggid.example.com%2Fpolicy
    &state=af0ifjsldkj
```

The AS validates whether the client is **authorized** to access each requested
resource. If not, it returns an `access_denied` error. When multiple resources
are specified, the issued token's scopes are the **intersection** of scopes
permitted across all requested resources.

### 2.2 Token Request

For `client_credentials` and `refresh_token` grants, the client includes
`resource` in the token request body:

```
POST /oauth/token HTTP/1.1
Host: ggid.example.com
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials
&client_id=s6BhdRkqt3
&client_secret=...
&resource=https://ggid.example.com/identity
```

The AS sets the token's `aud` claim to the resource value. For `authorization_code`
grants, the resource is inherited from the authorize request (the AS stores it
alongside the authorization code).

### 2.3 PAR Request (RFC 9126)

When using Pushed Authorization Requests, `resource` is included in the pushed
request body and validated server-side:

```
POST /oauth/par HTTP/1.1

client_id=s6BhdRkqt3
&response_type=code
&resource=https://ggid.example.com/identity
&redirect_uri=https%3A%2F%2Fclient.example.org%2Fcb
```

GGID already implements PAR — the `resource` parameter flows through identically.

### 2.4 Resource Identifier Format

- Must be an **absolute URI** (RFC 3986).
- **Must use HTTPS** scheme.
- Should be a stable, resolvable identifier for the resource server.
- Case-sensitive comparison as per RFC 3986 normalization.

```
https://ggid.example.com/identity    -- identity service
https://ggid.example.com/policy      -- policy service
https://ggid.example.com/org         -- org service
https://ggid.example.com/audit       -- audit service
```

---

## 3. Audience Binding in JWT

### 3.1 Token Claims

When a `resource` parameter is provided, the AS sets the JWT `aud` claim to the
resource value. For multiple resources, `aud` becomes an array:

```json
{
  "iss": "https://ggid.example.com",
  "sub": "user-uuid-here",
  "aud": "https://ggid.example.com/identity",
  "exp": 1716000000,
  "iat": 1715999100,
  "jti": "token-uuid",
  "scope": "openid profile read:users",
  "tenant_id": "00000000-0000-0000-0000-000000000001"
}
```

Multiple resources:

```json
{
  "aud": [
    "https://ggid.example.com/identity",
    "https://ggid.example.com/policy"
  ],
  "scope": "read:users"
}
```

### 3.2 Resource Server Validation

Each resource server validates the JWT signature, expiry, **and** audience:

1. Parse JWT, verify RS256 signature using AS JWKS.
2. Check `exp` — reject if expired.
3. Check `aud` — must include the RS's own resource identifier.
4. If `aud` does not match: return `401` with
   `WWW-Authenticate: Bearer error="invalid_token",
   error_description="audience mismatch"`.

This prevents a token minted for the `identity` service from being used at the
`policy` or `audit` service.

### 3.3 Introspection Response (RFC 7662)

When the RS uses token introspection, the response includes the `aud` claim:

```json
{
  "active": true,
  "aud": "https://ggid.example.com/identity",
  "scope": "read:users",
  "exp": 1716000000,
  "sub": "user-uuid-here"
}
```

GGID already exposes `aud` in its introspection response
(`IntrospectionResult.Aud` field at `oauth_service.go:541`).

---

## 4. GGID Integration

### 4.1 Current State

Examination of the GGID OAuth service reveals a **partial foundation** but
**no RFC 8707 support**:

| Aspect | Current State | RFC 8707 Gap |
|--------|--------------|--------------|
| `aud` claim in JWT | Set in `issueAccessToken()` — but caller hardcoded to `"ggid"` in some paths (token exchange `oauth_service.go:1270`, `"ggid"` literal) | Should be set to `resource` value |
| `resource` parameter parsing | **Not parsed** in authorize or token endpoints | Need to extract from request |
| Token exchange | `TokenExchangeRequestRFC8693` has `Resource` and `Audience` fields (`oauth_service.go:1055-1056`) but they are **unused** — exchange issues an opaque `"exchanged_" + uuid` token | Should set `aud` from `resource` |
| Discovery metadata | `GetDiscoveryConfig()` at `oauth_service.go:357` does **not** advertise `resource_indicators` | Add to discovery document |
| Gateway audience validation | Gateway validates JWT signature and expiry but does **not** check `aud` against target service | Need per-route audience enforcement |

### 4.2 Required Changes

1. **Token endpoint**: parse `resource` form parameter (repeatable).
2. **Authorize endpoint**: parse `resource` query parameter, store alongside auth code.
3. **Token issuance**: pass `resource` value(s) as `audience` to `issueAccessToken()`.
4. **Gateway middleware**: validate `aud` matches the target service identifier per route.
5. **Discovery**: advertise `resource_indicators` support.

### 4.3 Go Code — Token Issuance

```go
// TokenRequest extended with resource parameter
type AuthorizationCodeRequest struct {
    Code        string
    RedirectURI string
    ClientID    string
    Resource    []string // NEW: RFC 8707 resource indicators
}

// Modified issueAccessToken to accept multiple resources
func (s *OAuthService) issueAccessToken(
    userID, tenantID uuid.UUID,
    resources []string, // NEW: was single string
) (string, int, error) {
    now := time.Now()
    expiresAt := now.Add(15 * time.Minute)

    var aud any
    if len(resources) == 1 {
        aud = resources[0]
    } else {
        aud = resources
    }

    claims := jwt.MapClaims{
        "iss":       s.issuer,
        "sub":       userID.String(),
        "aud":       aud, // audience bound to requested resource(s)
        "iat":       now.Unix(),
        "exp":       expiresAt.Unix(),
        "jti":       uuid.New().String(),
        "tenant_id": tenantID.String(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = s.keyProvider.KeyID()
    signed, err := token.SignedString(s.keyProvider.PrivateKey())
    if err != nil {
        return "", 0, fmt.Errorf("sign access token: %w", err)
    }
    return signed, int(expiresAt.Sub(now).Seconds()), nil
}
```

### 4.4 Go Code — Gateway Validation

```go
// AuthMiddleware validates JWT audience per route
func (m *AuthMiddleware) validateAudience(
    claims jwt.MapClaims,
    targetService string,
) error {
    aud, ok := claims["aud"]
    if !ok {
        return errors.PermissionDenied("missing audience claim")
    }

    switch v := aud.(type) {
    case string:
        if v != targetService {
            return errors.PermissionDenied(
                "audience mismatch: token for %s, expected %s", v, targetService,
            )
        }
    case []any:
        found := false
        for _, a := range v {
            if s, ok := a.(string); ok && s == targetService {
                found = true
                break
            }
        }
        if !found {
            return errors.PermissionDenied(
                "audience mismatch: token not for %s", targetService,
            )
        }
    default:
        return errors.PermissionDenied("invalid audience claim type")
    }
    return nil
}

// Route-to-resource mapping
var routeAudience = map[string]string{
    "/api/v1/users":  "https://ggid.example.com/identity",
    "/api/v1/roles":  "https://ggid.example.com/policy",
    "/api/v1/orgs":   "https://ggid.example.com/org",
    "/api/v1/audit":  "https://ggid.example.com/audit",
}
```

### 4.5 Per-Service Resource Identifiers

| Service | Resource Identifier | Gateway Route |
|---------|-------------------|---------------|
| Identity | `https://ggid.example.com/identity` | `/api/v1/users` |
| Policy | `https://ggid.example.com/policy` | `/api/v1/roles` |
| Org | `https://ggid.example.com/org` | `/api/v1/orgs` |
| Audit | `https://ggid.example.com/audit` | `/api/v1/audit` |

### 4.6 Discovery Metadata Update

```go
// Add to GetDiscoveryConfig() return struct
func (s *OAuthService) GetDiscoveryConfig() *domain.OIDCDiscoveryConfig {
    // ... existing fields ...
    return &domain.OIDCDiscoveryConfig{
        // ... existing fields ...
        ResourceIndicatorsSupported: true, // NEW: RFC 8707
    }
}
```

---

## 5. Comparison: resource Parameter vs. aud Claim

| Dimension | `resource` Parameter (RFC 8707) | `aud` Claim (RFC 7519 JWT) |
|-----------|-------------------------------|--------------------------|
| **Direction** | Client → AS (request) | AS → RS (in token) |
| **Purpose** | Client requests target audience | Token carries the audience |
| **Without resource param** | AS sets `aud` to default or omits it | GGID currently hardcodes `"ggid"` |
| **With resource param** | Client controls which service receives the token | `aud` = the `resource` value(s) |
| **Security constraint** | AS validates client is authorized for the resource before issuing | RS validates `aud` matches its identifier |
| **Client cannot escalate** | AS rejects unauthorized `resource` requests | RS rejects mismatched `aud` |

The `resource` parameter is the **request-side** mechanism; `aud` is the
**token-side** mechanism. RFC 8707 connects them: `resource` in → `aud` out.

---

## 6. Security Benefits

1. **Token replay prevention.** A token scoped to `identity` fails validation
   at `policy`, `org`, and `audit`. An attacker who steals a token gets access
   to one service, not all four.

2. **Reduced blast radius.** Compartmentalized access means a leaked token
   only affects the service it was minted for. Incident response is scoped.

3. **Clearer audit trail.** The `aud` claim in every token and the introspection
   response reveals which service the token was intended for, improving forensic
   analysis.

4. **Scope intersection.** When requesting tokens for multiple resources, the
   AS grants only the intersection of permitted scopes — least privilege by
   default.

5. **Fine-grained access control.** Clients request exactly the access they
   need. No over-privileged tokens carrying blanket access to all GGID services.

---

## 7. Feature Comparison Table

| Feature | Without Resource (Current) | With Resource (RFC 8707) |
|---------|--------------------------|------------------------|
| **Audience control** | AS hardcodes `"ggid"` for all tokens | Client specifies target via `resource` |
| **Cross-service replay** | Token valid at any GGID service | Token rejected if `aud` mismatches |
| **Token scope** | Full client scopes everywhere | Intersection of scopes per resource |
| **Audit clarity** | `aud` always `"ggid"` — no per-service info | `aud` = specific service URI |
| **Gateway enforcement** | Signature + expiry only | Signature + expiry + audience check |
| **Implementation effort** | Zero (current state) | ~6-7 days (see roadmap) |

---

## 8. Implementation Roadmap

| Phase | Task | Effort | Dependencies |
|-------|------|--------|-------------|
| 1 | Parse `resource` parameter in token endpoint | ~2 days | None |
| 2 | Set `aud` claim from `resource` in `issueAccessToken()` | ~1 day | Phase 1 |
| 3 | Gateway audience validation per route | ~2 days | Phase 2 |
| 4 | Authorize endpoint `resource` parameter + auth code storage | ~1 day | Phase 1 |
| 5 | Discovery metadata: `resource_indicators` support | ~0.5 day | Phase 1 |

**Total effort**: ~6-7 days  
**Priority**: P1 — critical for microservices token isolation in multi-tenant deployments.

### Sequencing Notes

- Phases 1-2 can be done first for `client_credentials` grants (no user consent flow).
- Phase 4 (authorize endpoint) requires updating the auth code storage schema to
  persist `resource` alongside the code.
- Phase 3 (gateway) is independent and can proceed in parallel once Phase 2 lands.
- Phase 5 is trivial once Phase 1 is merged.

### Backward Compatibility

Existing clients that omit `resource` continue to receive tokens with the current
default `aud`. Gateway audience validation should be **opt-in per route** initially,
then enforced globally once all clients are updated.

---

*This document examines GGID OAuth service source at `services/oauth/internal/service/oauth_service.go` and `services/oauth/internal/server/server.go`.*
