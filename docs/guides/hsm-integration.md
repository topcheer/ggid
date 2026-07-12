# HSM Integration Guide

This guide covers integrating Hardware Security Modules (HSM) with GGID — PKCS#11, AWS KMS, Azure Key Vault, HashiCorp Vault, key wrapping, signing operations, and failover.

## Overview

HSMs provide hardware-based key storage and cryptographic operations. Keys never leave the HSM, providing the highest level of protection for signing keys and secrets.

## Integration Options

### GGID KeyProvider Interface

```go
type KeyProvider interface {
    PublicKey() crypto.PublicKey
    PrivateKey() crypto.PrivateKey  // nil if HSM signs
    KeyID() string
    Sign(data []byte) ([]byte, error)  // HSM-backed or local
}
```

## AWS KMS

### Configuration

```yaml
kms:
  provider: aws
  key_id: arn:aws:kms:us-east-1:123:key/abc-123
  region: us-east-1
```

### Implementation

```go
import "github.com/aws/aws-sdk-go-v2/service/kms"

type KMSKeyProvider struct {
    keyID     string
    client    *kms.Client
    publicKey crypto.PublicKey
}

func (k *KMSKeyProvider) Sign(data []byte) ([]byte, error) {
    resp, err := k.client.Sign(ctx, &kms.SignInput{
        KeyId:            &k.keyID,
        Message:          data,
        SigningAlgorithm: types.SigningAlgorithmSpecRsassaPssSha256,
    })
    return resp.Signature, err
}
```

## Azure Key Vault

```yaml
key_vault:
  provider: azure
  vault_url: https://ggid-kv.vault.azure.net
  key_name: jwt-signing-key
  tenant_id: xxx
  client_id: xxx
```

## HashiCorp Vault

### Transit Engine

```bash
# Create key in Vault
vault write transit/keys/ggid-jwt type=rsa-2048

# Sign
vault write transit/sign/ggid-jwt input=$(base64 <<< "$data")
```

### Configuration

```yaml
vault:
  address: https://vault.internal:8200
  auth_method: kubernetes
  transit_key: ggid-jwt
  role: ggid-auth
```

### Go Implementation

```go
type VaultKeyProvider struct {
    client   *vault.Client
    keyName  string
}

func (v *VaultKeyProvider) Sign(data []byte) ([]byte, error) {
    resp, err := v.client.Logical().Write("transit/sign/"+v.keyName, map[string]interface{}{
        "input": base64.StdEncoding.EncodeToString(data),
    })
    return base64.StdEncoding.DecodeString(resp.Data["signature"].(string))
}
```

## PKCS#11 (On-Prem HSM)

### Configuration

```yaml
pkcs11:
  library: /usr/lib/softhsm/libsofthsm2.so
  slot: 0
  pin: "${HSM_PIN}"
  key_label: ggid-jwt-signing
```

### Go Implementation

```go
import "github.com/ThalesIgnite/crypto11"

ctx, _ := crypto11.Configure(&crypto11.Config{
    Path:       "/usr/lib/softhsm/libsofthsm2.so",
    SlotNumber: &slot,
    Pin:        pin,
})

key, _ := ctx.FindKeyPair(nil, []byte("ggid-jwt-signing"))
signature, _ := key.Sign(rand.Reader, data, crypto.SHA256)
```

## Key Wrapping

For backup and transfer, keys are wrapped with a Key Encryption Key (KEK):

```
HSM generates DEK (Data Encryption Key)
  ↓
HSM wraps DEK with KEK (stored in separate HSM)
  ↓
Wrapped DEK exported for backup
  ↓
Only HSM can unwrap and use DEK
```

## Failover

### Multi-Region KMS

```
Region A: Primary KMS key → signs JWTs
Region B: Replica KMS key → standby (same key material)

If Region A KMS unavailable:
  → Failover to Region B KMS
  → Same key material → same signatures
  → No JWKS change needed
```

### Vault HA

```yaml
vault:
  ha: true
  replicas: 3
  storage: raft
  api_addr: https://vault.internal:8200
```

## Performance Comparison

| Method | Sign Latency | Throughput | Cost |
|--------|-------------|-----------|------|
| Local (software) | 0.01ms | 100K/s | Free |
| AWS KMS | 5-15ms | 5K/s | $0.03/10K |
| Azure Key Vault | 5-15ms | 5K/s | $0.03/10K |
| Vault Transit | 2-5ms | 20K/s | Infra cost |
| PKCS#11 HSM | 1-3ms | 50K/s | $10K-$100K |

**Recommendation**: Use software signing for development, Vault Transit for most production, PKCS#11 for regulated industries.

## See Also

- [Key Management Lifecycle](../research/key-management-lifecycle.md)
- [Password Pepper Deploy](password-pepper-deploy.md)
- [gRPC TLS Setup](grpc-tls-setup.md)
