# Certificate Lifecycle

Certificate inventory, issuance workflow, automated renewal, expiry monitoring, revocation (CRL/OCSP), chain of trust, pinning strategy, and CT monitoring.

## Certificate Types

| Type | Purpose | Lifetime | CA |
|------|---------|----------|-----|
| Public TLS | HTTPS for ggid.dev | 90 days | Let's Encrypt |
 Internal mTLS | Service-to-service | 90 days | Internal CA |
 SAML signing | Sign assertions | 365 days | Internal CA |
 SAML encryption | Encrypt assertions | 365 days | Internal CA |
 Client cert | mTLS client auth | 90 days | Internal CA |
 Code signing | Sign releases | 365 days | Internal CA |

## CA Hierarchy

```
Root CA (offline, self-signed)
  └── Intermediate CA (online, signs leaf certs)
       └── Leaf certificates (per service/domain)
```

Root CA private key is offline (HSM or air-gapped). Intermediate CA signs all leaf certs.

## Issuance Workflow

```
1. Generate CSR (Certificate Signing Request)
   → openssl req -new -key service.key -out service.csr
2. Submit CSR to CA
3. CA validates identity (domain ownership for public, internal policy for private)
4. CA signs certificate
5. Deploy certificate + key to service
6. Update trust store if new CA
```

## Automated Renewal (ACME / cert-manager)

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ggid-tls
spec:
  secretName: ggid-tls
  duration: 2160h      # 90 days
  renewBefore: 720h    # 30 days before
  dnsNames: ["ggid.dev", "*.ggid.dev"]
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
```

cert-manager handles: issuance, renewal, secret rotation, pod restart.

## Expiry Monitoring & Alerting

| Threshold | Action |
|-----------|--------|
| 60 days | Log info |
| 30 days | Alert #ops (renewal should trigger) |
| 7 days | Critical alert (renewal failed) |
| Expired | Page on-call (service down) |

```bash
GET /api/v1/admin/certificates/expiry
# → [
#   {"name": "ggid-tls", "expires": "2025-04-01", "days_remaining": 25},
#   {"name": "auth-mtls", "expires": "2025-03-15", "days_remaining": 8, "alert": true}
# ]
```

## Revocation

### CRL (Certificate Revocation List)

```bash
# Generate CRL
openssl ca -gencrl -out crl.pem
# Published at: https://ca.ggid.dev/crl.pem
```

### OCSP Stapling

```nginx
ssl_stapling on;
ssl_stapling_verify on;
# Server fetches OCSP response, includes in TLS handshake
# Client doesn't need to contact CA
```

| Method | Real-time | Client overhead | Use |
|--------|----------|----------------|-----|
| CRL | No (periodic download) | Large file | Bulk revocation |
| OCSP | Yes (per request) | Extra round trip | Real-time check |
| OCSP stapling | Yes (server-side) | None (in handshake) | Best practice |

## Chain of Trust

```
Client trusts Root CA (pre-installed in OS/browser)
  → Root CA signed Intermediate CA
  → Intermediate CA signed Leaf cert
  → Client verifies: leaf → intermediate → root (all valid)
```

If any link is broken (expired, revoked), the chain fails.

## Certificate Transparency (CT) Monitoring

```bash
# Monitor CT logs for unauthorized certs for ggid.dev
GET /api/v1/admin/certificates/ct-monitor
# → Scans CT logs (crt.sh, Google CT)
# → Alerts on certs issued for ggid.dev that weren't requested by GGID
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Cert expiring <30d | Renewal should trigger |
| Renewal failure | Any → manual intervention |
| OCSP fetch failure | Any → clients can't verify |
| Unauthorized CT cert | Any → CA compromise |
| Chain incomplete | Any → client trust error |

## See Also

- [Certificate Lifecycle Automation](certificate-lifecycle-automation.md)
- [mTLS Between Services](mtls-between-services.md)
- [Cryptography Key Rotation](cryptography-key-rotation.md)
- [Key Rotation Strategy](key-rotation-strategy.md)