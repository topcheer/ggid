# Secrets Management — Technical Guide

> Feature: Multi-Provider Secrets Management with Rotation
> Console: `/admin/secrets`

## What It Does

GGID's secrets management system centralizes cryptographic key and secret lifecycle across multiple providers (Vault, AWS KMS, environment variables). Secrets are referenced by URI scheme, enabling provider-agnostic access with automatic rotation and health monitoring.

## Secret URI Schemes

| Scheme | Provider | Example |
|--------|----------|---------|
| `vault://` | HashiCorp Vault | `vault://secret/data/ggid#encryption-key` |
| `aws-kms://` | AWS KMS | `aws-kms://alias/ggid-encryption-key` |
| `env://` | Environment variable | `env://GGID_ENCRYPTION_KEY` |
| `gcp-kms://` | Google Cloud KMS | `gcp-kms://projects/ggid/keyRings/global/cryptoKeys/enc` |
| `azure-kv://` | Azure Key Vault | `azure-kv://ggid-keyvault/encryption-key` |
| `file://` | Local file (dev only) | `file:///etc/ggid/secrets/enc-key` |

## SecretsProvider Interface

```go
type SecretsProvider interface {
    // Resolve retrieves a secret value by URI
    Resolve(ctx context.Context, uri string) ([]byte, error)

    // Health checks provider connectivity
    Health(ctx context.Context) error

    // Rotate generates a new secret value
    Rotate(ctx context.Context, uri string) ([]byte, error)
}
```

## Secret Categories

| Category | Example Secrets | Rotation Frequency |
|----------|----------------|-------------------|
| **Encryption keys** | KEK, DEK wrapping key | Annual |
| **Auth secrets** | JWT signing key, internal auth secret | 90 days |
| **Database** | PostgreSQL connection password | 90 days |
| **API keys** | External service keys (OTX, AbuseIPDB) | Per provider policy |
| **Certificates** | TLS certs, SAML signing cert | Per cert expiry |
| **OAuth** | Client secrets | On compromise |

## Rotation Schedule

Each secret has a configurable rotation schedule:

```json
{
  "uri": "vault://secret/data/ggid#jwt-signing-key",
  "rotation_days": 90,
  "last_rotated": "2026-04-18T00:00:00Z",
  "next_rotation": "2026-07-17T00:00:00Z",
  "status": "healthy"
}
```

When rotation triggers:
1. Generate new secret value.
2. Store in provider (new version).
3. Update application reference (hot reload).
4. Verify new secret works.
5. Mark old version for cleanup.

## Provider Health Monitoring

| Metric | Description |
|--------|-------------|
| `provider.available` | Can reach provider (1/0) |
| `provider.latency_ms` | Resolve latency |
| `secret.rotation_due` | Secrets past rotation date |
| `secret.resolve_errors` | Failed resolve attempts |

Health checked every 60 seconds. Failures trigger alerts.

## Fallback Chain

If primary provider is unavailable, GGID falls back:

```
vault://  →  env://  →  file://
```

The fallback ensures service continuity during Vault outages.

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/admin/secrets` | GET | List secrets (metadata only, no values) |
| `/api/v1/admin/secrets/:id/rotate` | POST | Trigger manual rotation |
| `/api/v1/admin/secrets/health` | GET | Provider health status |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List secrets (metadata only)
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/admin/secrets" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Check provider health
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/admin/secrets/health" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Trigger manual rotation
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/admin/secrets/jwt-signing-key/rotate" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Secret resolve fails | Provider down or secret deleted | Check provider health; verify secret exists |
| Rotation overdue | Rotation not triggered or failed | Manually trigger rotation; check provider logs |
| Fallback to env:// | Vault unreachable | Investigate Vault connectivity; check network policies |
| Latency spike on resolve | Provider overloaded or network issue | Check provider metrics; consider caching |

## Best Practices

- **Never log secret values**: Secrets metadata is safe to log, values are not.
- **Use Vault for production**: env:// and file:// are for development only.
- **Monitor rotation dates**: Alert when secrets are within 7 days of rotation due.
- **Test fallback**: Regularly verify env:// fallback works for critical secrets.
- **Rotate on compromise**: If any secret is suspected leaked, rotate immediately.
- **Separate secrets per tenant**: Use Vault paths with tenant IDs for isolation.
