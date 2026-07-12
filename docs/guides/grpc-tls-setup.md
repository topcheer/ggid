# gRPC TLS Configuration Guide

This guide covers configuring gRPC TLS for secure inter-service communication in GGID — certificate generation, mutual TLS (mTLS), cert rotation, and troubleshooting.

## Overview

GGID services communicate via gRPC. By default, gRPC uses plaintext. For production, all inter-service communication must be encrypted with TLS.

## Service gRPC Ports

| Service | gRPC Port |
|---------|----------|
| Identity | 50051 |
| Policy | 9070 |
| Org | 9071 |
| Audit | 9072 |

## Certificate Generation

### Generate CA

```bash
# Create root CA
openssl genrsa -out ca.key 4096
openssl req -new -x509 -key ca.key -sha256 -days 3650 \
  -out ca.crt \
  -subj "/CN=GGID Internal CA/O=GGID"
```

### Generate Server Certificate

```bash
# Generate key
openssl genrsa -out server.key 2048

# Create CSR with SANs (critical for gRPC)
cat > server.cnf << 'EOF'
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no
[req_distinguished_name]
CN = ggid-internal
[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = localhost
DNS.2 = *.ggid.svc.cluster.local
DNS.3 = identity
DNS.4 = policy
DNS.5 = org
DNS.6 = audit
IP.1 = 127.0.0.1
EOF

openssl req -new -key server.key -out server.csr -config server.cnf
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out server.crt -days 365 -sha256 \
  -extensions v3_req -extfile server.cnf
```

### Generate Client Certificate

```bash
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr \
  -subj "/CN=ggid-gateway/O=GGID"
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out client.crt -days 365 -sha256
```

## mTLS Configuration

### Server-Side (e.g., Policy Service)

```go
import (
    "crypto/tls"
    "crypto/x509"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

func newGRPCServer() *grpc.Server {
    // Load server cert
    cert, _ := tls.LoadX509KeyPair("server.crt", "server.key")

    // Load CA for client verification
    caPEM, _ := os.ReadFile("ca.crt")
    certPool := x509.NewCertPool()
    certPool.AppendCertsFromPEM(caPEM)

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientAuth:   tls.RequireAndVerifyClientCert, // mTLS
        ClientCAs:    certPool,
        MinVersion:   tls.VersionTLS12,
    }

    return grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
}
```

### Client-Side (e.g., Gateway)

```go
func newGRPCConn(target string) *grpc.ClientConn {
    cert, _ := tls.LoadX509KeyPair("client.crt", "client.key")

    caPEM, _ := os.ReadFile("ca.crt")
    certPool := x509.NewCertPool()
    certPool.AppendCertsFromPEM(caPEM)

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      certPool,
        ServerName:   "ggid-internal",
        MinVersion:   tls.VersionTLS12,
    }

    conn, _ := grpc.Dial(target,
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
    )
    return conn
}
```

## Docker Configuration

Mount certificates as volumes:

```yaml
# docker-compose.yaml
services:
  policy:
    volumes:
      - ./certs/ca.crt:/certs/ca.crt:ro
      - ./certs/server.crt:/certs/server.crt:ro
      - ./certs/server.key:/certs/server.key:ro
    environment:
      - GRPC_TLS_ENABLED=true
      - GRPC_TLS_CERT=/certs/server.crt
      - GRPC_TLS_KEY=/certs/server.key
      - GRPC_TLS_CA=/certs/ca.crt
      - GRPC_TLS_CLIENT_AUTH=require
```

## Kubernetes Configuration

```yaml
# k8s secret from certs
kubectl create secret generic ggid-grpc-certs \
  --namespace ggid \
  --from-file=ca.crt \
  --from-file=server.crt \
  --from-file=server.key \
  --from-file=client.crt \
  --from-file=client.key

# Mount in deployment
volumes:
  - name: grpc-certs
    secret:
      secretName: ggid-grpc-certs
volumeMounts:
  - name: grpc-certs
    mountPath: /certs
    readOnly: true
```

## Certificate Rotation

### Zero-Downtime Rotation

```
1. Generate new cert (signed by same CA)
2. Add new cert to cert pool (alongside old)
3. Deploy new cert to servers
4. Deploy new cert to clients
5. After grace period (24h), remove old cert
```
### cert-manager (Kubernetes)

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ggid-grpc-cert
  namespace: ggid
spec:
  secretName: ggid-grpc-certs
  duration: 2160h    # 90 days
  renewBefore: 360h  # 15 days before
  issuerRef:
    name: ggid-ca-issuer
    kind: ClusterIssuer
  dnsNames:
    - localhost
    - "*.ggid.svc.cluster.local"
  usages:
    - server auth
    - client auth
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| `connection error: desc = transport auth` | TLS not configured on one side | Enable TLS on both server and client |
| `certificate signed by unknown authority` | CA mismatch | Ensure both sides trust same CA |
| `cannot validate certificate for x` | Missing SAN | Add DNS name to SAN in cert |
| `tls: bad certificate` | Expired cert | Rotate certificate |
| `client certificate required` | mTLS not configured on client | Add client cert to client config |

### Debug gRPC TLS

```bash
# Test TLS handshake
openssl s_client -connect localhost:9070 -showcerts

# Check certificate details
openssl x509 -in server.crt -text -noout | grep -A2 "Subject Alternative Name"

# Test gRPC connection with debugging
GRPC_GO_LOG_SEVERITY=info GRPC_GO_LOG_VERBOSITY=99 \
  grpcurl -v localhost:9070 list
```

## Security Checklist

- [ ] CA private key stored securely (not in container)
- [ ] All server certs include SANs for all service DNS names
- [ ] mTLS enabled (client certs required)
- [ ] TLS 1.2+ minimum
- [ ] Certificate rotation automated (cert-manager or cron)
- [ ] CA cert distributed to all services
- [ ] Private keys mounted read-only
- [ ] gRPC plaintext disabled in production

## See Also

- [Key Management Lifecycle](../research/key-management-lifecycle.md)
- [Production Checklist](production-checklist.md)
- [Security Audit Checklist](security-audit-checklist.md)
