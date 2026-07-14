# Credential Vault Architecture

At-rest encryption (AES-256-GCM), key hierarchy (master→tenant→data), HSM integration, key rotation automation, access audit, memory zeroing, secret scoping, and backup/restore.

## Key Hierarchy

```
HSM Root Key (never leaves HSM)
  └── Master Encryption Key (MEK, per environment)
        └── Tenant Encryption Key (TEK, per tenant)
              └── Data Encryption Key (DEK, per record/table)
                    └── Encrypted Data (AES-256-GCM)
```

### Envelope Encryption

```go
func encryptData(plaintext []byte, tenantID string) ([]byte, error) {
    // 1. Get or generate DEK (encrypted by TEK)
    dek := generateAESKey() // 256-bit random
    encDEK := encryptWithKey(dek, getTEK(tenantID))

    // 2. Encrypt data with DEK
    ciphertext := aesGCMEncrypt(plaintext, dek)

    // 3. Store: encDEK + ciphertext
    return append(encDEK, ciphertext...), nil
}

func decryptData(blob []byte, tenantID string) ([]byte, error) {
    encDEK := blob[:256]
    ciphertext := blob[256:]

    // 1. Decrypt DEK with TEK
    dek := decryptWithKey(encDEK, getTEK(tenantID))

    // 2. Decrypt data with DEK
    return aesGCMDecrypt(ciphertext, dek)
}
```

## HSM Integration

```go
type HSMProvider struct {
    client *pkcs11.Ctx
    slot   uint
}

func (h *HSMProvider) Sign(data []byte) ([]byte, error) {
    // Signing key never leaves HSM
    session := h.openSession()
    defer h.closeSession(session)

    h.login(session)
    key := h.findKey(session, "ggid-mek")
    return h.sign(session, key, data)
}
```

| HSM Type | Use Case | Latency |
|----------|---------|---------|
| AWS CloudHSM | Cloud deployment | 2-5ms |
| Azure Key Vault | Azure deployment | 1-3ms |
| Thales Luna | On-prem enterprise | <1ms |
| Software (dev only) | Development | <0.1ms |

## Key Rotation Automation

| Key Type | Rotation Period | Grace Period | Method |
|----------|----------------|-------------|--------|
| MEK | 1 year | 30 days | Re-encrypt all TEKs |
| TEK | 90 days | 7 days | Re-encrypt all DEKs |
| DEK | Per-record (generated fresh) | — | Generated on write |
| JWT signing | 90 days | 15 days | Dual-publish in JWKS |
| DB credentials | 90 days | 0 (immediate) | Rotate + rolling restart |

### Rotation Process

```go
func rotateTEK(tenantID string) error {
    oldTEK := getTEK(tenantID)
    newTEK := generateAESKey()

    // 1. Re-encrypt all DEKs with new TEK
    deks := getAllDEKs(tenantID)
    for _, dek := range deks {
        plaintextDEK := decryptWithKey(dek.Encrypted, oldTEK)
        dek.Encrypted = encryptWithKey(plaintextDEK, newTEK)
        store.Update(dek)
    }

    // 2. Store new TEK (keep old for grace period)
    store.SetTEK(tenantID, newTEK, oldTEK, time.Now().Add(7*24*time.Hour))

    // 3. After grace period, remove old TEK
    time.AfterFunc(7*24*time.Hour, func() {
        store.RemoveOldTEK(tenantID)
    })

    audit.Log("key.rotated", map[string]interface{}{
        "type": "TEK", "tenant": tenantID,
    })
    return nil
}
```

## Memory Zeroing

```go
func encryptAndZero(plaintext []byte, key []byte) ([]byte, error) {
    defer func() {
        // Overwrite plaintext in memory
        for i := range plaintext {
            plaintext[i] = 0
        }
    }()
    return aesGCMEncrypt(plaintext, key)
}

// Use crypto/subtle for constant-time comparison
// Never log or expose keys
```

## Secret Scoping

| Scope | Who Can Access | Example |
|-------|---------------|---------|
| System | Platform services only | JWT signing key |
| Tenant | Services + tenant admin | Tenant encryption key |
| User | The user only | User's MFA secret |
| Application | OAuth client only | Client secret |

## Access Audit

```json
{
  "event": "vault.access",
  "caller": "auth-svc",
  "secret_type": "TEK",
  "tenant_id": "uuid",
  "action": "encrypt",
  "timestamp": "2025-01-15T10:00:00Z",
  "request_id": "req-uuid"
}
```

Every key access logged. Alert on unusual access patterns.

## Backup/Restore

```bash
# Backup encrypted key store (MEK wrapped by HSM)
vault backup --output /backups/vault-$(date +%Y%m%d).enc

# Restore (requires HSM to unwrap MEK)
vault restore --input /backups/vault-20250115.enc
# → MEK unwrapped by HSM → TEKs available → DEKs available → Data readable
```

**Without HSM access, backups are useless** — this is by design.

## Monitoring

| Metric | Alert |
|--------|-------|
| HSM latency | >10ms → degraded |
| Key rotation failures | Any → investigate |
| Unusual access pattern | Off-hours or bulk → alert |
| Grace period expiring | <1 day → ensure rotation complete |
| Backup failures | Any → data loss risk |

## See Also

- [Secrets Rotation Automation](secrets-rotation-automation.md)
- [Secret Sprawl Prevention](secret-sprawl-prevention.md)
- [Database Security](database-security.md)
- [Post-Quantum Crypto Migration](post-quantum-crypto-migration.md)
