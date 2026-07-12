# Database Encryption Guide

This guide covers encryption layers, PostgreSQL pgcrypto, per-column encryption for PII, key management hierarchy (DEK/KEK/MEK), search on encrypted data, index implications, performance benchmarks, and GGID's database encryption implementation.

## Encryption Layers

### Layer Overview

```
┌────────────────────────────────────────────┐
│  Application-Level Encryption              │  ← GGID encrypts before storing
├────────────────────────────────────────────┤
│  Column-Level Encryption (pgcrypto)         │  ← PostgreSQL encrypts columns
├────────────────────────────────────────────┤
│  Transparent Data Encryption (TDE)          │  ← Database encrypts files
├────────────────────────────────────────────┤
│  Disk Encryption (LUKS/dm-crypt)            │  ← OS encrypts disk
├────────────────────────────────────────────┤
│  Transport Encryption (TLS)                 │  ← Network encryption
└────────────────────────────────────────────┘
```

### Layer Comparison

| Layer | Scope | Performance Impact | Key Management |
|---|---|---|---|
| Transport (TLS) | In transit | Minimal | Certificates |
| Disk (LUKS) | At rest (full disk) | <2% | LUKS passphrase |
| TDE | At rest (DB files) | 3-5% | DB master key |
| Column-level | Specific columns | 5-15% | Column keys |
| Application | Before DB write | 5-20% | App-managed keys |

### Defense in Depth

GGID uses multiple layers:
1. **TLS** — All connections encrypted
2. **Disk encryption** — OS-level LUKS
3. **Application-level** — Encrypt PII before storage
4. **Column-level** — pgcrypto for sensitive columns

## PostgreSQL pgcrypto

### Installation

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

### Encryption Functions

```sql
-- Encrypt with passphrase
INSERT INTO users (email_encrypted)
VALUES (pgp_sym_encrypt('user@example.com', 'encryption-key'));

-- Decrypt
SELECT pgp_sym_decrypt(email_encrypted, 'encryption-key') AS email
FROM users WHERE id = 1;

-- Hash (one-way)
INSERT INTO users (password_hash)
VALUES (crypt('password123', gen_salt('bf', 12)));
```

### pg_stat Monitoring

```sql
-- Check pgcrypto usage
SELECT * FROM pg_stat_user_functions
WHERE schemaname = 'public'
AND funcname LIKE 'pgp%';

-- Check table sizes (encrypted vs unencrypted)
SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;
```

## Per-Column Encryption for PII

### Which Columns to Encrypt

| Column | Encrypt? | Method |
|---|---|---|
| email | Yes | pgp_sym_encrypt |
| phone | Yes | pgp_sym_encrypt |
| address | Yes | pgp_sym_encrypt |
| ssn | Yes | pgp_sym_encrypt (restricted) |
| password_hash | No (already hashed) | crypt() |
| username | No (needed for lookup) | Plain |
| tenant_id | No (needed for RLS) | Plain |
| created_at | No | Plain |

### Application-Level Encryption

```go
type EncryptedField struct {
    Ciphertext []byte
    Nonce      []byte
}

func encryptField(plaintext string, key []byte) (*EncryptedField, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
    return &EncryptedField{Ciphertext: ciphertext, Nonce: nonce}, nil
}

func decryptField(field *EncryptedField, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    plaintext, err := gcm.Open(nil, field.Nonce, field.Ciphertext, nil)
    if err != nil {
        return "", err
    }
    return string(plaintext), nil
}
```

### Model with Encrypted Fields

```go
type User struct {
    ID          string          `db:"id"`
    Username    string          `db:"username"`      // Plain (for lookup)
    Email       EncryptedField  `db:"email_encrypted"` // Encrypted
    Phone       EncryptedField  `db:"phone_encrypted"` // Encrypted
    TenantID    string          `db:"tenant_id"`     // Plain (for RLS)
    CreatedAt   time.Time       `db:"created_at"`    // Plain
}
```

## Key Management Hierarchy

### DEK / KEK / MEK

```
MEK (Master Encryption Key)
 │  ← Stored in HSM/KMS, never in application memory
 │
 ├── KEK (Key Encryption Key) per tenant
 │    ← Encrypts DEKs, stored encrypted in DB
 │
 │    ├── DEK (Data Encryption Key) for email column
 │    ├── DEK for phone column
 │    └── DEK for address column
```

### Key Types

| Key | Scope | Rotation | Storage |
|---|---|---|---|
| MEK | Global | 90 days | HSM/KMS (never in DB) |
| KEK | Per-tenant | 180 days | Encrypted in DB (encrypted by MEK) |
| DEK | Per-column per-tenant | 365 days | Encrypted in DB (encrypted by KEK) |

### Key Retrieval

```go
func getDEK(tenantID, column string) ([]byte, error) {
    // Check cache
    cacheKey := fmt.Sprintf("dek:%s:%s", tenantID, column)
    if dek, ok := keyCache.Get(cacheKey); ok {
        return dek.([]byte), nil
    }
    
    // Get encrypted DEK from DB
    encDEK, err := db.GetEncryptedDEK(tenantID, column)
    if err != nil {
        return nil, err
    }
    
    // Get KEK (from KMS, decrypts with MEK)
    kek, err := getKEK(tenantID)
    if err != nil {
        return nil, err
    }
    
    // Decrypt DEK with KEK
    dek, err := decryptWithKey(encDEK, kek)
    if err != nil {
        return nil, err
    }
    
    // Cache
    keyCache.Set(cacheKey, dek, 5*time.Minute)
    return dek, nil
}

func getKEK(tenantID string) ([]byte, error) {
    // Get encrypted KEK from DB
    encKEK, err := db.GetEncryptedKEK(tenantID)
    if err != nil {
        return nil, err
    }
    
    // Decrypt KEK with MEK from KMS
    kek, err := kms.Decrypt(encKEK, "MEK")
    if err != nil {
        return nil, err
    }
    
    return kek, nil
}
```

## Search on Encrypted Data

### Deterministic Encryption

Same plaintext → same ciphertext. Enables equality search:

```go
func encryptDeterministic(plaintext string, key []byte) string {
    block, _ := aes.NewCipher(key)
    // Use fixed IV for deterministic encryption
    iv := make([]byte, aes.BlockSize)  // All zeros
    mode := cipher.NewCBCEncrypter(block, iv)
    padded := pkcs7Pad([]byte(plaintext), aes.BlockSize)
    ciphertext := make([]byte, len(padded))
    mode.CryptBlocks(ciphertext, padded)
    return base64.StdEncoding.EncodeToString(ciphertext)
}
```

```sql
-- Can search with deterministic encryption
SELECT * FROM users
WHERE email_search_hash = encrypt_deterministic('user@example.com', key);
```

### Comparison

| Method | Equality Search | Range Search | Security |
|---|---|---|---|
| Deterministic | Yes | No | Lower (pattern leakage) |
| Order-preserving | Yes | Yes | Lowest (reveals order) |
| Search index | Yes | No | Medium (index of hashes) |
| Blind index | Yes | No | High (separate hash) |

### Blind Index (Recommended)

Store a separate HMAC hash for search, encrypted data separately:

```go
type User struct {
    EmailEncrypted  EncryptedField `db:"email_encrypted"`
    EmailBlindIndex string         `db:"email_blind_idx"` // HMAC for search
}

func computeBlindIndex(plaintext string, key []byte) string {
    h := hmac.New(sha256.New, key)
    h.Write([]byte(plaintext))
    return hex.EncodeToString(h.Sum(nil))
}
```

```sql
-- Search using blind index
SELECT * FROM users WHERE email_blind_idx = 'abc123hash...';
```

## Index Implications

### What Can Be Indexed

| Storage Method | Can Index? | Index Type |
|---|---|---|
| Plain text | Yes | B-tree, Hash, GIN |
| Deterministic encryption | Yes | B-tree (equality only) |
| Randomized encryption | No | None |
| Blind index | Yes | Hash (equality only) |
| HMAC | Yes | B-tree (equality only) |

### Index Strategy

```sql
-- Blind index for email search
CREATE INDEX idx_users_email_blind ON users (email_blind_idx);

-- Regular index on non-encrypted fields
CREATE INDEX idx_users_tenant ON users (tenant_id);
CREATE INDEX idx_users_username ON users (username);

-- Composite index for common queries
CREATE INDEX idx_users_tenant_username ON users (tenant_id, username);
```

### What Can't Be Indexed

- Randomized encrypted columns (each encryption produces different ciphertext)
- Range queries on encrypted data (without order-preserving encryption)
- LIKE queries on encrypted data

## Performance Benchmarks

### Encryption Overhead

| Operation | Plain | Encrypted | Overhead |
|---|---|---|---|
| INSERT 1 row | 0.2ms | 0.8ms | 4x |
| SELECT by ID | 0.1ms | 0.3ms | 3x |
| SELECT by email (indexed) | 0.1ms | 0.2ms | 2x (blind index) |
| SELECT by email (scan) | 50ms | 500ms | 10x |
| Bulk INSERT 1000 rows | 200ms | 800ms | 4x |
| Bulk SELECT 1000 rows | 100ms | 400ms | 4x |

### Key Retrieval Overhead

| Method | Time | Notes |
|---|---|---|
| Cache hit | 0.01ms | In-memory cache |
| Cache miss → KMS | 5-50ms | KMS API call |
| Cache miss → HSM | 10-100ms | HSM API call |

### Optimization Tips

1. **Cache DEKs** — Don't fetch from KMS on every operation
2. **Use blind indexes** — Enable fast equality search
3. **Batch operations** — Amortize key retrieval over multiple rows
4. **Encrypt only PII** — Don't encrypt non-sensitive columns
5. **Use connection pooling** — Reduce TLS handshake overhead

## GGID Database Encryption

### Configuration

```yaml
database:
  encryption:
    enabled: true
    layers:
      transport:
        tls: true
        min_version: "TLS1.2"
      disk:
        luks: true  # OS-managed
      column:
        enabled: true
        engine: "pgcrypto"  # or "application"
        columns:
          - name: "email"
            method: "aes-256-gcm"
            blind_index: true
          - name: "phone"
            method: "aes-256-gcm"
            blind_index: true
          - name: "address"
            method: "aes-256-gcm"
            blind_index: false
    key_management:
      provider: "aws-kms"  # or "hashicorp-vault", "azure-keyvault"
      master_key_id: "kms-key-uuid"
      dek_cache_ttl: 5m
      key_rotation:
        dek: 365d
        kek: 180d
        mek: 90d
```

### Encryption Service

```go
type EncryptionService struct {
    kms    KMSClient
    cache  KeyCache
    config EncryptionConfig
}

func (s *EncryptionService) Encrypt(tenantID, column string, plaintext string) (string, string, error) {
    dek, err := s.getDEK(tenantID, column)
    if err != nil {
        return "", "", err
    }
    
    encrypted, err := encryptAESGCM(plaintext, dek)
    if err != nil {
        return "", "", err
    }
    
    blindIndex := computeBlindIndex(plaintext, s.config.BlindIndexKey)
    
    return encrypted, blindIndex, nil
}

func (s *EncryptionService) Decrypt(tenantID, column string, ciphertext string) (string, error) {
    dek, err := s.getDEK(tenantID, column)
    if err != nil {
        return "", err
    }
    
    return decryptAESGCM(ciphertext, dek)
}
```

## Best Practices

1. **Encrypt PII at application level** — Don't rely solely on disk encryption
2. **Use blind indexes for search** — Enables equality search without decryption
3. **Cache DEKs** — KMS calls are expensive, cache for 5 minutes
4. **Rotate keys regularly** — DEK annually, KEK semi-annually, MEK quarterly
5. **Use KMS/HSM** — Never store master keys in the database
6. **Encrypt only sensitive columns** — Don't encrypt everything (performance)
7. **Use AES-256-GCM** — Authenticated encryption, no separate MAC needed
8. **Don't use deterministic encryption** — Use blind indexes instead
9. **Monitor encryption performance** — Track overhead and optimize
10. **Test key rotation** — Ensure rotation doesn't break decryption