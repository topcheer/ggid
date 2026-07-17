# Service Mesh & Microsegmentation: East-West Traffic Security for GGID

> **Focus**: Adding service mesh integration (Istio/Envoy or Cilium) to GGID's existing k3s deployment — providing automatic mTLS between services, L7 authorization policies, microsegmentation with default-deny zones, and east-west traffic observability. Advances the Networks pillar from "Traditional" to "Advanced" in CISA ZTMM 2.0.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§10), curl commands (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [The East-West Traffic Problem](#2-the-east-west-traffic-problem)
3. [GGID Current State: North-South Only](#3-ggid-current-state-north-south-only)
4. [Gap Analysis](#4-gap-analysis)
5. [Service Mesh Landscape](#5-service-mesh-landscape)
6. [Recommended Architecture: Istio + Envoy](#6-recommended-architecture-istio--envoy)
7. [Microsegmentation Design](#7-microsegmentation-design)
8. [SPIFFE/SPIRE Integration](#8-spiffespire-integration)
9. [Endpoint Precondition Check](#9-endpoint-precondition-check)
10. [API Design + Configuration](#10-api-design--configuration)
11. [Migration Path](#11-migration-path)
12. [Implementation Backlog with DoD](#12-implementation-backlog-with-dod)
13. [Competitive Differentiation](#13-competitive-differentiation)

---

## 1. Executive Summary

GGID's network security is entirely **north-south** (client → gateway → backend). Once traffic passes the gateway, it flows between backend services (identity, auth, oauth, policy, audit) over **unencrypted plain HTTP** with **no authentication or authorization**. Any pod in the Kubernetes cluster can call any other pod directly, bypassing the gateway entirely.

This is the **ZTMM Networks pillar at "Traditional" maturity** — the lowest of all 5 pillars. The gap:

- **No mTLS between services** — Internal traffic is plaintext
- **No service-to-service auth** — Any pod can call any pod
- **No L7 authorization** — No "service A can call service B on path X"
- **No microsegmentation** — No network isolation between zones
- **No east-west observability** — Internal traffic not traced or monitored

**Recommendation**: Deploy **Istio with Envoy sidecars** on GGID's k3s cluster, providing:
1. Automatic mTLS between all services (cert provisioning + rotation via SPIFFE)
2. L7 authorization policies (GGID policy service as external PDP for Envoy)
3. Microsegmentation (default-deny between service zones, explicit allows)
4. Distributed tracing (Jaeger) + traffic metrics (Prometheus)
5. GGID as **trust anchor** (issues SPIFFE IDs, validates workload identity)

**Estimated effort**: 3 sprints for MVP (Istio install + mTLS + authz policies + observability).

---

## 2. The East-West Traffic Problem

### North-South vs East-West

```
North-South (GGID has this ✅):
  Client → Gateway → Backend Service
  - TLS termination at gateway
  - JWT validation
  - Rate limiting, WAF
  - Policy enforcement (PDP)

East-West (GGID does NOT have this ❌):
  Gateway → Identity Service → Auth Service → Policy Service
  - No TLS (plain HTTP)
  - No authentication (any pod can call)
  - No authorization (no policy check)
  - No encryption (traffic visible on network)
  - No tracing (no visibility into service calls)
```

### Attack Vectors Without East-West Security

| Vector | How | Impact |
|--------|-----|--------|
| **Pod hijack** | Attacker compromises one pod → calls all services directly | Full data access |
| **Network sniffing** | Traffic between services is plaintext | Credential/data theft |
| **Lateral movement** | No segmentation → attacker moves freely | Blast radius = entire cluster |
| **Service impersonation** | No workload identity → fake service accepted | Data exfiltration |
| **Unauthorized access** | No service-level authz → any service calls any API | Privilege escalation |

---

## 3. GGID Current State: North-South Only

### Existing Network Security

| Component | File:Line | Status | Scope |
|-----------|-----------|--------|-------|
| TLS termination | `gateway/` | ✅ | North-south only |
| JWT auth middleware | `gateway/middleware/jwt_auth.go` | ✅ | North-south only |
| mTLS (OAuth client cert) | `oauth/service/jar_mtls.go:212` | ✅ | OAuth clients only |
| API key auth | `gateway/middleware/apikey.go:22` | ✅ | North-south only |
| Rate limiting | `gateway/middleware/` | ✅ | North-south only |
| WAF middleware | `gateway/middleware/` | ✅ | North-south only |
| gRPC inter-service | Various services | ⚠️ | Plain HTTP, no auth |

### What Exists (Partial)

GGID uses gRPC for some inter-service communication but **without TLS or authentication**:
```yaml
# Current inter-service config (NO security):
identity_service_url: "http://identity:8081"
auth_service_url: "http://auth:8082"
policy_service_url: "http://policy:8083"
```

### Service Inventory (K8s Deployment)

| Service | Port | Calls To |
|---------|------|----------|
| Gateway | 8080 | All services |
| Identity | 8081 | — |
| Auth | 8082 | Identity (user lookup) |
| OAuth | 8083 | Auth (credential check) |
| Policy | 8084 | — |
| Audit | 8085 | — |
| Org | 8086 | Identity (user lookup) |
| MCP | 8087 | All services |

---

## 4. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No inter-service mTLS** | Internal traffic plaintext |
| 2 | **No service identity** | Can't verify calling service |
| 3 | **No L7 authz policy** | No "service A can call service B" rules |
| 4 | **No microsegmentation** | All pods in same network namespace |
| 5 | **No distributed tracing** | Can't debug inter-service calls |
| 6 | **No east-west metrics** | Can't detect anomalous internal traffic |
| 7 | **No SPIFFE workload identity** | No cryptographic workload attestation |
| 8 | **No circuit breaking** | Cascading failures possible |

---

## 5. Service Mesh Landscape

| Mesh | Data Plane | Sidecar | mTLS | AuthZ Policy | Performance | Complexity |
|------|-----------|---------|------|-------------|-------------|------------|
| **Istio** | Envoy | Yes | Auto (SPIFFE) | L7 + L3/L4 | Medium overhead | High |
| **Linkerd** | Linkerd2-proxy | Yes | Auto | L7 only | Low overhead | Medium |
| **Consul Connect** | Envoy | Yes | Auto | L7 | Medium | Medium |
| **Cilium** | eBPF (kernel) | **No** | Auto | L3/L4/L7 | **Best** | Medium |
| **Kuma** | Envoy | Yes | Auto | L7 | Medium | Low |

### Recommendation: Istio (primary) + Cilium (future option)

**Istio** is the most mature, widely-deployed mesh with the richest policy model. For GGID:
- Envoy sidecars handle mTLS + L7 authz
- External authorization via gRPC call to GGID policy service (ExtAuthz)
- Distributed tracing (Jaeger) built-in
- Works on k3s (GGID's deployment target)

**Cilium** (eBPF, sidecarless) is the future but requires kernel 5.4+ and is more operationally complex. Keep as P2 option.

---

## 6. Recommended Architecture: Istio + Envoy

```
                    ┌──────────────────────────────────────────────┐
                    │              Istio Control Plane              │
                    │                                              │
                    │  ┌──────────┐ ┌──────────┐ ┌────────────┐   │
                    │  │ Istiod   │ │ Envoy    │ │ Jaeger     │   │
                    │  │ (Pilot + │ │ Filters  │ │ (Tracing)  │   │
                    │  │  Citadel │ │          │ │            │   │
                    │  │  Galley) │ │          │ │            │   │
                    │  └────┬─────┘ └────┬─────┘ └────────────┘   │
                    └───────┼────────────┼────────────────────────┘
                            │            │
                    ┌───────▼────────────▼────────────────────────┐
                    │           Data Plane (Sidecars)              │
                    │                                             │
                    │  ┌─────────┐  ┌─────────┐  ┌─────────┐      │
                    │  │ Gateway │  │Identity │  │  Auth   │      │
                    │  │ +Envoy  │  │ +Envoy  │  │ +Envoy  │      │
                    │  │ (PEP)   │  │         │  │         │      │
                    │  └────┬────┘  └────┬────┘  └────┬────┘      │
                    │       │  mTLS      │  mTLS      │  mTLS     │
                    │       │ ◄─────────►│ ◄─────────►│           │
                    │       │            │            │            │
                    │       │     ┌──────▼───┐  ┌─────▼────┐      │
                    │       │     │ Policy   │  │  OAuth   │      │
                    │       │     │ +Envoy   │  │ +Envoy   │      │
                    │       │     │ (ExtAuthz│  │          │      │
                    │       │     │  target) │  │          │      │
                    │       │     └──────────┘  └──────────┘      │
                    │                                             │
                    │  Every inter-service call:                  │
                    │    1. Envoy sidecar intercepts              │
                    │    2. mTLS established (SPIFFE certs)       │
                    │    3. L7 authz checked (ExtAuthz → GGID)    │
                    │    4. Request forwarded                     │
                    │    5. Response + metrics + trace            │
                    └─────────────────────────────────────────────┘
```

### GGID as External Authorization (ExtAuthz) Provider

```
Service A calls Service B:
  1. Envoy (A sidecar) intercepts outbound request
  2. Envoy establishes mTLS with Envoy (B sidecar)
  3. Envoy sends ExtAuthz check to GGID Policy Service:
     {
       "source": { "principal": "spiffe://ggid/ns/ggid/sa/identity" },
       "destination": { "principal": "spiffe://ggid/ns/ggid/sa/auth" },
       "request": { "path": "/api/v1/auth/check", "method": "POST" }
     }
  4. GGID Policy Service evaluates:
     - Is identity service allowed to call auth service on /check?
     - Check service-to-service policy table
     - Return: allow / deny
  5. If allow → Envoy forwards request
     If deny → Envoy returns 403
```

---

## 7. Microsegmentation Design

### Service Zones

```
┌─────────────────────────────────────────────────────────────┐
│  Zone: public (DMZ)                                         │
│  ┌──────────┐                                               │
│  │ Gateway  │ ← Only service exposed to internet            │
│  └────┬─────┘                                               │
├───────┼─────────────────────────────────────────────────────┤
│       │  Zone: application                                   │
│       ▼                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │ Identity │  │   Auth   │  │  OAuth   │                  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                  │
│       │              │              │                         │
├───────┼──────────────┼──────────────┼─────────────────────────┤
│       │  Zone: policy (core)                                 │
│       ▼              ▼              ▼                         │
│  ┌──────────────────────────────────────┐                    │
│  │         Policy Service (PDP)          │                    │
│  └──────────────────────────────────────┘                    │
├───────────────────────────────────────────────────────────────┤
│  Zone: data (restricted)                                     │
│  ┌──────────┐  ┌──────────┐                                 │
│  │  Audit   │  │   Org    │                                 │
│  └──────────┘  └──────────┘                                 │
└───────────────────────────────────────────────────────────────┘

Rules:
  public → application: ALLOW (gateway → services)
  application → policy: ALLOW (services → PDP)
  application → data: CONDITIONAL (only specific paths)
  data → application: DENY (data zone can't initiate calls)
  policy → data: DENY (PDP is stateless, no outbound)
  public → policy: DENY (can't bypass to PDP directly)
  public → data: DENY (can't access audit directly)
```

### Istio AuthorizationPolicy (L7)

```yaml
# Default deny all inter-service traffic
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: default-deny
  namespace: ggid
spec:
  {}
# (empty selector = deny all)

---
# Allow gateway → identity on specific paths
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: gateway-to-identity
  namespace: ggid
spec:
  selector:
    matchLabels:
      app: identity
  action: ALLOW
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/ggid/sa/gateway"]
    to:
    - operation:
        paths: ["/api/v1/identity/*", "/ggid.identity.v1.*/*"]

---
# Allow auth → identity (user lookup only)
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: auth-to-identity
  namespace: ggid
spec:
  selector:
    matchLabels:
      app: identity
  action: ALLOW
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/ggid/sa/auth"]
    to:
    - operation:
        paths: ["/api/v1/identity/users/*"]

---
# External AuthZ via GGID PDP
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: ext-authz-policy
  namespace: ggid
spec:
  selector:
    matchLabels:
      app: identity
  action: CUSTOM
  provider:
    name: ggid-pdp
  rules:
  - {}  # All requests go through GGID PDP
```

---

## 8. SPIFFE/SPIRE Integration

### SPIFFE Identity Model

```
SPIFFE ID format: spiffe://trust-domain/workload-identifier

GGID services:
  spiffe://ggid.dev/ns/ggid/sa/gateway
  spiffe://ggid.dev/ns/ggid/sa/identity
  spiffe://ggid.dev/ns/ggid/sa/auth
  spiffe://ggid.dev/ns/ggid/sa/oauth
  spiffe://ggid.dev/ns/ggid/sa/policy
  spiffe://ggid.dev/ns/ggid/sa/audit

Trust chain:
  GGID Root CA → Istio CA → Service certificates
  Each service gets an SVID (SPIFFE Verifiable Identity Document)
  SVID is an X.509 cert with SPIFFE URI SAN
```

### GGID as Trust Anchor

```
1. GGID operates the root CA for the mesh
2. Istio's Citadel (CA) is subordinate to GGID's CA
3. GGID validates workload identity before issuing SVIDs:
   - k8s service account token → verify with k8s API
   - Assign SPIFFE ID based on service identity
4. All mTLS certs chain back to GGID root CA
5. GGID can revoke trust for a compromised service
```

---

## 9. Endpoint Precondition Check

### Existing Infrastructure (Reuse)

| Component | File:Line | Status | Reuse |
|----------|-----------|--------|-------|
| Policy gRPC `Check()` | `policy/handler/policy_handler.go:138` | ✅ | ExtAuthz target |
| gRPC inter-service | Various | ✅ | Wrapped by Envoy |
| Gateway JWT auth | `gateway/middleware/jwt_auth.go` | ✅ | North-south PEP |
| Prometheus metrics | `gateway/middleware/metrics.go:50` | ✅ | Extend with mesh metrics |
| K8s deployment | `deploy/k8s/` | ✅ | Add Istio injection |

### New Components Required

| Component | Purpose | Priority |
|-----------|---------|----------|
| Istio control plane (istiod) | mTLS + policy distribution | P0 |
| Envoy sidecars (auto-injected) | Per-service PEP | P0 |
| ExtAuthz gRPC service | Envoy → GGID PDP adapter | P0 |
| Service mesh policy API | CRUD for service-to-service rules | P1 |
| SPIFFE ID registry | DB of workload identities | P1 |
| Mesh observability dashboard | Jaeger + Prometheus integration | P2 |

---

## 10. API Design + Configuration

### Service Mesh Policy (Declarative)

```bash
# Create a service-to-service authorization rule
curl -X POST https://ggid.corp.com/api/v1/mesh/policies \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "auth-can-read-users",
    "source_service": "auth",
    "destination_service": "identity",
    "allowed_paths": ["/api/v1/identity/users/*"],
    "allowed_methods": ["GET"],
    "action": "allow"
  }'

# List mesh policies
curl https://ggid.corp.com/api/v1/mesh/policies \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Check mesh status
curl https://ggid.corp.com/api/v1/mesh/status \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response:
{
  "mtls_status": "STRICT",
  "services_in_mesh": 8,
  "total_services": 8,
  "policies_enforced": 24,
  "last_config_sync": "2026-07-17T10:00:00Z"
}
```

### Istio PeerAuthentication (mTLS Strict)

```yaml
apiVersion: security.istio.io/v1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ggid
spec:
  mtls:
    mode: STRICT  # All inter-service traffic MUST use mTLS
```

---

## 11. Migration Path

### Phase 1: Install Istio (No disruption)

```bash
# 1. Install Istio on k3s
istioctl install --set profile=minimal

# 2. Label namespace for sidecar injection
kubectl label namespace ggid istio-injection=enabled

# 3. Rolling restart to inject sidecars
kubectl rollout restart deployment -n ggid
```

### Phase 2: Enable mTLS PERMISSIVE (Transition)

```yaml
# Allow both mTLS and plaintext (during transition)
apiVersion: security.istio.io/v1
kind: PeerAuthentication
spec:
  mtls:
    mode: PERMISSIVE
```

### Phase 3: Enable mTLS STRICT (Enforce)

```yaml
# After all services have sidecars
apiVersion: security.istio.io/v1
kind: PeerAuthentication
spec:
  mtls:
    mode: STRICT
```

### Phase 4: Add Authorization Policies (Zone isolation)

Apply default-deny → explicit allows (see §7).

### Phase 5: Enable ExtAuthz (GGID PDP)

Connect Envoy ExtAuthz to GGID policy service for per-request service authz.

---

## 12. Implementation Backlog with DoD

### P0 — Istio mTLS + Basic Policies (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Istio control plane install on k3s | ✅ istiod running ✅ Sidecar injection enabled ✅ All services have Envoy sidecars ✅ Zero downtime | 3d |
| 2 | mTLS STRICT enforcement | ✅ All inter-service traffic encrypted ✅ Plain HTTP rejected ✅ Cert auto-rotation ✅ ≥3 tests | 2d |
| 3 | Default-deny + zone policies | ✅ Default deny all ✅ Zone-based allows (public→app, app→policy) ✅ Data zone isolated ✅ ≥3 tests | 3d |
| 4 | Service-to-service authz policies | ✅ Per-path L7 rules ✅ Only allowed paths accessible ✅ ≥3 tests | 3d |
| 5 | Inter-service URL migration (HTTP→mTLS) | ✅ All service URLs work with mesh ✅ No plaintext connections ✅ ≥3 tests | 2d |

### P1 — ExtAuthz + SPIFFE + Observability (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 6 | ExtAuthz gRPC adapter (Envoy → GGID PDP) | ✅ Envoy calls GGID on every inter-service request ✅ Allow/deny enforced ✅ ≥3 tests | 3d |
| 7 | Mesh policy management API | ✅ CRUD for policies ✅ Translates to Istio CRDs ✅ DB-backed ✅ ≥3 tests | 3d |
| 8 | SPIFFE ID registry | ✅ DB of workload identities ✅ SVID verification ✅ ≥3 tests | 2d |
| 9 | Distributed tracing (Jaeger) | ✅ All inter-service calls traced ✅ Latency visible ✅ ≥3 tests | 2d |
| 10 | Mesh observability dashboard | ✅ Traffic metrics per service ✅ Policy decisions ✅ mTLS status | 2d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 11 | Cilium eBPF evaluation | Sidecarless alternative (kernel 5.4+) |
| 12 | Circuit breaking | Prevent cascading failures |
| 13 | Canary deployments | Traffic splitting for zero-downtime deploys |
| 14 | Multi-cluster mesh | Cross-cluster service-to-service auth |
| 15 | WASM Envoy filters | Custom L7 logic via WASM (connects to wasm-plugin research) |

---

## 13. Competitive Differentiation

| Feature | GGID (target) | Okta+Zscaler | Microsoft Entra | Cloudflare | AWS App Mesh |
|---------|---------------|--------------|-----------------|-----------|--------------|
| **Inter-service mTLS** | **Istio automatic** | Zscaler ZPA | Azure Mesh | Cloudflare Tunnels | App Mesh mTLS |
| **L7 authz policies** | **GGID PDP via ExtAuthz** | Network-level | Azure Policy | Cloudflare Rules | IAM roles |
| **Microsegmentation** | **Zone-based default-deny** | ZPA segments | NSG + ASG | Zero Trust segments | Security groups |
| **SPIFFE identity** | **GGID as trust anchor** | Custom | Managed Identity | Access Service Accounts | IRSA |
| **East-west observability** | **Jaeger + Prometheus** | ZIA logs | Azure Monitor | Cloudflare Analytics | CloudWatch |
| **GGID PDP integration** | **Native (ExtAuthz)** | No | No | No | No |
| **Open source** | **Yes (Istio + GGID)** | No | No | No | No |

**Key differentiator**: GGID would be the only open-source IAM that acts as **both north-south PEP (gateway) and east-west PDP (mesh ExtAuthz)** — a unified policy plane across all traffic directions, not just perimeter.

---

## References

- [Istio Service Mesh](https://istio.io/) — Production service mesh
- [Envoy Proxy ExtAuthz](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter) — External authorization
- [SPIFFE/SPIRE](https://spiffe.io/) — Workload identity standard
- [Cilium](https://cilium.io/) — eBPF-based networking
- [Linkerd](https://linkerd.io/) — Lightweight mesh
- [NIST SP 800-204A](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-204A.pdf) — Building secure microservices
- [CISA ZTMM 2.0: Networks Pillar](https://www.cisa.gov/zero-trust-maturity-model) — Network maturity requirements
- [GGID Policy gRPC Handler](../services/policy/internal/handler/policy_handler.go) — Check() RPC at line 138
- [GGID Zero Trust Maturity Assessment](./zero-trust-maturity-assessment.md) — Networks pillar at "Traditional"
- [GGID Continuous Authorization & PDP](./continuous-authorization-pdp.md) — Unified PDP architecture
- [GGID WASM Plugin Architecture](./wasm-plugin-architecture.md) — WASM Envoy filter connection
