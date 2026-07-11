# Service Mesh Integration for IAM Zero-Trust Architecture

> **Research Document** — Service mesh patterns for GGID's east-west security, mTLS,
> policy enforcement, and multi-cluster deployment.
>
> **Companion docs**: [`zero-trust-iam.md`](./zero-trust-iam.md) (microsegmentation,
> device posture), [`grpc-security-iam.md`](./grpc-security-iam.md) (gRPC hardening),
> [`observability-iam.md`](./observability-iam.md) (metrics/tracing).
>
> **Related code**: [`deploy/envoy/ggid-envoy.yaml`](../../deploy/envoy/ggid-envoy.yaml),
> [`deploy/helm/ggid/`](../../deploy/helm/ggid/),
> [`deploy/nginx/nginx.conf`](../../deploy/nginx/nginx.conf).

---

## Table of Contents

1. [Service Mesh Concepts for IAM](#1-service-mesh-concepts-for-iam)
2. [Istio Integration](#2-istio-integration)
3. [Linkerd Integration](#3-linkerd-integration)
4. [Zero-Trust East-West with mTLS Sidecar](#4-zero-trust-east-west-with-mtls-sidecar)
5. [JWT Validation at Proxy Layer](#5-jwt-validation-at-proxy-layer)
6. [Authorization Policy at Mesh Layer](#6-authorization-policy-at-mesh-layer)
7. [Observability via Mesh](#7-observability-via-mesh)
8. [Multi-Cluster IAM with Service Mesh](#8-multi-cluster-iam-with-service-mesh)
9. [GGID Service Mesh Roadmap](#9-ggid-service-mesh-roadmap)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. Service Mesh Concepts for IAM

### What a Service Mesh Provides

A service mesh is an infrastructure layer that handles service-to-service communication
transparently. For an IAM system like GGID — where seven microservices (gateway, identity,
auth, oauth, policy, org, audit) exchange authentication tokens, user data, and policy
decisions over the network — a mesh provides four critical capabilities:

| Capability | Benefit for IAM |
|---|---|
| **mTLS between services** | All east-west traffic encrypted and authenticated. No plaintext credentials on the wire. |
| **Traffic management** | Canary deployments for auth flows, circuit breakers to prevent cascade failures. |
| **Observability** | Per-service-pair latency, error rates, and distributed tracing without code changes. |
| **Policy enforcement** | Layer 7 authorization (who can call whom) enforced at the proxy, not the application. |

### Sidecar Proxy Model

Each pod runs a lightweight proxy (Envoy in Istio, linkerd2-proxy in Linkerd) alongside
the application container. All inbound and outbound traffic flows through the sidecar,
which handles TLS termination, routing, metrics, and policy enforcement.

```
┌─────────────────────────────────────┐
│              Pod                     │
│  ┌───────────┐   ┌───────────────┐  │
│  │  GGID App │◄──┤  Sidecar Proxy│  │
│  │  (Go)     │   │  (Envoy/etc)  │  │
│  │  :8080    │──►│  :15001       │  │
│  └───────────┘   └───────┬───────┘  │
│                          │          │
└──────────────────────────┼──────────┘
                           │ mTLS
                   ┌───────▼───────┐
                   │  Remote Pod   │
                   │  Sidecar      │
                   └───────────────┘
```

### Data Plane vs Control Plane

- **Data plane**: The sidecar proxies that intercept and forward traffic. Each proxy
  operates independently, making real-time routing decisions based on its configuration.
- **Control plane**: A centralized component (Istio's `istiod`, Linkerd's control
  plane) that distributes configuration, certificates, and policy to all sidecars.

```
┌─────────────────────────────────────────────────────┐
│                   Control Plane                      │
│  ┌─────────────┐  ┌──────────┐  ┌───────────────┐  │
│  │  Config API │  │ CA (mTLS)│  │ Policy Engine │  │
│  └──────┬──────┘  └────┬─────┘  └───────┬───────┘  │
└─────────┼──────────────┼────────────────┼──────────┘
          │   xDS/gRPC   │   cert agent   │ policy push
    ┌─────▼────┐   ┌────▼─────┐   ┌────▼─────┐
    │ Sidecar  │   │ Sidecar  │   │ Sidecar  │
    │ Gateway  │   │   Auth   │   │ Policy   │
    └──────────┘   └──────────┘   └──────────┘
         Data Plane (all sidecar proxies)
```

### Why IAM Benefits from Service Mesh

GGID's current architecture routes all external traffic through the Gateway (north-south),
but inter-service communication — Gateway to Auth, Auth to Identity, Policy to Org, Audit
publishing to NATS — flows over **plaintext HTTP/gRPC** as seen in the Docker Compose
configuration:

```yaml
# From deploy/docker-compose.yaml — plaintext service URLs
AUTH_SERVICE_URL: "http://auth:9001"
IDENTITY_SERVICE_URL: "http://identity:8080"
POLICY_SERVICE_URL: "http://policy:8070"
```

This is a zero-trust violation: an attacker who compromises one pod gains plaintext access
to all inter-service traffic. A service mesh encrypts this east-west communication
transparently, with **zero application code changes**.

---

## 2. Istio Integration

### Istio Architecture

Istio is the most feature-complete service mesh for Kubernetes. Its architecture:

| Component | Role |
|---|---|
| **Envoy sidecar** | Data plane proxy injected into each pod via mutating webhook |
| **istiod** | Control plane: combines Pilot (config), Citadel (CA), Galley (validation) |
| **istio-ingressgateway** | Entry point for external traffic (replaces or complements Nginx) |
| **istio-cni** | Optional CNI plugin for transparent traffic capture |

### mTLS Auto-Mounting

Istio automatically provisions and rotates mTLS certificates for every sidecar using
SPIFFE (Secure Production Identity Framework for Everyone). Each pod receives a
cryptographic identity in the format `spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>`.

Certificates are short-lived (default 24h) and rotated automatically. Application code
never sees the certificates — the sidecar handles the entire TLS handshake.

### PeerAuthentication for Strict mTLS

```yaml
# Enforce STRICT mTLS across the entire GGID namespace
apiVersion: security.istio.io/v1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ggid
spec:
  mtls:
    mode: STRICT
---
# Per-service override: Auth service requires STRICT mTLS
apiVersion: security.istio.io/v1
kind: PeerAuthentication
metadata:
  name: auth-mtls
  namespace: ggid
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: auth
  mtls:
    mode: STRICT
```

`STRICT` mode rejects any plaintext connection — if a pod tries to call the Auth service
without mTLS, the connection is refused at the sidecar level.

### GGID Go Service Deployment with Istio Injection

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-auth
  namespace: ggid
  labels:
    app.kubernetes.io/name: ggid
    app.kubernetes.io/component: auth
spec:
  replicas: 3
  selector:
    matchLabels:
      app.kubernetes.io/component: auth
  template:
    metadata:
      labels:
        app.kubernetes.io/name: ggid
        app.kubernetes.io/component: auth
      annotations:
        # Istio sidecar injection
        sidecar.istio.io/inject: "true"
        # Use Istio's proxy for both inbound and outbound
        traffic.sidecar.istio.io/includeInboundPorts: "9001"
        traffic.sidecar.istio.io/excludeOutboundPorts: "5432,6379,4222"
        # Resource hints for sidecar
        sidecar.istio.io/proxyCPU: "100m"
        sidecar.istio.io/proxyMemory: "128Mi"
        sidecar.istio.io/proxyCPULimit: "500m"
        sidecar.istio.io/proxyMemoryLimit: "256Mi"
    spec:
      serviceAccountName: ggid-auth
      containers:
        - name: auth
          image: ggid/auth:latest
          ports:
            - name: http
              containerPort: 9001
              protocol: TCP
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: ggid-db-secret
                  key: url
            - name: REDIS_ADDR
              value: "redis:6379"
            - name: JWT_PRIVATE_KEY_PATH
              value: "/configs/rsa_private.pem"
          # Application listens on HTTP — sidecar handles mTLS externally
          # No TLS code changes needed in the Go binary
          livenessProbe:
            httpGet:
              path: /healthz
              port: 9001
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /healthz
              port: 9001
            initialDelaySeconds: 5
            periodSeconds: 5
          resources:
            limits:
              cpu: 500m
              memory: 256Mi
            requests:
              cpu: 100m
              memory: 128Mi
```

### Traffic Flow with Istio

```
External Request
      │
      ▼
┌──────────────────┐
│ Istio Ingress    │
│ Gateway (Envoy)  │  ← TLS termination for north-south
│ :443             │  ← JWT validation (optional)
└────────┬─────────┘
         │ mTLS (Istio managed)
         ▼
┌──────────────────┐     mTLS     ┌──────────────────┐
│ Gateway Pod      │◄────────────►│ Auth Pod          │
│ ┌─────┐ ┌──────┐ │              │ ┌─────┐ ┌──────┐ │
│ │App  │ │Envoy │ │              │ │App  │ │Envoy │ │
│ │:8080│ │:15001│ │              │ │:9001│ │:15001│ │
│ └─────┘ └──────┘ │              │ └─────┘ └──────┘ │
└──────────────────┘              └──────────────────┘
         │ mTLS                          │ mTLS
         ▼                               ▼
┌──────────────────┐     mTLS     ┌──────────────────┐
│ Identity Pod     │◄────────────►│ Policy Pod        │
│ ┌─────┐ ┌──────┐ │              │ ┌─────┐ ┌──────┐ │
│ │App  │ │Envoy │ │              │ │App  │ │Envoy │ │
│ └─────┘ └──────┘ │              │ └─────┘ └──────┘ │
└──────────────────┘              └──────────────────┘
```

---

## 3. Linkerd Integration

### Linkerd Architecture

Linkerd is a lighter alternative to Istio. Its data plane proxy is written in Rust
(linkerd2-proxy), optimized for minimal resource overhead.

| Component | Role |
|---|---|
| **linkerd2-proxy** | Rust-based sidecar (≈10MB RSS, sub-millisecond latency overhead) |
| **linkerd-controller** | Control plane (config, identity, destination, proxy-injector) |
| **Jaeger** | Optional but natively integrated distributed tracing |
| **Grafana** | Built-in dashboards |

### mTLS via Service Identity

Linkerd derives workload identity from Kubernetes ServiceAccounts. No SPIRE or external
CA is needed — Linkerd uses the cluster's built-in PKI. Each pod's identity is
`<service-account>.<namespace>.serviceaccount.identity.linkerd.cluster.local`.

```yaml
# Enable mTLS for all GGID services — Linkerd default behavior
# Simply inject the proxy; mTLS is on by default
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-policy
  namespace: ggid
  labels:
    app.kubernetes.io/component: policy
spec:
  replicas: 3
  selector:
    matchLabels:
      app.kubernetes.io/component: policy
  template:
    metadata:
      labels:
        app.kubernetes.io/component: policy
      annotations:
        linkerd.io/inject: enabled
    spec:
      serviceAccountName: ggid-policy
      containers:
        - name: policy
          image: ggid/policy:latest
          ports:
            - name: http
              containerPort: 8070
            - name: grpc
              containerPort: 9070
          env:
            - name: DB_HOST
              value: postgres
            - name: NATS_URL
              value: "nats://nats:4222"
          resources:
            limits:
              cpu: 500m
              memory: 256Mi
```

### HTTPRoute for Traffic Splitting

Linkerd 2.14+ supports the Gateway API `HTTPRoute` for traffic splitting, enabling
canary deployments for GGID's Auth service:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: ggid-auth-canary
  namespace: ggid
spec:
  parentRefs:
    - name: ggid-gateway
      kind: Service
  hostnames:
    - "auth.ggid.internal"
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /api/v1/auth
      backendRefs:
        - name: ggid-auth-stable
          port: 9001
          weight: 90
        - name: ggid-auth-canary
          port: 9001
          weight: 10
```

### Istio vs Linkerd for GGID

| Criterion | Istio | Linkerd |
|---|---|---|
| Proxy language | C++ (Envoy) | Rust (linkerd2-proxy) |
| Memory per sidecar | 50-150MB | 10-40MB |
| Feature richness | AuthorizationPolicy, RequestAuthentication, EnvoyFilter | Basic mTLS, retry, timeout, tracing |
| JWT at proxy | Built-in (RequestAuthentication) | Not natively supported (needs ext-auth) |
| L7 authorization | Full (path, method, JWT claims) | Limited (service-level only) |
| Learning curve | Steep | Gentle |
| **GGID recommendation** | Better fit (JWT validation, L7 policy) | Good for mTLS-only phase |

---

## 4. Zero-Trust East-West with mTLS Sidecar

### How the Sidecar Handles mTLS

The sidecar proxy intercepts all traffic transparently via `iptables` rules set up at
pod creation time. For outbound traffic from the GGID Gateway to the Auth service:

1. Gateway's Go binary calls `http://auth:9001/login` (plaintext, no TLS).
2. The sidecar's outbound listener intercepts the connection.
3. Sidecar resolves `auth` to the target pod and establishes a **mTLS connection** to
   the target's sidecar.
4. Target sidecar terminates mTLS, forwards plaintext to Auth's Go binary on `:9001`.
5. Response flows back through the same mTLS tunnel.

```
Gateway Pod                          Auth Pod
┌────────────────┐                   ┌────────────────┐
│  Go App :8080  │                   │  Go App :9001  │
│      │         │                   │         │      │
│  http://auth   │                   │  plaintext     │
│      │         │                   │      │         │
│  ┌───▼───────┐ │    mTLS tunnel    │  ┌───▼───────┐ │
│  │  Envoy    │◄├──────────────────►┤  │  Envoy    │ │
│  │  Outbound │ │  SPIFFE identity  │  │  Inbound  │ │
│  │  :15001   │ │  cert auto-rotate │  │  :15006   │ │
│  └───────────┘ │                   │  └───────────┘ │
└────────────────┘                   └────────────────┘
```

### SPIFFE Identity Per Pod

Each pod receives a unique SPIFFE Verifiable Identity Document (SVID) in the format:

```
spiffe://cluster.local/ns/ggid/sa/ggid-auth
spiffe://cluster.local/ns/ggid/sa/ggid-gateway
spiffe://cluster.local/ns/ggid/sa/ggid-policy
```

The control plane CA signs these. SVIDs are short-lived (24h default) and rotated
automatically — there is no manual certificate management.

### Application-Level mTLS vs Sidecar mTLS

| Aspect | Application-Level mTLS | Sidecar mTLS |
|---|---|---|
| Code changes | Requires TLS config in every Go service | None |
| Certificate management | App must load/rotate certs | Fully automated by mesh |
| Go code example | `tls.Config{Certificates: ...}` per service | No changes |
| Failure surface | App bug can disable TLS | Sidecar enforces independently |
| Performance | No extra hop | Extra hop (localhost, negligible latency) |
| Debugging | Direct TLS logs | Sidecar access logs |

GGID currently uses plaintext between services. Adopting application-level mTLS would
require modifying all seven services to load TLS certificates — a significant change
affecting `DATABASE_URL`, `AUTH_SERVICE_URL`, and every inter-service client. The
sidecar approach eliminates this: **zero Go code changes**.

### Current GGID Inter-Service Communication (Plaintext)

From the Gateway's environment configuration:

```go
// services/gateway/cmd/main.go — no TLS between gateway and backend services
cfg.AuthServiceURL    // "http://auth:9001"
cfg.IdentityServiceURL // "http://identity:8080"
cfg.PolicyServiceURL   // "http://policy:8070"
cfg.OrgServiceURL      // "http://org:8071"
cfg.AuditServiceURL    // "http://audit:8072"
cfg.OAuthServiceURL    // "http://oauth:9005"
```

With a service mesh, these URLs remain `http://` — the sidecar transparently upgrades
them to `https://` on the wire.

---

## 5. JWT Validation at Proxy Layer

### Envoy JWT Auth Filter

GGID's Gateway currently validates JWTs in Go code via the JWKS client
(`middleware.NewJWKSClient`). With a service mesh, this validation can be offloaded to
the Envoy sidecar, which fetches the JWKS from GGID's OAuth service and validates
tokens before they reach the application.

The existing `deploy/envoy/ggid-envoy.yaml` already demonstrates this pattern:

```yaml
# From deploy/envoy/ggid-envoy.yaml — JWT auth filter
http_filters:
  - name: envoy.filters.http.jwt_authn
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication
      providers:
        ggid_jwt:
          issuer: "https://ggid.example.com"
          audiences:
            - "ggid-api"
          remote_jwks:
            http_uri:
              uri: "http://ggid-gateway:8080/.well-known/jwks.json"
              cluster: ggid_gateway
              timeout: 5s
            cache_duration: 300s
      rules:
        - match:
            prefix: "/api/v1/auth/login"
          # Public path — no JWT required
        - match:
            prefix: "/api/v1"
          requires:
            provider_name: ggid_jwt
```

### Istio RequestAuthentication (Native JWT Validation)

With Istio, JWT validation is configured declaratively via `RequestAuthentication`:

```yaml
apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata:
  name: ggid-jwt
  namespace: ggid
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: ggid
  jwtRules:
    - issuer: "https://ggid.example.com"
      audiences:
        - "ggid-api"
      jwksUri: "http://ggid-oauth:9005/.well-known/jwks.json"
      # Forward the JWT payload as headers to the application
      outputPayloadToHeader: "x-jwt-payload"
      # Map specific claims to headers
      outputClaimToHeaders:
        - header: "x-tenant-id"
          claim: "tenant_id"
        - header: "x-user-id"
          claim: "sub"
        - header: "x-scopes"
          claim: "scope"
      forwardOriginalToken: false
```

### Per-Route JWT Requirements with AuthorizationPolicy

```yaml
# Public routes: no JWT required (login, register, JWKS, discovery)
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: ggid-public-routes
  namespace: ggid
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: gateway
  action: ALLOW
  rules:
    - to:
        - operation:
            paths:
              - /api/v1/auth/login
              - /api/v1/auth/register
              - /api/v1/auth/refresh
              /.well-known/*
              /oauth/authorize
              /oauth/token
```

### Benefits of Proxy-Layer JWT Validation

1. **Crypto offloading**: RSA signature verification happens in Envoy's native code,
   reducing Go GC pressure on the Gateway.
2. **JWKS caching**: Envoy caches the JWKS for 300s (configurable), reducing fetches
   to the OAuth service. Currently GGID refreshes every 15 minutes in Go.
3. **Defense in depth**: Even if the Go application has a JWT validation bug, the
   sidecar rejects invalid tokens before they reach the app.
4. **Tenant context injection**: JWT claims (`tenant_id`, `sub`, `scope`) are forwarded
   as headers, simplifying application code.

---

## 6. Authorization Policy at Mesh Layer

### Istio AuthorizationPolicy

Istio's `AuthorizationPolicy` enforces L7 access control based on:
- **Source identity** (SPIFFE SVID, service account, namespace)
- **Request properties** (path, method, JWT claims)
- **Destination** (service, port)

This creates a **second authorization layer**: the mesh decides whether a request may
reach the service, and the service's own RBAC engine decides whether the user is
authorized for the specific action.

### GGID Service Access Matrix

| From | To | Allowed Paths |
|---|---|---|
| Gateway | Auth | `/api/v1/auth/*`, `/healthz` |
| Gateway | Identity | `/api/v1/users/*`, `/healthz` |
| Gateway | Policy | `/api/v1/roles/*`, `/api/v1/permissions/*`, `/healthz` |
| Gateway | Org | `/api/v1/orgs/*`, `/healthz` |
| Gateway | Audit | `/api/v1/audit/*`, `/healthz` |
| Gateway | OAuth | `/oauth/*`, `/.well-known/*`, `/healthz` |
| Auth | Identity | `/api/v1/users/*` (user lookup during login) |
| Policy | Org | `/api/v1/orgs/*` (org hierarchy for ABAC) |
| Any | Audit | NATS only (not HTTP) |

### Mesh AuthorizationPolicy Examples

```yaml
# Only Gateway pods can call the Auth service
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: auth-allow-gateway-only
  namespace: ggid
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: auth
  action: ALLOW
  rules:
    - from:
        - source:
            principals:
              - "cluster.local/ns/ggid/sa/ggid-gateway"
            # Also allow the Auth service to call itself (healthchecks)
              - "cluster.local/ns/ggid/sa/ggid-auth"
      to:
        - operation:
            paths:
              - /api/v1/auth/*
              - /healthz
            methods:
              - GET
              - POST
---
# Only Gateway and Auth pods can call Identity
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: identity-allow-gateway-auth
  namespace: ggid
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: identity
  action: ALLOW
  rules:
    - from:
        - source:
            principals:
              - "cluster.local/ns/ggid/sa/ggid-gateway"
              - "cluster.local/ns/ggid/sa/ggid-auth"
      to:
        - operation:
            paths:
              - /api/v1/users/*
              - /healthz
---
# Policy service: allow Gateway (north-south) and Auth (token validation)
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: policy-allow-gateway
  namespace: ggid
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: policy
  action: ALLOW
  rules:
    - from:
        - source:
            principals:
              - "cluster.local/ns/ggid/sa/ggid-gateway"
      to:
        - operation:
            paths:
              - /api/v1/roles/*
              - /api/v1/permissions/*
              - /healthz
---
# Deny all other traffic to backend services (default deny)
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: default-deny
  namespace: ggid
spec:
  # Empty selector = applies to all workloads in namespace
  {}
```

### Defense in Depth: Mesh Policy + App RBAC

```
Request ──► Envoy Sidecar ──► Go Application
              │                    │
              ▼                    ▼
        Mesh Policy           App RBAC Engine
        (L7: who can          (L7: user X can
         reach service Y)      perform action Z
                               on resource R)
```

The mesh layer answers: "Is the calling service allowed to reach this endpoint?"
The application layer answers: "Is this user allowed to perform this operation?"

An attacker who compromises the Policy pod cannot call the Auth service directly —
the mesh policy rejects the connection at the sidecar. And even if they reach the Auth
service through legitimate channels, the app-level JWT validation still requires a
valid token with appropriate scopes.

---

## 7. Observability via Mesh

### Distributed Tracing (Automatic)

Istio and Linkerd automatically inject trace headers (W3C Trace Context /
BAGGAGE) and export spans to Jaeger or Zipkin. GGID's current observability setup
(see [`observability-iam.md`](./observability-iam.md)) would gain:

- **End-to-end traces**: A single login request spans Gateway → Auth → Identity →
  Redis (session store). The mesh generates the full span tree without any Go
  instrumentation changes.
- **Latency waterfall**: Identify which service hop is the bottleneck (e.g., Auth
  waiting on LDAP query).

```yaml
# Istio tracing configuration
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  meshConfig:
    enableTracing: true
    extensionProviders:
      - name: jaeger
        zipkin:
          address: jaeger-collector.observability.svc:9411
    defaultProviders:
      tracing:
        - jaeger
    # Sample 100% of auth/login requests, 10% of others
    defaultConfig:
      tracing:
        sampling: 10
```

### Per-Service-Pair Metrics

The mesh sidecar generates metrics for every service-to-service interaction:

| Metric | Description |
|---|---|
| `istio_requests_total` | Total requests by source, destination, response code |
| `istio_request_duration_milliseconds` | Latency histogram per service pair |
| `istio_tcp_connections_opened_total` | TCP connection establishment |
| `envoy_cluster_upstream_rq_xx` | Upstream success/error rates |

These metrics are scraped by Prometheus and visualized in Grafana or Kiali:

```
# Golden signal dashboard — per GGID service pair
Gateway → Auth:      1000 req/s, 2ms p99, 0.01% 5xx
Gateway → Identity:   800 req/s, 5ms p99, 0.00% 5xx
Gateway → Policy:     500 req/s, 3ms p99, 0.02% 5xx
Auth     → Identity:  200 req/s, 4ms p99, 0.00% 5xx
Policy   → Org:        50 req/s, 8ms p99, 0.00% 5xx
```

### Access Logs from Envoy

Each sidecar produces structured access logs:

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "upstream_cluster": "outbound|9001||auth.ggid.svc.cluster.local",
  "downstream_remote_address": "10.0.1.5:38920",
  "route_name": "default",
  "request": {
    "method": "POST",
    "path": "/api/v1/auth/login",
    "scheme": "https",
    "user_agent": "GGID-Console/1.0"
  },
  "response": {
    "code": 200,
    "duration_ms": 12,
    "bytes_sent": 1024
  },
  "tls": {
    "sni": "auth.ggid.svc.cluster.local",
    "cipher": "ECDHE-RSA-AES256-GCM-SHA384",
    "version": "TLSv1.3"
  }
}
```

### Kiali for Mesh Visualization

Kiali provides a real-time service graph showing traffic flow, error rates, and
mTLS status for the entire GGID mesh:

```
                    ┌─────────┐
          1000 req/s│         │ 0.01% 5xx
               ┌───►│  Auth   │───┐
               │    │  :9001  │   │
               │    └─────────┘   │
┌──────────┐   │                  │   ┌──────────┐
│ Gateway  │───┤                  ├──►│ Identity │
│  :8080   │   │    ┌─────────┐   │   │  :8081   │
└──────────┘   ├───►│ Policy  │   │   └──────────┘
               │    │  :8070  │   │
               │    └─────────┘   │
               │                  │   ┌──────────┐
               └───►──────────────┴──►│  Audit   │
                                       │  :8072   │
                                       └──────────┘
    Green = mTLS active    Red = errors    Yellow = high latency
```

This complements application-level observability: the mesh provides infrastructure-level
visibility (network, TLS, routing) while GGID's own metrics provide business-level
insights (login success rate, JWT validation time, RBAC decision latency).

---

## 8. Multi-Cluster IAM with Service Mesh

### Cross-Cluster mTLS

For multi-region GGID deployments, a service mesh enables seamless cross-cluster
communication with the same mTLS guarantees as intra-cluster traffic.

### Shared Trust Domain (SPIFFE)

All clusters share the same SPIFFE trust domain (`cluster.local` or a custom domain like
`ggid.example.com`). Pods in cluster A can verify pods in cluster B because they share
the same root of trust.

```
┌─────────────────────── SPIFFE Trust Domain: ggid.example.com ───────────────────────┐
│                                                                                      │
│  ┌─────────────────────────────┐          ┌─────────────────────────────┐          │
│  │    Region: US-East (Primary) │          │    Region: EU-West (Failover)│          │
│  │                              │          │                              │          │
│  │  Cluster: ggid-us            │          │  Cluster: ggid-eu            │          │
│  │  ┌─────┐ ┌─────┐ ┌─────┐   │  mTLS    │  ┌─────┐ ┌─────┐ ┌─────┐   │          │
│  │  │ GW  │ │Auth │ │ ID  │   │◄────────►│  │ GW  │ │Auth │ │ ID  │   │          │
│  │  └─────┘ └─────┘ └─────┘   │  feder.  │  └─────┘ └─────┘ └─────┘   │          │
│  │     ┌──────────┐            │  trust   │     ┌──────────┐            │          │
│  │     │ Postgres │            │          │     │ Postgres │            │          │
│  │     │ (Primary)│            │          │     │ (Replica)│            │          │
│  │     └──────────┘            │          │     └──────────┘            │          │
│  └─────────────────────────────┘          └─────────────────────────────┘          │
│                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

### Istio Multi-Cluster Configuration

Istio supports two multi-cluster models:

1. **Primary-Remote**: One cluster runs the control plane; remote clusters connect to it.
2. **Multi-Primary**: Each cluster runs its own control plane, federated via shared trust.

```yaml
# Multi-cluster service entry for cross-region Gateway failover
apiVersion: networking.istio.io/v1
kind: ServiceEntry
metadata:
  name: ggid-auth-remote
  namespace: ggid
spec:
  hosts:
    - auth.ggid.global
  location: MESH_INTERNAL
  ports:
    - number: 9001
      name: http
      protocol: HTTP
  resolution: DNS
  endpoints:
    - address: ggid-eu-ingressgateway.eu-west.istio-gateways.svc.cluster.local
      ports:
        http: 15443
      locality: eu-west-1
      weight: 100
---
# Locality-preferring failover: try local first, failover to remote
apiVersion: networking.istio.io/v1
kind: DestinationRule
metadata:
  name: ggid-auth-failover
  namespace: ggid
spec:
  host: auth.ggid.svc.cluster.local
  trafficPolicy:
    loadBalancer:
      simple: LEAST_REQUEST
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 10s
      baseEjectionTime: 30s
```

### Multi-Region Failover for GGID

```
User in EU                        User in US
     │                                 │
     ▼                                 ▼
┌──────────┐                    ┌──────────┐
│ Global   │                    │ Global   │
│ LB (DNS) │                    │ LB (DNS) │
└────┬─────┘                    └────┬─────┘
     │                               │
     ▼                               ▼
┌──────────┐    mTLS failover  ┌──────────┐
│ ggid-eu  │◄─────────────────►│ ggid-us  │
│ Gateway  │   (if EU down)    │ Gateway  │
└────┬─────┘                   └────┬─────┘
     │                               │
     ▼                               ▼
┌──────────┐  async replication ┌──────────┐
│ Postgres │◄──────────────────►│ Postgres │
│ EU-West  │   (logical repl.)  │ US-East  │
└──────────┘                    └──────────┘
```

---

## 9. GGID Service Mesh Roadmap

### Current State Assessment

**Existing infrastructure** (reviewed from `deploy/`):

| Component | Current State | Mesh-Ready? |
|---|---|---|
| Docker Compose (`docker-compose.yaml`) | 13 services, plaintext HTTP between services | No (no mesh for local dev) |
| Helm chart (`deploy/helm/ggid/`) | K8s deployments, services, NetworkPolicies, HPA | Partially — needs injection annotations |
| Envoy config (`deploy/envoy/ggid-envoy.yaml`) | Standalone Envoy with JWT filter, rate limit, circuit breaker | Template only — not deployed |
| Nginx (`deploy/nginx/nginx.conf`) | TLS termination, security headers, rate limiting | Replaced by Istio ingress gateway |
| NetworkPolicy (`networkpolicy.yaml`) | L3/L4: restricts pod-to-pod by port | Complemented (not replaced) by mesh L7 policy |

**What's missing for mesh adoption**:
1. No Istio/Linkerd installation manifests or Helm values.
2. No `sidecar.istio.io/inject` annotations in Helm deployment templates.
3. No `PeerAuthentication`, `AuthorizationPolicy`, or `RequestAuthentication` resources.
4. No service-level `ServiceAccount` definitions (Istio derives identity from SA).
5. No distributed tracing backend (Jaeger/Zipkin) configured for K8s.

### Phased Adoption Plan

#### Phase 1: Sidecar Injection + Permissive mTLS (Week 1-2)

- [ ] Install Istio in `ggid` namespace.
- [ ] Add `istio-injection=enabled` label to namespace.
- [ ] Create dedicated `ServiceAccount` per service:
  ```yaml
  apiVersion: v1
  kind: ServiceAccount
  metadata:
    name: ggid-gateway
    namespace: ggid
  ---
  apiVersion: v1
  kind: ServiceAccount
  metadata:
    name: ggid-auth
    namespace: ggid
  # ... one per service
  ```
- [ ] Set `PeerAuthentication` to `PERMISSIVE` mode (accept both plaintext and mTLS).
- [ ] Verify services still work with mesh enabled.

#### Phase 2: Strict mTLS (Week 3)

- [ ] Switch `PeerAuthentication` to `STRICT` mode.
- [ ] Verify all GGID services communicate via mTLS.
- [ ] Remove plaintext service URLs from ConfigMaps (sidecar handles routing).

#### Phase 3: AuthorizationPolicy (Week 4)

- [ ] Deploy `AuthorizationPolicy` resources (see Section 6).
- [ ] Implement default-deny with explicit allow rules per service pair.
- [ ] Test: verify compromised pod cannot reach unauthorized services.

#### Phase 4: RequestAuthentication + JWT at Proxy (Week 5-6)

- [ ] Deploy `RequestAuthentication` for JWT validation at Envoy (see Section 5).
- [ ] Map JWT claims to headers (`x-tenant-id`, `x-user-id`).
- [ ] Configure per-route JWT requirements (public vs protected paths).
- [ ] Verify Gateway's Go code still validates JWT (defense in depth).

#### Phase 5: Observability (Week 7-8)

- [ ] Deploy Jaeger for distributed tracing.
- [ ] Configure Prometheus to scrape Envoy metrics.
- [ ] Deploy Kiali for mesh visualization.
- [ ] Create Grafana dashboards for mesh metrics.

---

## 10. Gap Analysis & Recommendations

### Gap 1: No mTLS Between Services (P0)

**Current state**: All inter-service communication is plaintext HTTP
(`http://auth:9001`, `http://identity:8080`).

**Risk**: An attacker who compromises any pod can sniff credentials, JWTs, and
PII traversing the network.

**Recommendation**: Deploy Istio with `PeerAuthentication: STRICT` in the `ggid`
namespace.

**Effort**: 2 weeks (Phase 1-2). No Go code changes required.

### Gap 2: No Service-to-Service Authorization (P1)

**Current state**: NetworkPolicy restricts by port (L3/L4), but any pod in the
namespace can call any other pod on the allowed port.

**Risk**: A compromised Audit pod can call Auth's login endpoint, or a compromised
Policy pod can read user data from Identity.

**Recommendation**: Deploy `AuthorizationPolicy` resources that restrict each service
to only the callers that legitimately need access (see Section 6 access matrix).

**Effort**: 1 week. Requires defining per-service ServiceAccounts.

### Gap 3: JWT Validation Only at Application Layer (P1)

**Current state**: JWT validation happens in Go code (`middleware.NewJWKSClient`)
with no proxy-layer enforcement.

**Risk**: If the Go JWT validation has a bug (e.g., algorithm confusion), invalid
tokens reach backend services.

**Recommendation**: Deploy `RequestAuthentication` for Envoy-side JWT validation.
Keep the Go validation as a second layer (defense in depth).

**Effort**: 1 week. Envoy JWT filter config template already exists at
`deploy/envoy/ggid-envoy.yaml`.

### Gap 4: No Distributed Tracing (P2)

**Current state**: No end-to-end tracing for login/register flows that span
Gateway → Auth → Identity → Redis.

**Risk**: Latency bottlenecks and failure points are hard to diagnose without
correlated traces across services.

**Recommendation**: Deploy Jaeger with Istio's automatic trace injection.
No Go code changes needed — Envoy injects W3C trace context headers.

**Effort**: 3 days (deploy Jaeger + configure Istio tracing).

### Gap 5: No Multi-Cluster Capability (P3)

**Current state**: Single-cluster deployment via Helm chart. No cross-region
failover.

**Risk**: Single region outage = complete IAM outage.

**Recommendation**: Phase for future. First establish single-cluster mesh
stability, then add federated trust domains for multi-region.

**Effort**: 4-6 weeks. Requires PostgreSQL logical replication, federated
Istio control planes, and global load balancing.

### Summary: Effort & Impact Matrix

| # | Gap | Priority | Effort | Impact |
|---|---|---|---|---|
| 1 | mTLS between services | P0 | 2 weeks | Eliminates plaintext credential leakage |
| 2 | Service-to-service authz | P1 | 1 week | Prevents lateral movement |
| 3 | JWT at proxy layer | P1 | 1 week | Defense in depth for token validation |
| 4 | Distributed tracing | P2 | 3 days | Faster incident response |
| 5 | Multi-cluster | P3 | 4-6 weeks | Regional failover |

**Total estimated effort**: 8-10 weeks for full mesh adoption (Phases 1-5).
Phase 1-2 (mTLS) delivers the highest security ROI with zero application code changes.

---

## References

- [SPIFFE specification](https://github.com/spiffe/spiffe/blob/main/standards/X509-SVID.md)
- [Istio Security Best Practices](https://istio.io/latest/docs/ops/best-practices/security/)
- [Linkerd mTLS](https://linkerd.io/2/features/automatic-mtls/)
- [Envoy JWT Authentication Filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/jwt_authn_filter)
- [NIST SP 800-207 Zero Trust Architecture](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-207.pdf)
- [Existing GGID Envoy config](../../deploy/envoy/ggid-envoy.yaml)
- [GGID Helm chart](../../deploy/helm/ggid/)
- [GGID zero-trust research](./zero-trust-iam.md)
- [GGID gRPC security research](./grpc-security-iam.md)
- [GGID observability research](./observability-iam.md)
