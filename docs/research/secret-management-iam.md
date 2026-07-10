# Secret Management for IAM Systems

> Research document for the GGID project — security architecture review of secret
> lifecycle, storage, rotation, leakage prevention, and key hierarchy.

**Status:** Draft
**Audience:** GGID architects, platform engineers, security reviewers
**Last Updated:** 2025

---

## Table of Contents

1. [Secret Types in IAM](#1-secret-types-in-iam)
2. [HashiCorp Vault Integration](#2-hashicorp-vault-integration)
3. [Cloud KMS Integration](#3-cloud-kms-integration)
4. [Secret Rotation Automation](#4-secret-rotation-automation)
5. [Dev/Prod Secret Separation](#5-devprod-secret-separation)
6. [Secret Leakage Prevention](#6-secret-leakage-prevention)
7. [Key Hierarchy & Compartmentalization](#7-key-hierarchy--compartmentalization)
8. [GGID Secret Handling Audit](#8-ggid-secret-handling-audit)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. Secret Types in IAM

An IAM system is a high-value target because it concentrates the keys to the
entire identity kingdom. The secrets it manages fall into several categories
with distinct sensitivity levels and rotation cadences.

### 1.1 Classification Matrix

| Secret Type | Sensitivity | Rotation Frequency | Current GGID Handling |
|---|---|---|---|
| JWT signing private key (RSA/ECDSA) | Critical | 90 days | PEM file at `configs/rsa_private.pem` |
| JWT HMAC secret (HS256) | Critical | 90 days | `JWT_SECRET` env var (Helm) |
| Database credentials | Critical | 30 days (or dynamic) | `DATABASE_URL` env var |
| Redis password | High | 30-90 days | `REDIS_PASSWORD` env var |
| LDAP bind password | High | 30-90 days | `LDAP_BIND_PASSWORD` env var |
| OAuth client secrets | High | 90 days | Not yet implemented |
| SAML signing certificate/key | High | 365 days | Not yet implemented |
| Encryption key for PII at rest | Critical | 365 days | `pkg/crypto` AES-256-GCM, key via parameter |
| API keys (third-party integrations) | Medium | 90 days | Not yet implemented |
| Social provider secrets (Google, GitHub) | High | On compromise | Hardcoded in test fixtures |
| WebAuthn attestation CA keys | Medium | 365 days | Not yet implemented |

### 1.2 Sensitivity Tiers

**Tier 1 — Root of Trust:** JWT signing keys, master encryption keys.
Compromise allows token forgery and data decryption. Must be stored in HSM or
KMS-backed Vault. Rotation requires a multi-day overlap window.

**Tier 2 — Infrastructure Credentials:** Database, Redis, LDAP, NATS
credentials. Compromise allows data exfiltration and privilege escalation.
Should use dynamic, short-lived credentials where possible.

**Tier 3 — Integration Secrets:** OAuth client secrets, social provider keys,
webhook signing secrets. Compromise affects specific integrations. Rotated
through admin workflows.

**Tier 4 — Derived/Runtime Secrets:** Session tokens, refresh tokens,
verification codes. These are ephemeral and should never be persisted in
plaintext.

---

## 2. HashiCorp Vault Integration

Vault provides a unified secrets management layer with dynamic secret
generation, encryption-as-a-service, and audit logging.

### 2.1 KV Secrets Engine v2 — Static Secret Storage

Store long-lived secrets (OAuth client secrets, SAML keys) in Vault KV v2:

```bash
# Enable KV v2
vault secrets enable -path=ggid kv-v2

# Store OAuth client secrets
vault kv put ggid/oauth/google \
  client_id="123456.apps.googleusercontent.com" \
  client_secret="GOCSPX-abcdef123456"

# Store LDAP bind credentials
vault kv put ggid/ldap \
  bind_dn="cn=ggid,ou=service,dc=corp,dc=local" \
  bind_password="s3cr3t-bind-pass"
```

### 2.2 Go Code — Fetching Secrets at Startup

```go
package secrets

import (
	"context"
	"fmt"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
)

// VaultClient wraps the Vault API client for GGID secret access.
type VaultClient struct {
	client *vaultapi.Client
}

// NewVaultClient creates a Vault client using VAULT_ADDR and VAULT_TOKEN
// (or VAULT_APPROLE_ROLE_ID / VAULT_APPROLE_SECRET_ID for AppRole auth).
func NewVaultClient(ctx context.Context) (*VaultClient, error) {
	config := vaultapi.DefaultConfig()
	config.Address = os.Getenv("VAULT_ADDR") // e.g. https://vault.corp.local:8200

	client, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create vault client: %w", err)
	}

	// AppRole authentication (recommended for services)
	roleID := os.Getenv("VAULT_APPROLE_ROLE_ID")
	secretID := os.Getenv("VAULT_APPROLE_SECRET_ID")
	if roleID != "" && secretID != "" {
		resp, err := client.Logical().Write("auth/approle/login", map[string]interface{}{
			"role_id":   roleID,
			"secret_id": secretID,
		})
		if err != nil {
			return nil, fmt.Errorf("vault approle login: %w", err)
		}
		client.SetToken(resp.Auth.ClientToken)
	}

	return &VaultClient{client: client}, nil
}

// GetSecret reads a secret from KV v2. Returns the data map.
func (vc *VaultClient) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	secret, err := vc.client.Logical().Read(fmt.Sprintf("ggid/data/%s", path))
	if err != nil {
		return nil, fmt.Errorf("read secret %s: %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret %s not found", path)
	}
	// KV v2 wraps data under "data" key
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected secret format for %s", path)
	}
	return data, nil
}

// GetLDAPCredentials fetches LDAP bind credentials from Vault.
func (vc *VaultClient) GetLDAPCredentials(ctx context.Context) (bindDN, bindPassword string, err error) {
	data, err := vc.GetSecret(ctx, "ldap")
	if err != nil {
		return "", "", err
	}
	bindDN, _ = data["bind_dn"].(string)
	bindPassword, _ = data["bind_password"].(string)
	return bindDN, bindPassword, nil
}
```

### 2.3 Dynamic Database Credentials

The Vault Database secrets engine generates short-lived, unique database users
on demand, eliminating static DB passwords:

```bash
# Enable database secrets engine
vault secrets enable database

# Configure PostgreSQL connection
vault write database/config/ggid-postgres \
  plugin_name=postgresql-database-plugin \
  connection_url="postgresql://{{username}}:{{password}}@db:5432/ggid?sslmode=disable" \
  allowed_roles="ggid-auth-role" \
  username="vault-admin" \
  password="vault-admin-pass"

# Create role with 1h TTL, max 24h
vault write database/roles/ggid-auth-role \
  db_name=ggid-postgres \
  creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; \
    GRANT ggid_app TO \"{{name}}\";" \
  default_ttl=1h max_ttl=24h
```

Service retrieves credentials on startup:

```go
// GetDynamicDBCredentials fetches short-lived DB credentials from Vault.
// Call this on startup and on credential expiry (auto-renew via LeaseID).
func (vc *VaultClient) GetDynamicDBCredentials(ctx context.Context, role string) (username, password string, leaseID string, err error) {
	secret, err := vc.client.Logical().Read(fmt.Sprintf("database/creds/%s", role))
	if err != nil {
		return "", "", "", fmt.Errorf("get dynamic creds for %s: %w", role, err)
	}
	username, _ = secret.Data["username"].(string)
	password, _ = secret.Data["password"].(string)
	leaseID = secret.LeaseID
	return username, password, leaseID, nil
}
```

### 2.4 Transit Engine — Encryption-as-a-Service

The Transit engine performs cryptographic operations inside Vault, so the
encryption key never leaves the HSM. This is ideal for PII encryption:

```bash
# Create a named encryption key
vault write -f transit/keys/ggid-pii type=aes256-gcm96

# Set a rotation schedule (quarterly)
vault write transit/keys/ggid-pii/config auto_rotate_period=7776000  # 90 days
```

```go
// EncryptPII encrypts plaintext using Vault Transit engine.
// The encryption key never leaves Vault.
func (vc *VaultClient) EncryptPII(ctx context.Context, plaintext []byte) (string, error) {
	ciphertext, err := vc.client.Logical().Write("transit/encrypt/ggid-pii", map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	})
	if err != nil {
		return "", fmt.Errorf("vault transit encrypt: %w", err)
	}
	ct, _ := ciphertext.Data["ciphertext"].(string) // "vault:v1:..."
	return ct, nil
}

// DecryptPII decrypts a Transit-encrypted ciphertext.
func (vc *VaultClient) DecryptPII(ctx context.Context, ciphertext string) ([]byte, error) {
	result, err := vc.client.Logical().Write("transit/decrypt/ggid-pii", map[string]interface{}{
		"ciphertext": ciphertext,
	})
	if err != nil {
		return nil, fmt.Errorf("vault transit decrypt: %w", err)
	}
	ptB64, _ := result.Data["plaintext"].(string)
	return base64.StdEncoding.DecodeString(ptB64)
}
```

---

## 3. Cloud KMS Integration

Cloud KMS providers manage the root key material inside FIPS-140-2 Level 3
HSMs. They are ideal for the Key Encryption Key (KEK) in an envelope
encryption scheme.

### 3.1 Envelope Encryption Pattern

Instead of sending all data to KMS for encryption (latency + cost), generate a
local Data Encryption Key (DEK), encrypt data locally, then encrypt the DEK
with KMS:

```
Plaintext Data
     |
     v
[DEK (local, ephemeral)] --AES-GCM--> Ciphertext Data
     |
     v
[KMS Encrypt(KEK, DEK)] --> Encrypted DEK
     |
     v
Store: (Ciphertext Data, Encrypted DEK)
```

### 3.2 Go Code — Envelope Encryption with AWS KMS

```go
package envelope

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// KMSClient wraps the AWS KMS client for envelope encryption.
type KMSClient struct {
	client  *kms.Client
	keyID   string // KMS key ARN (the KEK)
}

// NewKMSClient creates a KMS client for envelope encryption.
func NewKMSClient(client *kms.Client, keyID string) *KMSClient {
	return &KMSClient{client: client, keyID: keyID}
}

// EncryptedPayload holds ciphertext and the encrypted DEK.
type EncryptedPayload struct {
	Ciphertext   []byte // AES-256-GCM encrypted data
	EncryptedDEK []byte // DEK encrypted by KMS
}

// Encrypt performs envelope encryption.
func (k *KMSClient) Encrypt(ctx context.Context, plaintext []byte) (*EncryptedPayload, error) {
	// 1. Generate a 256-bit DEK locally
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("generate DEK: %w", err)
	}

	// 2. Encrypt data with DEK (AES-256-GCM)
	block, err := aes.NewCipher(dek)
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
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// 3. Encrypt the DEK with KMS (KEK never leaves AWS)
	result, err := k.client.Encrypt(ctx, &kms.EncryptInput{
		KeyId:   &k.keyID,
		Plaintext: dek,
		EncryptionAlgorithm: types.EncryptionAlgorithmSpecSymmetricDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("kms encrypt DEK: %w", err)
	}

	return &EncryptedPayload{
		Ciphertext:   ciphertext,
		EncryptedDEK: result.CiphertextBlob,
	}, nil
}

// Decrypt performs envelope decryption.
func (k *KMSClient) Decrypt(ctx context.Context, payload *EncryptedPayload) ([]byte, error) {
	// 1. Decrypt the DEK via KMS
	result, err := k.client.Decrypt(ctx, &kms.DecryptInput{
		KeyId: &k.keyID,
		CiphertextBlob: payload.EncryptedDEK,
		EncryptionAlgorithm: types.EncryptionAlgorithmSpecSymmetricDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("kms decrypt DEK: %w", err)
	}
	dek := result.Plaintext

	// 2. Decrypt data with the recovered DEK
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(payload.Ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := payload.Ciphertext[:nonceSize], payload.Ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("aes-gcm decrypt: %w", err)
	}
	return plaintext, nil
}
```

### 3.3 Provider Comparison

| Feature | AWS KMS | Google Cloud KMS | Azure Key Vault |
|---|---|---|---|
| Key types | AES, RSA, ECC | AES, RSA, EC | AES, RSA, EC |
| HSM backing | CloudHSM / FIPS 140-2 L3 | Cloud HSM / FIPS 140-2 L3 | Managed HSM / FIPS 140-2 L3 |
| Key rotation | Automatic (1 yr) or manual | Automatic or manual | Automatic (90 days) or manual |
| Envelope encrypt | `Encrypt`/`Decrypt` API | `Encrypt`/`Decrypt` API | `Wrap`/`Unwrap` API |
| Free tier | 20K req/mo | 10K req/mo | 10K req/mo |
| Go SDK | aws-sdk-go-v2 | cloud.google.com/go/kms | azkeys SDK |

---

## 4. Secret Rotation Automation

### 4.1 JWT Signing Key Rotation

JWT key rotation follows a **publish-overlap-revoke** sequence to avoid
invalidating active tokens:

```
Phase 1 (Publish):  Add new key to JWKS → new key signs new tokens
Phase 2 (Overlap):  Both keys active → gateway accepts both
Phase 3 (Revoke):   Old key removed from JWKS after max token TTL expires
```

```go
package keyrotation

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
)

// SigningKey represents a JWT signing key with lifecycle metadata.
type SigningKey struct {
	KID        string    `json:"kid"`
	PrivateKey *rsa.PrivateKey
	Status     string    `json:"status"` // "active", "overlap", "revoked"
	CreatedAt  time.Time `json:"created_at"`
	RotatedAt  time.Time `json:"rotated_at,omitempty"`
}

// KeyManager manages JWT signing key lifecycle.
type KeyManager struct {
	keys map[string]*SigningKey
}

// RotateKey performs zero-downtime key rotation:
// 1. Current active key → overlap status
// 2. Generate new key → active status
// 3. After maxTokenTTL, overlap key → revoked (and removed from JWKS)
func (km *KeyManager) RotateKey(ctx context.Context, maxTokenTTL time.Duration) error {
	// Find current active key and move to overlap
	for _, k := range km.keys {
		if k.Status == "active" {
			k.Status = "overlap"
			k.RotatedAt = time.Now()
		}
		if k.Status == "overlap" {
			// Schedule revocation after max token TTL
			go func(key *SigningKey) {
				select {
				case <-time.After(maxTokenTTL):
					key.Status = "revoked"
				case <-ctx.Done():
					return
				}
			}(k)
		}
	}

	// Generate new active key
	token, err := crypto.GenerateRandomToken(8)
	if err != nil {
		return err
	}
	newKey := &SigningKey{
		KID:        "key-" + token[:12],
		Status:     "active",
		CreatedAt:  time.Now(),
	}
	km.keys[newKey.KID] = newKey
	return nil
}

// JWKSResponse is the JSON Web Key Set published at /.well-known/jwks.json.
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	KID string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// GetJWKS returns the public key set for all non-revoked keys.
func (km *KeyManager) GetJWKS() JWKSResponse {
	var keys []JWK
	for _, k := range km.keys {
		if k.Status == "revoked" || k.PrivateKey == nil {
			continue
		}
		keys = append(keys, JWK{
			KID: k.KID,
			Kty: "RSA",
			Use: "sig",
			Alg: "RS256",
			N:   base64.RawURLEncoding.EncodeToString(k.PrivateKey.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.PrivateKey.E)).Bytes()),
		})
	}
	return JWKSResponse{Keys: keys}
}
```

### 4.2 Database Credential Rotation via Vault

```go
// StartDBCredentialRenewer auto-renews Vault dynamic DB credentials.
// On renewal failure, it triggers a credential refresh + reconnect.
func StartDBCredentialRenewer(ctx context.Context, vc *VaultClient, leaseID string, onRenew func(newLeaseID string)) {
	go func() {
		for {
			secret, err := vc.client.Sys().Lookup(leaseID)
			if err != nil {
				// Lease expired or revoked → trigger reconnect
				newUser, newPass, newLease, err := vc.GetDynamicDBCredentials(ctx, "ggid-auth-role")
				if err != nil {
					time.Sleep(30 * time.Second)
					continue
				}
				_ = newUser
				_ = newPass
				leaseID = newLease
				onRenew(newLease)
				continue
			}
			renewIn := secret.TTL / 2
			select {
			case <-time.After(renewIn):
				_, err := vc.client.Sys().Renew(leaseID, 3600)
				if err != nil {
					// Renewal failed → fetch new creds
					continue
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
```

### 4.3 LDAP/OAuth Secret Rotation Workflow

```
1. Admin initiates rotation in Console
2. New secret generated → stored in Vault (versioned)
3. Service reads new secret from Vault → hot-reloads provider config
4. Old secret kept in Vault for rollback window (7 days)
5. After window, old secret deleted → provider-side revocation
6. Audit event logged: "secret.rotated" with actor, target, timestamp
```

---

## 5. Dev/Prod Secret Separation

### 5.1 The Problem

GGID currently uses environment variables for all environments. In development,
secrets live in `.env` files or Helm `secrets.yaml`. In production, this is
insufficient because:

- Environment variables are visible via `/proc/<pid>/environ` to any process user
- No audit trail of who accessed which secret
- No automatic rotation
- No secret versioning or rollback

### 5.2 Environment-Agnostic Secret Loading

```go
package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// SecretProvider abstracts secret loading across environments.
type SecretProvider interface {
	GetSecret(ctx context.Context, key string) (string, error)
}

// EnvSecretProvider reads from environment variables (dev only).
type EnvSecretProvider struct{}

func (e *EnvSecretProvider) GetSecret(_ context.Context, key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("secret %s not found in environment", key)
	}
	return v, nil
}

// VaultSecretProvider reads from HashiCorp Vault (prod).
type VaultSecretProvider struct {
	client *VaultClient
}

func (v *VaultSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("vault secret key must be 'path/field', got %s", key)
	}
	data, err := v.client.GetSecret(ctx, parts[0])
	if err != nil {
		return "", err
	}
	val, ok := data[parts[1]].(string)
	if !ok {
		return "", fmt.Errorf("field %s not found in secret %s", parts[1], parts[0])
	}
	return val, nil
}

// NewSecretProvider returns the appropriate provider based on environment.
func NewSecretProvider(ctx context.Context) SecretProvider {
	switch os.Getenv("SECRET_BACKEND") {
	case "vault":
		vc, err := NewVaultClient(ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to init vault: %v", err))
		}
		return &VaultSecretProvider{client: vc}
	default:
		return &EnvSecretProvider{}
	}
}
```

### 5.3 Kubernetes: Secrets vs Vault CSI Provider

| Approach | Pros | Cons |
|---|---|---|
| Kubernetes Secrets (etcd) | Simple, native | Base64 encoded (not encrypted by default), no rotation |
| Sealed Secrets (Bitnami) | Encrypted in git, decrypted by controller | Requires controller, no dynamic secrets |
| Vault CSI Provider | Dynamic secrets, auto-rotation, audit trail | Requires Vault + CSI driver |
| External Secrets Operator | Syncs from Vault/AWS SM/GCP SM to k8s secrets | Still materializes secrets in etcd |

**Recommendation for GGID:** Use Vault CSI Provider for Tier 1-2 secrets,
External Secrets Operator for Tier 3-4.

---

## 6. Secret Leakage Prevention

### 6.1 Redacting Secrets from Structured Logs

GGID already has `pkg/pii.Obfuscate()` for masking PII. Extend this to
redact known secret patterns:

```go
package secretredact

import (
	"regexp"
	"strings"
	"sync"
)

var (
	jwtPattern      = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}`)
	bearerPattern   = regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9._-]+`)
	passwordPattern = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|api_key)["\s:=]+([^\s"]+)`)
	basicAuthPattern = regexp.MustCompile(`(?i)basic\s+[a-zA-Z0-9+/=]+`)
	connStrPattern   = regexp.MustCompile(`(postgres|redis|mongodb|amqp)://[^:]+:([^@]+)@`)
)

// SecretRedactor provides thread-safe secret masking for log output.
type SecretRedactor struct {
	mu      sync.RWMutex
	patterns []*regexp.Regexp
}

func NewSecretRedactor() *SecretRedactor {
	return &SecretRedactor{
		patterns: []*regexp.Regexp{
			jwtPattern, bearerPattern, passwordPattern,
			basicAuthPattern, connStrPattern,
		},
	}
}

// Redact masks known secret patterns in a log string.
func (r *SecretRedactor) Redact(s string) string {
	for _, p := range r.patterns {
		s = p.ReplaceAllStringFunc(s, func(match string) string {
			if strings.Contains(strings.ToLower(match), "password") ||
				strings.Contains(strings.ToLower(match), "secret") ||
				strings.Contains(strings.ToLower(match), "token") ||
				strings.Contains(strings.ToLower(match), "api_key") {
				return p.ReplaceAllString(match, "${1}[REDACTED]")
			}
			return "[REDACTED]"
		})
	}

	// Mask connection string passwords
	s = connStrPattern.ReplaceAllString(s, "${1}://***:***@")

	return s
}
```

### 6.2 Git Pre-Commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.18.0
    hooks:
      - id: gitleaks
        args: ['--verbose', '--redact']

  - repo: https://github.com/trufflesecurity/trufflehog
    rev: v3.73.0
    hooks:
      - id: trufflehog
        args: ['--no-update', '--fail', '--no-verification', 'git', 'file://.']
```

### 6.3 CI/CD Pipeline Secret Scanning

```yaml
# .github/workflows/secret-scan.yml
name: Secret Scanning
on: [push, pull_request]

jobs:
  gitleaks:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # full history for scan
      - name: Gitleaks
        uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  trufflehog:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: TruffleHog OSS
        uses: trufflesecurity/trufflehog@main
        with:
          path: .
          extra_args: --only-verified --fail
```

---

## 7. Key Hierarchy & Compartmentalization

### 7.1 Three-Tier Key Hierarchy

```
Master Key (KEK) — stored in HSM/KMS, never in application memory
     |
     +-- Service DEK — per-service data encryption key
     |       |
     |       +-- Tenant Key A — derived via HKDF for tenant A
     |       +-- Tenant Key B — derived via HKDF for tenant B
     |       +-- Tenant Key C — derived via HKDF for tenant C
     |
     +-- Signing Key — RSA/ECDSA for JWT signing
     +-- Session Key — HMAC for CSRF/session tokens
```

### 7.2 Why Per-Tenant Keys?

In a multi-tenant IAM system, per-tenant encryption keys provide:

1. **Blast radius containment:** Compromise of one tenant's key does not expose
   other tenants' data
2. **Key revocation:** Revoking a tenant's key instantly makes their data
   unreadable (for offboarding)
3. **Regulatory compliance:** GDPR right-to-be-forgotten can be implemented by
   destroying the tenant key

### 7.3 HKDF for Tenant-Specific Keys

```go
package tenantkeys

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/hkdf"
	"io"
)

// TenantKeyManager derives per-tenant encryption keys from a master key
// using HKDF (RFC 5869). The master key never leaves the KMS.
type TenantKeyManager struct {
	masterKey []byte // fetched from KMS at startup
}

// NewTenantKeyManager creates a manager with the given master DEK.
func NewTenantKeyManager(masterKey []byte) *TenantKeyManager {
	return &TenantKeyManager{masterKey: masterKey}
}

// DeriveKey generates a 256-bit key unique to the given tenant.
// The info string ensures domain separation between key purposes.
func (tkm *TenantKeyManager) DeriveKey(tenantID, purpose string) ([]byte, error) {
	info := []byte(fmt.Sprintf("ggid-tenant-%s-%s", tenantID, purpose))
	reader := hkdf.New(sha512.New384, tkm.masterKey, []byte(tenantID), info)
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("hkdf derive: %w", err)
	}
	return key, nil
}

// DeriveSigningKey generates an HMAC-SHA256 key for tenant-specific signing.
func (tkm *TenantKeyManager) DeriveSigningKey(tenantID string) ([]byte, error) {
	return tkm.DeriveKey(tenantID, "signing")
}

// FingerprintKey returns a non-reversible fingerprint for audit logging.
func (tkm *TenantKeyManager) FingerprintKey(tenantID, purpose string) string {
	key, _ := tkm.DeriveKey(tenantID, purpose)
	h := sha256.Sum256(key)
	return "fp:" + hex.EncodeToString(h[:8])
}

// VerifyIntegrity uses HMAC to verify data integrity with a tenant-specific key.
func (tkm *TenantKeyManager) ComputeHMAC(tenantID string, data []byte) ([]byte, error) {
	key, err := tkm.DeriveSigningKey(tenantID)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil), nil
}
```

---

## 8. GGID Secret Handling Audit

### 8.1 Current State — `pkg/crypto`

GGID's `pkg/crypto/crypto.go` provides:
- **Argon2id** password hashing with 64MB/3 iterations/2 parallelism — strong.
- **AES-256-GCM** encryption via `AESEncrypt`/`AESDecrypt`.
- `GenerateRandomToken` using `crypto/rand` — cryptographically secure.

**Finding 1 — Key derivation uses SHA-256 hash:**
```go
func hashKey(key []byte) []byte {
    h := sha256.Sum256(key)
    return h[:]
}
```
The `AESEncrypt`/`AESDecrypt` functions hash the provided key with SHA-256
before use. This means any-length key is accepted and silently normalized.
There is no key strength validation — a 1-byte key produces a valid AES key.
While the hash output is 32 bytes (correct for AES-256), the effective entropy
is bounded by the input key, and SHA-256 is not a proper KDF (no salt, no
iterations, no work factor).

**Severity:** Medium. Should use HKDF or require exactly 32 bytes.

**Finding 2 — No key lifecycle management:**
The crypto package treats keys as opaque byte slices. There is no concept of
key versioning, rotation, or provenance. The caller is responsible for
storage, and there is no audit trail of which key encrypted what.

**Severity:** Medium. Acceptable for library layer; must be addressed at
service layer.

### 8.2 Current State — Service `main.go` Files

**Auth Service** (`services/auth/cmd/main.go`):
- JWT keys loaded from PEM files at `configs/rsa_private.pem` /
  `configs/rsa_public.pem`.
- If the PEM file does not exist, `loadOrCreatePrivateKey()` **generates a new
  2048-bit RSA key** and writes it to disk with `0600` permissions.
- LDAP credentials loaded from environment variables:
  `LDAP_URL`, `LDAP_BIND_DN`, `LDAP_BIND_PASSWORD`, etc.
- Database URL defaults to `postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable`.

**Finding 3 — Auto-generated RSA key on first boot:**
```go
func loadOrCreatePrivateKey(path string) (*rsa.PrivateKey, error) {
    if data, err := os.ReadFile(path); err == nil {
        return parsePrivateKey(data)
    }
    key, err := rsa.GenerateKey(rand.Reader, 2048)
    // ... writes to configs/rsa_private.pem
}
```
In a multi-instance deployment, each instance generates a different key pair,
causing JWT validation failures. There is no warning log or startup check.

**Severity:** High. Should fail loudly if key file is missing in production.

**Finding 4 — Hardcoded default database credentials:**
```go
URL: "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable"
```
Multiple services (`auth`, `identity`, `oauth`) have this default. While
convenient for development, these credentials could be used in production if
`DATABASE_URL` is not set.

**Severity:** Medium. Should panic if `DATABASE_URL` is empty in production mode.

**Finding 5 — Redis password in config struct:**
```go
type RedisConfig struct {
    Password string `yaml:"password"`
}
```
Loaded from environment but stored in a plain struct field. No masking in
logs, no zeroing after use.

**Severity:** Low. Standard pattern but should be documented.

### 8.3 Current State — Gateway Configuration

The gateway loads JWT public key from PEM file or JWKS URL:
```go
JWKSURL:       "", // empty = use local public key
PublicKeyPath: "configs/rsa_public.pem",
```

**Finding 6 — No HMAC secret validation:**
The `GRPCInterceptorConfig` has a `JWTSecret` field for HMAC validation:
```go
if cfg.JWTSecret != "" { ... }
```
If `JWTSecret` is empty, JWT validation is **silently skipped**. There is no
warning that the gateway is running without auth.

**Severity:** High in production. Should log a warning or refuse to start.

### 8.4 Current State — Helm Secrets

```yaml
# deploy/helm/ggid/templates/secrets.yaml
stringData:
  secret: {{ .Values.jwt.secret | default (randAlphaNum 32) | quote }}
```

**Finding 7 — Auto-generated JWT secret stored as Kubernetes Secret:**
The Helm chart auto-generates a 32-char JWT secret if not provided. This is
stored as a Kubernetes Secret (base64 in etcd). On Helm upgrade without
explicit value, the secret may regenerate, invalidating all tokens.

**Severity:** Medium. Should persist across upgrades and be backed up.

### 8.5 Positive Findings

- `pkg/pii` provides masking for emails, phones, IPs, UUIDs, SSNs, credit cards
- PEM files written with `0600` permissions and `configs/` dir with `0700`
- `crypto/rand` used for all randomness (not `math/rand`)
- Argon2id parameters are production-grade (64MB memory, 3 iterations)
- Gateway has JWKS refresh with 15-minute interval

---

## 9. Gap Analysis & Recommendations

### 9.1 What GGID Currently Lacks

| Gap | Impact | Priority |
|---|---|---|
| No external secret store (Vault/KMS) | All secrets in env vars / PEM files | P0 |
| No automated key rotation | Manual rotation = downtime or skipped | P0 |
| No per-tenant encryption keys | Single key compromise = full breach | P1 |
| No secret leakage scanning in CI | Secrets can leak to git/CI logs | P1 |
| No audit trail for secret access | No forensic capability post-incident | P1 |
| Auto-generated keys on first boot | Multi-instance key mismatch | P1 |
| No KMS-backed envelope encryption | DEK stored alongside ciphertext | P2 |
| No dynamic database credentials | Static DB passwords never rotate | P2 |

### 9.2 Implementation Roadmap

#### Action 1: Integrate Vault Secret Provider (Effort: 2 weeks)

Create `pkg/secrets/` package with the `SecretProvider` interface (Section 5.2).
Wire all services to use `SecretProvider` instead of direct `os.Getenv()` for
sensitive values. In dev mode, fall back to environment variables. In
production, require `VAULT_ADDR` + AppRole credentials.

**Deliverables:**
- `pkg/secrets/provider.go` — interface + env/vault implementations
- Service `main.go` updates to use provider for DB URL, Redis password, LDAP creds
- Documentation: `docs/operations/vault-setup.md`

#### Action 2: Fix JWT Key Bootstrapping (Effort: 3 days)

Replace `loadOrCreatePrivateKey()` with `loadPrivateKeyOrFail()`:
- If `JWT_PRIVATE_KEY_PATH` is set and file missing → panic with clear error
- If not set and `GO_ENV != production` → auto-generate with warning log
- Add health check endpoint that verifies key is loaded and matches gateway's
  public key

**Deliverables:**
- Patch `services/auth/internal/service/token_service.go`
- Add startup validation in `main.go`

#### Action 3: Add Secret Redaction to Logging (Effort: 1 week)

Extend `pkg/pii` with `SecretRedactor` (Section 6.1). Wrap `log.Printf` calls
or use a structured logger (slog) with a redacting handler. Add gitleaks +
trufflehog pre-commit hooks and CI pipeline scanning.

**Deliverables:**
- `pkg/pii/redact.go` — secret pattern redaction
- `.pre-commit-config.yaml` — gitleaks + trufflehog
- `.github/workflows/secret-scan.yml` — CI scanning

#### Action 4: Per-Tenant Key Derivation (Effort: 2 weeks)

Implement `TenantKeyManager` (Section 7.3) in `pkg/crypto`. Store the master
DEK in KMS (envelope encryption). Derive per-tenant keys for PII encryption.
Wire into `pkg/pii` so encrypted PII fields use tenant-specific keys.

**Deliverables:**
- `pkg/crypto/tenant_keys.go` — HKDF-based key derivation
- Integration with existing `AESEncrypt`/`AESDecrypt` (accept derived key)
- Migration plan for existing encrypted data (re-encrypt with tenant keys)

#### Action 5: Vault Transit for Encryption-as-a-Service (Effort: 1 week)

When Vault is available, use the Transit engine for PII encryption instead of
local AES. This ensures the encryption key never lives in application memory.
Fall back to local `AESEncrypt` when Vault is unavailable (dev mode).

**Deliverables:**
- `pkg/pii/transit.go` — Vault Transit integration
- `pkg/pii/encryptor.go` — unified interface (local AES vs Vault Transit)
- Configuration: `PII_ENCRYPTION_BACKEND=local|vault`

### 9.3 Summary

GGID's cryptographic primitives are sound (Argon2id, AES-256-GCM, crypto/rand),
but the **secret management layer is missing**. Secrets are loaded from
environment variables and PEM files with no rotation, no external store, and
no audit trail. The highest-impact improvements are:

1. **Vault integration** — central secret store with dynamic credentials
2. **Fix key bootstrapping** — prevent multi-instance key mismatches
3. **Per-tenant encryption keys** — limit blast radius with HKDF derivation
4. **Secret redaction + scanning** — prevent leaks in logs and source control
5. **Transit encryption** — keep encryption keys in HSM, not application memory

These improvements bring GGID from "cryptographically correct but operationally
manual" to "enterprise-grade secret lifecycle management."

---

## References

- [HashiCorp Vault Documentation](https://developer.hashicorp.com/vault/docs)
- [AWS KMS Developer Guide](https://docs.aws.amazon.com/kms/latest/developerguide/)
- [RFC 5869 — HKDF](https://datatracker.ietf.org/doc/html/rfc5869)
- [NIST SP 800-57 — Key Management](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)
- [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html)
- [Gitleaks](https://github.com/gitleaks/gitleaks)
- [TruffleHog](https://github.com/trufflesecurity/trufflehog)
- [Vault CSI Provider](https://developer.hashicorp.com/vault/docs/platform/k8s/csi)
- [External Secrets Operator](https://external-secrets.io/)
- [GGID Security Checklist](../security-checklist.md)
- [GGID Token Lifecycle](../token-lifecycle.md)
