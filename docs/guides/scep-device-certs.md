# SCEP Device Certificates — Technical Guide

> Feature: Internal CA + SCEP Enrollment for Managed Devices
> Location: `services/auth/internal/server/trust_store_handler.go`
> Endpoints: `/api/v1/auth/certificates/*`

## What It Does

GGID includes an internal Certificate Authority (CA) that issues device certificates via SCEP (Simple Certificate Enrollment Protocol). Managed devices obtain X.509 certificates for mutual TLS authentication, eliminating password-based device auth and enabling certificate-based zero-trust access.

## Architecture

```
MDM (Mobile Device Management)
         ↓
   SCEP Profile Push to Device
         ↓
   Device generates Key Pair + CSR
         ↓
   GGID Internal CA
   ┌────────────────────────┐
   │ Verify enrollment challenge │
   │ ↓                          │
   │ Sign CSR → Issue certificate │
   │ ↓                          │
   │ Return signed certificate   │
   └────────────────────────┘
         ↓
   Device uses cert for mTLS auth
         ↓
   Gateway validates cert against CA
```

## Certificate Lifecycle

### 1. Enrollment

Device requests a certificate via SCEP:

- **Challenge**: Pre-shared challenge password (distributed by MDM).
- **CSR**: Device generates a Certificate Signing Request with its public key.
- **Signing**: GGID CA signs the CSR, returning an X.509 certificate.
- **Validity**: Typically 1 year (configurable).

**API:** `POST /api/v1/auth/certificates/csr` — generate or submit a CSR.

### 2. Authentication

Devices authenticate using mTLS:

- Device presents its certificate during TLS handshake.
- Gateway validates against GGID CA trust store.
- Certificate subject (CN) maps to device identity.
- Certificate fingerprint linked to device posture record.

### 3. Renewal

- MDM triggers renewal before expiration (typically at 80% of validity).
- Device generates new CSR.
- Old certificate remains valid until expiration.
- Overlap period ensures zero downtime.

### 4. Revocation

- Admin revokes certificate via console or API.
- Certificate added to Certificate Revocation List (CRL).
- Gateway checks CRL on each mTLS handshake.
- Revoked connections rejected immediately.

## Trust Store

GGID maintains a trust store of:
- **Internal CA certificate**: Root CA for device certs.
- **Intermediate CAs**: Optional intermediate signing CAs.
- **External CAs**: Trusted third-party CAs for federation.

**API:** `GET/POST /api/v1/auth/certificates/trust-store`

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/auth/certificates/csr` | POST | Generate or submit CSR |
| `/api/v1/auth/certificates/trust-store` | GET/POST | Manage CA trust store |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# Generate CSR
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/auth/certificates/csr" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"common_name":"device-laptop-001","organization":"Acme Corp"}'

# List trust store
NEW_TOKEN="your-jwt-token"
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/auth/certificates/trust-store" \
  -H "Authorization: Bearer $NEW_TOKEN" -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| SCEP enrollment fails | Challenge password mismatch or CA down | Verify challenge; check CA service status |
| mTLS handshake fails | Certificate expired or revoked | Renew certificate; check CRL status |
| CRL not updating | CRL distribution point unreachable | Check CRL endpoint; consider OCSP stapling |
| CSR rejected | Invalid subject or key too weak | Use RSA 2048+ or EC P-256+; verify subject format |

## Best Practices

- **Use MDM for enrollment**: Automate via MDM SCEP profiles — no manual cert install.
- **Short validity**: 1-year certs with auto-renewal better than 10-year certs.
- **Monitor expiration**: Alert on certs expiring within 30 days.
- **Revoke immediately**: On device loss or employee termination, revoke certs first.
- **Use EC over RSA**: EC P-256 certs are smaller and faster than RSA.
- **Pin to CA**: Gateway should only trust the internal CA, not public CAs.
