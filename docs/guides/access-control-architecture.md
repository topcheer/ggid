# Access Control Architecture

This guide covers RBAC vs ABAC vs PBAC comparison, enforcement layers, PDP/PEP/PIP architecture, attribute store, decision caching, hierarchical RBAC, and GGID's access control architecture.

## RBAC vs ABAC vs PBAC

### Comparison

| Aspect | RBAC | ABAC | PBAC |
|---|---|---|---|
| Basis | Roles | Attributes | Policies |
| Granularity | Coarse | Fine | Fine |
| Complexity | Low | Medium | High |
| Performance | Fast | Medium | Medium |
| Flexibility | Low | High | Very High |
| Best for | Simple org | Fine-grained | Complex rules |

### When to Use Each

| Scenario | Recommended |
|---|---|
| Simple role hierarchy | RBAC |
| Department + clearance + time | ABAC |
| Complex policy with exceptions | PBAC |
| Mix of simple and complex | RBAC + ABAC (hybrid) |

### GGID Approach: Hybrid RBAC + ABAC

GGID uses RBAC as the base (roles assigned to users) with ABAC for fine-grained policy enforcement (attributes evaluated at decision time).

## Enforcement Layers

### Three-Layer Model

```
Request → Layer 1: Gateway (coarse) → Layer 2: Service (medium) → Layer 3: Data (fine)
```

### Layer 1: Gateway (Coarse)

- JWT validation (is token valid?)
- Route authorization (can user access this endpoint?)
- Rate limiting (is user within limits?)
- Tenant check (does JWT tenant match request?)

```go
func gatewayAuthz(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Validate JWT
        claims := validateJWT(r)
        
        // Check route authorization
        if !hasRouteAccess(claims, r.URL.Path) {
            writeError(w, 403, "route_forbidden")
            return
        }
        
        // Tenant check
        if claims.TenantID != getTenantFromRequest(r) {
            writeError(w, 403, "tenant_mismatch")
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### Layer 2: Service (Medium)

- Resource authorization (can user access this specific resource?)
- Action authorization (can user perform this action?)
- ABAC policy evaluation (do attributes allow this?)

```go
func serviceAuthz(user *User, action string, resource *Resource) error {
    // RBAC check
    if !user.HasPermission(action) {
        return ErrPermissionDenied
    }
    
    // ABAC check
    ctx := buildEvalContext(user, action, resource)
    decision := policyEngine.Evaluate(ctx)
    if decision != "permit" {
        return ErrPolicyDenied
    }
    
    return nil
}
```

### Layer 3: Data (Fine)

- Row-Level Security (RLS) in database
- Column-level access (PII fields)
- Data classification checks

```sql
-- PostgreSQL RLS
CREATE POLICY tenant_isolation ON users
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

## PDP vs PEP vs PIP

### Architecture

```
┌──────┐     ┌──────┐     ┌──────┐     ┌──────┐
│ PEP  │────▶│ PDP  │────▶│ PIP  │     │ PAP  │
│(Enf) │     │(Dec) │     │(Info)│     │(Admin)│
└──────┘     └──────┘     └──────┘     └──────┘
   ↑              ↑
   │              │
   └──decision────┘
```

### Policy Decision Point (PDP)

Evaluates policies and returns a decision (permit/deny):

```go
type PDP struct {
    policies  []Policy
    pip       *PIP
    cache     DecisionCache
}

func (p *PDP) Decide(request *AuthzRequest) string {
    // Check cache
    if cached, ok := p.cache.Get(request); ok {
        return cached
    }
    
    // Gather attributes from PIP
    ctx := p.pip.GatherAttributes(request)
    
    // Evaluate policies
    decision := p.evaluate(ctx)
    
    // Cache decision
    p.cache.Set(request, decision, 5*time.Minute)
    
    return decision
}
```

### Policy Enforcement Point (PEP)

Intercepts requests and enforces PDP decisions:

```go
type PEP struct {
    pdp *PDP
}

func (p *PEP) Enforce(user *User, action string, resource *Resource) error {
    request := &AuthzRequest{
        Subject: user,
        Action: action,
        Resource: resource,
    }
    
    decision := p.pdp.Decide(request)
    
    if decision != "permit" {
        audit.Log("access_denied", user.ID, action, resource.ID)
        return ErrAccessDenied
    }
    
    audit.Log("access_granted", user.ID, action, resource.ID)
    return nil
}
```

### Policy Information Point (PIP)

Gathers attributes needed for decision:

```go
type PIP struct {
    jwtStore    JWTStore
    ldapClient  LDAPClient
    dbClient    DBClient
    geoService  GeoService
}

func (p *PIP) GatherAttributes(request *AuthzRequest) *EvalContext {
    ctx := &EvalContext{}
    
    // Subject attributes (from JWT)
    ctx.Subject = map[string]interface{}{
        "role":        request.Subject.Role,
        "tenant_id":   request.Subject.TenantID,
        "mfa_verified": request.Subject.MFAVerified,
    }
    
    // Resource attributes (from DB)
    ctx.Resource = map[string]interface{}{
        "type":        request.Resource.Type,
        "owner":       request.Resource.OwnerID,
        "sensitivity": request.Resource.Classification,
        "tenant_id":   request.Resource.TenantID,
    }
    
    // Environment attributes (computed)
    ctx.Environment = map[string]interface{}{
        "time":    classifyTime(time.Now()),
        "network": classifyNetwork(request.IP),
        "device":  classifyDevice(request.UserAgent),
    }
    
    return ctx
}
```

### Policy Administration Point (PAP)

Manages policy creation, modification, and deployment:

```bash
POST /api/v1/policy/policies
{
  "name": "confidential-data-access",
  "effect": "permit",
  "condition": "subject.mfa_verified == true AND resource.sensitivity == 'confidential'"
}
```

## Attribute Store

### Attribute Sources

| Attribute | Source | Cache TTL |
|---|---|---|
| subject.role | JWT claim | Token lifetime |
| subject.tenant_id | JWT claim | Token lifetime |
| subject.department | LDAP | 5 min |
| subject.manager | HR system | 15 min |
| resource.type | DB metadata | 5 min |
| resource.owner | DB metadata | 5 min |
| resource.sensitivity | DB classification | 5 min |
| environment.time | System clock | No cache |
| environment.network | IP classification | 1 min |
| environment.device | User agent | No cache |

## Decision Caching

### Cache Strategy

```go
type DecisionCache struct {
    cache Cache
    ttl   time.Duration
}

func (c *DecisionCache) Get(req *AuthzRequest) (string, bool) {
    key := c.buildKey(req)
    if val, ok := c.cache.Get(key); ok {
        return val.(string), true
    }
    return "", false
}

func (c *DecisionCache) buildKey(req *AuthzRequest) string {
    // Key: subject_id + action + resource_id + resource_tenant
    return fmt.Sprintf("authz:%s:%s:%s:%s",
        req.Subject.ID, req.Action, req.Resource.ID, req.Resource.TenantID)
}

func (c *DecisionCache) Invalidate(userID string) {
    // Invalidate all decisions for user (on role change, etc.)
    c.cache.DeletePattern("authz:" + userID + ":*")
}
```

### Cache Invalidation Triggers

| Event | Invalidation |
|---|---|
| Role assignment change | All decisions for user |
| Policy update | All decisions for affected scope |
| Tenant config change | All decisions in tenant |
| Resource classification change | All decisions for resource |

## Hierarchical RBAC

### Role Hierarchy

```
platform-admin
    ├── security-admin
    │   ├── user-admin
    │   │   └── user-reader
    │   └── audit-reader
    ├── tenant-admin
    │   ├── role-admin
    │   └── config-admin
    └── developer
        └── api-reader
```

### Permission Inheritance

```go
func getEffectivePermissions(roleID string) []string {
    role := getRole(roleID)
    perms := make(map[string]bool)
    
    // Add role's own permissions
    for _, p := range role.Permissions {
        perms[p] = true
    }
    
    // Add parent role permissions (inheritance)
    if role.ParentID != "" {
        parentPerms := getEffectivePermissions(role.ParentID)
        for _, p := range parentPerms {
            perms[p] = true
        }
    }
    
    return keys(perms)
}
```

### Role Hierarchy Configuration

```yaml
rbac:
  hierarchy:
    platform-admin:
      permissions: ["*"]
      inherits: []
    security-admin:
      permissions: ["security.*", "audit.*"]
      inherits: ["user-admin"]
    user-admin:
      permissions: ["users.*", "roles.*"]
      inherits: ["user-reader"]
    user-reader:
      permissions: ["users.read"]
      inherits: []
```

## GGID Access Control Architecture

### Full Stack

```
Client Request
    │
    ▼
┌─────────┐
│ Gateway  │ Layer 1: JWT validation, route authz, tenant check
│ (PEP 1) │
└────┬────┘
     │
     ▼
┌─────────┐
│ Service  │ Layer 2: RBAC + ABAC policy evaluation
│ (PEP 2) │
└────┬────┘
     │
     ▼
┌─────────┐
│  PDP     │ Policy Decision Point
│ Engine   │ - Evaluate policies
│          │ - Check decision cache
└────┬────┘
     │
     ▼
┌─────────┐
│  PIP     │ Policy Information Point
│          │ - Gather subject attributes (JWT/LDAP)
│          │ - Gather resource attributes (DB)
│          │ - Gather environment attributes
└────┬────┘
     │
     ▼
┌─────────┐
│ Database │ Layer 3: Row-Level Security
│ (PEP 3) │ - Tenant isolation
│          │ - Data classification
└─────────┘
```

### Configuration

```yaml
access_control:
  model: "hybrid"  # RBAC + ABAC
  enforcement:
    gateway: true    # Layer 1
    service: true    # Layer 2
    data: true       # Layer 3 (RLS)
  pdp:
    enabled: true
    cache:
      enabled: true
      ttl: 5m
      max_entries: 100000
  pip:
    attribute_sources:
      jwt: true
      ldap: false
      database: true
    cache_ttl: 5m
  rbac:
    hierarchy: true
    inheritance: true
  abac:
    combiner: "permit-unless-deny"
  audit:
    log_all_decisions: true
    log_trace: true
```

## Best Practices

1. **Layer defense** — Don't rely on a single enforcement point
2. **Cache decisions** — Policy evaluation is expensive
3. **Invalidate on change** — Clear cache when roles/policies change
4. **Use PDP/PEP separation** — Decouple decision from enforcement
5. **Audit all decisions** — Full trail for compliance
6. **Default deny** — When no policy matches, deny
7. **Keep policies simple** — Complex policies are hard to maintain
8. **Test with dry-run** — Verify policy changes before deploying
9. **Monitor performance** — Track decision latency
10. **Use RBAC as base** — Add ABAC for fine-grained exceptions