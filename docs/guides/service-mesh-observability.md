# Service Mesh Observability

Istio/Linkerd telemetry, distributed tracing integration, metrics pipeline, golden signals, traffic analysis, and anomaly detection.

## Overview

Service mesh provides automatic observability for all inter-service traffic without application code changes.

## Telemetry Sources

### Istio

```yaml
# Istio telemetry config
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: default-telemetry
spec:
  tracing:
  - randomSamplingPercentage: 10.0
  accessLogging:
  - providers:
    - name: otel
  metrics:
  - providers:
    - name: prometheus
```

### Linkerd

```yaml
# Linkerd auto-injects proxy that collects:
# - Request volume per service
# - Latency (P50, P95, P99)
# - Success rate
# - TCP connections
```

## Golden Signals

| Signal | Metric | Alert |
|--------|--------|-------|
| **Latency** | `istio_request_duration_milliseconds` | P99 > 500ms |
| **Traffic** | `istio_requests_total` | QPS change >50% |
| **Errors** | `istio_requests_total{response_code=~"5.."}` | >1% error rate |
| **Saturation** | `container_cpu_usage_seconds_total` | >80% CPU |

### Prometheus Queries

```promql
# Request rate by service
sum(rate(istio_requests_total[1m])) by (destination_service)

# Error rate
sum(rate(istio_requests_total{response_code=~"5.."}[1m])) by (destination_service)
/ sum(rate(istio_requests_total[1m])) by (destination_service)

# P99 latency
histogram_quantile(0.99, sum(rate(istio_request_duration_milliseconds_bucket[1m])) by (le, destination_service))

# Success rate
sum(rate(istio_requests_total{response_code!~"5.."}[1m])) by (destination_service)
/ sum(rate(istio_requests_total[1m])) by (destination_service)
```

## Distributed Tracing Integration

### Istio + OpenTelemetry

```yaml
# Istio sends spans to OTel collector
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
spec:
  tracing:
  - providers:
    - name: otel
    randomSamplingPercentage: 10.0
    customTags:
      tenant_id:
        header:
          name: x-tenant-id
```

### Span Enrichment

Mesh adds:
- Source/destination service names
- Request protocol (HTTP/gRPC)
- Response code
- Latency
- Custom headers (tenant_id, user_id)

Application adds:
- Business logic spans (DB queries, cache lookups)
- Audit trail correlation

## Traffic Analysis

### Service Dependency Map

```
Gateway → Auth (95% of traffic)
       → Identity (60%)
       → Policy (40%)
       → OAuth (20%)
       → Audit (100%, fire-and-forget)

Auth → Identity (30% verify user)
    → Redis (90% session check)
    → LDAP (10% when LDAP login)
```

### Traffic Anomaly Detection

```yaml
anomaly_detection:
  rules:
    - name: "unusual_service_traffic"
      condition: "request_rate > baseline * 3"
      action: alert

    - name: "new_connection_pattern"
      condition: "new source-destination pair not in 7d baseline"
      action: log + review

    - name: "protocol_anomaly"
      condition: "gRPC service receiving HTTP requests"
      action: alert
```

## Metrics Pipeline

```
Sidecar Proxy → Prometheus (scrape /stats) → Grafana Dashboards
                    │
                    ├── Alertmanager (golden signal alerts)
                    └── Long-term storage (Thanos/Mimir)
```

### Grafana Dashboard Panels

| Panel | Query |
|-------|-------|
| Service mesh overview | Success rate + latency per service |
| Top error services | `sort_desc(sum(rate(errors)) by (service))` |
| Traffic flow (Sankey) | Source → destination request volume |
| Slowest endpoints | P99 latency sorted descending |
| Mesh health | Proxy version, config sync status |

## mTLS Telemetry

```promql
# mTLS coverage
sum(istio_tcp_connections_opened_total{
  connection_security_policy="mutual_tls"
}) by (destination_service)
/ sum(istio_tcp_connections_opened_total) by (destination_service)
# Target: 100% mTLS coverage
```

## Circuit Breaker Metrics

```yaml
circuit_breaker:
  metrics:
    - name: "connections_pending"
      query: "envoy_cluster_upstream_pending_connections"
      alert: "> 10 → circuit breaker opening"

    - name: "outlier_detection_ejections"
      query: "envoy_cluster_outlier_detection_ejections_total"
      alert: "> 0 → unhealthy upstream ejected"
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Proxy config sync failures | Any → mesh misconfigured |
| mTLS coverage | <100% → security gap |
| Unreachable services | Any → network policy or DNS issue |
| High mesh latency overhead | >2ms added → investigate |
| Data plane version mismatch | Mixed proxy versions → upgrade |

## See Also

- [Service Mesh Integration](service-mesh-integration.md)
- [Distributed Tracing Setup](distributed-tracing-setup.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
- [Gateway Architecture](gateway-architecture.md)
