# Canary Deployment Strategy

Gradual rollout, traffic splitting, automated rollback on error rate, policy-based promotion, per-tenant canary, and monitoring checkpoints.

## Overview

Canary deployment releases new code to a small percentage of users first, monitoring for errors before full rollout. This limits blast radius of bad deployments.

## Deployment Phases

```
Canary (5%) → Checkpoint → Canary (25%) → Checkpoint → Canary (50%) → Full (100%)
     ↓             ↓              ↓              ↓
   Monitor      Auto-rollback if error rate > threshold
```

## Traffic Splitting

### Kubernetes (Istio)

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: identity-svc
spec:
  http:
  - route:
    - destination:
        host: identity-svc
        subset: stable
      weight: 95
    - destination:
        host: identity-svc
        subset: canary
      weight: 5
```

### Gateway-Based

```go
func canaryMiddleware(stable, canary http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Hash user ID for consistent routing
        userID := getUserID(r)
        hash := fnv32(userID) % 100

        if hash < canaryPercent {
            canary.ServeHTTP(w, r)
            w.Header().Set("X-Deployment", "canary")
        } else {
            stable.ServeHTTP(w, r)
            w.Header().Set("X-Deployment", "stable")
        }
    })
}
```

## Per-Tenant Canary

Deploy canary to specific tenants first (friendly customers, internal teams):

```yaml
canary_config:
  strategy: "per-tenant"
  phases:
    - phase: 1
      tenants: ["internal-test"]      # Internal team first
      percentage: 100
      duration: "1h"

    - phase: 2
      tenants: ["beta-customer-1", "beta-customer-2"]
      percentage: 100
      duration: "4h"

    - phase: 3
      tenants: "*"                    # All tenants
      percentage: 5
      duration: "2h"

    - phase: 4
      percentage: 25
      duration: "4h"

    - phase: 5
      percentage: 100
```

## Automated Rollback

### Error Rate Monitoring

```go
type CanaryMonitor struct {
    errorRateThreshold float64 // e.g., 0.02 (2%)
    latencyThreshold   time.Duration // e.g., 500ms
    window             time.Duration // e.g., 5 min
}

func (m *CanaryMonitor) Evaluate() DeploymentDecision {
    metrics := getCanaryMetrics(m.window)

    // Compare canary vs stable
    if metrics.CanaryErrorRate > m.errorRateThreshold &&
       metrics.CanaryErrorRate > metrics.StableErrorRate*2 {
        return Rollback // Canary error rate >2x stable
    }

    if metrics.CanaryP99Latency > m.latencyThreshold &&
       metrics.CanaryP99Latency > metrics.StableP99Latency*1.5 {
        return Rollback // Canary 50% slower
    }

    if metrics.CanaryCrashRate > 0 {
        return Rollback // Any crashes
    }

    return Promote
}
```

### Checkpoint Evaluation

| Checkpoint | Duration | Metrics Evaluated |
|-----------|----------|-------------------|
| Phase 1→2 | 1 hour | Error rate, latency, crash rate |
| Phase 2→3 | 4 hours | Error rate, latency, user complaints |
| Phase 3→4 | 2 hours | Error rate, latency, CPU/memory |
| Phase 4→5 | 4 hours | Full comparison vs stable |

### Rollback Action

```bash
# Instant rollback
kubectl patch virtualservice identity-svc --type='json' \
  -p='[{"op":"replace","path":"/spec/http/0/route/0/weight","value":100},
       {"op":"replace","path":"/spec/http/0/route/1/weight","value":0}]'

# Scale down canary pods
kubectl scale deployment identity-svc-canary --replicas=0

# Alert
alert.Send("canary_rolled_back", map[string]interface{}{
    "reason": "error_rate_exceeded",
    "canary_error_rate": 0.045,
    "stable_error_rate": 0.001,
})
```

## Promotion Criteria

All of these must pass before promoting:

| Check | Threshold | Metric Source |
|-------|-----------|--------------|
| Error rate | Canary ≤ stable + 0.5% | Prometheus |
| P99 latency | Canary ≤ stable × 1.2 | OpenTelemetry |
| Crash rate | 0 crashes | Kubernetes events |
| Memory usage | No leak (flat or decreasing) | cAdvisor |
| Audit log integrity | Hash chain valid | Audit service |
| Health checks | All passing | Kubernetes probes |
| User complaints | 0 new tickets | Support system |

## Implementation Pipeline

```yaml
# CI/CD pipeline (GitHub Actions / Argo)
deploy_canary:
  steps:
    - name: deploy-canary-5pct
      run: kubectl apply -f canary-5pct.yaml

    - name: wait-and-evaluate
      run: ./scripts/evaluate-canary.sh --duration 1h
      # Exits non-zero if rollback needed

    - name: promote-to-25pct
      if: success()
      run: kubectl apply -f canary-25pct.yaml

    - name: auto-rollback
      if: failure()
      run: ./scripts/rollback-canary.sh
```

## Monitoring Dashboard

```
Canary Deployment: identity-svc v2.3.1
┌─────────────────────────────────────┐
│ Canary: 5% (phase 1/5)              │
│ Duration: 45min / 1h                │
│                                     │
│ Error rate:   0.1% (stable: 0.1%) ✓ │
│ P99 latency:  12ms (stable: 11ms) ✓ │
│ Crashes:      0 ✓                   │
│ Memory:       Stable ✓              │
│                                     │
│ [Promote]  [Hold]  [Rollback]       │
└─────────────────────────────────────┘
```

## See Also

- [Zero-Downtime Deployment](zero-downtime-deployment.md)
- [Gateway Architecture](gateway-architecture.md)
- [Distributed Tracing Setup](distributed-tracing-setup.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
