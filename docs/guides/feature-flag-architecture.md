# Feature Flag Architecture

Flag types, evaluation engine, rollout strategies, A/B testing, kill switches, gradual rollout, and per-tenant flags.

## Flag Types

| Type | Purpose | TTL | Example |
|------|---------|-----|---------|
| Release | Gate new features | Days-weeks | `new_auth_flow` |
| Ops | Operational control | Permanent | `maintenance_mode` |
| Experiment | A/B testing | Weeks | `checkout_layout_v2` |
| Permission | Entitlement | Permanent | `premium_analytics` |

## Evaluation Engine

```go
type FlagEngine struct {
    store FlagStore
    cache *ristretto.Cache
}

func (e *FlagEngine) Evaluate(ctx context.Context, flagKey string, user UserContext) (bool, error) {
    flag := e.store.Get(flagKey)
    if flag == nil { return false, nil } // Default: off
    
    switch flag.Strategy {
    case "boolean":
        return flag.Enabled, nil
        
    case "percentage":
        return hashPercentage(user.ID, flag.Key) < flag.Percentage, nil
        
    case "allowlist":
        return contains(flag.AllowedUsers, user.ID), nil
        
    case "tenant":
        return contains(flag.AllowedTenants, user.TenantID), nil
        
    case "attribute":
        return evalAttributeRule(flag.Rule, user.Attributes), nil
        
    case "kill_switch":
        return flag.Enabled && !flag.Killed, nil
        
    default:
        return false, nil
    }
}

// Consistent hashing ensures same user always gets same result
func hashPercentage(userID, flagKey string) float64 {
    h := fnv.New32()
    h.Write([]byte(userID + flagKey))
    return float64(h.Sum32()%10000) / 100.0 // 0.0 - 99.99
}
```

## Rollout Strategies

### Gradual Rollout

```yaml
flag: "new_auth_flow"
strategy: "percentage"
rollout:
  - day_1: 1%     # Canary
  - day_2: 5%     # If no errors
  - day_3: 25%    # Quarter of users
  - day_5: 50%    # Half
  - day_7: 100%   # Full rollout
auto_rollback:
  error_rate_threshold: 0.02
  latency_increase_threshold: 1.5
```

### Per-Tenant Rollout

```yaml
flag: "premium_analytics"
strategy: "tenant"
allowed_tenants:
  - "internal-test"      # Day 1
  - "beta-customer-1"    # Day 3
  - "enterprise-tier"    # Day 7
  - "*"                  # Day 14
```

### Ring Rollout

```yaml
flag: "new_db_engine"
rings:
  - ring: 0          # Internal
    users: ["internal-*"]
  - ring: 1          # Early adopters
    tenants: ["beta-*"]
  - ring: 2          # 10% of all
    percentage: 10
  - ring: 3          # 50%
    percentage: 50
  - ring: 4          # 100%
    percentage: 100
```

## A/B Testing

```yaml
flag: "checkout_layout"
strategy: "experiment"
variants:
  - name: "control"
    weight: 50
    value: "v1_layout"
  - name: "treatment"
    weight: 50
    value: "v2_layout"
    
metrics:
  primary: "checkout_completion_rate"
  secondary: ["time_to_checkout", "error_rate"]
  duration: "14_days"
```

### Implementation

```go
variant := flagEngine.GetVariant(ctx, "checkout_layout", userContext)
switch variant {
case "v1_layout":
    renderCheckoutV1(w)
case "v2_layout":
    renderCheckoutV2(w)
}

// Track which variant user saw
analytics.Track("checkout_viewed", map[string]interface{}{
    "variant": variant,
    "user_id": userContext.ID,
})
```

## Kill Switches

```yaml
flag: "oauth_introspection"
type: "kill_switch"
enabled: true
killed: false    # Set to true to instantly disable

# When killed=true:
# - Feature returns false for ALL users
# - No evaluation logic runs
# - Near-zero latency
# - Logged immediately
```

### Emergency Kill

```bash
# Instantly disable a feature
POST /api/v1/admin/flags/oauth_introspection/kill
{"reason": "performance degradation", "killed_by": "on-call"}
# → All requests now skip introspection, fall back to JWT-only
```

## Configuration

### Flag Definition

```bash
POST /api/v1/admin/flags
{
  "key": "new_auth_flow",
  "type": "release",
  "strategy": "percentage",
  "percentage": 5,
  "description": "New authentication flow with WebAuthn-first UX",
  "created_by": "product-team",
  "expires_at": "2025-03-01"
}
```

### Client-Side Evaluation

```javascript
// SDK evaluates flags client-side (bootstrapped from server)
const flags = ggid.getFlags({ user });
if (flags.newAuthFlow) {
    renderNewAuthFlow();
} else {
    renderLegacyAuthFlow();
}
```

## Hot Reload

```go
// Flags reload from DB every 30 seconds (no restart needed)
func (e *FlagEngine) StartReloader(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    for {
        select {
        case <-ticker.C:
            e.reloadFlags()
        case <-ctx.Done():
            return
        }
    }
}

func (e *FlagEngine) reloadFlags() {
    flags := e.store.LoadAll()
    e.cache.Clear()
    for _, f := range flags {
        e.flags.Store(f.Key, f)
    }
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Flag evaluation latency | <1ms (cached) |
| Flag changes | Log all with who/when |
| Expired flags | Auto-disable + alert |
| Unused flags | No evaluations in 7 days → cleanup |
| Experiment significance | p < 0.05 → declare winner |

## See Also

- [Canary Deployment Strategy](canary-deployment-strategy.md)
- [Policy Hot Reload](policy-hot-reload.md)
- [Zero-Downtime Deployment](zero-downtime-deployment.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
