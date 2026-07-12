# Service Mesh Integration Guide

Istio/Linkerd mTLS, sidecar patterns, traffic policies, identity propagation, circuit breaking, and observability integration.

## Overview

A service mesh provides infrastructure-level mTLS, traffic management, and observability without changing application code. GGID integrates with Istio and Linkerd via sidecar proxies.

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   Service Mesh                    │
│                                                   │
│  ┌─────────┐     ┌─────────┐     ┌─────────┐    │
│  │ Gateway  │     │  Auth   │     │ Policy  │    │
│  │  + Envoy │←mTLS│ + Envoy │←mTLS│ + Envoy │    │
│  │  sidecar │     │ sidecar │     │ sidecar │    │
│  └────┬────┘     └────┬────┘     └────┬────┘    │
│       │               │               │          │
│       └───────────────┴───────────────┘          │
│                      │                            │
│              ┌───────┴───────┐                   │
│              │  Control Plane │                   │
│              │  (Istiod/Linkerd)│                  │
│              └───────────────┘                   │
└─────────────────────────────────────────────────┘
```

## Istio Integration

### Installation

```bash
# Install Istio with default profile
istioctl install --set profile=default

# Label namespace for auto-injection
kubectl label namespace ggid istio-injection=enabled

# Restart pods to get sidecars
kubectl rollout restart deployment -n ggid
```

### PeerAuthentication (mTLS)

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ggid
spec:
  mtls:
    mode: STRICT  # Enforce mTLS for all internal traffic
```

| Mode | Behavior |
|------|----------|
| STRICT | Only mTLS connections accepted (recommended) |
| PERMISSIVE | Both mTLS and plaintext accepted (migration only) |
| DISABLE | No mTLS (not recommended) |

### AuthorizationPolicy

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: gateway-to-services
  namespace: ggid
spec:
  selector:
    matchLabels:
      app: auth
  action: ALLOW
  rules:
    - from:
        - source:
            principals: ["cluster.local/ns/ggid/sa/gateway-sa"]
      to:
        - operation:
            methods: ["GET", "POST"]
            paths: ["/api/v1/auth/*"]
```

This ensures only the Gateway service account can call Auth service.

## Linkerd Integration

### Installation

```bash
linkerd install | kubectl apply -f -
linkerd check

# Annotate namespace for injection
kubectl annotate namespace ggid linkerd.io/inject=enabled
```

### mTLS (Automatic)

Linkerd enables mTLS automatically between meshed services — no configuration needed. Verify:

```bash
linkerd -n ggid viz edges
# Shows mTLS status between all service pairs
```

## Traffic Management

### Request Routing

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: gateway-routing
  namespace: ggid
spec:
  hosts: ["gateway"]
  http:
    # Canary: 10% traffic to v2
    - match:
        - headers:
            x-canary:
              exact: "true"
      route:
        - destination:
            host: gateway
            subset: v2
    - route:
        - destination:
            host: gateway
            subset: v1
          weight: 90
        - destination:
            host: gateway
            subset: v2
          weight: 10
```

### Circuit Breaking (Istio)

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: auth-circuit-breaker
  namespace: ggid
spec:
  host: auth
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 50
      http:
        http1MaxPendingRequests: 100
        maxRequestsPerConnection: 10
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
```

## Identity Propagation

### JWT → mTLS Identity

```
External client → JWT (user identity)
Gateway → validates JWT, extracts user
Gateway → mTLS (service identity: gateway-sa)
Gateway → propagates X-User-ID header + JWT to backend
Backend → mTLS verifies gateway is the caller (trust)
Backend → JWT verified (user identity)
```

### Header Propagation

```yaml
# Istio — propagate tracing + identity headers
apiVersion: networking.istio.io/v1beta1
kind: EnvoyFilter
metadata:
  name: propagate-identity
  namespace: ggid
spec:
  configPatches:
    - applyTo: HTTP_FILTER
      match:
        proxy:
          proxyVersion: "1.20"
      patch:
        operation: INSERT_BEFORE
        value:
          name: envoy.filters.http.header_to_metadata
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.http.header_to_metadata.v3.Config
            request_rules:
              - header: X-User-ID
                on_present:
                  metadata_namespace: ggid
                  key: user_id
```

## Observability Integration

### Distributed Tracing

```yaml
# Istio — automatically generates spans
apiVersion: tracing.istio.io/v1
kind: TracingConfig
metadata:
  name: default
spec:
  provider: jaeger
  sampling: 10  # 10% sampling
```

Each request gets a trace ID propagated through all services automatically.

### Metrics

The mesh adds infrastructure metrics on top of GGID's application metrics:

| Metric | Source | Description |
|--------|--------|-------------|
| `istio_requests_total` | Envoy | Total requests between services |
| `istio_request_duration_milliseconds` | Envoy | Inter-service latency |
| `istio_tcp_connections_opened_total` | Envoy | TCP connections |
| `istio_request_bytes` | Envoy | Request payload size |

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "GGID Service Mesh",
    "panels": [
      {"title": "mTLS Status", "query": "sum(istio_requests_total{security_policy=\"mutual_tls\"})"},
      {"title": "Service-to-Service Latency", "query": "histogram_quantile(0.99, istio_request_duration_milliseconds_bucket)"},
      {"title": "Circuit Breakers Open", "query": "sum(upstream_rq_pending_overflow)"},
      {"title": "Request Success Rate", "query": "sum(rate(istio_requests_total{response_code!~\"5..\"}[5m]))"}
    ]
  }
}
```

## Traffic Policies

### Retry Policy

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: retry-policy
  namespace: ggid
spec:
  hosts: ["auth"]
  http:
    - retries:
        attempts: 3
        perTryTimeout: 2s
        retryOn: "5xx,reset,connect-failure"
    - route:
        - destination:
            host: auth
```

### Timeout Policy

```yaml
http:
  - timeout: 5s  # Max time for entire request
    route:
      - destination:
          host: auth
```

## Migration: Application mTLS → Mesh mTLS

GGID has application-level mTLS (gRPC TLS between services). If migrating to mesh-managed mTLS:

```yaml
# Phase 1: Enable mesh in PERMISSIVE mode (accepts both)
PeerAuthentication:
  mtls:
    mode: PERMISSIVE

# Phase 2: Verify all traffic is meshed
linkerd viz edges -n ggid  # Check all pairs show mTLS

# Phase 3: Switch to STRICT mode
PeerAuthentication:
  mtls:
    mode: STRICT

# Phase 4: Remove application-level TLS config
# (Go from TLS-in-app to TLS-in-mesh)
```

| Benefit of Mesh mTLS | Detail |
|----------------------|--------|
| Automatic rotation | Mesh handles cert lifecycle |
| No code changes | TLS at infrastructure layer |
| Centralized policy | Control plane manages all services |
| Mutual identity | SPIFFE IDs for service identity |

## Monitoring

| Metric | Alert |
|--------|-------|
| mTLS failures | Any → cert expiry or config issue |
| Sidecar injection failures | Pods without sidecar |
| Circuit breaker trips | Any → backend unhealthy |
| Mesh control plane down | Critical → all routing stops |
| Certificate expiry | <7 days → auto-rotation check |

## See Also

- [Gateway Architecture](gateway-architecture.md)
- [gRPC vs REST](grpc-vs-rest.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
- [Multi-Region Deployment](multi-region-deployment.md)
- [Zero-Downtime Deployment](zero-downtime-deployment.md)
