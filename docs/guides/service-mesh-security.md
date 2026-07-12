# Service Mesh Security

Istio/Linkerd mTLS automation, sidecar proxy auth, authorization policies (JWT claims → mesh rules), zero-trust east-west security, and Envoy token validation.

## Overview

Service mesh secures all east-west (service-to-service) traffic automatically via mTLS and authorization policies, without application code changes.

## mTLS Automation

### Istio

```yaml
# PeerAuthentication: enforce STRICT mTLS
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ggid
spec:
  mtls:
    mode: STRICT   # Reject non-mTLS traffic
```

### Cert Rotation

| Aspect | Istio Default |
|--------|--------------|
| Cert validity | 24 hours |
| Rotation | Automatic (istiod) |
| CA | Self-signed (prod: external CA) |
| Key type | ECDSA P-256 |

## Authorization Policies

### JWT Claim → Mesh Rule

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: identity-svc-authz
  namespace: ggid
spec:
  selector:
    matchLabels:
      app: identity
  action: ALLOW
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/ggid/sa/gateway"]  # Only gateway
    to:
    - operation:
        methods: ["GET", "POST", "PATCH", "DELETE"]
        paths: ["/api/v1/users/*"]
    when:
    - key: request.auth.claims[scope]
      values: ["*users:read*", "*users:write*"]
```

### Per-Service Rules

| Service | Allowed Callers | Scope Check |
|---------|----------------|-------------|
| Identity | Gateway only | `users:read` or `users:write` |
| Auth | Gateway only | No scope (auth handles its own) |
| Policy | Gateway, Identity, Auth | `policy:read` or `policy:evaluate` |
| Audit | All services (write only) | None (trusted internal) |

## Zero-Trust East-West

### Deny All by Default

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: deny-all
  namespace: ggid
spec: {}  # Empty spec = deny all traffic
```

Then add explicit ALLOW policies per service.

### L7 Policy (HTTP-level)

```yaml
rules:
- to:
  - operation:
      paths: ["/api/v1/admin/*"]
  when:
  - key: request.auth.claims[scope]
    values: ["*admin:*"]
  - key: request.auth.claims[tenant_id]
    values: ["${request.headers[x-tenant-id]}"]
```

## Envoy Filter for Token Validation

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: jwt-validation
  namespace: ggid
spec:
  configPatches:
  - applyTo: HTTP_FILTER
    match:
      proxy:
        proxyVersion: "^1.*"
    patch:
      operation: INSERT_BEFORE
      value:
        name: envoy.filters.http.jwt_authn
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication
          providers:
            ggid:
              issuer: "https://auth.ggid.dev"
              jwks_remote_uri: "https://auth.ggid.dev/.well-known/jwks.json"
              audiences: ["ggid-gateway"]
          rules:
          - match: {prefix: "/api/"}
            requires: {provider_name: "ggid"}
```

Mesh validates JWT at proxy level — requests never reach application if token is invalid.

## Mutual SLO

| Metric | Target | Enforcement |
|--------|--------|-------------|
| mTLS coverage | 100% | PeerAuthentication STRICT |
| Policy coverage | 100% services | Deny-all + explicit allows |
| Cert rotation | 0 manual | Automatic via istiod |
| Policy violations | 0 | Audit log + alert |

## Monitoring

| Metric | Alert |
|--------|-------|
| mTLS failures | Any → misconfigured pod |
| Policy denials | >1% → investigate |
| Cert rotation failures | Any → cert management issue |
| Unauthorized connection attempts | Spike → possible lateral movement |

## See Also

- [Service Mesh Integration](service-mesh-integration.md)
- [Service Mesh Observability](service-mesh-observability.md)
- [mTLS Between Services](mtls-between-services.md)
- [Zero Trust Network Design](zero-trust-network-design.md)
