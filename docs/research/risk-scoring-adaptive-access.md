# Risk Scoring & Adaptive Access

> Risk-based authentication model for GGID: score requests and enforce step-up auth.

---

## Competitor Analysis

### Okta ISPM
- Continuous risk scoring (0-100)
- Factors: IP reputation, device, location, behavior
- Auto-remediation (force MFA, revoke session)

### Auth0 Anomaly Detection
- Brute-force detection (5+ failures)
- Impossible travel (geo-distance check)
- Stuffed credentials (HaveIBeenPwned)

### Azure AD Conditional Access
- Signal-based policies (user, device, location, app)
- Session risk: low/medium/high
- Auto-apply MFA when risk > threshold

---

## GGID Risk Score Design

### Scoring Factors

| Factor | Weight | Data Source |
|--------|--------|-------------|
| Failed login count | 30 | Audit events (Redis counter) |
| New IP address | 20 | Session IP vs history |
| Impossible travel | 25 | Geo-distance between logins |
| New device | 15 | WebAuthn credential ID |
| Off-hours access | 10 | Time of day |

### Risk Calculation (Go)

```go
type RiskEngine struct {
    redis *redis.Client
}

func (e *RiskEngine) Score(ctx context.Context, userID, ip string) int {
    score := 0

    // Failed login count (last hour)
    fails := e.redis.Get(ctx, "login_fails:"+userID).Val()
    if n, _ := strconv.Atoi(fails); n >= 5 {
        score += 30
    }

    // New IP
    known := e.redis.SIsMember(ctx, "known_ips:"+userID, ip).Val()
    if !known {
        score += 20
    }

    // Off-hours
    hour := time.Now().Hour()
    if hour < 6 || hour > 22 {
        score += 10
    }

    return score // 0-100
}
```

### Adaptive Actions

| Score | Action |
|-------|--------|
| 0-20 | Allow (low risk) |
| 21-50 | Require MFA (medium risk) |
| 51-80 | Require WebAuthn (high risk) |
| 81-100 | Deny + alert admin (critical) |

### ABAC Integration

```bash
curl -X POST .../api/v1/policies \
  -d '{
    "name": "High risk requires WebAuthn",
    "effect": "deny",
    "actions": ["*"],
    "resources": ["*"],
    "priority": 300,
    "condition": "risk_score > 50 AND mfa_verified == false"
  }'
```

---

## Implementation Estimate

| Component | Effort |
|-----------|--------|
| Risk engine (scoring) | 2 days |
| IP history tracking (Redis) | 1 day |
| Impossible travel (geo-distance) | 1 day |
| ABAC condition integration | 1 day |
| Console UI (risk dashboard) | 2 days |
| **Total** | **7 days** |

Priority: P2.

---

*See: [Realtime Alerting Design](realtime-alerting-design.md) | [ABAC Policy](../guides/abac-policy.md) | [Security Overview](../architecture/security-overview.md)*

*Last updated: 2025-07-11*
