# Scope Explosion Prevention: Designing a Manageable Scope Taxonomy

> OAuth scope lifecycle security: taxonomy design, consent, deduplication,
> dynamic registration, hierarchy, and per-route enforcement.
> For standard claim definitions, see [oidc-claims-and-scopes.md](./oidc-claims-and-scopes.md).

---

## 1. Overview

OAuth scopes define what an access token **can do** — they limit a client's reach.
Two failure modes exist at opposite ends of the spectrum:

| Anti-Pattern | Cause | Consequence |
|---|---|---|
| **Scope explosion** | Too many fine-grained scopes (hundreds) | Maintenance nightmare, nobody knows what a scope does, clients request everything, developers bypass checks |
| **Scope starvation** | Too few or overly broad scopes | Every token is effectively `admin`, least-privilege is impossible, a leaked token grants full access |

The goal is a **balanced taxonomy** with proper enforcement at every layer:
issuance (token only contains granted-scoped claims), transport (token carries
accurate scope claims), and enforcement (gateway denies requests lacking required
scopes).

---

## 2. Scope Taxonomy Design

### Hierarchical Scopes

Dot-notation scopes create an implicit inheritance tree:

```
admin
  └── identity
        ├── users
        │     ├── read
        │     ├── write
        │     └── delete
        └── roles
              ├── read
              └── write
```

- **Wildcard expansion**: `identity:users:*` implies all `identity:users:*` sub-scopes.
- **Inheritance**: `admin` implies `identity:*` implies `identity:users:read`.
- **Trade-off**: powerful for administrators, but harder to audit — a single
  wildcard grants everything beneath it.

### Flat Scopes (OAuth Standard)

Space-separated, no hierarchy: `"read write admin"` (RFC 6749). Simpler, but at
scale (50+ APIs) flat scopes become unmanageable.

### GGID Recommended Taxonomy

```text
# Standard OIDC (RFC 6749 / OpenID Connect Core)
openid                    # OIDC identity (issues id_token)
profile                   # Basic profile claims
email                     # Email claims
offline_access            # Refresh token grant

# GGID resource scopes (service:resource:action)
identity:users:read       # List/view users
identity:users:write      # Create/update users
identity:users:delete     # Delete users
identity:roles:read       # List roles
identity:roles:write      # Manage roles
policy:rules:read         # View policies
policy:rules:write        # Modify policies
org:read                  # View organizations
org:write                 # Manage organizations
audit:read                # Query audit logs
```

### Scope Hierarchy Table

| Scope | Implies | Typical Use Case |
|---|---|---|
| `admin` | All scopes | Super-admin, break-glass |
| `identity:*` | All `identity:*` scopes | Identity administrator |
| `identity:users:*` | `read` + `write` + `delete` | User manager |
| `identity:users:read` | (leaf) | Read-only user list for dashboards |
| `audit:read` | (leaf) | Compliance auditor (read-only) |

**Current GGID state**: The discovery endpoint (`GetDiscovery`) advertises only
`["openid", "profile", "email", "offline_access"]` — four standard OIDC scopes.
No GGID-specific resource scopes are defined or validated.

---

## 3. Scope Consent

### Explicit Consent

When a client first requests a scope, the authorization server should present a
consent screen showing exactly what data and actions the client will gain:

1. User reviews requested scopes and their human-readable descriptions.
2. User approves or denies **per-scope** (not all-or-nothing).
3. Granted scopes are stored keyed by `(user_id, client_id)`.
4. Subsequent authorization requests reuse previously granted scopes silently
   (incremental consent) or prompt only for **newly requested** scopes.

### Granular Consent (Progressive)

Requesting `admin` on first login is suspicious. Instead:

- **First login**: request `openid profile` only.
- **User clicks "Manage Users"**: client requests `identity:users:read`.
- **User clicks "Delete User"**: client requests `identity:users:delete`.

This progressive approach reduces user suspicion and limits blast radius of a
compromised token.

### Dynamic / Risk-Based Scoping

The authorization server evaluates context before granting a scope:

| Factor | Low risk (grant) | High risk (deny / step-up) |
|---|---|---|
| Source IP | Known corporate IP | Anonymous VPN / TOR |
| Device | Managed device with MFA | Unknown browser |
| Time | Business hours | 03:00 AM |
| Scope sensitivity | `read` | `delete` / `admin` |

High-risk combinations trigger step-up authentication (e.g., re-prompt WebAuthn)
or time-limit the granted scope (write expires in 1 hour).

### Consent Storage Schema

```sql
CREATE TABLE oauth_consents (
    user_id      UUID        NOT NULL,
    client_id    TEXT        NOT NULL,
    scopes       TEXT[]      NOT NULL DEFAULT '{}',  -- explicitly granted scopes
    granted_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ,
    PRIMARY KEY (user_id, client_id)
);

CREATE INDEX idx_consents_client ON oauth_consents (client_id);
```

**Current GGID state**: No consent table exists. No consent flow is implemented.

---

## 4. Scope Deduplication

### Problem

A client may send `scope=read write read admin read`. Duplicates must be
collapsed to `read write admin` before storage and token issuance. Failing to
deduplicate causes:

- Bloated token payloads (JWT `scope` claim grows).
- Confusion in audit logs (same scope listed multiple times).
- Authorization header size issues for REST proxies.

### Implementation

```go
// deduplicateScopes removes duplicate scopes while preserving order.
func deduplicateScopes(scopes []string) []string {
    seen := make(map[string]bool, len(scopes))
    result := make([]string, 0, len(scopes))
    for _, s := range scopes {
        if !seen[s] {
            seen[s] = true
            result = append(result, s)
        }
    }
    return result
}
```

### Scope Validation Rules

At the authorize and token endpoints, the AS should enforce:

1. **Reject unknown scopes** — scope must exist in the registry.
2. **Reject unauthorized scopes** — client's registration must allow this scope.
3. **Reject empty scope** — unless the endpoint is a public discovery API.
4. **Deduplicate** — before storing on the auth code or token.
5. **Intersect with user consent** — token scope = min(requested, consented, client-allowed).

**Current GGID state**: `oauth_service.go` stores `req.Scope` directly on the
authorization code (line 255) with no validation, deduplication, or registry
lookup. `strings.Fields` splits the scope string but does not filter unknowns.

---

## 5. Dynamic Scope Registration

### Problem

New API features need new scopes. Hard-coding them in the AS configuration is
slow and requires a deployment. Per-tenant custom scopes (tenant A has
`billing:read`, tenant B does not) require a flexible registry.

### Pattern

A **scope registry** acts as the single source of truth. The AS validates every
requested scope against the registry before issuing a token.

```go
type ScopeRegistry struct {
    scopes map[string]ScopeDefinition
    mu     sync.RWMutex
}

type ScopeDefinition struct {
    Name        string   // e.g. "identity:users:write"
    Description string   // human-readable, shown on consent screen
    Claims      []string // OIDC claims this scope grants
    Default     bool     // granted without explicit consent?
    AdminOnly   bool     // requires admin approval?
}
```

Tenants can register custom scopes in a namespaced format:
`{tenant_id}:custom:{resource}:{action}`. The AS validates that a scope is
either a built-in (`identity:*`, `policy:*`, etc.) or a registered tenant scope.

### Registry Storage

```sql
CREATE TABLE oauth_scope_registry (
    scope_name   TEXT        NOT NULL,
    tenant_id    UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    description  TEXT        NOT NULL,
    claims       TEXT[]      NOT NULL DEFAULT '{}',
    is_default   BOOLEAN     NOT NULL DEFAULT false,
    admin_only   BOOLEAN     NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (scope_name, tenant_id)
);
```

**Current GGID state**: No scope registry. Dynamic client registration
(`RegisterClient` / RFC 7591) accepts any scope string and stores it on the
client without validation.

---

## 6. Scope-to-Claim Mapping Enforcement

### The Principle

Each OAuth scope grants specific OIDC claims. The AS **must** ensure the token
contains only claims for explicitly granted scopes. A token with `scope=openid`
must never contain `email` or `name`.

| Scope | Claims Granted |
|---|---|
| `openid` | `sub` |
| `profile` | `name`, `family_name`, `given_name`, `picture`, `preferred_username` |
| `email` | `email`, `email_verified` |
| `identity:users:read` | `groups` (for authorization) |

### Enforcement Pattern

```go
func (s *OAuthService) buildClaims(user *domain.User, scopes []string) jwt.MapClaims {
    claims := jwt.MapClaims{"sub": user.ID}
    for _, scope := range scopes {
        def, ok := s.scopeRegistry.Lookup(scope)
        if !ok {
            continue // unknown scope — skip (shouldn't happen post-validation)
        }
        for _, claimName := range def.Claims {
            if val := user.GetClaim(claimName); val != nil {
                claims[claimName] = val
            }
        }
    }
    return claims
}
```

### GGID Gap Analysis

GGID's `ExchangeAuthorizationCode` issues tokens but there is no evidence of
claim filtering by scope in `oauth_service.go`. The discovery endpoint lists
`ClaimsSupported` (sub, email, name, picture, groups, preferred_username,
updated_at) without binding them to specific scopes. **Any token may currently
carry all user attributes regardless of the granted scope.**

---

## 7. Scope Enforcement at Gateway

### Per-Route Scope Requirements

The gateway is the natural enforcement point. Each route is annotated with
required scopes, and a middleware checks the JWT's scope claim before forwarding:

```go
type RouteScope struct {
    Pattern   string   // e.g. "/api/v1/users"
    Methods   []string // e.g. {"POST", "PUT", "DELETE"}
    Required  []string // e.g. {"identity:users:write"}
}

var routeScopes = []RouteScope{
    {Pattern: "/api/v1/users", Methods: []string{"GET"},
     Required: []string{"identity:users:read"}},
    {Pattern: "/api/v1/users", Methods: []string{"POST", "PUT", "PATCH", "DELETE"},
     Required: []string{"identity:users:write"}},
    {Pattern: "/api/v1/roles", Methods: []string{"GET"},
     Required: []string{"identity:roles:read"}},
    {Pattern: "/api/v1/audit", Methods: []string{"GET"},
     Required: []string{"audit:read"}},
}
```

### Middleware

```go
func ScopeEnforcement(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := ClaimsFromContext(r.Context())
        required := matchRouteScopes(r.URL.Path, r.Method)
        if required != nil && !hasAllScopes(claims.Scopes, required) {
            w.Header().Set("WWW-Authenticate",
                `Bearer error="insufficient_scope", `+
                    `error_description="token lacks required scope"`)
            writeJSONError(w, "insufficient_scope", http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func hasAllScopes(have, need []string) bool {
    set := make(map[string]bool, len(have))
    for _, s := range have {
        set[s] = true
    }
    for _, s := range need {
        if !set[s] {
            return false
        }
    }
    return true
}
```

### GGID Current State

The gateway's `jwt_claims.go` **extracts** scopes from the JWT (handling both
`scope` string and `scopes` array formats) and forwards them as the `X-Scopes`
header. The `apikey.go` middleware has a `HasScope()` helper, but it only
applies to API-key authentication and returns `true` (unrestricted) for JWT-based
requests.

**No per-route scope enforcement exists.** The gateway validates JWT signature
and expiry but does not check whether the token has the required scope for the
targeted endpoint. Any valid JWT can access any route.

---

## 8. GGID Scope Audit

| Feature | GGID Status | Gap | Priority |
|---|---|---|---|
| Scope taxonomy defined | Partial — OIDC only (`openid profile email offline_access`) | No GGID resource scopes (`identity:*`, `policy:*`) | P0 |
| Scope hierarchy (wildcard) | No | No dot-notation or `*` expansion | P2 |
| Scope consent UI | No | No consent table, no consent screen | P1 |
| Scope deduplication | No | Raw `strings.Fields` stored on auth code | P0 |
| Dynamic scope registration | Partial — RFC 7591 accepts scope string | No registry, no validation of requested scopes | P2 |
| Scope-to-claim enforcement | No | Token claims not filtered by granted scope | P1 |
| Per-route scope requirement | No | Gateway validates JWT only, no scope check | P0 |
| Scope registry/storage | No | No `oauth_scope_registry` table | P2 |
| Scope validation at authorize | No | `req.Scope` stored directly (line 255) | P0 |

**Summary**: GGID currently treats scopes as opaque strings with no validation,
no hierarchy, no consent, and no enforcement. The only scope-aware code is the
gateway's claim extraction (for downstream headers) and the API-key `HasScope()`
helper.

---

## 9. Implementation Roadmap

| Phase | Deliverable | Priority | Effort |
|---|---|---|---|
| 1 | Define GGID scope taxonomy (`identity:*`, `policy:*`, `org:*`, `audit:*`) and update discovery endpoint | P0 | ~1 day |
| 2 | Per-route scope enforcement middleware in gateway with `insufficient_scope` (403) response | P0 | ~2 days |
| 3 | Scope deduplication + validation at authorize/token endpoints (reject unknown scopes) | P0 | ~1 day |
| 4 | Scope-to-claim mapping enforcement in token issuance (`buildClaims` filters by scope) | P1 | ~1 day |
| 5 | Consent storage (`oauth_consents` table) + consent screen in OAuth authorize flow | P1 | ~3 days |
| 6 | Scope registry (`oauth_scope_registry` table) + per-tenant custom scope registration | P2 | ~2 days |
| 7 | Scope hierarchy — dot-notation wildcard expansion (`identity:users:*`) | P2 | ~1 day |

**Total estimated effort**: ~11 days. Phases 1-3 are critical security gaps and
should be addressed before any production deployment.
