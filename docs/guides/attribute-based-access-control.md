# Attribute-Based Access Control

This guide covers the ABAC policy model, attribute categories, policy combinators, XACML comparison, policy expression language, nested conditions, attribute sourcing, performance optimization, and GGID's ABAC engine.

## Overview

ABAC makes access control decisions based on attributes of the subject (user), resource, action, and environment. Unlike RBAC (which uses roles only), ABAC can express fine-grained policies like "allow managers to approve expenses under $10,000 during business hours from corporate network."

## Policy Model

### Four Attribute Categories

```
┌─────────────────────────────────────────────┐
│              ACCESS DECISION                  │
├──────────┬──────────┬──────────┬───────────┤
│ Subject  │ Resource │ Action   │ Environment│
│ Attributes│ Attributes│Attributes│ Attributes │
├──────────┼──────────┼──────────┼───────────┤
│ role     │ type     │ name     │ time       │
│ dept     │ owner    │ category │ location   │
│ clearance│ sensitivity│ method  │ network    │
│ manager  │ tenant   │          │ device     │
│ country  │ tags     │          │ session    │
└──────────┴──────────┴──────────┴───────────┘
```

### Subject Attributes

| Attribute | Source | Example |
|---|---|---|
| `role` | JWT claim | "admin", "developer" |
| `department` | JWT claim / LDAP | "Engineering", "Sales" |
| `clearance` | DB | "secret", "top-secret" |
| `manager` | HR system | "true", "false" |
| `country` | IP geolocation | "US", "UK" |
| `mfa_verified` | Session | true, false |
| `tenant_id` | JWT claim | "tenant-uuid" |

### Resource Attributes

| Attribute | Source | Example |
|---|---|---|
| `type` | Resource metadata | "document", "api", "database" |
| `owner` | Resource metadata | "user-uuid" |
| `sensitivity` | Classification | "public", "confidential", "restricted" |
| `tenant_id` | Resource metadata | "tenant-uuid" |
| `tags` | Resource labels | ["finance", "pii"] |
| `department` | Resource metadata | "Engineering" |

### Action Attributes

| Attribute | Values |
|---|---|
| `name` | read, write, delete, create, execute, approve |
| `category` | read-only, modify, admin |

### Environment Attributes

| Attribute | Source | Example |
|---|---|---|
| `time` | System clock | "business_hours", "after_hours" |
| `day` | System clock | "weekday", "weekend" |
| `location` | IP geolocation | "corporate", "remote", "public" |
| `network` | IP classification | "corporate", "vpn", "public" |
| `device` | Device fingerprint | "managed", "byod", "unknown" |
| `session_age` | Session metadata | "fresh", "old" |

## Policy Combinators

### Combining Algorithms

When multiple policies apply, a combinator determines the final decision:

| Combinator | Logic | Default |
|---|---|---|
| permit-unless-deny | If any deny → deny; else permit | GGID default |
| deny-unless-permit | If any permit → permit; else deny | Conservative |
| permit-overrides | Any permit wins over deny | Liberal |
| deny-overrides | Any deny wins over permit | Most common |
| first-applicable | First matching rule wins | Ordered |

### Examples

```yaml
policies:
  - name: "allow-managers-business-hours"
    combiner: "permit-unless-deny"
    rules:
      - effect: permit
        condition: "subject.role == 'manager' AND env.time == 'business_hours'"
      - effect: deny
        condition: "subject.role != 'manager'"
```

## XACML Comparison

| Aspect | XACML | GGID ABAC |
|---|---|---|
| Format | XML | YAML/JSON |
| Complexity | Very high | Moderate |
| Performance | Slow (XML parsing) | Fast (native Go) |
| Standardization | OASIS standard | Custom |
| Interoperability | High (cross-vendor) | GGID-specific |
| Policy exchange | XML-based | YAML/JSON |
| Learning curve | Steep | Gentle |

### When to Use XACML

- Cross-vendor policy exchange required
- Regulatory compliance mandates XACML
- Integration with XACML-based PDPs

### When to Use GGID ABAC

- Internal policy management
- Performance is critical
- Developer-friendly policy authoring
- Integration with GGID's audit and risk engine

## Policy Expression Language

### Syntax

```
<attribute> <operator> <value>
```

### Operators

| Operator | Description | Example |
|---|---|---|
| `==` | Equal | `subject.role == "admin"` |
| `!=` | Not equal | `subject.dept != "Sales"` |
| `in` | Contained in | `subject.role in ["admin", "manager"]` |
| `not in` | Not contained in | `subject.country not in ["CN", "RU"]` |
| `>` | Greater than | `resource.amount > 10000` |
| `<` | Less than | `resource.sensitivity < "confidential"` |
| `>=` | Greater or equal | `subject.clearance >= "secret"` |
| `<=` | Less or equal | `env.time <= "18:00"` |
| `contains` | Array contains | `resource.tags contains "pii"` |
| `matches` | Regex match | `resource.name matches "^user.*"` |

### Logical Operators

| Operator | Example |
|---|---|
| `AND` | `subject.role == "admin" AND env.time == "business_hours"` |
| `OR` | `subject.role == "admin" OR subject.role == "manager"` |
| `NOT` | `NOT env.location == "public"` |

## Nested Conditions

### AND/OR Nesting

```yaml
policy:
  name: "expense-approval"
  effect: permit
  condition:
    and:
      - subject.role in ["manager", "director"]
      - action.name == "approve"
      - resource.type == "expense"
      - or:
          - and:
              - resource.amount <= 10000
              - env.time == "business_hours"
          - and:
              - resource.amount <= 50000
              - subject.role == "director"
              - env.network == "corporate"
```

This reads as:
- Manager or director
- Action is approve
- Resource is expense
- AND either:
  - Amount ≤ $10,000 during business hours, OR
  - Amount ≤ $50,000 by director on corporate network

### Condition Tree

```go
type Condition interface {
    Evaluate(ctx *EvalContext) bool
}

type AndCondition struct {
    Conditions []Condition
}

func (c *AndCondition) Evaluate(ctx *EvalContext) bool {
    for _, cond := range c.Conditions {
        if !cond.Evaluate(ctx) {
            return false
        }
    }
    return true
}

type OrCondition struct {
    Conditions []Condition
}

func (c *OrCondition) Evaluate(ctx *EvalContext) bool {
    for _, cond := range c.Conditions {
        if cond.Evaluate(ctx) {
            return true
        }
    }
    return false
}

type Comparison struct {
    Attribute string
    Operator  string
    Value     interface{}
}

func (c *Comparison) Evaluate(ctx *EvalContext) bool {
    actual := ctx.GetAttribute(c.Attribute)
    return compare(actual, c.Operator, c.Value)
}
```

## Attribute Sourcing

### JWT Claims (Primary)

```go
func getSubjectAttributes(token string) map[string]interface{} {
    claims := parseJWT(token)
    return map[string]interface{}{
        "role":        claims["roles"],
        "department":  claims["department"],
        "tenant_id":   claims["tenant_id"],
        "mfa_verified": claims["mfa_verified"],
        "sub":         claims["sub"],
    }
}
```

### LDAP Directory

```go
func getLDAPAttributes(userDN string) map[string]interface{} {
    entry, _ := ldap.Search(userDN)
    return map[string]interface{}{
        "manager":    entry.GetAttributeValue("manager"),
        "department": entry.GetAttributeValue("departmentNumber"),
        "title":      entry.GetAttributeValue("title"),
        "country":    entry.GetAttributeValue("c"),
    }
}
```

### Database Lookup

```go
func getResourceAttributes(resourceID string) map[string]interface{} {
    resource := db.GetResource(resourceID)
    return map[string]interface{}{
        "type":        resource.Type,
        "owner":       resource.OwnerID,
        "sensitivity": resource.Classification,
        "tags":        resource.Tags,
        "tenant_id":   resource.TenantID,
    }
}
```

### Environment (Computed)

```go
func getEnvironmentAttributes(r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "time":     classifyTime(time.Now()),
        "day":      classifyDay(time.Now()),
        "location": classifyLocation(clientIP(r)),
        "network":  classifyNetwork(clientIP(r)),
        "device":   classifyDevice(r),
    }
}
```

## Performance Optimization

### Attribute Caching

```go
type AttributeCache struct {
    cache Cache
    ttl   time.Duration
}

func (c *AttributeCache) GetSubjectAttributes(userID string) map[string]interface{} {
    key := "abac:subject:" + userID
    if val, ok := c.cache.Get(key); ok {
        return val.(map[string]interface{})
    }
    attrs := fetchSubjectAttributes(userID)
    c.cache.Set(key, attrs, c.ttl)
    return attrs
}
```

### Policy Indexing

```go
type PolicyIndex struct {
    byAction  map[string][]*Policy
    byResource map[string][]*Policy
}

func (idx *PolicyIndex) FindPolicies(action, resourceType string) []*Policy {
    // Only evaluate policies that match the action and resource type
    actionPolicies := idx.byAction[action]
    resourcePolicies := idx.byResource[resourceType]
    return intersect(actionPolicies, resourcePolicies)
}
```

### Decision Caching

```go
func (e *ABACEngine) Evaluate(ctx *EvalContext) string {
    // Cache key: hash of all attributes + action + resource
    cacheKey := buildCacheKey(ctx)
    if decision, ok := e.decisionCache.Get(cacheKey); ok {
        return decision.(string)
    }

    // Evaluate
    decision := e.evaluatePolicies(ctx)

    // Cache (short TTL — attributes may change)
    e.decisionCache.Set(cacheKey, decision, 5*time.Minute)

    return decision
}
```

### Performance Targets

| Metric | Target |
|---|---|
| Policy evaluation | <1ms |
| Attribute lookup (cached) | <0.1ms |
| Attribute lookup (uncached) | <5ms |
| Decision cache hit rate | >90% |

## GGID ABAC Engine

### Policy Definition

```yaml
abac:
  policies:
    - name: "confidential-data-access"
      description: "Access to confidential data requires MFA + corporate network"
      effect: permit
      condition:
        and:
          - resource.sensitivity == "confidential"
          - subject.mfa_verified == true
          - env.network in ["corporate", "vpn"]
          - action.name in ["read", "write"]

    - name: "restricted-data-access"
      description: "Restricted data requires security clearance + managed device"
      effect: permit
      condition:
        and:
          - resource.sensitivity == "restricted"
          - subject.clearance >= "secret"
          - subject.mfa_verified == true
          - env.device == "managed"
          - env.network == "corporate"
          - action.name in ["read"]

    - name: "after-hours-block"
      description: "Block write operations after hours for non-admins"
      effect: deny
      condition:
        and:
          - env.time == "after_hours"
          - subject.role not in ["admin", "security-admin"]
          - action.category == "modify"

    - name: "default-deny"
      description: "Deny by default"
      effect: deny
      condition: "true"

  combiner: "permit-unless-deny"
```

### Configuration

```yaml
abac:
  enabled: true
  combiner: "permit-unless-deny"
  caching:
    attributes:
      enabled: true
      ttl: 5m
    decisions:
      enabled: true
      ttl: 5m
      max_entries: 100000
  performance:
    target_eval_time: "1ms"
    target_cache_hit: 0.9
  audit:
    log_all_decisions: true
    log_trace: true  # Include evaluation trace
```

### API

```bash
POST /api/v1/policy/evaluate
Authorization: Bearer <token>

{
  "subject": { "role": "manager", "department": "Engineering" },
  "resource": { "type": "document", "sensitivity": "confidential" },
  "action": { "name": "read" },
  "environment": { "time": "business_hours", "network": "corporate" }
}

Response:
{
  "decision": "permit",
  "matched_policy": "confidential-data-access",
  "trace": [ ... ],
  "evaluated_at": "2026-07-12T10:00:00Z"
}
```

## Best Practices

1. **Start with RBAC, add ABAC for fine-grained** — RBAC as base, ABAC for exceptions
2. **Use permit-unless-deny** — Default to open, deny specific cases
3. **Cache attributes** — Attribute lookups are expensive
4. **Cache decisions** — Same inputs → same output
5. **Index policies** — Don't evaluate all policies for every request
6. **Audit all decisions** — Full trace for compliance
7. **Test policies with dry-run** — Verify before deploying
8. **Keep policies readable** — Complex nesting is hard to maintain
9. **Version policies** — Track changes, enable rollback
10. **Monitor performance** — ABAC can be slow if not optimized
