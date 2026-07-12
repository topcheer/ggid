# Certificate Lifecycle Automation

ACME integration, cert-manager with Let's Encrypt, wildcard vs SAN, ECDSA P-256, automated renewal, OCSP stapling, CRL, and cert pinning.

## Automated Renewal

| Cert Type | Provider | Rotation | Renew Before |
|-----------|----------|----------|-------------|
| Public TLS | Let's Encrypt (ACME) | 90 days | 30 days |
| Internal mTLS | Internal CA (cert-manager) | 90 days | 15 days |
| SAML signing | Internal CA | 365 days | 30 days |
| Client certs | Internal CA | 90 days | 15 days |

## cert-manager + ACME

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: security@ggid.dev
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ggid-tls
spec:
  secretName: ggid-tls
  duration: 2160h    # 90 days
  renewBefore: 720h  # 30 days
  dnsNames:
    - ggid.dev
    - "*.ggid.dev"
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
```

## ECDSA Preference

| Algorithm | Key Size | Signature Size | Security Level |
|-----------|---------|---------------|---------------|
| RSA-2048 | 2048 bits | 256 bytes | 112-bit |
| ECDSA P-256 | 256 bits | 64 bytes | 128-bit |
| ECDSA P-384 | 384 bits | 96 bytes | 192-bit |

GGID uses ECDSA P-256: smaller keys, faster signing, same security.

## OCSP Stapling

```nginx
ssl_stapling on;
ssl_stapling_verify on;
ssl_trusted_certificate /certs/ca.crt;
resolver 8.8.8.8 valid=300s;
```

Server fetches OCSP response and includes it in TLS handshake — client doesn't need to contact CA.

## Cert Pinning

For mobile apps (can't rotate CA trust easily):

```swift
// Pin the public key hash (survives cert rotation within same key)
let pinnedKeyHash = "sha256/base64-of-public-key"
// Validate on every TLS connection
```

Pin the **public key**, not the certificate — survives cert renewal as long as key stays same.

## Monitoring

| Metric | Alert |
|--------|-------|
| Cert expiry <30 days | Renewal should trigger |
| Cert renewal failures | Any → manual intervention |
| OCSP fetch failures | Any → clients can't verify |
| Cert chain incomplete | Any → client trust error |

## See Also

- [mTLS Between Services](mtls-between-services.md)
- [Cryptography Key Rotation](cryptography-key-rotation.md)
- [Secrets Rotation Automation](secrets-rotation-automation.md)
