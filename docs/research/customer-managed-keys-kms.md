# Customer-Managed Keys (CMK) & KMS Integration: Envelope Encryption, BYOK/HYOK, and Per-Tenant Key Isolation for GGID

> **Focus**: Extending GGID's existing KeyProvider abstraction (which handles JWT signing keys) to support full data-at-rest encryption with customer-managed keys — envelope encryption (DEK/KEK hierarchy), BYOK (Bring Your Own Key), HYOK (Hold Your Own Key), automated key rotation, per-tenant key isolation, China compliance (SM2/SM3/SM4), and FIPS 140-2 Level 3 HSM support.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§11), curl commands (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: KeyProvider Abstraction](#2-ggid-current-state-keyprovider-abstraction)
3. [Gap Analysis](#3-gap-analysis)
4. [KMS Landscape](#4-kms-landscape)
5. [Envelope Encryption Architecture](#5-envelope-encryption-architecture)
6. [BYOK and HYOK Patterns](#6-byok-and-hyok-patterns)
7. [Proposed Architecture: DataKeyProvider](#7-proposed-architecture-datakeyprovider)
8. [Endpoint Precondition Check](#8-endpoint-precondition-check)
9. [API Design + Curl Commands](#9-api-design--curl-commands)
10. [Database Schema](#10-database-schema)
11. [China Compliance: SM2/SM3/SM4](#11-china-compliance-sm2sm3sm4)
12. [FIPS 140-2 HSM Requirements](#12-fips-140-2-hsm-requirements)
13. [Implementation Backlog with DoD](#13-implementation-backlog-with-dod)
14. [Competitive Differentiation](#14-competitive-differentiation)

---

## 1. Executive Summary

Data encryption is the last line of defense in Zero Trust. Even if an attacker breaches the network, steals a database backup, or compromises an admin account — encrypted data is useless without the key. Customer-Managed Keys (CMK) give enterprises control over their encryption keys, ensuring that not even GGID operators can access tenant data without the customer's key.

GGID has a **mature KeyProvider abstraction** (`pkg/crypto/key_provider.go:39`) supporting:
- Local PEM files ✅
- PKCS#11 HSM ✅
- AWS KMS (asymmetric signing) ✅
- GCP Cloud KMS ✅
- Azure Key Vault ✅
- HashiCorp Vault Transit ✅
- Chinese SM2/SM3 ✅

However, this KeyProvider is **signing-only** — it provides `crypto.Signer` for JWT/SAML/assertion signing. It does **not** support:
1. **Data encryption** — No `Encrypt()/Decrypt()` interface
2. **Envelope encryption** — No DEK/KEK hierarchy
3. **Per-tenant keys** — One key pair per service, not per tenant
4. **BYOK** — No API for customers to import their own keys
5. **Automated rotation** — No key rotation lifecycle management
6. **Field-level encryption** — No selective encryption of sensitive columns
7. **Key metadata DB** — No tracking of key versions, rotations, or access

**Recommendation**: Extend the KeyProvider abstraction with a **DataKeyProvider** interface that adds `GenerateDataKey()`, `Encrypt()`, and `Decrypt()` — implementing envelope encryption with per-tenant CMK, BYOK key import, automated rotation, and field-level encryption for sensitive data.

**Estimated effort**: 3 sprints for MVP (DataKeyProvider + envelope encryption + per-tenant keys + BYOK API).

---

## 2. GGID Current State: KeyProvider Abstraction

### Existing Components

| Component | File:Line | Status | Capability |
|-----------|-----------|--------|------------|
| KeyProvider interface | `pkg/crypto/key_provider.go:39` | **Implemented** ✅ | `Signer()`, `Public()`, `Metadata()` |
| Provider types | `key_provider.go:61` | **7 providers** ✅ | local, pkcs11, aws, gcp, azure, vault, sm2 |
| NewKeyProvider factory | `key_provider.go:147` | **Works** ✅ | Switch on provider type |
| AWS KMS provider | `key_provider_aws.go:17` | **Works** ✅ | SigV4 HTTP, GetPublicKey, Sign |
| SM2 config | `key_provider.go:82` | **Works** ✅ | SM2 key pair from PEM |
| Algorithm whitelist | `pkg/crypto/alg_whitelist.go` | **Works** ✅ | RS256/ES256/EdDSA/SM2SM3 |
| Secret broker | `identity/server/secret_broker.go` | **DB-backed** ✅ | Zero-trust secret injection |
| Crypto helpers | `pkg/crypto/crypto.go` | **Works** ✅ | Hash, HMAC, random generation |

### What the KeyProvider CAN Do (Today)

```go
type KeyProvider interface {
    Metadata() KeyMetadata         // {KeyID, Algorithm, Use}
    Public() crypto.PublicKey      // For JWKS, verification
    Signer() crypto.Signer         // For JWT signing
    Close() error
}
```

### What It CANNOT Do

```go
// Missing: data encryption operations
type DataKeyProvider interface {
    GenerateDataKey(ctx context.Context, context string) (plaintextKey, encryptedKey []byte, err error)
    Decrypt(ctx context.Context, encryptedKey []byte, context string) (plaintextKey []byte, err error)
    Encrypt(ctx context.Context, plaintext []byte, context string) (ciphertext []byte, err error)
}
```

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No data encryption** | Sensitive fields stored in plaintext in PostgreSQL |
| 2 | **No envelope encryption** | No DEK/KEK hierarchy for scalable field encryption |
| 3 | **No per-tenant keys** | All tenants share one signing key; no isolation |
| 4 | **No BYOK** | Customers can't bring their own encryption keys |
| 5 | **No rotation** | Keys never rotate; compromised key = permanent exposure |
| 6 | **No field-level encryption** | PII columns (email, phone, SSN) stored in plaintext |
| 7 | **No key metadata DB** | No tracking of key versions, usage, access audit |
| 8 | **No SM4 data encryption** | SM2 signing exists but no SM4 (symmetric) for data |

---

## 4. KMS Landscape

| KMS Provider | Key Types | BYOK | HYOK | Envelope | HSM Level | China Region |
|-------------|-----------|------|------|----------|-----------|-------------|
| **AWS KMS** | RSA, ECC, AES | ✅ | Via CloudHSM | ✅ | FIPS 140-2 L3 (CloudHSM) | ✅ (cn-north) |
| **Azure Key Vault** | RSA, EC, AES | ✅ | Via Managed HSM | ✅ | FIPS 140-2 L3 (MHSM) | ✅ (Azure China) |
| **Google Cloud KMS** | RSA, EC, AES | ✅ | Via External Key Mgr | ✅ | FIPS 140-2 L3 (Cloud HSM) | ✅ (GCP China) |
| **HashiCorp Vault** | Any | ✅ | Transit engine | ✅ | Via PKCS#11 | Self-hosted |
| **IBM Cloud HSM** | RSA, ECC, AES | ✅ | Native | ✅ | FIPS 140-2 L3 | ❌ |
| **Alibaba KMS** | RSA, EC, SM2/SM4 | ✅ | ❌ | ✅ | FIPS 140-2 L3 | ✅ (native) |

---

## 5. Envelope Encryption Architecture

### The DEK/KEK Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                    Envelope Encryption                       │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │  Customer    │    │  Key         │    │  Data        │   │
│  │  Master Key  │───▶│  Encryption  │───▶│  Encryption  │   │
│  │  (CMK/KEK)   │    │  Key (DEK)   │    │  Key (DEK)   │   │
│  │  In KMS/HSM  │    │  Ephemeral   │    │  Encrypts    │   │
│  │  Never leaves│    │  Per record  │    │  field data  │   │
│  └──────────────┘    └──────────────┘    └──────────────┘   │
│                                                             │
│  Flow:                                                      │
│  1. App calls KMS: GenerateDataKey()                        │
│     → KMS returns: plaintext_DEK + encrypted_DEK            │
│  2. App encrypts data with plaintext_DEK (AES-256-GCM)      │
│  3. App stores: encrypted_data + encrypted_DEK              │
│  4. App discards plaintext_DEK                              │
│                                                             │
│  Decrypt:                                                   │
│  1. App reads: encrypted_data + encrypted_DEK               │
│  2. App calls KMS: Decrypt(encrypted_DEK)                   │
│     → KMS returns: plaintext_DEK                            │
│  3. App decrypts data with plaintext_DEK                    │
│  4. App discards plaintext_DEK                              │
└─────────────────────────────────────────────────────────────┘
```

### Why Envelope Encryption?

| Approach | Problem | Envelope Solution |
|----------|---------|-------------------|
| **KMS encrypts all data** | KMS API call per record (slow, 10ms+) | KMS called once per DEK (cached) |
| **Static key in app** | Key compromise = all data exposed | DEK is ephemeral; KEK never leaves KMS |
| **Single key for all** | Key rotation = re-encrypt everything | Rotate KEK; old DEKs still decryptable |
| **No key isolation** | All tenants share one key | Per-tenant CMK; DEKs never cross tenants |

---

## 6. BYOK and HYOK Patterns

### BYOK (Bring Your Own Key)

```
Customer generates key → imports to KMS → GGID uses for envelope encryption

1. Customer generates RSA/ECC key in their KMS
2. Customer wraps key with GGID public key (import wrapper)
3. Customer calls GGID API: POST /api/v1/kms/keys/import
   Body: { wrapped_key, algorithm, tenant_id }
4. GGID stores wrapped key reference (key never in plaintext in GGID)
5. GGID uses KMS GenerateDataKey with imported CMK
6. Customer can revoke key at any time → all data undecryptable
```

### HYOK (Hold Your Own Key)

```
Key NEVER leaves customer's HSM. GGID delegates all crypto operations.

1. Customer configures external KMS endpoint
2. GGID sends Encrypt/Decrypt requests to customer's KMS proxy
3. Customer's KMS performs crypto in HSM; returns only ciphertext/plaintext
4. GGID never sees the key material
5. Requires: always-online connectivity to customer KMS
6. Latency: +5-20ms per operation (network hop to customer KMS)
```

### BYOK vs HYOK Comparison

| Feature | BYOK | HYOK |
|---------|------|------|
| Key location | Stored in GGID's KMS | Never leaves customer HSM |
| Key material | Encrypted at rest in KMS | Customer HSM only |
| Latency | Standard (~1ms) | +5-20ms (external call) |
| Offline | ✅ Works if KMS available | ❌ Requires customer KMS online |
| Key rotation | GGID manages | Customer manages |
| Use case | Standard enterprise | Regulated / sovereign cloud |

---

## 7. Proposed Architecture: DataKeyProvider

### New Interface

```go
// DataKeyProvider extends KeyProvider with data encryption operations.
type DataKeyProvider interface {
    KeyProvider // Existing signing capability

    // GenerateDataKey creates a new DEK encrypted by the CMK.
    // Returns plaintext DEK (for immediate use) + encrypted DEK (for storage).
    GenerateDataKey(ctx context.Context, context string) (plaintext, encrypted []byte, err error)

    // DecryptKey decrypts an encrypted DEK back to plaintext.
    DecryptKey(ctx context.Context, encryptedKey []byte, context string) ([]byte, error)

    // Encrypt encrypts small data directly with the CMK (for non-envelope use).
    Encrypt(ctx context.Context, plaintext []byte, context string) ([]byte, error)

    // Decrypt decrypts data encrypted with Encrypt().
    Decrypt(ctx context.Context, ciphertext []byte, context string) ([]byte, error)

    // RotateKey initiates key rotation for the CMK.
    RotateKey(ctx context.Context) (newKeyID string, err error)
}
```

### Per-Tenant Key Hierarchy

```
┌─────────────────────────────────────────────┐
│  Root Master Key (RMK)                      │
│  — GGID platform key, in KMS               │
│  — Encrypts all tenant KEKs                 │
├─────────────────────────────────────────────┤
│  Tenant KEK (CMK) per tenant                │
│  — One CMK per tenant, in KMS               │
│  — Imported (BYOK) or generated             │
│  — Encrypts DEKs for that tenant            │
├─────────────────────────────────────────────┤
│  Data Encryption Key (DEK) per record       │
│  — Ephemeral AES-256-GCM key                │
│  — Generated by KMS GenerateDataKey         │
│  — Encrypts individual fields               │
├─────────────────────────────────────────────┤
│  Encrypted Field Data                       │
│  — AES-256-GCM ciphertext                   │
│  — Stored alongside encrypted DEK           │
└─────────────────────────────────────────────┘
```

### Field-Level Encryption

```go
// Encrypts a sensitive field before storing in PostgreSQL
func (s *EncryptionService) EncryptField(ctx context.Context, tenantID uuid.UUID, table, column string, plaintext []byte) (*EncryptedField, error) {
    cmk, err := s.getTenantCMK(ctx, tenantID)
    if err != nil {
        return nil, err
    }
    
    // Generate DEK
    plaintextDEK, encryptedDEK, err := cmk.GenerateDataKey(ctx, fmt.Sprintf("tenant:%s:table:%s:col:%s", tenantID, table, column))
    if err != nil {
        return nil, err
    }
    defer wipeKey(plaintextDEK)
    
    // Encrypt data with DEK
    ciphertext, nonce, err := aesGCMEncrypt(plaintextDEK, plaintext)
    if err != nil {
        return nil, err
    }
    
    return &EncryptedField{
        Ciphertext:   ciphertext,
        Nonce:        nonce,
        EncryptedDEK: encryptedDEK,
        KeyID:        cmk.Metadata().KeyID,
        Algorithm:    "AES-256-GCM",
    }, nil
}
```

---

## 8. Endpoint Precondition Check

### Existing Infrastructure (Reuse)

| Component | File:Line | Status | Reuse |
|----------|-----------|--------|-------|
| KeyProvider interface | `pkg/crypto/key_provider.go:39` | ✅ | Extend with DataKeyProvider |
| KMS configs (AWS/GCP/Azure/Vault) | `key_provider.go:61` | ✅ | Add encryption operations |
| AWS KMS provider | `key_provider_aws.go:17` | ✅ | Add GenerateDataKey/Decrypt |
| SM2 provider | `key_provider.go:82` | ✅ | Add SM4 symmetric |
| Algorithm whitelist | `pkg/crypto/alg_whitelist.go` | ✅ | Add SM4, AES-256-GCM |
| Secret broker | `identity/server/secret_broker.go` | ✅ | Use for key material injection |
| Crypto helpers | `pkg/crypto/crypto.go` | ✅ | AES-GCM helpers |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| `/api/v1/kms/keys` | GET | List tenant keys | P0 |
| `/api/v1/kms/keys` | POST | Create/import CMK | P0 |
| `/api/v1/kms/keys/{id}` | GET | Get key metadata | P0 |
| `/api/v1/kms/keys/{id}` | DELETE | Delete/revoke key | P0 |
| `/api/v1/kms/keys/{id}/rotate` | POST | Rotate key | P0 |
| `/api/v1/kms/keys/import` | POST | BYOK key import | P1 |
| `/api/v1/kms/keys/{id}/test` | POST | Test encrypt/decrypt | P1 |
| `/api/v1/kms/keys/{id}/audit` | GET | Key usage audit | P1 |

---

## 9. API Design + Curl Commands

### Create CMK

```bash
curl -X POST https://ggid.corp.com/api/v1/kms/keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "tenant-cmk-primary",
    "provider": "aws",
    "config": {
      "key_spec": "RSA_4096",
      "region": "us-east-1",
      "usage": "ENCRYPT_DECRYPT"
    },
    "rotation_days": 90,
    "description": "Primary CMK for tenant data encryption"
  }'

# Response:
{
  "key_id": "kms_7f3a2b1c-...",
  "name": "tenant-cmk-primary",
  "provider": "aws",
  "kms_key_id": "arn:aws:kms:us-east-1:...:key/...",
  "status": "active",
  "rotation_days": 90,
  "created_at": "2026-07-17T10:00:00Z"
}
```

### BYOK Key Import

```bash
curl -X POST https://ggid.corp.com/api/v1/kms/keys/import \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "customer-imported-key",
    "wrapped_key": "base64-encoded-wrapped-key-material",
    "import_token": "base64-import-token-from-kms",
    "algorithm": "RSA_4096",
    "provider": "aws"
  }'

# Response:
{
  "key_id": "kms_9e8f7g6h-...",
  "status": "imported",
  "kms_key_id": "arn:aws:kms:us-east-1:...:key/...",
  "expires_at": "2027-07-17T00:00:00Z"
}
```

### Rotate Key

```bash
curl -X POST https://ggid.corp.com/api/v1/kms/keys/kms_7f3a/rotate \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response:
{
  "key_id": "kms_7f3a2b1c-...",
  "new_version": 2,
  "previous_version": 1,
  "old_version_retired": true,
  "old_version_decrypt_until": "2027-07-17T00:00:00Z",
  "rotated_at": "2026-10-17T10:00:00Z"
}
```

### Key Audit Trail

```bash
curl "https://ggid.corp.com/api/v1/kms/keys/kms_7f3a/audit?limit=50" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response:
{
  "events": [
    {
      "action": "generate_data_key",
      "request_id": "req_abc",
      "context": "tenant:uuid:table:users:col:ssn",
      "success": true,
      "timestamp": "2026-07-17T10:05:00Z"
    },
    {
      "action": "decrypt",
      "request_id": "req_def",
      "context": "tenant:uuid:table:users:col:ssn",
      "success": true,
      "timestamp": "2026-07-17T10:05:01Z"
    }
  ]
}
```

---

## 10. Database Schema

```sql
-- Tenant CMK registry
CREATE TABLE kms_keys (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(128) NOT NULL,
    description         TEXT,

    -- Provider
    provider            VARCHAR(32) NOT NULL,         -- 'aws', 'gcp', 'azure', 'vault', 'pkcs11', 'sm2'
    kms_key_id          VARCHAR(512),                 -- KMS-specific key ARN/ID
    algorithm           VARCHAR(32),                  -- 'RSA_4096', 'ECC_NIST_P256', 'SM2', etc.
    key_usage           VARCHAR(32) DEFAULT 'ENCRYPT_DECRYPT',

    -- Versioning
    current_version     INT DEFAULT 1,
    rotation_days       INT,                          -- Auto-rotation interval

    -- BYOK
    imported            BOOLEAN DEFAULT false,
    import_expires_at   TIMESTAMPTZ,

    -- State
    status              VARCHAR(16) DEFAULT 'active', -- 'active', 'rotating', 'retired', 'revoked'

    created_by          UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, name)
);

-- Key version history (for rotation tracking)
CREATE TABLE kms_key_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id              UUID NOT NULL REFERENCES kms_keys(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL,
    version             INT NOT NULL,
    kms_key_version_id  VARCHAR(512),                 -- KMS-specific version ID
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    retired_at          TIMESTAMPTZ,
    UNIQUE(key_id, version)
);

-- Key usage audit log
CREATE TABLE kms_audit_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    key_id              UUID NOT NULL,
    action              VARCHAR(32) NOT NULL,         -- 'generate_data_key', 'decrypt', 'encrypt', 'rotate', 'import'
    context             VARCHAR(512),                 -- 'tenant:X:table:Y:col:Z'
    request_id          VARCHAR(128),
    success             BOOLEAN DEFAULT true,
    error_message       TEXT,
    ip_address          VARCHAR(45),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Encrypted field metadata (tracks which columns are encrypted)
CREATE TABLE kms_encrypted_fields (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    table_name          VARCHAR(128) NOT NULL,
    column_name         VARCHAR(128) NOT NULL,
    key_id              UUID NOT NULL REFERENCES kms_keys(id),
    algorithm           VARCHAR(32) DEFAULT 'AES-256-GCM',
    enabled             BOOLEAN DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, table_name, column_name)
);

-- Indexes
CREATE INDEX idx_kms_keys_tenant ON kms_keys (tenant_id, status);
CREATE INDEX idx_kms_versions_key ON kms_key_versions (key_id, version DESC);
CREATE INDEX idx_kms_audit_tenant_time ON kms_audit_log (tenant_id, created_at DESC);
CREATE INDEX idx_kms_audit_key ON kms_audit_log (tenant_id, key_id, created_at DESC);
CREATE INDEX idx_kms_fields_tenant ON kms_encrypted_fields (tenant_id, table_name);
```

---

## 11. China Compliance: SM2/SM3/SM4

### GM/T Standards Overview

| Standard | Algorithm | Use Case | GGID Status |
|----------|-----------|----------|-------------|
| GB/T 32918.2-2016 | **SM2** | Asymmetric signing/diffie-hellman | ✅ `key_provider.go:26` |
| GB/T 32905-2016 | **SM3** | Hash (256-bit) | ✅ Used with SM2SM3 |
| GB/T 32907-2016 | **SM4** | Symmetric block cipher (128-bit key) | ❌ Not implemented |
| GB/T 38635-2020 | **SM9** | Identity-based encryption | ❌ Not implemented |

### SM4 Data Encryption (Needed)

```go
// SM4 is China's equivalent of AES-128
// Required for government/financial deployments in China

type SM4DataKeyProvider struct {
    keyStore SM4KeyStore
}

func (p *SM4DataKeyProvider) GenerateDataKey(ctx context.Context, context string) ([]byte, []byte, error) {
    // Generate SM4 key (128-bit)
    plaintextKey := make([]byte, 16)
    rand.Read(plaintextKey)
    
    // Encrypt with SM4-ECB or SM4-CBC using KEK
    encryptedKey := sm4Encrypt(p.kek, plaintextKey)
    
    return plaintextKey, encryptedKey, nil
}
```

### China Deployment Architecture

```
┌──────────────────────────────────────────────┐
│  China Region Deployment                     │
│                                              │
│  SM2 — JWT signing (asymmetric)             │
│  SM3 — Hash (replaces SHA-256)              │
│  SM4 — Data encryption (replaces AES)       │
│                                              │
│  KMS: Alibaba Cloud KMS or on-prem HSM      │
│  Compliance: 等保2.0 Level 3, 密评要求       │
│                                              │
│  Config:                                     │
│  key_provider:                               │
│    provider: "sm2"                           │
│    sm2:                                      │
│      private_key_path: /keys/sm2_priv.pem   │
│    data_provider: "sm4"                     │
│    sm4:                                      │
│      kek_source: "alibaba_kms"              │
└──────────────────────────────────────────────┘
```

---

## 12. FIPS 140-2 HSM Requirements

### HSM Levels

| Level | Requirements | Use Case |
|-------|-------------|----------|
| **Level 1** | Software-only encryption | Development |
| **Level 2** | Tamper-evident + role auth | General enterprise |
| **Level 3** | Tamper-resistant + identity auth | Financial, government |
| **Level 4** | Tamper-responsive + environmental | High-security |

### FIPS 140-2 Level 3 Requirements for GGID

```
1. Keys generated inside HSM (never exposed as plaintext)
2. Private keys never leave HSM boundary
3. Tamper-resistant physical protection
4. Identity-based authentication (not just passwords)
5. Audit trail of all key operations

GGID's PKCS#11 provider (key_provider.go:97) supports HSM via:
  - PKCS#11 library path
  - Slot/token selection
  - PIN-based authentication
  - Key stored in HSM (private key never extracted)
```

### GGID PKCS#11 Config

```yaml
key_provider:
  provider: "pkcs11"
  pkcs11:
    lib_path: "/usr/lib/softhsm/libsofthsm2.so"
    slot_label: "ggid-token"
    pin: "${HSM_PIN}"
    key_label: "ggid-signing-key"
```

---

## 13. Implementation Backlog with DoD

### P0 — DataKeyProvider + Envelope Encryption (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | DataKeyProvider interface | ✅ Interface defined ✅ AES-256-GCM implementation ✅ go build PASS ✅ ≥3 tests | 2d |
| 2 | KMS key registry DB schema | ✅ CREATE TABLE in migration ✅ No in-memory map ✅ ≥3 tests | 1d |
| 3 | Per-tenant CMK management | ✅ CRUD API for keys ✅ DB-backed ✅ Per-tenant isolation ✅ ≥3 tests | 4d |
| 4 | Envelope encryption service | ✅ GenerateDataKey → AES-GCM encrypt field ✅ Decrypt path ✅ DEK wipe after use ✅ ≥3 tests | 3d |
| 5 | AWS KMS envelope encryption | ✅ GenerateDataKey API call ✅ Decrypt API call ✅ SigV4 auth ✅ ≥3 tests | 3d |
| 6 | Key management API | ✅ 6 endpoints registered ✅ curl test PASS ✅ ≥3 tests | 2d |

### P1 — BYOK + Rotation + SM4 (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | BYOK key import | ✅ Import wrapped key to KMS ✅ Key never in plaintext ✅ ≥3 tests | 3d |
| 8 | Automated key rotation | ✅ Cron-based rotation (rotation_days) ✅ Old version still decrypts ✅ ≥3 tests | 3d |
| 9 | SM4 data encryption | ✅ SM4 GenerateDataKey ✅ SM4 encrypt/decrypt ✅ ≥3 tests | 3d |
| 10 | Field-level encryption integration | ✅ Sensitive columns encrypted in DB ✅ Transparent to app layer ✅ ≥3 tests | 3d |

### P2 — HYOK + Audit + Console (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 11 | HYOK (external KMS proxy) | ✅ Delegate encrypt/decrypt to external KMS ✅ ≥3 tests | 4d |
| 12 | Key usage audit API | ✅ All key operations logged ✅ Filter/query audit trail ✅ ≥3 tests | 2d |
| 13 | Console key management UI | ✅ List/create/rotate keys ✅ Audit trail view ✅ BYOK import wizard | 3d |

### P3 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 14 | HYOK via Vault Transit | HashiCorp Vault as external KMS |
| 15 | SM9 identity-based encryption | China IBE standard |
| 16 | Key access justification | Require reason for decrypt operations |
| 17 | Multi-region key replication | Cross-region KMS sync |
| 18 | Quantum-resistant KMS | ML-KEM for key exchange (PQC) |

---

## 14. Competitive Differentiation

| Feature | GGID (target) | Okta | Microsoft Entra | AWS IAM | Azure AD |
|---------|---------------|------|-----------------|---------|----------|
| **KMS providers** | **6 (AWS/GCP/Azure/Vault/PKCS11/SM2)** | Limited | Azure KV | AWS KMS | Azure KV |
| **Envelope encryption** | **DEK/KEK** | No | Yes | Yes | Yes |
| **Per-tenant CMK** | **Yes** | No | Partial | Via account | Partial |
| **BYOK** | **Yes** | No | Yes | Yes | Yes |
| **HYOK** | **Yes** | No | Yes (Ext Key) | No | Yes |
| **SM2/SM3/SM4** | **Yes** | No | No | No | No |
| **FIPS 140-2 L3** | **Via PKCS#11** | Yes | Yes | Yes (CloudHSM) | Yes (MHSM) |
| **Open source** | **Yes (Apache 2.0)** | No | No | No | No |

**Key differentiator**: GGID would be the only open-source IAM with native SM2/SM3/SM4 support for China compliance AND support for 6 KMS providers including cloud KMS, Vault, PKCS#11 HSM, and Chinese GM standards — all from one unified KeyProvider abstraction.

---

## References

- [AWS KMS Envelope Encryption](https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#enveloping) — DEK/KEK pattern
- [Azure Key Vault BYOK](https://learn.microsoft.com/en-us/azure/key-vault/keys/byok-specification) — Key import
- [HashiCorp Vault Transit](https://developer.hashicorp.com/vault/docs/secrets/transit) — Encryption-as-a-service
- [GB/T 32918.2-2016 (SM2)](http://www.gmbz.org.cn/main/view.html?download=20211008020537598527.html) — Chinese GM standard
- [GB/T 32907-2016 (SM4)](http://www.gmbz.org.cn/main/view.html?download=20211008020538558527.html) — SM4 cipher
- [FIPS 140-2](https://csrc.nist.gov/publications/detail/fips/140/2/final) — Cryptographic module requirements
- [GGID KeyProvider Interface](../pkg/crypto/key_provider.go) — Signing key abstraction at line 39
- [GGID AWS KMS Provider](../pkg/crypto/key_provider_aws.go) — KMS signing at line 17
- [GGID SM2 Config](../pkg/crypto/key_provider.go) — SM2 at line 82
- [GGID PKCS#11 Config](../pkg/crypto/key_provider.go) — HSM at line 97
- [GGID Algorithm Whitelist](../pkg/crypto/alg_whitelist.go) — Supported algorithms
- [GGID Secret Broker](../services/identity/internal/server/secret_broker.go) — Zero-trust secret injection
- [GGID PQC Post-Quantum Research](./pqc-post-quantum-cryptography.md) — Quantum-safe crypto
- [GGID HSM/KMS Integration](./hsm-kms-integration.md) — Previous HSM research
- [GGID Zero Trust Maturity Assessment](./zero-trust-maturity-assessment.md) — Data pillar P0 gap
