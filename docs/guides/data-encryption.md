# Data Encryption — Technical Guide

> Feature: Envelope Encryption with Per-Tenant Keys
> Location: `pkg/crypto/data_key_provider.go`

## What It Does

GGID uses envelope encryption to protect sensitive data at rest. Each tenant gets unique Data Encryption Keys (DEKs) that are encrypted by a master Key Encryption Key (KEK). This provides cryptographic isolation between tenants and enables key rotation without re-encrypting all data.

## Architecture

```
┌───────────────────────────────────────────┐
│              Key Hierarchy                  │
│                                             │
│  KEK (Key Encryption Key)                   │
│  ├── AES-256 or SM4                         │
│  ├── Stored in KMS / Vault / local          │
│  │                                          │
│  ├── Encrypts → DEK₁ (Tenant A)            │
│  ├── Encrypts → DEK₂ (Tenant B)            │
│  └── Encrypts → DEK₃ (Tenant C)            │
│                                             │
│  DEK (Data Encryption Key)                  │
│  ├── AES-256-GCM or SM4-GCM                │
│  ├── Per-tenant, per-record                │
│  └── Encrypts → field-level data           │
└───────────────────────────────────────────┘
```

### Envelope Encryption Flow

1. **Generate DEK**: System creates a random 32-byte DEK for the tenant.
2. **Encrypt DEK**: DEK is encrypted using the KEK (AES-256 or SM4).
3. **Encrypt Data**: Plaintext data is encrypted using the DEK (AES-256-GCM or SM4-GCM).
4. **Store**: Encrypted data + encrypted DEK are stored together (base64-encoded).
5. **Decrypt**: Encrypted DEK is decrypted with KEK, then data is decrypted with DEK.

## Cipher Algorithms

| Algorithm | Use Case | Standard |
|-----------|----------|----------|
| **AES-256-GCM** | International deployments | FIPS 197 / NIST SP 800-38D |
| **SM4-GCM** | China compliance (GB/T 32907) | Chinese national standard |

Both algorithms use GCM mode for authenticated encryption (confidentiality + integrity).

## DataKeyProvider Interface

```go
type DataKeyProvider interface {
    // Generate a new DEK for the tenant
    GenerateDataKey(ctx context.Context, tenantID string) (plaintextDEK, encryptedDEK []byte, err error)

    // Decrypt a stored encrypted DEK
    DecryptDataKey(ctx context.Context, encryptedDEK []byte) (plaintextDEK []byte, err error)

    // Encrypt a field value: generate DEK → AES-256-GCM → base64
    EncryptField(ctx context.Context, tenantID string, plaintext []byte) (ciphertext string, err error)

    // Decrypt a value produced by EncryptField
    DecryptField(ctx context.Context, ciphertext string) (plaintext []byte, err error)
}
```

## Configuration

### Creating an Envelope Encryption Provider

```go
import "github.com/ggid/ggid/pkg/crypto"

// International deployment (AES-256-GCM)
provider := crypto.NewEnvelopeEncryptionProvider(kekMaterial)

// China compliance (SM4-GCM)
provider := crypto.NewEnvelopeEncryptionProvider(kekMaterial).WithSM4()
```

### Key Material Sources

The KEK material can come from:
- **Local**: `loadEncryptionKey()` — reads from environment variable or file.
- **AWS KMS**: Managed KEK with automatic rotation.
- **GCP KMS**: Google Cloud Key Management Service.
- **Azure Key Vault**: Microsoft managed keys.
- **HashiCorp Vault**: Self-hosted or Vault Cloud.
- **PKCS#11**: Hardware Security Module (HSM).

## Usage Example

```go
ctx := context.Background()
tenantID := "00000000-0000-0000-0000-000000000001"

// Encrypt a sensitive field
plaintext := []byte("user@example.com")
ciphertext, err := provider.EncryptField(ctx, tenantID, plaintext)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Encrypted:", ciphertext)
// Output: base64(GCM_ciphertext + nonce + encryptedDEK)

// Decrypt later
decrypted, err := provider.DecryptField(ctx, ciphertext)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Decrypted:", string(decrypted))
// Output: user@example.com
```

## Key Rotation

DEK rotation: Generate a new DEK for new records. Old records remain encrypted with old DEKs (still decryptable).

KEK rotation: Re-encrypt all stored DEKs with the new KEK. This is less frequent but more impactful.

## Security Properties

- **Tenant isolation**: Each tenant has unique DEKs — compromise of one tenant's data doesn't affect others.
- **Forward secrecy**: If a DEK is compromised, only records encrypted with that specific DEK are affected.
- **Authenticated encryption**: GCM mode provides both confidentiality and integrity — tampered ciphertext is detected.
- **Random nonces**: Each encryption uses a cryptographically random 12-byte nonce (GCM standard).

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| DecryptField fails | Wrong KEK or corrupted ciphertext | Verify KEK material matches encryption key; check for data corruption |
| "cipher: message authentication failed" | Tampered ciphertext or wrong key | GCM authentication tag verification failed — data may be corrupted or tampered |
| SM4 not available | Build without SM4 support | Ensure Go crypto backend supports SM4; use AES-256-GCM as fallback |
| Key rotation breaks decryption | Old DEKs not re-encrypted | Maintain key version history; re-encrypt DEKs when rotating KEK |

## Best Practices

- **Protect the KEK**: The KEK is the root of trust — store in a hardware HSM or cloud KMS.
- **Rotate DEKs regularly**: Generate new DEKs for each encryption operation (envelope pattern).
- **Never log plaintext keys**: Ensure DEKs and KEK material are never logged.
- **Use base64 for storage**: Encrypted output is base64-encoded for safe storage in JSON/DB.
- **Tenant-scoped keys**: Always pass the correct tenant ID to ensure key isolation.
- **Audit key usage**: Log key generation and usage events (without exposing key material).
