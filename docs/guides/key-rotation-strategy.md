# Key Rotation Strategy

Rotation frequency by key type, automated pipeline, zero-downtime dual-key period, key versioning, emergency rotation, HSM/KMS integration, and audit trail.

## Rotation Frequency

| Key Type | Frequency | Grace Period | Rationale |
|----------|-----------|-------------|-----------|
| JWT signing | 90 days | 15 days | Token forgery prevention |
| SAML signing | 365 days | 30 days | Metadata-driven, slower cycle |
| SAML encryption | 365 days | 30 days | Match signing |
| mTLS service cert | 90 days | 15 days | cert-manager auto |
| DB encryption (DEK) | Per write | — | Fresh key each encryption |
| Tenant encryption (TEK) | 90 days | 7 days | Re-wrap DEKs |
| Master encryption (MEK) | 365 days | 30 days | Re-wrap TEKs |
| OAuth client secret | 180 days | 7 days | Rolling replacement |
| API key | 90 days | 7 days | Rolling replacement |
| Cookie signing | 90 days | 7 days | JWT-aligned |
| WebAuthn RP key | 365 days | 30 days | Credential re-registration needed |

## Zero-Downtime Rotation (Dual-Key Period)

```
Day 0:   New key generated + published alongside old
         JWKS: [old-key, new-key]
         Accept tokens from either; sign with new only

Day 1+:  All new tokens signed with new key
         Old tokens still verified (15min TTL expires naturally)

Day 15:  Old key removed from JWKS
         Old key revoked from signing capability
```

### Implementation

```go
type KeyManager struct {
    current  *SigningKey   // Signs new tokens
    previous *SigningKey   // Verifies old tokens (grace)
}

func (km *KeyManager) Rotate() error {
    newKey, err := km.generateKey()  // HSM/KMS backed
    if err != nil { return err }

    km.previous = km.current
    km.current = newKey
    publishJWKS(km.current, km.previous)

    // Schedule old key removal
    time.AfterFunc(gracePeriod, func() {
        km.previous = nil
        publishJWKS(km.current)
        audit.Log("key.removed", newKey.ID)
    })

    audit.Log("key.rotated", map[string]interface{}{
        "old_kid": km.previous.ID,
        "new_kid": km.current.ID,
    })
    return nil
}
```

## Key Versioning

```
kid: "jwt-sig-2025-01-001"  (type-year-sequence)
kid: "jwt-sig-2025-04-002"  (next rotation)
kid: "saml-sig-2025-001"
kid: "mtls-auth-2025-03-001"
```

Versioning enables audit trail and rollback identification.

## Automated Pipeline

```yaml
rotation_pipeline:
  schedule: "0 3 1 * *"  # Monthly check
  steps:
    - query: keys_expiring_within_15_days
    - generate: new_key via HSM/KMS
    - publish: dual JWKS (old + new)
    - switch: signing to new key only
    - wait: grace_period (15 days)
    - remove: old key from JWKS
    - audit: log rotation event
    - alert: security team of completion
```

## Emergency Rotation (Compromise)

```bash
POST /api/v1/admin/keys/emergency-rotate
{
  "key_id": "jwt-sig-2025-01-001",
  "reason": "Suspected key compromise",
  "invalidate_tokens": true,
  "notify": ["security@corp.com", "ciso@corp.com"]
}
```

### Emergency Steps

1. Revoke compromised key immediately (remove from JWKS)
2. All tokens signed by that key → invalid (force re-auth)
3. Generate new key immediately
4. Publish new JWKS
5. Notify all resource servers to re-fetch JWKS
6. Security incident + audit log

## HSM/KMS Integration

```go
// Key generated inside HSM — private key never leaves
func (km *KeyManager) generateKey() (*SigningKey, error) {
    handle, err := hsm.GenerateECDSA("P-256")
    pubKey, err := hsm.ExportPublic(handle)
    return &SigningKey{
        ID:        "jwt-sig-" + time.Now().Format("2006-01") + "-001",
        PublicKey: pubKey,
        Handle:    handle,
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
    }, nil
}
```

| Provider | Latency | FIPS |
|----------|---------|------|
| AWS KMS | 5-15ms | ✅ |
| GCP KMS | 3-10ms | ✅ |
| Azure Key Vault | 3-10ms | ✅ |
| Local HSM | <0.1ms | ✅ |

## Audit Trail

```json
{
  "event": "key.rotated",
  "key_type": "jwt_signing",
  "old_kid": "jwt-sig-2025-01-001",
  "new_kid": "jwt-sig-2025-04-002",
  "trigger": "scheduled",
  "grace_period_days": 15,
  "timestamp": "2025-04-01T03:00:00Z"
}
```

Retention: 7 years (compliance).

## Monitoring

| Metric | Alert |
|--------|-------|
| Key nearing expiry | <15 days → rotate now |
| Rotation automation failure | Any → manual intervention |
| Emergency rotation | Any → security incident |
| JWKS fetch failures | Any → clients can't verify |

## See Also

- [Cryptography Key Rotation](cryptography-key-rotation.md)
- [Credential Vault Architecture](credential-vault-architecture.md)
- [KMS Integration Guide](kms-integration-guide.md)
- [Secrets Rotation Automation](secrets-rotation-automation.md)
- [Certificate Lifecycle Automation](certificate-lifecycle-automation.md)
