# Data Key Management — Technical Guide

> Feature: Envelope Encryption with Per-Tenant Data Keys
> Location: `pkg/crypto/data_key_provider.go`

## What It Does

GGID uses envelope encryption to protect sensitive data at rest. Each tenant gets unique Data Encryption Keys (DEKs) that are encrypted by a Key Encryption Key (KEK). This provides cryptographic isolation between tenants and enables key rotation without re-encrypting all data.

## Key Hierarchy

```
KEK (Key Encryption Key)
├── AES-256-GCM (international) or SM4-GCM (China compliance)
├── Stored in KMS / Vault / local environment
├── 32-byte key material (hashed from source)
│
├── Encrypts → DEK₁ (Tenant A, per-record)
├── Encrypts → DEK₂ (Tenant B, per-record)
└── Encrypts → DEK₃ (Tenant C, per-record)

DEK (Data Encryption Key)
├── AES-256-GCM or SM4-GCM
├── Random 32-byte key, generated per encryption operation
└── Encrypts → field-level plaintext data
```

## Envelope Encryption Flow

1. **Generate DEK**: System creates a cryptographically random 32-byte DEK.
2. **Encrypt DEK**: DEK is encrypted using the KEK → produces `encryptedDEK`.
3. **Encrypt Data**: Plaintext is encrypted using the DEK with AES-256-GCM → produces `ciphertext + nonce + authTag`.
4. **Package**: Base64-encode `ciphertext + nonce + encryptedDEK` into a single string.
5. **Store**: The package string is stored in the database column.
6. **Decrypt**: Decode base64 → decrypt DEK with KEK → decrypt data with DEK.

## DataKeyProvider Interface

```go
type DataKeyProvider interface {
    // Generate a new DEK for the tenant.
    // Returns (plaintextDEK, encryptedDEK, error).
    GenerateDataKey(ctx context.Context, tenantID string) (plaintextDEK, encryptedDEK []byte, err error)

    // Decrypt a stored encrypted DEK back to plaintext.
    DecryptDataKey(ctx context.Context, encryptedDEK []byte) (plaintextDEK []byte, err error)

    // Encrypt a field value: generate DEK → AES-256-GCM → base64.
    EncryptField(ctx context.Context, tenantID string, plaintext []byte) (ciphertext string, err error)

    // Decrypt a value produced by EncryptField.
    DecryptField(ctx context.Context, ciphertext string) (plaintext []byte, err error)
}
```

## Cipher Algorithms

| Algorithm | Use Case | Standard | Key Size |
|-----------|----------|----------|----------|
| **AES-256-GCM** | International deployments | FIPS 197 / NIST SP 800-38D | 256-bit |
| **SM4-GCM** | China compliance (GB/T 32907) | Chinese national standard | 128-bit |

Both use GCM mode for authenticated encryption (confidentiality + integrity).

## Configuration

### Creating a Provider

```go
import "github.com/ggid/ggid/pkg/crypto"

// International (AES-256-GCM) — default
provider := crypto.NewEnvelopeEncryptionProvider(kekMaterial)

// China compliance (SM4-GCM)
provider := crypto.NewEnvelopeEncryptionProvider(kekMaterial).WithSM4()
```

### KEK Material Sources

| Source | Use Case |
|--------|----------|
| **Local** | Development — read from `GGID_ENCRYPTION_KEY` env var |
| **AWS KMS** | Production — managed KEK with automatic rotation |
| **GCP KMS** | Google Cloud deployments |
| **Azure Key Vault** | Microsoft Azure deployments |
| **HashiCorp Vault** | Self-hosted or Vault Cloud |
| **PKCS#11 HSM** | Hardware Security Module for highest assurance |

## Usage Examples

### Encrypt and Decrypt a Field

```go
ctx := context.Background()
tenantID := "00000000-0000-0000-0000-000000000001"

// Encrypt
plaintext := []byte("user@example.com")
ciphertext, err := provider.EncryptField(ctx, tenantID, plaintext)
// ciphertext = "base64(gcm_ciphertext + nonce + encryptedDEK)"

// Decrypt (later, possibly in a different service)
decrypted, err := provider.DecryptField(ctx, ciphertext)
// decrypted = []byte("user@example.com")
```

### Generate and Use DEK Directly

```go
// Generate a DEK for bulk operations
plaintextDEK, encryptedDEK, err := provider.GenerateDataKey(ctx, tenantID)
// Use plaintextDEK for multiple field encryptions in a batch
// Store encryptedDEK alongside the encrypted records

// Later, recover the DEK
recoveredDEK, err := provider.DecryptDataKey(ctx, encryptedDEK)
```

## Tenant Key Lifecycle

### Key Generation
- Each `EncryptField` call generates a unique DEK.
- DEKs are never reused across records — each encrypted value has its own key.

### Key Rotation
- **DEK rotation**: Automatic — new records get new DEKs. Old records remain decryptable with their stored encryptedDEKs.
- **KEK rotation**: Manual process:
  1. Generate new KEK.
  2. For each tenant, decrypt all stored encryptedDEKs with old KEK.
  3. Re-encrypt DEKs with new KEK.
  4. Update provider with new KEK.

### Key Revocation (Tenant Offboarding)
- Delete tenant's KEK access → all encrypted data becomes permanently unreadable.
- Optionally: delete all stored encryptedDEKs for the tenant.

## Security Properties

- **Tenant isolation**: Each tenant's DEKs are unique — compromise of one tenant's data doesn't affect others.
- **Per-record uniqueness**: Each encrypted field has a unique DEK — blast radius of key compromise is minimal.
- **Authenticated encryption**: GCM mode detects tampering — modified ciphertext fails decryption.
- **Random nonces**: 12-byte GCM nonce is cryptographically random per encryption.
- **No key logging**: DEKs and KEK material are never serialized or logged.

## Encrypted Output Format

```
base64(
  [1 byte: version]      // 0x01 = AES-256-GCM, 0x02 = SM4-GCM
  [12 bytes: nonce]      // GCM nonce
  [N bytes: ciphertext]  // encrypted plaintext
  [16 bytes: authTag]    // GCM authentication tag (embedded in ciphertext by Go)
  [M bytes: encryptedDEK] // DEK encrypted with KEK
)
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| DecryptField fails | Wrong KEK or corrupted data | Verify KEK material; check for data corruption |
| "cipher: message authentication failed" | Tampered ciphertext or wrong key | GCM tag verification failed — data may be corrupted |
| SM4 not available | Go build without SM4 support | Use AES-256-GCM as fallback; ensure crypto backend supports SM4 |
| KEK rotation breaks decryption | Old DEKs not re-encrypted | Maintain key version history; re-encrypt all DEKs when rotating KEK |
| Performance issues | Too many DEK generations | Use GenerateDataKey for batch operations to amortize KEK operations |

## Best Practices

- **Protect the KEK**: The KEK is the root of trust — store in HSM or cloud KMS.
- **Rotate KEK annually**: Schedule annual KEK rotation with DEK re-encryption.
- **Never log keys**: Ensure DEKs and KEK material never appear in logs.
- **Use base64 for storage**: Encrypted output is base64-encoded for DB-safe storage.
- **Tenant-scoped keys**: Always pass the correct tenant ID for key isolation.
- **Audit key operations**: Log key generation and usage events (without key material).
- **Test decryption after rotation**: Always verify that historical data remains decryptable after KEK rotation.
