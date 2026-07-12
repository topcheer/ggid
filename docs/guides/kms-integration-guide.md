# KMS Integration Guide

AWS KMS, GCP KMS, Azure Key Vault, HashiCorp Vault Transit — envelope encryption flow, CMK rotation, IAM policy, key aliasing, performance, and fallback.

## Provider Comparison

| Provider | Latency | Key Types | FIPS | Cost |
|----------|---------|-----------|------|------|
| AWS KMS | 5-15ms | RSA, ECC, AES | ✅ | $1/key/month |
| GCP KMS | 3-10ms | RSA, ECC | ✅ | $0.06/key/month |
| Azure Key Vault | 3-10ms | RSA, ECC | ✅ | $0.03/key/10k ops |
| HashiCorp Vault | <1ms (local) | All | ✅ (ent) | Self-hosted |

## Envelope Encryption Flow

```
1. App requests data encryption
2. GGID generates DEK locally (AES-256)
3. GGID sends DEK to KMS for encryption: KMS.Encrypt(CMK, DEK) → encrypted DEK
4. GGID encrypts data with DEK locally
5. Store: encrypted DEK + encrypted data
6. DEK is zeroed from memory after use
```

```go
func encryptWithKMS(plaintext []byte, cmkID string) (*EncryptedBlob, error) {
    // 1. Generate DEK
    dek := make([]byte, 32)
    rand.Read(dek)
    
    // 2. Encrypt DEK with KMS CMK
    encDEK, err := kms.Encrypt(&kms.EncryptInput{
        KeyId:     aws.String(cmkID),
        Plaintext: dek,
    })
    if err != nil { return nil, err }
    
    // 3. Encrypt data with DEK locally
    ciphertext := aesGCMEncrypt(plaintext, dek)
    
    // 4. Zero DEK
    zeroBytes(dek)
    
    return &EncryptedBlob{
        EncryptedDEK: encDEK.CiphertextBlob,
        Ciphertext:   ciphertext,
    }, nil
}
```

## CMK Rotation

| Provider | Auto-Rotation | Period |
|----------|--------------|--------|
| AWS KMS | ✅ Configurable | 365 days |
| GCP KMS | ✅ Automatic | 90 days |
| Azure Key Vault | ✅ Configurable | 90 days |

Rotation rotates the backing key — existing encrypted data remains decryptable.

## Key Aliasing Convention

```
alias/ggid-prod-mek          # Master encryption key
alias/ggid-prod-tek-{tenant} # Per-tenant encryption key
alias/ggid-jwt-signing       # JWT signing key
alias/ggid-saml-signing      # SAML signing key
```

## IAM Policy (AWS)

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["kms:Encrypt", "kms:Decrypt", "kms:GenerateDataKey"],
    "Resource": "arn:aws:kms:us-east-1:*:key/alias/ggid-prod-*"
  }, {
    "Effect": "Deny",
    "Action": "kms:DeleteKey",
    "Resource": "*"
  }]
}
```

## Performance: Local HSM vs Cloud KMS

| Operation | Local HSM | Cloud KMS |
|-----------|----------|-----------|
| Encrypt | <0.1ms | 5-15ms |
| Decrypt | <0.1ms | 5-15ms |
| Batch (1000) | <1ms | 5-15s |

**Mitigation**: Envelope encryption means KMS is only called for DEK wrap/unwrap, not per-data-item.

## Fallback Strategy

```go
type KMSFallback struct {
    primary  KMSProvider  // Cloud KMS
    fallback KMSProvider  // Vault Transit or software
}

func (k *KMSFallback) Encrypt(key []byte) ([]byte, error) {
    result, err := k.primary.Encrypt(key)
    if err != nil {
        log.Warn("KMS primary failed, using fallback", err)
        return k.fallback.Encrypt(key)
    }
    return result, nil
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| KMS latency | >50ms → degraded |
| KMS errors | >1% → check IAM/quota |
| CMK nearing rotation | <30 days → verify auto-rotation |
| Fallback usage | Any → primary KMS issue |

## See Also

- [Credential Vault Architecture](credential-vault-architecture.md)
- [Cryptography Key Rotation](cryptography-key-rotation.md)
- [Secrets Rotation Automation](secrets-rotation-automation.md)
- [Database Security](database-security.md)
