# OIDC Scope Management and Enforcement for IAM Systems

> Scope registry, consent management, scope-role separation, downgrade attack
> defense, dynamic registration validation, token-level enforcement, lifecycle
> management, and GGID gap analysis.
>
> Related documents:
> - [oidc-claims-and-scopes.md](./oidc-claims-and-scopes.md) — Standard OIDC
>   scope definitions, claim types, subject identifiers.
> - [scope-explosion-prevention.md](./scope-explosion-prevention.md) — Scope
>   taxonomy design, hierarchical scopes, deduplication, gateway enforcement.

---

## 1. Scope Registry Design

A scope registry is the authoritative source for every scope an authorization
server recognises. Without one, scope strings are free-text: typos pass
validation, documentation drifts, and downstream services cannot resolve what a
scope means. A registry maps each scope name to a structured descriptor.

### Scope Descriptor

```go
// ScopeTier classifies a scope's origin and trust level.
type ScopeTier int

const (
    TierStandard ScopeTier = iota // openid, profile, email — RFC 6749/OpenID Core
    TierInternal                  // platform-internal scopes (e.g. ggid:admin)
    TierCustom                    // tenant-defined via dynamic registration
)

// ScopeDescriptor holds metadata for a single scope.
type ScopeDescriptor struct {
    Name            string            // e.g. "profile", "identity:users:read"
    DisplayName     string            // human-readable label for consent UI
    Description     string            // what data or access this scope grants
    Claims          []string          // OIDC claims this scope unlocks
    Tier            ScopeTier
    RequiresConsent bool              // force interactive consent even if previously granted
    Deprecated      bool              // marked for removal
    SunsetDate      *time.Time        // when deprecated scopes stop working
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// ScopeRegistry is the authoritative scope store.
type ScopeRegistry struct {
    mu     sync.RWMutex
    scopes map[string]*ScopeDescriptor
}

func NewScopeRegistry() *ScopeRegistry {
    r := &ScopeRegistry{scopes: make(map[string]*ScopeDescriptor)}
    r.seedStandard()
    return r
}

// seedStandard registers the built-in OIDC scopes.
func (r *ScopeRegistry) seedStandard() {
    standard := []*ScopeDescriptor{
        {Name: "openid", DisplayName: "Sign you in", Description: "Authenticate your identity", Claims: []string{"sub"}, Tier: TierStandard, RequiresConsent: false},
        {Name: "profile", DisplayName: "Profile information", Description: "Name, photo, and other profile details", Claims: []string{"name", "family_name", "given_name", "picture", "preferred_username", "updated_at"}, Tier: TierStandard, RequiresConsent: false},
        {Name: "email", DisplayName: "Email address", Description: "Your email and verification status", Claims: []string{"email", "email_verified"}, Tier: TierStandard, RequiresConsent: false},
        {Name: "address", DisplayName: "Address", Description: "Your postal address", Claims: []string{"address"}, Tier: TierStandard, RequiresConsent: true},
        {Name: "phone", DisplayName: "Phone number", Description: "Your phone number and verification status", Claims: []string{"phone_number", "phone_number_verified"}, Tier: TierStandard, RequiresConsent: true},
        {Name: "offline_access", DisplayName: "Background access", Description: "Access your data when you are not using the application", Claims: []string{}, Tier: TierStandard, RequiresConsent: true},
    }
    for _, s := range standard {
        s.CreatedAt = time.Now()
        r.scopes[s.Name] = s
    }
}
```

### Registration API

Custom scopes are registered through an authenticated admin or client
registration endpoint. The registry validates name format and rejects collisions
with standard scopes:

```go
var reservedScopePrefixes = []string{"openid", "profile", "email", "address", "phone", "offline_access"}

// Register adds a new custom scope to the registry.
func (r *ScopeRegistry) Register(sd *ScopeDescriptor) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if err := validateScopeName(sd.Name); err != nil {
        return err
    }
    if _, exists := r.scopes[sd.Name]; exists {
        return fmt.Errorf("scope %q already registered", sd.Name)
    }
    if isReservedPrefix(sd.Name) {
        return fmt.Errorf("scope %q collides with a reserved standard scope", sd.Name)
    }
    sd.Tier = TierCustom
    sd.CreatedAt = time.Now()
    sd.UpdatedAt = sd.CreatedAt
    r.scopes[sd.Name] = sd
    return nil
}

func validateScopeName(name string) error {
    if name == "" || len(name) > 128 {
        return fmt.Errorf("scope name must be 1-128 characters")
    }
    // Allow alphanumeric, colon, underscore, dot, hyphen
    for _, ch := range name {
        if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
            (ch >= '0' && ch <= '9') || ch == ':' || ch == '_' ||
            ch == '.' || ch == '-') {
            return fmt.Errorf("scope name contains invalid character %q", ch)
        }
    }
    return nil
}

func isReservedPrefix(name string) bool {
    for _, prefix := range reservedScopePrefixes {
        if name == prefix || strings.HasPrefix(name, prefix+":") {
            return true
        }
    }
    return false
}

// Lookup returns the descriptor for a scope, or nil if not found.
func (r *ScopeRegistry) Lookup(name string) *ScopeDescriptor {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.scopes[name]
}

// ResolveClaims expands a list of scope names into the full set of claims.
func (r *ScopeRegistry) ResolveClaims(scopeNames []string) []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    seen := make(map[string]bool)
    var claims []string
    for _, name := range scopeNames {
        if sd, ok := r.scopes[name]; ok {
            for _, c := range sd.Claims {
                if !seen[c] {
                    seen[c] = true
                    claims = append(claims, c)
                }
            }
        }
    }
    return claims
}
```

### DB-Backed Registry

For multi-tenant deployments, scopes persist in PostgreSQL:

```sql
CREATE TABLE oauth_scopes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    description     TEXT NOT NULL,
    claims          TEXT[] NOT NULL DEFAULT '{}',
    tier            SMALLINT NOT NULL DEFAULT 2, -- 0=standard,1=internal,2=custom
    requires_consent BOOLEAN NOT NULL DEFAULT true,
    deprecated      BOOLEAN NOT NULL DEFAULT false,
    sunset_date     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, name)
);
```

---

## 2. Consent Flow Implementation

OIDC requires user consent for scopes beyond the basic identity set. GGID's
current consent implementation (in `services/oauth/internal/server/server.go`)
is minimal: it checks whether any scope is outside a hardcoded `basicScopes` set
and returns `consent_required` if so. The user must re-send `consent=true` as a
query parameter. There is no persistent consent record.

### Consent Record Model

```go
// ConsentRecord stores a user's consent decision for a client+scope combination.
type ConsentRecord struct {
    ID         uuid.UUID
    TenantID   uuid.UUID
    UserID     uuid.UUID
    ClientID   uuid.UUID
    Scopes     []string // scopes the user explicitly granted
    GrantedAt  time.Time
    RevokedAt  *time.Time
    ExpiresAt  *time.Time // optional: consent can expire
}

// ConsentStore manages consent persistence.
type ConsentStore interface {
    GetConsent(ctx context.Context, tenantID, userID, clientID uuid.UUID) (*ConsentRecord, error)
    SaveConsent(ctx context.Context, record *ConsentRecord) error
    RevokeConsent(ctx context.Context, tenantID, userID, clientID uuid.UUID) error
}
```

### Granular Consent Handler

Granular consent lets the user accept some scopes and reject others. The
authorization server issues a token containing only the accepted scopes, not the
full requested set:

```go
// EvaluateConsent determines which scopes to grant based on stored consent
// and the current request. Returns (grantedScopes, needsInteraction).
func EvaluateConsent(
    store ConsentStore,
    ctx context.Context,
    tenantID, userID, clientID uuid.UUID,
    requestedScopes []string,
    registry *ScopeRegistry,
) (granted []string, needsInteraction bool, err error) {
    // 1. Look up existing consent.
    existing, err := store.GetConsent(ctx, tenantID, userID, clientID)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return nil, false, err
    }

    consentSet := make(map[string]bool)
    if existing != nil && existing.RevokedAt == nil {
        for _, s := range existing.Scopes {
            consentSet[s] = true
        }
    }

    // 2. Check each requested scope.
    var unconsented []string
    for _, scope := range requestedScopes {
        sd := registry.Lookup(scope)
        if sd == nil {
            // Unknown scope — reject silently rather than erroring.
            continue
        }
        if sd.Deprecated {
            // Skip deprecated scopes; they should not appear in new grants.
            continue
        }
        if !sd.RequiresConsent || consentSet[scope] {
            granted = append(granted, scope)
        } else {
            unconsented = append(unconsented, scope)
        }
    }

    // 3. If any scope needs consent, trigger the consent UI.
    if len(unconsented) > 0 {
        return granted, true, nil
    }
    return granted, false, nil
}

// RecordConsent saves a granular consent decision.
func RecordConsent(
    store ConsentStore,
    ctx context.Context,
    tenantID, userID, clientID uuid.UUID,
    acceptedScopes []string,
) error {
    record := &ConsentRecord{
        ID:        uuid.New(),
        TenantID:  tenantID,
        UserID:    userID,
        ClientID:  clientID,
        Scopes:    acceptedScopes,
        GrantedAt: time.Now(),
    }
    return store.SaveConsent(ctx, record)
}

// RevokeConsentForClient removes all consent for a user+client pair.
// This should also revoke all active tokens for that combination.
func RevokeConsentForClient(
    store ConsentStore,
    tokenRevoker TokenRevoker,
    ctx context.Context,
    tenantID, userID, clientID uuid.UUID,
) error {
    if err := store.RevokeConsent(ctx, tenantID, userID, clientID); err != nil {
        return err
    }
    return tokenRevoker.RevokeByClientAndUser(ctx, tenantID, clientID, userID)
}
```

### Consent Revocation Endpoint

Users should be able to revoke consent at any time. This is typically exposed
at `/oauth/consent/revoke` and should cascade to token revocation:

```go
// HandleConsentRevoke processes POST /oauth/consent/revoke
func HandleConsentRevoke(store ConsentStore, revoker TokenRevoker) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            ClientID string `json:"client_id"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeError(w, http.StatusBadRequest, "invalid_request")
            return
        }
        // userID comes from JWT middleware context
        claims := middleware.ClaimsFromContext(r.Context())
        userID, _ := uuid.Parse(claims.Subject)
        clientUUID, _ := uuid.Parse(req.ClientID)
        tc, _ := tenant.FromContext(r.Context())

        if err := RevokeConsentForClient(store, revoker, r.Context(),
            tc.TenantID, userID, clientUUID); err != nil {
            writeError(w, http.StatusInternalServerError, "server_error")
            return
        }
        writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
    }
}
```

---

## 3. Scope-Role Separation

### Why They Must Not Merge

| Aspect | OAuth Scopes | RBAC Roles |
|--------|-------------|------------|
| Purpose | Delegated access (what a client can do on behalf of a user) | Internal authorization (what a user can do in the system) |
| Granted by | Resource owner (consent) | Administrator (assignment) |
| Lifetime | Token-bound (short-lived) | Session-bound (long-lived) |
| Carried in | Access token `scope` claim | Backend authorization check |
| Revocation | Token revocation or expiry | Role removal |

The most common mistake in IAM implementations is mapping OAuth scopes directly
to RBAC roles — e.g., issuing a `role:admin` scope that the gateway treats as
full administrative access. This collapses two independent security dimensions
into one and makes least-privilege enforcement impossible.

### Dual Enforcement Pattern

A resource endpoint may require **both** a valid scope (delegated permission)
and a valid role (internal permission). The scope proves the client is
authorized to call this category of API; the role proves the user is authorized
to perform this specific action.

```go
// DualAuthz enforces both OAuth scope and RBAC role.
type DualAuthz struct {
    roleChecker RoleChecker
}

type RoleChecker interface {
    HasRole(ctx context.Context, tenantID, userID uuid.UUID, role string) (bool, error)
}

// RequireScopeAndRole returns middleware that checks both conditions.
func (d *DualAuthz) RequireScopeAndRole(requiredScope, requiredRole string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := middleware.ClaimsFromContext(r.Context())

            // 1. Check scope (from JWT).
            if !contains(claims.Scopes, requiredScope) {
                writeRFC6750Error(w, "insufficient_scope",
                    "The request requires higher privileges than provided by the access token.")
                return
            }

            // 2. Check role (from backend RBAC).
            tc, _ := tenant.FromContext(r.Context())
            userID, _ := uuid.Parse(claims.Subject)
            hasRole, err := d.roleChecker.HasRole(r.Context(), tc.TenantID, userID, requiredRole)
            if err != nil || !hasRole {
                writeRFC6750Error(w, "insufficient_privilege",
                    "User does not have the required role.")
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func contains(slice []string, val string) bool {
    for _, s := range slice {
        if s == val {
            return true
        }
    }
    return false
}

func writeRFC6750Error(w http.ResponseWriter, errorCode, description string) {
    w.Header().Set("WWW-Authenticate",
        fmt.Sprintf(`Bearer error="%s", error_description="%s"`, errorCode, description))
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusForbidden)
    json.NewEncoder(w).Encode(map[string]string{
        "error":             errorCode,
        "error_description": description,
    })
}
```

### Example: DELETE /api/v1/users/:id

```go
// Route registration:
//   Scope: identity:users:write (proves the client has user-write delegation)
//   Role:  user_manager (proves the user can manage users)
adminRouter.Delete("/api/v1/users/:id",
    dualAuthz.RequireScopeAndRole("identity:users:write", "user_manager"),
    deleteUserHandler,
)
```

---

## 4. Scope Downgrade Attacks

### Attack Scenario

1. User initially grants client `app1` scopes `openid profile identity:users:write`.
2. An attacker who compromises `app1` re-initiates the flow requesting only
   `openid profile`.
3. The server issues a token with the narrower scope set.
4. The attacker performs actions using the original broad-scoped token but
   the audit trail shows only `openid profile` — making malicious actions
   appear unauthorized and harder to attribute.

More dangerously, if the authorization server's audit logic assumes "the most
recent token reflects all granted permissions," the downgrade creates a false
audit baseline.

### Detection Strategy

Compare the currently requested scopes against the previously granted scopes
for the same user+client pair. If the new request is a strict subset, flag it:

```go
// ScopeDowngradeResult describes a detected downgrade attempt.
type ScopeDowngradeResult struct {
    IsDowngrade     bool
    PreviouslyGranted []string
    CurrentlyRequested []string
    RemovedScopes   []string
}

// DetectScopeDowngrade compares the current request against stored consent.
func DetectScopeDowngrade(
    store ConsentStore,
    ctx context.Context,
    tenantID, userID, clientID uuid.UUID,
    requestedScopes []string,
) (*ScopeDowngradeResult, error) {
    existing, err := store.GetConsent(ctx, tenantID, userID, clientID)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return &ScopeDowngradeResult{IsDowngrade: false}, nil
        }
        return nil, err
    }
    if existing == nil || existing.RevokedAt != nil {
        return &ScopeDowngradeResult{IsDowngrade: false}, nil
    }

    reqSet := make(map[string]bool)
    for _, s := range requestedScopes {
        reqSet[s] = true
    }

    var removed []string
    for _, granted := range existing.Scopes {
        if !reqSet[granted] {
            removed = append(removed, granted)
        }
    }

    // Downgrade = at least one previously granted scope was dropped,
    // AND no new scope was added.
    newScopesAdded := false
    grantedSet := make(map[string]bool)
    for _, s := range existing.Scopes {
        grantedSet[s] = true
    }
    for _, s := range requestedScopes {
        if !grantedSet[s] {
            newScopesAdded = true
        }
    }

    isDowngrade := len(removed) > 0 && !newScopesAdded

    return &ScopeDowngradeResult{
        IsDowngrade:        isDowngrade,
        PreviouslyGranted:  existing.Scopes,
        CurrentlyRequested: requestedScopes,
        RemovedScopes:      removed,
    }, nil
}
```

### Response Policy

When a downgrade is detected, the server should **not** silently narrow scopes.
Recommended policies:

1. **Log and warn** — Record the downgrade in the audit log with a `scope_downgrade`
   event type. Allow the flow to continue but flag for review.
2. **Require re-consent** — Force the user to explicitly consent to the narrower
   set, acknowledging that previously granted permissions will be dropped.
3. **Preserve maximum** — Issue the token with the union of previously granted
   and currently requested scopes (least-surprise for the user).

```go
// Policy: log downgrade and force re-consent.
func HandleDowngrade(result *ScopeDowngradeResult, auditLog AuditLogger,
    tenantID, userID, clientID uuid.UUID) bool {
    if !result.IsDowngrade {
        return false // no downgrade, proceed normally
    }
    auditLog.Log(ctx, AuditEvent{
        Type:    "scope_downgrade_detected",
        TenantID: tenantID,
        UserID:   userID,
        ClientID: clientID,
        Details: map[string]any{
            "previously_granted": result.PreviouslyGranted,
            "currently_requested": result.CurrentlyRequested,
            "removed_scopes":      result.RemovedScopes,
        },
    })
    return true // signal that re-consent is needed
}
```

---

## 5. Dynamic Scope Registration (RFC 7591)

RFC 7591 allows clients to register themselves dynamically. A malicious client
could attempt to register a scope that collides with a standard scope or an
internal platform scope to gain unintended access.

### Validation During Registration

```go
// ValidateDynamicScopes checks scope strings submitted during RFC 7591
// registration. Returns the sanitized scope list or an error.
func ValidateDynamicScopes(
    registry *ScopeRegistry,
    requestedScopes []string,
    allowDynamic bool,
) ([]string, error) {
    if !allowDynamic {
        // Only pre-registered scopes are allowed.
        var valid []string
        for _, s := range requestedScopes {
            if registry.Lookup(s) != nil {
                valid = append(valid, s)
            } else {
                return nil, fmt.Errorf("scope %q is not registered (dynamic scopes disabled)", s)
            }
        }
        return valid, nil
    }

    // Dynamic scopes allowed: validate format and collision.
    var valid []string
    for _, s := range requestedScopes {
        if err := validateScopeName(s); err != nil {
            return nil, fmt.Errorf("invalid scope %q: %w", s, err)
        }
        // Reject if it shadows a standard/internal scope.
        existing := registry.Lookup(s)
        if existing != nil && existing.Tier != TierCustom {
            return nil, fmt.Errorf("scope %q collides with a protected scope", s)
        }
        // Reject scope squatting on reserved prefixes.
        if isReservedPrefix(s) {
            return nil, fmt.Errorf("scope %q uses a reserved prefix", s)
        }
        valid = append(valid, s)
    }
    return valid, nil
}
```

### Rich Authorization Requests (RFC 9396)

RAR allows clients to request specific authorization details beyond flat scope
strings. Each detail type can carry its own actions:

```go
// AuthorizationDetail represents a RAR authorization detail entry.
type AuthorizationDetail struct {
    Type    string         `json:"type"`
    Actions []string       `json:"actions,omitempty"`
    Locations []string     `json:"locations,omitempty"`
    Extra   map[string]any `json:"-"`
}

// ParseAuthorizationDetails extracts RAR details from an authorization request.
func ParseAuthorizationDetails(raw json.RawMessage) ([]AuthorizationDetail, error) {
    var details []AuthorizationDetail
    if err := json.Unmarshal(raw, &details); err != nil {
        return nil, fmt.Errorf("invalid authorization_details: %w", err)
    }

    // Validate each detail type against a registry of known types.
    for i, d := range details {
        if d.Type == "" {
            return nil, fmt.Errorf("authorization_details[%d]: type is required", i)
        }
        if !isValidDetailType(d.Type) {
            return nil, fmt.Errorf("authorization_details[%d]: unknown type %q", i, d.Type)
        }
    }
    return details, nil
}

var allowedDetailTypes = map[string]bool{
    "payment_initiation": true,
    "account_information": true,
    "identity:users":      true,
    "identity:roles":      true,
}

func isValidDetailType(t string) bool {
    return allowedDetailTypes[t]
}
```

---

## 6. Token-Level Scope Enforcement

The access token's `scope` claim is the contract between the authorization
server and the resource server. The resource server must enforce that the
token's scopes are sufficient for the requested endpoint.

### RFC 6750 Error Response

When scope is insufficient, RFC 6750 section 3.1 mandates the
`insufficient_scope` error with HTTP 403 and a `WWW-Authenticate` header:

```
HTTP/1.1 403 Forbidden
WWW-Authenticate: Bearer realm="api",
  error="insufficient_scope",
  error_description="Token does not have the identity:users:write scope"
Content-Type: application/json

{"error":"insufficient_scope","error_description":"...","scope":"identity:users:write"}
```

### Scope Enforcement Middleware

```go
// ScopeRequirement defines what scopes an endpoint requires.
type ScopeRequirement struct {
    // AnyOf: token must have at least one of these scopes.
    AnyOf []string
    // AllOf: token must have all of these scopes.
    AllOf []string
}

// RequireScopes returns middleware that validates token scopes.
func RequireScopes(req ScopeRequirement) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := middleware.ClaimsFromContext(r.Context())

            tokenScopes := make(map[string]bool)
            for _, s := range claims.Scopes {
                tokenScopes[s] = true
            }

            // Check AllOf requirements.
            for _, required := range req.AllOf {
                if !tokenScopes[required] {
                    writeInsufficientScope(w, required)
                    return
                }
            }

            // Check AnyOf requirements.
            if len(req.AnyOf) > 0 {
                satisfied := false
                for _, required := range req.AnyOf {
                    if tokenScopes[required] {
                        satisfied = true
                        break
                    }
                }
                if !satisfied {
                    writeInsufficientScope(w, strings.Join(req.AnyOf, " "))
                    return
                }
            }

            next.ServeHTTP(w, r)
        })
    }
}

func writeInsufficientScope(w http.ResponseWriter, requiredScope string) {
    w.Header().Set("WWW-Authenticate",
        fmt.Sprintf(`Bearer error="insufficient_scope", scope="%s"`, requiredScope))
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusForbidden)
    json.NewEncoder(w).Encode(map[string]string{
        "error":             "insufficient_scope",
        "error_description": "The request requires higher privileges than provided by the access token.",
        "scope":             requiredScope,
    })
}
```

### Hierarchical Scope Matching

If the taxonomy uses hierarchical scopes (see `scope-explosion-prevention.md`),
the enforcement middleware must support wildcard expansion:

```go
// HasHierarchicalScope checks if the token grants a required scope,
// considering parent scope inheritance.
// e.g., token scope "identity:users:*" satisfies requirement "identity:users:read".
func HasHierarchicalScope(tokenScopes []string, required string) bool {
    for _, s := range tokenScopes {
        if s == required || s == "*" {
            return true
        }
        // Check wildcard: "identity:users:*" matches "identity:users:read"
        if strings.HasSuffix(s, ":*") {
            prefix := strings.TrimSuffix(s, "*")
            if strings.HasPrefix(required, prefix) {
                return true
            }
        }
        // Check parent: "identity" matches "identity:users:read"
        if strings.HasPrefix(required, s+":") {
            return true
        }
    }
    return false
}
```

---

## 7. Scope Lifecycle Management

Scopes are not immutable. Over time, scopes are deprecated, renamed, or
deleted. Each lifecycle event must be handled carefully to avoid breaking
existing tokens or creating security gaps.

### Scope Deprecation

```go
// DeprecateScope marks a scope as deprecated with an optional sunset date.
func (r *ScopeRegistry) DeprecateScope(name string, sunsetDate time.Time) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    sd, ok := r.scopes[name]
    if !ok {
        return fmt.Errorf("scope %q not found", name)
    }
    sd.Deprecated = true
    sd.SunsetDate = &sunsetDate
    sd.UpdatedAt = time.Now()

    // Log the deprecation event.
    // New authorization requests will skip deprecated scopes (see EvaluateConsent).
    return nil
}

// IsSunsetPast returns true if a deprecated scope's sunset date has passed.
func (sd *ScopeDescriptor) IsSunsetPast() bool {
    return sd.Deprecated && sd.SunsetDate != nil && time.Now().After(*sd.SunsetDate)
}
```

### Scope Deletion and Token Revocation

Deleting a scope must cascade to revoke all tokens that contain it:

```go
// DeleteScope removes a scope and revokes all tokens containing it.
func DeleteScope(
    registry *ScopeRegistry,
    tokenStore TokenStore,
    auditLog AuditLogger,
    ctx context.Context,
    scopeName string,
) error {
    // 1. Find all active tokens with this scope.
    tokens, err := tokenStore.FindTokensWithScope(ctx, scopeName)
    if err != nil {
        return fmt.Errorf("find tokens with scope: %w", err)
    }

    // 2. Revoke each token.
    for _, token := range tokens {
        if err := tokenStore.Revoke(ctx, token.ID); err != nil {
            // Log but continue — partial revocation is better than none.
            auditLog.Log(ctx, AuditEvent{
                Type:   "scope_deletion_revocation_failed",
                Details: map[string]any{"token_id": token.ID, "scope": scopeName},
            })
            continue
        }
    }

    // 3. Remove from registry.
    registry.mu.Lock()
    delete(registry.scopes, scopeName)
    registry.mu.Unlock()

    // 4. Audit.
    auditLog.Log(ctx, AuditEvent{
        Type:    "scope_deleted",
        Details: map[string]any{"scope": scopeName, "tokens_revoked": len(tokens)},
    })
    return nil
}
```

### Scope Migration (Renaming)

Renaming a scope without breaking existing tokens requires an alias period:

```go
// MigrateScope renames oldScope to newScope, keeping both active during
// the migration window so existing tokens continue to work.
func MigrateScope(
    registry *ScopeRegistry,
    oldName, newName string,
    migrationWindow time.Duration,
) error {
    registry.mu.Lock()
    defer registry.mu.Unlock()

    old, ok := registry.scopes[oldName]
    if !ok {
        return fmt.Errorf("scope %q not found", oldName)
    }

    // Create the new scope as a copy.
    newSD := *old
    newSD.Name = newName
    newSD.CreatedAt = time.Now()
    newSD.UpdatedAt = newSD.CreatedAt
    if err := validateScopeName(newName); err != nil {
        return err
    }
    registry.scopes[newName] = &newSD

    // Mark old scope as deprecated with sunset at end of migration window.
    sunset := time.Now().Add(migrationWindow)
    old.Deprecated = true
    old.SunsetDate = &sunset
    old.UpdatedAt = time.Now()

    // Register an alias so token enforcement accepts both names.
    registry.aliases[oldName] = newName
    return nil
}

// ResolveAlias returns the canonical scope name, resolving migration aliases.
func (r *ScopeRegistry) ResolveAlias(name string) string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    if canonical, ok := r.aliases[name]; ok {
        return canonical
    }
    return name
}
```

---

## 8. GGID Scope Enforcement Gap Analysis

This section examines the actual GGID source code to identify what exists and
what is missing in scope management and enforcement.

### 8.1 OAuth Service — Token Issuance (`services/oauth/`)

**What exists:**

- `OAuthService.CreateAuthorizationCode` (`oauth_service.go:201`) stores
  `req.Scope` directly into the authorization code without any validation
  against a registry. Any scope string is accepted.
- `ExchangeAuthorizationCode` (`oauth_service.go:291`) passes `code.Scope`
  through to the token response as a space-joined string. However, the
  `issueAccessToken` method (`oauth_service.go:402`) does **not** include the
  scope claim in the JWT. The access token contains only `iss`, `sub`, `aud`,
  `iat`, `exp`, `jti`, and `tenant_id` — no `scope` field.
- `IntrospectToken` (`oauth_service.go:546`) attempts to read `scope` from
  claims but it will always be empty because `issueAccessToken` never sets it.
- Consent handling in `server.go:233-254` uses a hardcoded `basicScopes` map.
  Consent is a single boolean query parameter (`consent=true`) — no granular
  consent, no persistent storage, no revocation.

**Critical gaps:**

| Gap | Impact | Severity |
|-----|--------|----------|
| Access token has no `scope` claim | Resource servers cannot enforce scopes — the token is effectively scope-unaware | Critical |
| No scope registry or validation at issuance | Arbitrary scope strings accepted; typos and fake scopes pass through | High |
| No persistent consent storage | Users cannot review or revoke consent; re-consent required every time | High |
| Consent is binary (all or nothing) | Users cannot selectively grant scopes | Medium |
| No scope downgrade detection | Audit trail can be weakened by requesting narrower scopes | Medium |

### 8.2 Gateway — JWT Middleware (`services/gateway/`)

**What exists:**

- `JWTClaimExtraction` middleware (`jwt_claims.go:104`) extracts `scope` from
  the JWT payload (both string and array forms) and sets it as the `X-Scopes`
  header for downstream services. This is well-implemented.
- `APIKeyAuth` middleware (`apikey.go:22`) validates API keys and extracts
  scopes into context via `APIKeyScopesKey`.
- `HasScope` (`apikey.go:60`) checks whether a specific scope exists in the
  context. However, it **returns `true` when no scopes are in context** (line
  63: `return true // No scope restriction if not using API key`). This means
  any JWT-authenticated request bypasses scope checks entirely.

**Critical gaps:**

| Gap | Impact | Severity |
|-----|--------|----------|
| `HasScope` returns `true` for JWT requests | JWT-authenticated requests never fail scope checks — scope enforcement is effectively disabled | Critical |
| No per-route scope requirements | The router (`router.go`) does not attach scope requirements to any route; all authenticated routes are equally accessible | Critical |
| No `insufficient_scope` error response | Gateway does not emit RFC 6750 compliant scope error responses | Medium |
| API key scopes and JWT scopes use different code paths | Inconsistent enforcement between the two authentication methods | Medium |

### 8.3 The `HasScope` Security Hole — Detailed

```go
// Current implementation (apikey.go:59-71):
func HasScope(ctx context.Context, scope string) bool {
    scopes, ok := ctx.Value(APIKeyScopesKey).([]string)
    if !ok {
        return true // BUG: returns true when no API key scopes present
    }
    // ...
}
```

This function was designed only for API key scope checking. When a request
arrives via JWT (not API key), `APIKeyScopesKey` is never set, so `ok` is
`false`, and the function returns `true` — granting access regardless of the
required scope. Any code calling `HasScope` for JWT-authenticated requests is
silently insecure.

The fix is to also check JWT scopes:

```go
func HasScope(ctx context.Context, scope string) bool {
    // Check API key scopes.
    if scopes, ok := ctx.Value(APIKeyScopesKey).([]string); ok {
        for _, s := range scopes {
            if s == scope || s == "*" {
                return true
            }
        }
        return false
    }
    // Check JWT scopes.
    if claims, ok := ctx.Value(claimsKey).(JWTCClaims); ok {
        for _, s := range claims.Scopes {
            if s == scope || s == "*" {
                return true
            }
        }
        return false
    }
    // No authentication context at all — deny.
    return false
}
```

---

## 9. Gap Analysis and Recommendations

### Summary of Findings

| Area | Current State | Risk |
|------|---------------|------|
| Scope registry | Does not exist — scopes are unvalidated strings | High |
| Token scope claim | Not included in access token JWT | Critical |
| Consent management | Binary, ephemeral, no persistence | High |
| Scope enforcement (gateway) | `HasScope` always returns `true` for JWT | Critical |
| Per-route scope requirements | Not implemented | Critical |
| Scope downgrade detection | Not implemented | Medium |
| Scope lifecycle (deprecate/migrate) | Not implemented | Low |
| Dynamic scope validation (RFC 7591) | Not implemented | Medium |

### Action Items

| # | Action | Effort | Priority |
|---|--------|--------|----------|
| 1 | **Include `scope` claim in access token JWT** — Modify `issueAccessToken` to add `"scope": strings.Join(code.Scope, " ")` to the JWT claims map. | 1 hour | P0 |
| 2 | **Fix `HasScope` to check JWT scopes** — Add JWT scope lookup so the function does not return `true` when no API key context exists. | 1 hour | P0 |
| 3 | **Implement scope registry with validation** — Add a `ScopeRegistry` that validates scope strings at authorization code creation time. Seed with standard OIDC scopes. | 1-2 days | P1 |
| 4 | **Add per-route scope requirements in gateway router** — Attach `ScopeRequirement` to each route and enforce via middleware. Start with admin-only routes (`/api/v1/users` DELETE, `/api/v1/roles` POST). | 2-3 days | P1 |
| 5 | **Implement persistent consent with revocation** — Store consent records per user+client+scope in PostgreSQL. Add consent revocation endpoint that cascades to token revocation. | 2-3 days | P1 |
| 6 | **Add scope downgrade detection** — Compare requested scopes against stored consent before code issuance. Log downgrades in audit trail. | 1 day | P2 |

### Architectural Recommendation

The scope enforcement gap is architectural: the OAuth service issues tokens
without scope claims, and the gateway has no mechanism to enforce scopes per
route. Fixing this requires changes across two services:

1. **OAuth service**: Embed `scope` in the JWT during `issueAccessToken`.
2. **Gateway**: Implement `RequireScopes` middleware and attach scope
   requirements to protected routes in `router.go`.

Until both are done, any code relying on `HasScope` for JWT-authenticated
requests is providing a false sense of security.

---

## References

- [RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749) — OAuth 2.0 Framework
- [RFC 6750](https://datatracker.ietf.org/doc/html/rfc6750) — Bearer Token Usage (insufficient_scope error)
- [RFC 7591](https://datatracker.ietf.org/doc/html/rfc7591) — OAuth 2.0 Dynamic Client Registration
- [RFC 7592](https://datatracker.ietf.org/doc/html/rfc7592) — OAuth 2.0 Dynamic Client Registration Management
- [RFC 9396](https://datatracker.ietf.org/doc/html/rfc9396) — OAuth 2.0 Rich Authorization Requests
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html) — Requesting Claims using Scope Values
- [oidc-claims-and-scopes.md](./oidc-claims-and-scopes.md) — Standard scope and claim reference
- [scope-explosion-prevention.md](./scope-explosion-prevention.md) — Scope taxonomy design and hierarchy
