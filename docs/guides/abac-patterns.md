# ABAC Design Patterns

Common ABAC patterns — condition composition, attribute precedence, time-based access, geo-fencing, device posture, decision caching.

## Condition Composition Patterns

### Pattern: Department + Clearance

```json
{
  "operator": "and",
  "items": [
    {"attribute": "department", "operator": "eq", "value": "engineering"},
    {"attribute": "clearance", "operator": "gte", "value": 3}
  ]
}
```

### Pattern: Manager OR Owner

```json
{
  "operator": "or",
  "items": [
    {"attribute": "user_id", "operator": "eq", "value": "{{resource.owner_id}}"},
    {"attribute": "department", "operator": "eq", "value": "management"}
  ]
}
```

### Pattern: Business Hours + Trusted Network

```json
{
  "operator": "and",
  "items": [
    {"attribute": "time", "operator": "in", "value": "business_hours"},
    {"attribute": "ip_address", "operator": "cidr", "value": "10.0.0.0/8"}
  ]
}
```

## Time-Based Access

| Pattern | Rule | Use Case |
|---------|------|----------|
| Business hours | Mon-Fri 9-18 | Standard access |
| Maintenance window | Sat 2-4 AM | Admin operations |
| Trading hours | 9:30-16:00 ET (NYSE) | Financial |
| Off-hours block | 22:00-06:00 | Sensitive ops |

## Geo-Fencing

```json
{"attribute": "geo_country", "operator": "in", "value": ["US", "CA", "UK", "AU"]}
```

Or deny specific countries:
```json
{"attribute": "geo_country", "operator": "not_in", "value": ["CN", "RU", "IR", "KP"]}
```

## Decision Caching

```go
cacheKey := fmt.Sprintf("policy:%s:%s:%s", userID, resource, action)
if cached := redis.Get(cacheKey); cached != nil { return cached }
result := policyEngine.Check(...)
redis.Set(cacheKey, result, 5*time.Minute)
// Invalidate on: role change, policy update, user update
```

## Best Practices

1. Start with RBAC, add ABAC selectively
2. Keep nesting <= 3 levels
3. Cache decisions (5 min TTL)
4. Test with dry-run before deploy
5. Review quarterly

## See Also

- [ABAC Condition Builder](abac-condition-builder.md)
- [Policy API](../api/policy.md)
