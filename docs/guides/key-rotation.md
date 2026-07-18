# Key Rotation — Technical Guide

> Feature: Automated Key Rotation with Dual-Key Grace Period
> Location: `pkg/crypto/`, `services/oauth/internal/service/`, `services/auth/internal/server/`

## What It Does

GGID supports automated rotation of all cryptographic keys and secrets with a dual-key grace period that ensures zero-downtime transitions. During rotation, both old and new keys are valid simultaneously, allowing gradual migration without breaking existing sessions or tokens.

## Key Categories

### 1. JWT Signing Keys

OAuth service uses RSA or ECDSA keys to sign JWTs.

**Rotation flow:**
1. Generate new key pair.
2. Add new key to JWKS endpoint (both keys active).
3. New tokens signed with new key.
4. Old tokens still validated with old key.
5. After grace period (default: 24h), remove old key.

**Config:**
```json
{
  "rotation_days": 90,
  "grace_period_hours": 24,
  "algorithm": "RS256",
  "key_size": 2048
}
```

### 2. SCEP Device Certificates

Internal CA signing certificate for device enrollment.

**Rotation flow:**
1. Generate new CA key pair.
2. Issue new CA certificate with overlap period.
3. Both old and new CA certificates in trust store.
4. New device enrollments use new CA.
5. Old devices renew against new CA.
6. After all devices renewed, revoke old CA.

**Timeline:** Typically 30-day overlap to allow device renewal.

### 3. Webhook HMAC Secrets

Secrets used to sign webhook deliveries.

**Rotation flow:**
1. Generate new secret.
2. Send webhooks with new signature + include `X-GGID-Previous-Signature` header.
3. Receivers verify against new or old secret.
4. After 7 days, stop sending old signature.

### 4. Internal Auth Secret

Secret used for inter-service authentication.

**Rotation flow:**
1. Set `INTERNAL_AUTH_PREV_SECRET` to current secret value.
2. Set `INTERNAL_AUTH_SECRET` to new value.
3. Services accept both secrets during transition.
4. After all services restarted with new secret, remove `PREV_SECRET`.

## Dual-Key Grace Period

The grace period is the critical safety mechanism:

```
Time ────────────────────────────────────────────→
         │                    │                    │
    New key active      Old key expires     Grace period ends
         │                    │                    │
    ─────┼────────────────────┼────────────────────┼─────
         │   Both keys valid   │   New key only    │
         │   (grace period)    │                   │
```

During grace period:
- New operations use the new key.
- Existing tokens/sessions validated with either key.
- No user-facing disruption.

## Rotation Schedule

| Key Type | Default Rotation | Grace Period |
|----------|-----------------|-------------|
| JWT signing | 90 days | 24 hours |
| SCEP CA | 365 days | 30 days |
| Webhook HMAC | 90 days | 7 days |
| Internal auth | 90 days | 48 hours |
| KEK (encryption) | 365 days | Manual |
| TLS certificates | Per cert expiry | Per overlap |

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/oauth/jwks` | GET | JWKS (shows current + grace keys) |
| `/api/v1/admin/secrets/:id/rotate` | POST | Trigger manual rotation |

### curl Examples

```bash
TOKEN="your-jwt-token"

# Check JWKS (verify dual keys during rotation)
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/oauth/jwks" | python3 -m json.tool

# Trigger manual key rotation
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/admin/secrets/jwt-signing-key/rotate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Token validation fails after rotation | Grace period too short or JWKS not updated | Extend grace period; verify JWKS endpoint returns both keys |
| Device enrollment fails after CA rotation | Devices not updated with new CA | Push MDM profile update with new CA cert |
| Webhook verification fails | Receiver not updated with new secret | Include previous-signature header; verify receiver handles dual secrets |
| Service auth fails | INTERNAL_AUTH_PREV_SECRET not set | Set PREV_SECRET env var before changing SECRET |

## Best Practices

- **Never rotate without grace period**: Always maintain dual-key overlap.
- **Test rotation in staging**: Verify the full rotation cycle before production.
- **Monitor JWKS key count**: During rotation, JWKS should have 2+ keys. After grace, back to 1.
- **Log rotation events**: Record when rotation starts, grace period ends, old key removed.
- **Coordinate with receivers**: Notify webhook receivers before HMAC rotation.
- **Automate where possible**: JWT and webhook secrets should auto-rotate on schedule.
