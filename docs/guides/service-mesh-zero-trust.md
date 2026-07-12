# Service Mesh Zero Trust

This guide covers Istio/Linkerd zero-trust patterns, mTLS everywhere, authorization policies, workload identity (SPIFFE), traffic encryption, ingress/egress security, and GGID's service mesh deployment.

## Zero-Trust Service Mesh

### Core Principles

| Principle | Implementation |
|---|---|
| No implicit trust | Every connection authenticated |
| mTLS everywhere | All inter-service traffic encrypted |
| Deny-by-default | No traffic allowed without explicit policy |
| Workload identity | SPIFFE IDs for service identity |
| Continuous verification | Every request authorized |
| Least privilege | Only required ports/paths allowed |

## Istio Zero-Trust Patterns

### mTLS Automatic Cert Rotation

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ggid
spec:
  mtls:
    mode: STRICT  # All traffic must use mTLS
```

Istio automatically:
- Issues certificates per workload
- Rotates certificates (15 min default)
- Distributes CA root to all workloads
- Enforces mTLS on all connections

### Authorization Policies (Deny-by-Default)

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: deny-all
  namespace: ggid
spec: {}  # Empty spec = deny all
---
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: allow-gateway-to-auth
  namespace: ggid
spec:
  selector:
    matchLabels:
      app: auth
  rules:
    - from:
        - source:
            principals: ["cluster.local/ns/ggid/sa/gateway"]
      to:
        - operation:
            methods: ["GET", "POST"]
            paths: ["/api/v1/auth/*"]
```

### Workload Identity (SPIFFE)

```
SPIFFE ID: spiffe://ggid.example.com/ns/ggid/sa/auth-service
```

| Service | SPIFFE ID |
|---|---|
| Gateway | spiffe://ggid.example.com/ns/ggid/sa/gateway |
| Auth | spiffe://ggid.example.com/ns/ggid/sa/auth |
| OAuth | spiffe://ggid.example.com/ns/ggid/sa/oauth |
| Identity | spiffe://ggid.example.com/ns/ggid/sa/identity |
| Policy | spiffe://ggid.example.com/ns/ggid/sa/policy |
| Audit | spiffe://ggid.example.com/ns/ggid/sa/audit |

## Linkerd Zero-Trust Patterns

```yaml
# Linkerd mTLS (automatic)
apiVersion: linkerd.io/v1alpha2
kind: Server
metadata:
  name: auth-server
spec:
  podSelector:
    matchLabels:
      app: auth
  port: 8080
  proxyProtocol: HTTP/2
```

Linkerd automatically enables mTLS between all meshed services.

## Traffic Encryption

| Traffic | Encryption | Method |
|---|---|---|
| Client → Ingress | TLS 1.2+ | Standard TLS |
| Ingress → Service | mTLS | Service mesh |
| Service → Service | mTLS | Service mesh |
| Service → Database | TLS | PostgreSQL TLS |
| Service → Redis | TLS | Redis TLS |
| Service → NATS | TLS | NATS TLS |

## Ingress Gateway Security

```yaml
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: jwt-auth
  namespace: ggid
spec:
  selector:
    matchLabels:
      istio: ingressgateway
  jwtRules:
    - issuer: "https://auth.ggid.example.com"
      jwksUri: "https://auth.ggid.example.com/.well-known/jwks.json"
      audiences: ["ggid-api"]
```

## Egress Gateway Control

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: allow-hibp-api
spec:
  hosts: ["api.haveibeenpwned.com"]
  ports: [{number: 443, name: https, protocol: HTTPS}]
  resolution: DNS
  location: MESH_EXTERNAL
```

Only explicitly allowed external services are reachable.

## GGID Service Mesh Deployment

```yaml
service_mesh:
  platform: "istio"  # or "linkerd"
  mtls:
    mode: "STRICT"
    cert_rotation: 15m
  authorization:
    default: "deny-all"
    policies: "explicit-allow"
  workload_identity:
    spiffe:
      enabled: true
      trust_domain: "ggid.example.com"
  ingress:
    jwt_validation: true
    rate_limiting: true
  egress:
    allowed_external: ["api.haveibeenpwned.com", "smtp.example.com"]
    deny_by_default: true
  observability:
    tracing: true
    metrics: true
    access_logs: true
```

## Best Practices

1. **STRICT mTLS** — Don't allow plaintext inter-service traffic
2. **Deny-by-default** — No traffic without explicit allow policy
3. **Use SPIFFE IDs** — Stable workload identity independent of IP
4. **Control egress** — Don't let services call arbitrary external APIs
5. **Validate JWT at ingress** — Reject invalid tokens at gateway
6. **Rotate certs automatically** — Short-lived certs reduce risk
7. **Enable tracing** — Full request tracing for debugging
8. **Monitor mesh health** — Track mTLS success rate, policy denials
9. **Version policies carefully** — Bad policy can block all traffic
10. **Test in staging first** — Verify mesh policies before production