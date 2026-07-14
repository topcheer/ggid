# KMS Integration

KMS providers, key hierarchy, envelope encryption, key rotation via KMS, signing key management, HSM integration, FIPS 140-2 compliance, and GGID KMS abstraction layer.

## Providers

| Provider | Latency | FIPS 140-2 | Key Types | Cost |
|----------|---------|-----------|-----------|------|
| AWS KMS | 5-15ms | ✅ L2 | RSA, ECC, AES | $1/key/mo |
| GCP KMS | 3-10ms | ✅ L3 | RSA, ECC | $0.06/key/mo |
| Azure Key Vault | 3-10ms | ✅ L3 | RSA, ECC | $0.03/10k ops |
| HashiCorp Vault | <1ms | ✅ (ent) | All | Self-hosted |

## Key Hierarchy

```
Master Encryption Key (MEK) — KMS/HSM backed, never leaves
  └── Tenant Encryption Key (TEK) — Per-tenant, wrapped by MEK
       └── Data Encryption Key (DEK) — Per-record, wrapped by TEK
            └── Actual data — AES-256-GCM encrypted with DEK
```

## Envelope Encryption

```go
func encrypt(plaintext []byte, tekID string) (*Blob, error) {
    // 1. Generate fresh DEK
    dek := make([]byte, 32)
    rand.Read(dek)

    // 2. Wrap DEK with TEK (via KMS)
    encDEK, err := kms.Encrypt(tekID, dek)
    if err != nil { return nil, err }

    // 3. Encrypt data locally with DEK
    ciphertext := aesGCM(plaintext, dek)

    // 4. Zero DEK from memory
    zero(dek)

    return &Blob{EncDEK: encDEK, Ciphertext: ciphertext}, nil
}
```

## GGID KMS Abstraction Layer

```go
type KMSProvider interface {
    Encrypt(keyID string, plaintext []byte) ([]byte, error)
    Decrypt(keyID string, ciphertext []byte) ([]byte, error)
    GenerateKey(keyType string) (string, error)
    RotateKey(keyID string) (string, error)
    GetKeyMetadata(keyID string) (*KeyMetadata, error)
}

// Multi-provider with fallback
type KMSManager struct {
    primary  KMSProvider
    fallback KMSProvider
}

func (m *KMSManager) Encrypt(keyID string, data []byte) ([]byte, error) {
    result, err := m.primary.Encrypt(keyID, data)
    if err != nil {
        log.Warn("KMS primary failed, using fallback", err)
        return m.fallback.Encrypt(keyID, data)
    }
    return result, nil
}
```

## Key Rotation via KMS

```yaml
rotation:
  schedule: "0 3 1 * *"
  keys:
    - key_id: "alias/ggid-mek"
      rotate_every: 365d
      provider: aws-kms
    - key_id: "alias/ggid-tek-{tenant}"
      rotate_every: 90d
      provider: aws-kms
    - key_id: "alias/ggid-jwt-signing"
      rotate_every: 90d
      provider: vault-transit
```

## Signing Key Management

```go
// JWT signing keys via Vault Transit
func signJWT(payload []byte) (string, error) {
    sig, err := vault.Transit.Sign("jwt-signing-key", payload)
    if err != nil { return "", err }
    return base64url(payload) + "." + base64url(sig), nil
}
```

Private key never leaves Vault — only signature is returned.

## HSM Integration

| HSM | Interface | Use Case |
|-----|-----------|---------|
| AWS CloudHSM | PKCS#11 | High-throughput signing |
| Thales Luna | PKCS#11 | On-prem compliance |
| YubiHSM 2 | PKCS#11 | Small-scale, dev |

## FIPS 140-2 Compliance

| Level | Requirement | GGID |
|-------|------------|------|
| Level 1 | Software crypto | Not used |
| Level 2 | Tamper-evident | AWS KMS |
| Level 3 | Tamper-resistant + ID-based auth | AWS CloudHSM, GCP KMS |

## Monitoring

| Metric | Alert |
|--------|-------|
| KMS latency | >50ms → degraded |
| KMS errors | >1% → check IAM/quota |
| Key nearing rotation | <30 days → verify auto |
| Fallback usage | Any → primary issue |

## See Also

- [KMS Integration Guide](kms-integration-guide.md)
- [Credential Vault Architecture](credential-vault-architecture.md)
- [Cryptography Key Rotation](cryptography-key-rotation.md)
- [Key Rotation Strategy](key-rotation-strategy.md)
