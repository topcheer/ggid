# ADR-002: gRPC for Internal Communication + REST for External API

- **Status:** Accepted
- **Date:** 2024-01-20

## Context

GGID services need to communicate with each other internally and expose APIs
to external clients (browsers, mobile apps, SDKs, SCIM clients).

Options considered:

1. **gRPC only** — Protocol Buffers for everything, gRPC-Gateway for REST
2. **REST only** — JSON/HTTP for all communication
3. **Hybrid** — gRPC for internal service-to-service, REST for external

### Forces

- External clients (browsers, curl) expect REST/JSON
- SCIM 2.0 specification mandates HTTP/REST
- Internal calls benefit from gRPC's binary efficiency and code generation
- OAuth2/OIDC flows are inherently REST-based
- Maintaining two API definitions (proto + OpenAPI) adds overhead

## Decision

We chose **hybrid**: dual-protocol services (gRPC + REST).

Each service that needs inter-service communication exposes:
- **gRPC server** (Protobuf) — for internal service-to-service calls
- **REST server** (`net/http`) — for external clients via the Gateway

The Gateway only speaks REST. It proxies REST requests to backend services'
REST endpoints. gRPC is reserved for future service mesh scenarios.

Protocol definitions live in `proto/` and generated code in `api/gen/`.

## Consequences

### Positive

- External API is standard REST/JSON — works with curl, fetch, Postman
- gRPC available for high-performance internal calls when needed
- SCIM 2.0 compliance is straightforward (REST handlers)
- Swagger/OpenAPI tooling works out of the box

### Negative

- Dual protocol means maintaining both gRPC handlers and REST handlers
- Some logic duplication between proto types and REST JSON marshaling
- Proto compilation step (`make proto`) adds build complexity

### Neutral

- Generated proto types could be used in REST handlers to reduce duplication
- buf generates Go code for 6 proto definitions
