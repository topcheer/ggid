# mTLS Between Services

Cert generation, cert-manager integration, per-service config, rotation automation, troubleshooting, and performance impact.

## Overview

All inter-service communication in GGID uses mutual TLS (mTLS) to prevent man-in-the-middle attacks and ensure service identity.

## Certificate Architecture

```
                    GGID Internal CA (cert-manager)
                           │
          ┌────────────────┼────────────────┐
          │                │                │
     Gateway Cert     Auth Cert      Identity Cert
     (CN: gateway)    (CN: auth)    (CN: identity)
```

Each service has its own certificate with CN matching the service DNS name.

## cert-manager Integration

### Cluster Issuer

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: ggid-internal-ca
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ggid-ca
spec:
  isCA: true
  secretName: ggid-ca-secret
  issuerRef: {name: ggid-internal-ca}
  commonName: "GGID Internal CA"
```

### Per-Service Certificate

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: auth-tls
spec:
  secretName: auth-tls
  duration: 2160h    # 90 days
  renewBefore: 360h  # Renew 15 days before
  dnsNames:
    - auth.ggid.svc.cluster.local
    - auth
  issuerRef: {name: ggid-internal-ca}
  usages: [server auth, client auth]
```

## gRPC mTLS Configuration

### Server

```go
creds, _ := credentials.NewServerTLSFromFile("/certs/tls.crt", "/certs/tls.key")
srv := grpc.NewServer(
    grpc.Creds(creds),
    grpc.UnaryInterceptor(authInterceptor),
)
```

### Client

```go
creds, _ := credentials.NewClientTLSFromFile("/certs/ca.crt", "")
conn, _ := grpc.Dial("auth.ggid.svc:9080",
    grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
        Certificates: []tls.Certificate{clientCert},
        RootCAs:      caPool,
        ServerName:   "auth.ggid.svc.cluster.local",
        MinVersion:   tls.VersionTLS13,
    })),
)
```

## Rotation Automation

| Setting | Value |
|---------|-------|
| Certificate duration | 90 days |
| Renew before expiry | 15 days |
| Algorithm | ECDSA P-256 |
| Key size | 256 bits |
| TLS minimum | 1.3 |

cert-manager handles rotation automatically. Pods pick up new certs via mounted Secret volume (auto-updated).

## Performance Impact

| Metric | Without mTLS | With mTLS | Overhead |
|--------|-------------|-----------|----------|
| Connection setup | 0.5ms | 2ms | +1.5ms |
| Request latency | 5ms | 5.1ms | +0.1ms |
| CPU per connection | baseline | +2% | Minimal |
| Memory per connection | baseline | +10KB | Minimal |

Connection pooling amortizes handshake cost — only first request pays.

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| `certificate signed by unknown authority` | CA cert mismatch | Ensure client trusts internal CA |
| `cannot validate certificate` | Wrong DNS name in cert | Check `dnsNames` in Certificate spec |
| `certificate expired` | Rotation failed | Check cert-manager pod logs |
| `handshake failure` | TLS version mismatch | Both sides must support TLS 1.3 |

## Monitoring

| Metric | Alert |
|--------|-------|
| Certificate expiry | <15 days → renewing (cert-manager) |
| TLS handshake failures | >1% → cert or config issue |
| cert-manager errors | Any → investigate |

## See Also

- [Service Mesh Integration](service-mesh-integration.md)
- [gRPC Interceptor Patterns](grpc-interceptor-patterns.md)
- [Secrets Rotation Automation](secrets-rotation-automation.md)
- [Gateway Architecture](gateway-architecture.md)
