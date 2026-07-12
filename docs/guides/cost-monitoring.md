# Cost Monitoring

Per-service cost attribution, resource tagging, idle detection, right-sizing, budget alerts, and FinOps dashboard.

## Cost Attribution

### Resource Tagging

```yaml
tags:
  project: "ggid"
  environment: "prod"
  service: "auth"
  team: "platform"
  cost_center: "CC-1001"
```

All resources (EC2, RDS, S3, NATS, LB) tagged for accurate per-service attribution.

### Per-Service Cost

| Service | Monthly Est. | Resources |
|---------|-------------|-----------|
| Gateway | $200 | 3 pods + LB |
| Auth | $300 | 3 pods + Redis |
| Identity | $250 | 2 pods + DB share |
| Policy | $200 | 2 pods + Redis |
| Audit | $350 | 2 pods + NATS + DB storage |
| Console | $100 | 1 pod + CDN |
| PostgreSQL | $500 | RDS instance |
| Redis | $150 | ElastiCache |
| NATS | $100 | JetStream storage |
| **Total** | **$2150** | |

## Idle Detection

```bash
# Find idle resources
GET /api/v1/admin/cost/idle-resources
# → [
#   {"resource": "staging-db-replica", "cpu_avg": "2%", "monthly_cost": 80, "recommendation": "scale_down"},
#   {"resource": "dev-redis", "connections": 0, "monthly_cost": 30, "recommendation": "terminate"},
#   {"resource": "old-snapshots", "count": 120, "monthly_cost": 45, "recommendation": "cleanup"}
# ]
```

## Right-Sizing

```yaml
right_sizing_rules:
  - metric: cpu_avg
    threshold: "<20% for 14 days"
    action: "recommend smaller instance"
    
  - metric: memory_avg
    threshold: "<30% for 14 days"
    action: "reduce memory request"
    
  - metric: cpu_p99
    threshold: ">90% regularly"
    action: "increase replicas or size"
```

## Budget Alerts

```yaml
budgets:
  - name: "production-monthly"
    limit: 3000
    alerts:
      - threshold: 80%
        channel: "#finops"
        message: "Production budget at 80%"
      - threshold: 100%
        channel: "#finops"
        page: true
        message: "Production budget exceeded"
  
  - name: "development-monthly"
    limit: 500
    alerts:
      - threshold: 90%
        action: "auto-scale-down-non-critical"
```

## FinOps Dashboard

```
┌──────────────────────────────────────┐
│  FINOPS DASHBOARD                     │
│                                       │
│  This Month: $1,847 / $3,000 (62%)   │
│  ████████████░░░░░░░░░░               │
│                                       │
│  By Service:                          │
│  PostgreSQL  ████████████ $500        │
│  Auth        ██████ $300              │
│  Audit       ███████ $350             │
│  ...                                  │
│                                       │
│  Potential Savings:                   │
│  • Scale down staging DB replica: $80 │
│  • Clean old snapshots: $45           │
│  • Right-size dev Redis: $20          │
│  Total: $145/month                    │
│                                       │
│  Trend: ↑5% vs last month             │
└──────────────────────────────────────┘
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Monthly spend | >80% budget → warn |
| Daily spend | >daily_avg × 2 → anomaly |
| Idle resources | Any → recommend cleanup |
| Cost per user | Track trend |

## See Also

- [Auto-Scaling Strategy](auto-scaling-strategy.md)
- [Infrastructure as Code](infrastructure-as-code.md)
- [Connection Pool Tuning](connection-pool-tuning.md)
- [SRE Practices](sre-practices.md)
