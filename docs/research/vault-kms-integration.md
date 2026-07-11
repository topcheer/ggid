# Vault/KMS Integration Architecture

## Status: PROPOSED (P2)

## Problem

GGID currently stores JWT signing keys as PEM files on disk (via `OAUTH_SIGNING_KEY_PATH`). This has limitations:
- No key rotation without restart
- No audit trail for key access
- Keys stored alongside application code
- No HSM-grade protection for private keys

## Proposed Architecture

### Phase 1: HashiCorp Vault Transit Engine

Use Vault's Transit engine for cryptographic operations. Keys never leave Vault.

```
GGID Services → Vault Transit API → Sign/Verify JWT
                                      ↑
                                   HSM (optional)
```

**Benefits**:
- Keys never touch disk or memory in plaintext
- Automatic key rotation with versioned keys
- Audit log for all crypto operations
- Works with cloud KMS (AWS KMS, GCP KMS, Azure Key Vault) via Vault plugins

**Implementation**:
1. New `VaultKeyProvider` implementing `domain.KeyProvider` interface
2. `PublicKey()` → reads from Vault (cached with 5-min TTL)
3. `PrivateKey()` → returns a reference, actual signing delegated to Vault
4. `KeyID()` → returns Vault key name + version
5. JWT signing: send header+payload to Vault Transit `/sign` endpoint

**Config**:
```env
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=s.xxx           # or VAULT_ROLE_ID + VAULT_SECRET_ID for AppRole
VAULT_TRANSIT_KEY=ggid-jwt-signing
VAULT_KEY_TYPE=rsa-2048
```

### Phase 2: Cloud KMS Direct Integration

For teams not using Vault, integrate directly with cloud KMS:

| Cloud | Service | SDK |
|-------|---------|-----|
| AWS | KMS | aws-sdk-go-v2/service/kms |
| GCP | Cloud KMS | google.golang.org/api/cloudkms/v1 |
| Azure | Key Vault | github.com/Azure/azure-sdk-for-go |

**Interface**:
```go
type KMSKeyProvider struct {
    client    KMSClient
    keyID     string
    publicKey *rsa.PublicKey // cached
}

func (k *KMSKeyProvider) Sign(data []byte) ([]byte, error) {
    return k.client.Sign(k.keyID, data)
}
```

### Phase 3: Database Encryption at Rest

- TDE (Transparent Data Encryption) for PostgreSQL
- Application-level field encryption for PII columns
- Envelope encryption: DEK encrypted by KEK from Vault/KMS

## Migration Path

1. **No-op mode** (current): PEM files on disk
2. **Hybrid mode**: Read keys from Vault but sign locally
3. **Full Vault mode**: All signing via Transit engine
4. **HSM mode**: Vault PKCS#11 backend or cloud HSM

## Security Considerations

- Vault token rotation via AppRole + response wrapping
- Break-glass procedure: emergency PEM fallback
- Key versioning for zero-downtime rotation
- JWKS endpoint must expose all active key versions

## References

- [HashiCorp Vault Transit Engine](https://developer.hashicorp.com/vault/docs/secrets/transit)
- [AWS KMS Asymmetric Signing](https://docs.aws.amazon.com/kms/latest/developerguide/asymmetric.html)
- [GCP Cloud KMS](https://cloud.google.com/kms/docs)
