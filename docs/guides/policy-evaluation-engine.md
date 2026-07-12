# Policy Evaluation Engine Design

RBAC fast-path, ABAC CEL evaluation, caching layer, decision tree optimization, hot path analysis, and performance benchmarks.

## Architecture

```
Request → Policy Engine
    │
    ├── 1. RBAC Fast-Path (cached, <0.1ms)
    │      └── Check role → permission map in cache
    │
    ├── 2. ABAC Evaluation (if RBAC inconclusive)
    │      └── Evaluate CEL conditions against attributes
    │
    ├── 3. Precedence Resolution
    │      └── deny > allow, higher priority wins
    │
    ├── 4. Decision Logging
    │      └── Record to audit pipeline
    │
    └── Return: Allow / Deny
```

## RBAC Fast-Path

Most access checks are simple RBAC lookups. The fast-path handles these in <0.1ms:

```go
func (e *Engine) evaluateRBAC(userID, resource, action string) (Decision, bool) {
    // Check cache first
    key := fmt.Sprintf("rbac:%s:%s:%s", userID, resource, action)
    if decision, ok := e.cache.Get(key); ok {
        return decision.(Decision), true // Cache hit
    }
    
    // Resolve effective permissions (with inheritance)
    roles := e.getUserRoles(userID)
    for _, role := range roles {
        perms := e.resolvePermissions(role) // Memoized per request
        for _, perm := range perms {
            if matchResource(perm.Resource, resource) && matchAction(perm.Actions, action) {
                e.cache.Set(key, Allow, 5*time.Minute)
                return Allow, true
            }
        }
    }
    
    return Deny, false // Fall through to ABAC
}
```

### Fast-Path Coverage

~85% of all access decisions are resolved by RBAC fast-path with cache hit.

## ABAC CEL Evaluation

For decisions that need context beyond roles:

```go
func (e *Engine) evaluateABAC(ctx EvalContext) Decision {
    matchingPolicies := e.findMatchingPolicies(ctx.Resource, ctx.Action)
    
    decisions := []PolicyDecision{}
    for _, policy := range matchingPolicies {
        // Use compiled CEL program (cached)
        prog := e.celCache.GetOrCompile(policy.Condition)
        
        env := map[string]interface{}{
            "user":     ctx.UserAttributes,
            "resource": ctx.ResourceAttributes,
            "action":   ctx.Action,
            "time":     time.Now(),
        }
        
        result, err := prog.Eval(env)
        if err != nil {
            audit.Log("policy.eval_error", policy, err)
            continue
        }
        
        if result.Value().(bool) {
            decisions = append(decisions, PolicyDecision{
                Policy: policy.Name,
                Effect: policy.Effect,
            })
        }
    }
    
    return resolvePrecedence(decisions)
}
```

### CEL Compilation Cache

```go
type CELCache struct {
    programs map[string]cel.Program
    mu       sync.RWMutex
}

func (c *CELCache) GetOrCompile(condition string) cel.Program {
    c.mu.RLock()
    if prog, ok := c.programs[condition]; ok {
        c.mu.RUnlock()
        return prog
    }
    c.mu.RUnlock()
    
    // Compile (expensive — only once per condition)
    ast, _ := cel.Compile(condition)
    prog, _ := cel.Program(ast)
    
    c.mu.Lock()
    c.programs[condition] = prog
    c.mu.Unlock()
    
    return prog
}
```

## Decision Tree Optimization

### Policy Indexing

Instead of evaluating all policies for every request, index by resource + action:

```go
type PolicyIndex struct {
    byResource map[string][]*Policy  // resource → policies
    byPattern  []*Policy              // wildcard policies
}

func (idx *PolicyIndex) Find(resource, action string) []*Policy {
    candidates := idx.byResource[resource]    // Exact match
    candidates = append(candidates, idx.byPattern...)  // Wildcards
    
    // Filter by action
    var matching []*Policy
    for _, p := range candidates {
        if p.MatchesAction(action) {
            matching = append(matching, p)
        }
    }
    return matching
}
```

### Optimization Impact

| Without Index | With Index |
|--------------|-----------|
| Evaluate 500 policies | Evaluate 5-10 policies |
| 5ms per evaluation | 0.5ms per evaluation |

## Caching Layer

### Three-Level Cache

```
L1: Per-request memoization (role → permissions map)
    ↓ miss
L2: Decision cache (ristretto, 5min TTL)
    ↓ miss
L3: PostgreSQL (authoritative)
```

### Cache Key Strategy

```go
// L2 decision cache
key := hash(userID + "|" + resource + "|" + action)
// For ABAC, include attribute hash:
key := hash(userID + "|" + resource + "|" + action + "|" + attrHash)
```

### Cache Invalidation

| Event | Scope |
|-------|-------|
| Role assigned/revoked | All decisions for user |
| Policy created/updated | All decisions for resource type |
| User attributes changed | All decisions for user |
| Global policy change | Flush entire cache |

```go
func (e *Engine) OnRoleChange(userID string) {
    e.cache.DelPrefix("user:" + userID + ":")
    audit.Log("cache_invalidated", userID)
}
```

## Hot Path Analysis

Identify which decisions happen most frequently:

```go
type DecisionMetrics struct {
    mu     sync.Mutex
    counts map[string]int  // resource:action → count
}

func (m *DecisionMetrics) Record(resource, action string) {
    key := resource + ":" + action
    m.mu.Lock()
    m.counts[key]++
    m.mu.Unlock()
}

// Report top 20 hottest decisions
func (m *DecisionMetrics) TopN(n int) []DecisionStat {
    // Sort by count descending
    // These are candidates for permanent cache pre-warming
}
```

### Pre-Warming Strategy

```go
// On startup, pre-evaluate top 100 hottest decisions
func (e *Engine) PreWarm() {
    hotPaths := metrics.TopN(100)
    for _, hp := range hotPaths {
        // Evaluate and cache for all users who need it
        for _, userID := range hp.FrequentUsers {
            e.evaluateRBAC(userID, hp.Resource, hp.Action)
        }
    }
}
```

## Precedence Resolution

```go
func resolvePrecedence(decisions []PolicyDecision) Decision {
    // Sort by priority (descending)
    sort.Slice(decisions, func(i, j int) bool {
        return decisions[i].Priority > decisions[j].Priority
    })
    
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
    return Deny  // Default deny (secure by default)
}
```

## Performance Benchmarks

| Scenario | Latency (p50) | Latency (p99) | Cache Hit Rate |
|----------|-------------|-------------|----------------|
| RBAC only (cached) | 0.05ms | 0.1ms | 95% |
| RBAC only (uncached) | 1.2ms | 3.0ms | — |
| RBAC + ABAC (cached) | 0.3ms | 0.8ms | 80% |
| RBAC + ABAC (uncached) | 2.1ms | 5.0ms | — |
| Full eval + logging | 4.5ms | 8.0ms | — |
| Policy with 10 CEL conditions | 1.5ms | 3.5ms | — |
| Policy with 50 CEL conditions | 5.0ms | 12ms | — |

### Benchmark Environment

- Single policy engine instance
- 1000 users, 50 roles, 500 policies
- Ristretto 100K capacity cache
- PostgreSQL 16 with indexes

## Decision Logging

```json
{
  "decision_id": "dec-uuid",
  "timestamp": "2025-01-15T10:30:00.123Z",
  "user_id": "uuid",
  "resource": "users",
  "action": "delete",
  "decision": "deny",
  "reason": "no_matching_allow_policy",
  "policies_evaluated": ["allow-engineering", "deny-offhours"],
  "evaluation_time_ms": 0.8,
  "cache_hit": false,
  "rbac_resolved": false,
  "abac_resolved": true
}
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Decision latency p99 | <5ms | >10ms → optimize |
| Cache hit rate | >90% | <80% → increase TTL or capacity |
| Policy evaluation errors | 0 | Any → CEL syntax bug |
| Deny rate | Track baseline | Spike → possible attack or policy misconfig |

## See Also

- [Policy Engine Internals](policy-engine-internals.md)
- [RBAC Design Patterns](rbac-design-patterns.md)
- [Conditional Access](conditional-access.md)
- [Access Request Lifecycle](access-request-lifecycle.md)
