# Competitive Research: API Gateway Landscape (Kong / APISIX / Envoy)

> Research date: January 2025
> Scope: Features from Kong Gateway 3.x, Apache APISIX 3.12, Envoy Gateway 1.1+
> Goal: Identify capabilities GGID Gateway could adopt

---

## Executive Summary

The API gateway landscape has converged on three dominant patterns:
1. **AI Gateway** — LLM proxying, token rate limiting, prompt engineering, semantic caching
2. **Plugin ecosystems** — Wasm extensions, marketplace plugins, dynamic loading
3. **Kubernetes-native** — Gateway API (GAMMA), declarative CRD-based configuration

Our GGID Gateway already has: JWT auth, RBAC, tenant isolation, CORS, rate limiting, circuit breaker, gRPC-Web, GraphQL proxy, WebSocket, canary routing, health scoring, Prometheus metrics, audit logging, API key rotation, sticky sessions, and more. This document identifies gaps and opportunities.

---

## 1. Kong Gateway 3.x

### Current Release: Kong AI Gateway 3.11 (April 2025)

### Key Features

| Feature | Description | GGID Status |
|---------|-------------|-------------|
| **AI Prompt Compression** | Compresses prompts before sending to LLM, reducing token spend by 30-50% | Not applicable (non-AI gateway) |
| **AWS Bedrock Guardrails** | Content filtering, prompt injection detection via AWS Bedrock | Not applicable |
| **Multimodal AI Proxy** | Proxy requests to text, image, audio, and video LLM models | Not applicable |
| **Semantic Caching** | Cache LLM responses by semantic similarity (not exact match) | Not applicable |
| **DeGraphQL Plugin** | Converts REST API endpoints to GraphQL queries automatically | **Gap** — we have GraphQL→REST, not REST→GraphQL |
| **OAS Validation** | Validate incoming requests against OpenAPI spec at the gateway | **Gap** — could add OAS validation middleware |
| **Canary Release** | Percentage-based traffic splitting with health-gated promotion | **Have** (canary.go) |
| **Rate Limiting (Advanced)** | Consumer-level, plugin-level, and response-code-aware rate limiting | **Have** (basic), gap in consumer-level limits |
| **mTLS** | Mutual TLS between gateway and upstream services | **Gap** — upstream TLS not implemented |
| **DecK** | Declarative configuration management (GitOps-style) | **Have** (config reload via SIGHUP/admin API) |

### What GGID Could Adopt

1. **OAS Validation middleware** — Validate request/response against OpenAPI 3.0 spec. Blocks malformed requests before they reach the backend. Reduces backend error handling load.

2. **DeGraphQL pattern** — Some users have GraphQL backends but want REST APIs. The gateway translates REST paths into GraphQL queries. Complements our existing GraphQL→REST resolver.

3. **Consumer-level rate limiting** — Rate limit per API consumer (identified by API key or JWT subject), not just per IP or tenant. Useful for metered API plans.

---

## 2. Apache APISIX 3.12

### Current Release: 3.12.0 (April 2025)

### Key Features

| Feature | Description | GGID Status |
|---------|-------------|-------------|
| **AI Proxy (ai-proxy)** | Proxy to OpenAI/DeepSeek-compatible LLM services | Not applicable |
| **AI Proxy Multi (ai-proxy-multi)** | Multi-model load balancing with weighted routing, retries, fallbacks | Not applicable, but pattern applicable to multi-backend |
| **AI Rate Limiting (ai-rate-limiting)** | Token-based rate limiting (not request-based) | Not applicable |
| **AI Request Rewrite** | LLM-driven request transformation (redact PII, enrich data) | Not applicable, but pattern interesting |
| **AI Prompt Decorator** | Inject system prompts before/after user input | Not applicable |
| **AI Prompt Template** | Fill-in-the-blank prompt templates with variables | Not applicable |
| **AI Prompt Guard** | Allow/deny pattern matching on prompts | Not applicable |
| **AI Content Moderation** | AWS Comprehend integration for toxic content detection | Not applicable |
| **AI RAG** | Retrieval-augmented generation via Azure AI Search | Not applicable |
| **Wasm Plugin Support** | Write plugins in any language compiled to WebAssembly | **Gap** — no plugin extension mechanism |
| **Dynamic Upstream** | Change upstream nodes without reload via etcd watch | **Have** (admin API toggle + reload) |
| **Health Check** | Passive + active health checks with circuit breaker | **Have** (health scoring + circuit breaker) |
| **Key-Value Store** | Built-in KV store for plugin state sharing across instances | **Gap** — no shared plugin state |
| **Service Mesh** | Built-in service mesh mode (east-west traffic) | **Gap** — we have Envoy sidecar template only |
| **Observability** | TTFT (time to first token), token usage, error rates in access log | **Have** (Prometheus metrics, structured logging) |

### What GGID Could Adopt

1. **Wasm plugin system** — Allow users to write custom middleware in Rust/Go/AssemblyScript compiled to Wasm. This is the #1 most-requested gateway extensibility feature. The Proxy-Wasm ABI standard allows plugins to run in any Wasm-compatible runtime.

2. **Multi-backend weighted routing** — Our current proxy is single-backend per route. APISIX's `ai-proxy-multi` pattern of weighted routing across multiple backends with health checks and failover could be generalized to any backend type (not just LLMs).

3. **Passive health checks** — We have active health checks (probing) but no passive health checking (tracking failure rates from real requests and automatically circuit-breaking). This would complement our existing health scoring.

4. **Plugin state sharing** — APISIX uses a shared KV store (etcd) for plugin state across gateway instances. For our multi-instance deployments, a Redis-backed plugin state store would enable consistent rate limiting and circuit breaking across instances.

---

## 3. Envoy Gateway 1.1 / Envoy Proxy 1.31

### Current Release: Envoy Gateway 1.2+ (late 2024), Envoy Proxy 1.33 (early 2025)

### Key Features

| Feature | Description | GGID Status |
|---------|-------------|-------------|
| **Gateway API (GAMMA)** | Full Kubernetes Gateway API implementation with CRD-based routing | **Gap** — no K8s-native config |
| **Filter Chaining** | Define and order HTTP filters declaratively | **Have** (middleware chain, but not declarative) |
| **Gradual mTLS Rollout** | Incrementally enforce mTLS between client and gateway | **Gap** — no mTLS support |
| **ExtProc** | Call external gRPC process for request/response processing | **Gap** — no external processing hook |
| **Wasm Extensions** | EnvoyExtensionPolicy for Wasm filter injection | **Gap** — same as APISIX |
| **HTTP/3 Happy Eyeballs** | Automatic HTTP/3 with fallback to HTTP/2 | **Gap** — HTTP/2 only |
| **Backend Traffic Policy** | Reusable traffic policies (timeouts, retries, circuit breaking) across routes | **Partial** — per-route timeouts, but not reusable policies |
| **Grafana Dashboard** | Pre-built Grafana dashboards for Envoy metrics | **Gap** — we have Prometheus but no pre-built dashboards |
| **Zipkin Tracing** | Distributed tracing integration | **Have** (OpenTelemetry tracing) |
| **Route Metadata** | Attach arbitrary metadata to routes for observability | **Gap** — no route metadata |
| **Service Mesh Integration** | Route to Service Cluster IP targets (east-west traffic) | **Gap** — north-south only |
| **Redis Proxy** | Built-in Redis protocol proxy with command filtering | **Gap** — not applicable for IAM |

### What GGID Could Adopt

1. **Backend Traffic Policies** — Instead of configuring timeouts/retries/circuit-breaking per-route, define reusable traffic policies that can be attached to multiple routes. This reduces config duplication and enables consistent policy enforcement.

2. **ExtProc pattern** — An external processing hook that calls a gRPC service for request/response transformation. More flexible than Wasm for complex transformations. Could be used for custom auth flows, request enrichment, or compliance checks.

3. **Route Metadata** — Allow attaching key-value metadata to routes. This metadata could be used for:
   - Observability: tag metrics with route owner, environment, cost center
   - Routing decisions: metadata-based routing rules
   - Documentation: auto-generate docs from route metadata

4. **Pre-built Grafana dashboards** — Package a Grafana dashboard JSON that visualizes GGID Gateway metrics (request rate, latency, error rate, circuit breaker state, health scores).

5. **HTTP/3 (QUIC) support** — Envoy's happy eyeballs implementation automatically upgrades to HTTP/3 when the client supports it. This reduces connection latency (0-RTT) and improves throughput. Go's `net/http` has experimental QUIC support.

---

## Feature Gap Analysis — Priority Recommendations

### High Priority (implement next)

| # | Feature | Effort | Impact |
|---|---------|--------|--------|
| 1 | **OAS Validation middleware** | Medium | Blocks malformed requests, reduces backend errors |
| 2 | **Backend Traffic Policies** | Medium | Reusable timeout/retry/circuit-breaker configs |
| 3 | **Passive health checks** | Small | Auto-detect unhealthy backends from real traffic |
| 4 | **Pre-built Grafana dashboard** | Small | Out-of-box observability |

### Medium Priority (next quarter)

| # | Feature | Effort | Impact |
|---|---------|--------|--------|
| 5 | **Route metadata** | Small | Better observability and routing flexibility |
| 6 | **Consumer-level rate limiting** | Medium | Per-API-key/per-subject rate limits |
| 7 | **Multi-backend weighted routing** | Medium | Load balance across multiple backends per route |
| 8 | **ExtProc hook** | Large | External request/response transformation via gRPC |

### Low Priority (future consideration)

| # | Feature | Effort | Impact |
|---|---------|--------|--------|
| 9 | **Wasm plugin system** | Large | Full extensibility, ecosystem play |
| 10 | **HTTP/3 support** | Large | Modern transport, reduced latency |
| 11 | **mTLS to upstream** | Medium | Zero-trust backend communication |
| 12 | **Service mesh mode** | Large | East-west traffic management |

---

## Comparison Matrix

| Capability | Kong 3.x | APISIX 3.12 | Envoy Gateway 1.2 | **GGID Gateway** |
|-----------|----------|-------------|-------------------|------------------|
| AI Gateway | Yes (3.11) | Yes (3.12) | No | No |
| Wasm Plugins | Yes | Yes | Yes | **No** |
| Gateway API (K8s) | Yes (via operator) | Yes (Ingress) | Yes (native) | **No** |
| GraphQL Proxy | Yes (plugin) | No | No | **Yes** |
| gRPC-Web | Yes | No | Yes | **Yes** |
| Circuit Breaker | Yes | Yes | Yes | **Yes** |
| Rate Limiting | Yes (advanced) | Yes (advanced) | Yes | **Yes (basic)** |
| Multi-tenant | No | No | No | **Yes (native)** |
| RBAC at Gateway | No | No | No | **Yes** |
| WebSocket | Yes | Yes | Yes | **Yes** |
| Canary Routing | Yes | Yes | Yes | **Yes** |
| Hot Reload | Yes | Yes (etcd) | Yes (CRD) | **Yes (SIGHUP)** |
| Prometheus | Yes | Yes | Yes | **Yes** |
| OTel Tracing | Yes | Yes | Yes | **Yes** |
| mTLS | Yes | Yes | Yes | **No** |
| Health Scoring | No | No | No | **Yes** |
| Sticky Sessions | No | Yes | Yes | **Yes** |
| API Key Rotation | Yes | No | No | **Yes** |
| Audit Logging | No | No | No | **Yes (NATS)** |

### GGID Unique Advantages

1. **Native multi-tenancy** — Only gateway with built-in tenant isolation, tenant_id injection, and per-tenant rate limiting
2. **IAM-native** — RBAC enforcement at the gateway layer, JWT verification with JWKS, API key lifecycle management
3. **Health scoring** — Novel backend health scoring algorithm (success rate + latency + decay) not found in competitors
4. **Audit-first architecture** — Async NATS-based audit logging built into the gateway
5. **GraphQL + gRPC-Web** — Both protocol adaptations built-in without plugins

---

## Conclusion

GGID Gateway is competitive on core gateway functionality (routing, auth, observability, protocol support) and leads in IAM-specific features (multi-tenancy, RBAC, audit). The main gaps are:

1. **Extensibility** (Wasm plugins, ExtProc) — This is the biggest competitive disadvantage. Kong, APISIX, and Envoy all support custom plugins.
2. **K8s-native configuration** (Gateway API) — GGID is config-file based, not CRD-based.
3. **Advanced traffic policies** — Reusable backend policies, passive health checks, consumer-level rate limits.

The AI Gateway wave (Kong 3.11, APISIX 3.12) is a separate product category and not directly relevant to GGID's IAM focus, but the plugin architecture patterns (prompt guard, request rewrite) could inspire similar middleware patterns for IAM (e.g., request sanitization, PII redaction at the gateway).
