# Policy Engine Internals

RBAC evaluation, ABAC condition matching, policy precedence, caching, decision logging, performance optimization, and dry-run architecture.

## Architecture

```
Request → Policy Engine
    │
    ├── Collect Context (user attributes, resource attrs, environment)
    ├── Evaluate RBAC (role → permission)
    ├── Evaluate ABAC (attribute conditions)
    ├── Apply Precedence (deny overrides allow)
    ├── Decision: Allow / Deny
    │
    ├── → Cache decision (if cacheable)
    ├── → Log decision
    └── → Return to caller
```

## RBAC Evaluation

### Role-Permission Mapping

```go
type Role struct {
    ID          string
    Name        string
    Permissions []Permission
    Inherits    []string  // Parent role IDs
}

type Permission struct {
    Resource  string   // "users", "roles", "*"
    Actions   []string // "read", "write", "delete", "*"
}
```

### Evaluation

```go
func (e *Engine) evaluateRBAC(userID, resource, action string) bool {
    roles := e.getUserRoles(userID)
    for _, role := range roles {
        perms := e.resolvePermissions(role) // Includes inherited
        for _, perm := range perms {
            if e.matchResource(perm.Resource, resource) &&
               contains(perm.Actions, action) || contains(perm.Actions, "*") {
                return true
            }
        }
    }
    return false
}
```

### Role Inheritance

```
Super Admin → Tenant Admin → Department Admin → User
    ↓              ↓                ↓
  everything   tenant-scoped     dept-scoped
```

Inheritance resolution is memoized per request to avoid repeated DB lookups.

## ABAC Condition Matching

### Policy Definition

```yaml
policies:
  - name: "allow-engineering-access"
    effect: allow
    condition: |
      user.department == "engineering"
      && resource.type == "project"
      && action == "read"
      && time.hour >= 9 && time.hour < 18

  - name: "deny-offhours-delete"
    effect: deny
    condition: |
      action == "delete"
      && (time.hour < 6 || time.hour > 22)
      && user.clearance != "admin"
```

### CEL Evaluation

```go
func (e *Engine) evaluateABAC(ctx context.Context, policy Policy) (bool, error) {
    env := map[string]interface{}{
        "user":     ctx.UserAttributes,
        "resource": ctx.ResourceAttributes,
        "action":   ctx.Action,
        "time":     time.Now(),
        "env":      ctx.Environment,
    }

    prog, err := cel.Compile(policy.Condition)
    if err != nil {
        return false, fmt.Errorf("policy compile error: %w", err)
    }

    result, err := prog.Eval(env)
    return result.Value().(bool), err
}
```

### Attribute Sources

| Attribute | Source | Example |
|-----------|--------|---------|
| `user.department` | PostgreSQL users table | "engineering" |
| `user.clearance` | JWT claim | "secret" |
| `user.risk_score` | Redis (real-time) | 25 |
| `resource.type` | Request path | "project" |
| `resource.owner` | Resource DB record | user UUID |
| `time.hour` | System clock | 14 |
| `env.ip_range` | Request IP | "10.0.0.0/8" |

## Policy Precedence

### Evaluation Order

```
1. Explicit DENY (highest priority)
2. Conditional DENY
3. Explicit ALLOW
4. Conditional ALLOW
5. Default DENY (lowest — secure by default)
```

### Conflict Resolution

```go
func resolvePolicies(decisions []PolicyDecision) Decision {
    for _, d := range decisions {
        if d.Effect == Deny {
            return Deny  // Any deny wins
        }
    }
    for _, d := range decisions {
        if d.Effect == Allow {
            return Allow
        }
    }
    return Deny  // Default deny
}
```

### Priority Override

```yaml
policies:
  - name: "break-glass-override"
    priority: 1000        # Higher = evaluated first
    effect: allow
    condition: "user.break_glass == true"

  - name: "normal-access"
    priority: 100
    effect: allow
    condition: "..."
```

## Caching Layer

### Decision Cache

```go
type DecisionCache struct {
    cache *ristretto.Cache  // In-memory LRU
    ttl   time.Duration
}

func (dc *DecisionCache) Get(userID, resource, action string) (Decision, bool) {
    key := hash(userID + "|" + resource + "|" + action)
    if d, ok := dc.cache.Get(key); ok {
        return d.(Decision), true
    }
    return Deny, false
}
```

### Cache Invalidation

| Event | Invalidation Scope |
|-------|-------------------|
| Role assigned/revoked | All decisions for that user |
| Policy created/updated/deleted | All decisions for that resource type |
| User attributes changed | All decisions for that user |
| Time-based (hour boundary) | Time-dependent policies only |

```go
// On role change
func (e *Engine) OnRoleChange(userID string) {
    e.cache.DelPrefix("user:" + userID)
    audit.Log("cache_invalidated", userID)
}
```

### Cache Hit Rate

| Scenario | Expected Hit Rate |
|----------|------------------|
| Read-heavy (users listing) | >95% |
| Write-heavy (admin operations) | 60-70% |
| First request for user | 0% (cold) |
| Repeated identical request | 100% |

## Decision Logging

```json
{
  "decision_id": "dec-uuid",
  "timestamp": "2025-01-15T10:30:00Z",
  "user_id": "uuid",
  "resource": "users",
  "action": "read",
  "decision": "allow",
  "policies_evaluated": [
    {"name": "allow-engineering-access", "matched": true, "effect": "allow"},
    {"name": "deny-offhours-delete", "matched": false}
  ],
  "evaluation_time_ms": 0.8,
  "cache_hit": false
}
```

Decisions are logged to the audit pipeline (NATS JetStream) for compliance and forensic analysis.

## Performance Optimization

| Technique | Impact | Implementation |
|-----------|--------|----------------|
| Decision cache | 95%+ latency reduction | Ristretto LRU, 5min TTL |
| Policy compilation cache | Avoid recompiling CEL | Compile once, eval many |
| Role permission memoization | Avoid DB lookups | Per-request map |
| Batch attribute lookup | Reduce DB round-trips | Single query for all attrs |
| Early termination | Skip remaining on deny | Short-circuit evaluation |

### Benchmark Targets

| Operation | Target | Measured |
|-----------|--------|----------|
| RBAC only (cached) | <0.1ms | 0.05ms |
| RBAC + ABAC (cached) | <0.5ms | 0.3ms |
| RBAC + ABAC (uncached) | <5ms | 2.1ms |
| Full eval with logging | <8ms | 4.5ms |

## Dry-Run Architecture

Dry-run evaluates a policy without enforcing it — for testing and impact analysis:

```bash
POST /api/v1/policy/dry-run
{
  "policy_id": "pol-new-restrict",
  "test_users": ["user-1", "user-2", "user-3"],
  "test_actions": ["users:read", "users:write"],
  "test_resources": ["users"]
}
# → {
#   "results": [
#     {"user":"user-1","action":"users:read","current":"allow","dry_run":"deny","changed":true},
#     {"user":"user-2","action":"users:read","current":"allow","dry_run":"allow","changed":false}
#   ],
#   "summary": {"total":6,"changed":1,"unchanged":5}
# }
```

### How Dry-Run Works

```
1. Evaluate current policy set → record decisions
2. Add/update test policy in memory (not persisted)
3. Re-evaluate with new policy set → record decisions
4. Compare before/after → report changes
5. Discard in-memory changes
```

This lets admins see the blast radius of a policy change before applying it.

## See Also

- [Conditional Access](conditional-access.md)
- [OAuth Client Scoped Permissions](oauth-client-scoped-permissions.md)
- [Delegated Administration](delegated-administration.md)
- [Audit Query API](audit-query-api.md)
