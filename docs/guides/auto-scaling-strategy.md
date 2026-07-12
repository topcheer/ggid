# Auto-Scaling Strategy

HPA/VPA/KEDA, metrics sources, scaling cooldown, predictive scaling, cost-aware scaling, scale-to-zero, and multi-metric.

## Scaling Approaches

| Tool | Mechanism | Best For |
|------|-----------|---------|
| HPA (Horizontal Pod Autoscaler) | Scale replicas by CPU/memory | Standard workloads |
| VPA (Vertical Pod Autoscaler) | Adjust CPU/memory requests | Stateful services |
| KEDA | Scale by event-driven metrics (queue depth, NATS) | Async workers |
| Cluster Autoscaler | Add/remove nodes | Cluster-level capacity |

## Horizontal Pod Autoscaler (HPA)

### Multi-Metric Config

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: gateway
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target: {type: Utilization, averageUtilization: 70}
  - type: Resource
    resource:
      name: memory
      target: {type: Utilization, averageUtilization: 80}
  - type: Pods
    pods:
      metric: {name: http_requests_per_second}
      target: {type: AverageValue, averageValue: "1000"}
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 30
      policies:
      - {type: Percent, value: 100, periodSeconds: 30}  # Double in 30s
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - {type: Percent, value: 25, periodSeconds: 60}   # Reduce 25%/min
```

### Per-Service Scaling Profiles

| Service | Min | Max | Primary Metric | Scale Trigger |
|---------|-----|-----|---------------|---------------|
| Gateway | 3 | 20 | HTTP req/s | >1000 req/s/pod |
| Auth | 2 | 10 | CPU | >70% |
| Identity | 2 | 8 | CPU | >70% |
| OAuth | 2 | 10 | CPU | >70% |
| Policy | 2 | 8 | CPU + Redis ops | >70% CPU |
| Audit | 2 | 6 | NATS queue depth | >1000 pending |
| Console | 1 | 3 | CPU | >70% |

## KEDA (Event-Driven)

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: audit-worker-scaler
spec:
  scaleTargetRef:
    name: audit-worker
  minReplicaCount: 1
  maxReplicaCount: 10
  pollingInterval: 10
  cooldownPeriod: 60
  triggers:
  - type: nats-jetstream
    metadata:
      natsServerMonitoringEndpoint: "nats:8222"
      streamName: "AUDIT_EVENTS"
      consumerName: "AUDIT_WORKER"
      lagThreshold: "100"
```

## Scale-to-Zero

For non-critical services (audit enrichment, analytics):

```yaml
scale_to_zero:
  enabled: true
  services: ["analytics-worker", "report-generator"]
  idle_timeout: "300s"      # Scale to 0 after 5 min idle
  warmup_time: "10s"         # Time to handle first request
  keep_min_during_business_hours: true  # 9am-6pm always min 1
```

### Considerations

| Factor | Impact |
|--------|--------|
| Cold start | 5-15s delay for first request |
| NATS durability | Messages buffered, no data loss |
| Health check | KEDA scales up on trigger before request |
| User experience | Only for background workers, not user-facing |

## Predictive Scaling

```yaml
predictive_scaling:
  enabled: true
  service: gateway
  model: "seasonal_naive"
  history: "30d"
  forecast_horizon: "1h"
  
  patterns:
    - description: "Morning rush 8-10am"
      scale_up_before: "07:45"
      target_replicas: 15
    - description: "Lunch dip 12-1pm"
      scale_down_at: "12:00"
      target_replicas: 5
```

## Cost-Aware Scaling

```yaml
cost_aware:
  max_monthly_budget: 5000  # USD
  current_spend: 3200
  
  rules:
    - if: spend > budget * 0.8
      then: "reduce_max_replicas by 20%"
      alert: "cost_threshold_approaching"
    
    - if: spend > budget
      then: "force_scale_down_to_min"
      alert: "budget_exceeded"
    
    - prefer_spot_instances: true
    - prefer_smaller_instances: true
```

## Scaling Cooldown

| Phase | Cooldown | Rationale |
|-------|---------|-----------|
| Scale up | 30s | React quickly to traffic |
| Scale down | 5 min | Don't flap — ensure traffic is truly reduced |
| Post-incident | 10 min | Avoid oscillation after spike |

## Monitoring

| Metric | Alert |
|--------|-------|
| HPA unable to scale | Max replicas reached → increase limit |
| Frequent scale up/down | Flapping → increase cooldown |
| Pods pending | Cluster autoscaler too slow |
| Cost overrun | >80% of budget → scale down |

## See Also

- [Canary Deployment Strategy](canary-deployment-strategy.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
- [Connection Pool Tuning](connection-pool-tuning.md)
- [SRE Practices](sre-practices.md)
