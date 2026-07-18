# GGID Monitoring

Prometheus alerts + Grafana dashboards for GGID observability.

## Files

- `alerts.yml` — Prometheus alerting rules (auth failures, latency, session anomalies)
- `grafana-dashboard-ggid-overview.json` — Pre-built Grafana dashboard (services, requests, latency, errors)
- `grafana-dashboard-auth-metrics.json` — Auth deep dive (login success/failure, MFA, sessions, risk evals)
- `grafana-dashboard-api-performance.json` — API perf (latency percentiles, slowest routes, payload sizes, GC)
- `grafana-dashboard-security-overview.json` — Security posture (attack trends, auth failures, rate limits, anomalies)

## Setup

### Prometheus Alerts

```bash
# Copy alerts to Prometheus rules directory
kubectl create configmap ggid-alerts --from-file=alerts.yml -n monitoring

# Or add to existing PrometheusRule CRD
cat alerts.yml | kubectl apply -f -
```

### Grafana Dashboards

Import all 4 dashboards:

```bash
# Create ConfigMap with all dashboards
kubectl create configmap ggid-dashboards \
  --from-file=grafana-dashboard-ggid-overview.json \
  --from-file=grafana-dashboard-auth-metrics.json \
  --from-file=grafana-dashboard-api-performance.json \
  --from-file=grafana-dashboard-security-overview.json \
  -n monitoring
kubectl label configmap ggid-dashboards grafana_dashboard=1 -n monitoring
```

Or import individually via Grafana UI: Dashboards > Import > Upload JSON.

## Metrics Endpoints

Each GGID service exposes Prometheus metrics at `/metrics`:
- Gateway: `http://gateway:8080/metrics`
- Auth: `http://auth:8082/metrics`
- Identity: `http://identity:8081/metrics`

## Key Alerts

| Alert | Trigger |
|-------|---------|
| `HighAuthFailureRate` | >5 auth failures/min for 5m |
| `HighLatency` | p99 latency >2s for 10m |
| `SessionAnomaly` | >100 session revocations in 5m |
| `NHIExcessiveRisk` | NHI risk score >80 |
