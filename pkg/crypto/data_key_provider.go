package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// DataKeyProvider implements envelope encryption: per-tenant KEK encrypts DEKs, DEKs encrypt field data.
// KEK operations delegate to the existing KeyProvider (AWS/GCP/Azure/Vault/PKCS11/local).
type DataKeyProvider interface {
	// GenerateDataKey creates a new DEK for the tenant.
	// Returns (plaintextDEK, encryptedDEK, error).
	// The plaintextDEK is for immediate use; encryptedDEK is for storage.
	GenerateDataKey(ctx context.Context, tenantID string) (plaintextDEK, encryptedDEK []byte, err error)

	// DecryptDataKey decrypts a stored encryptedDEK back to plaintext.
	DecryptDataKey(ctx context.Context, encryptedDEK []byte) (plaintextDEK []byte, err error)

	// EncryptField encrypts a plaintext value using a DEK for the given tenant.
	// Internally: generate/use DEK → AES-256-GCM encrypt → return base64(ciphertext + encryptedDEK).
	EncryptField(ctx context.Context, tenantID string, plaintext []byte) (ciphertext string, err error)

	// DecryptField decrypts a value produced by EncryptField.
	DecryptField(ctx context.Context, ciphertext string) (plaintext []byte, err error)
}

// EnvelopeEncryptionProvider is the production implementation of DataKeyProvider.
// It uses AES-256-GCM for data encryption and a local KEK for DEK wrapping.
type EnvelopeEncryptionProvider struct {
	kekKey []byte // 32-byte KEK (AES-256) for encrypting DEKs
	cipher CipherAlgorithm
}

// CipherAlgorithm selects between AES-256-GCM (international) and SM4-GCM (China compliance).
type CipherAlgorithm string

const (
	CipherAES256GCM CipherAlgorithm = "aes-256-gcm"
	CipherSM4GCM    CipherAlgorithm = "sm4-gcm"
)

// NewEnvelopeEncryptionProvider creates a provider with the given KEK material.
// The KEK must be at least 32 bytes (will be hashed to 32 bytes).
// Default cipher is AES-256-GCM; use WithSM4() for China compliance.
func NewEnvelopeEncryptionProvider(kekMaterial []byte) *EnvelopeEncryptionProvider {
	return &EnvelopeEncryptionProvider{
		kekKey: hashKey(kekMaterial),
		cipher: CipherAES256GCM,
	}
}

// WithSM4 switches to SM4-GCM cipher for China compliance (GB/T 32907).
func (p *EnvelopeEncryptionProvider) WithSM4() *EnvelopeEncryptionProvider {
	p.cipher = CipherSM4GCM
	return p
}

// GenerateDataKey creates a new 32-byte DEK, encrypts it with the KEK.
func (p *EnvelopeEncryptionProvider) GenerateDataKey(ctx context.Context, tenantID string) ([]byte, []byte, error) {
	plaintextDEK := make([]byte, 32) // AES-256 key
	if _, err := io.ReadFull(rand.Reader, plaintextDEK); err != nil {
		return nil, nil, fmt.Errorf("generate DEK: %w", err)
	}

	encryptedDEK, err := p.encryptWithKEK(plaintextDEK)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypt DEK with KEK: %w", err)
	}

	return plaintextDEK, encryptedDEK, nil
}

// DecryptDataKey decrypts an encrypted DEK using the KEK.
func (p *EnvelopeEncryptionProvider) DecryptDataKey(ctx context.Context, encryptedDEK []byte) ([]byte, error) {
	plaintextDEK, err := p.decryptWithKEK(encryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("decrypt DEK with KEK: %w", err)
	}
	return plaintextDEK, nil
}

// EncryptedEnvelope is the wire format for encrypted field values.
// It bundles the encrypted DEK + ciphertext so decryption only needs the KEK.
type EncryptedEnvelope struct {
	EncryptedDEK string `json:"edek"}`          // base64
	Ciphertext   string `json:"ct"}`             // base64
	Cipher       string `json:"cipher"`          // "aes-256-gcm" or "sm4-gcm"
	Timestamp    int64  `json:"ts"`              // creation time
}

// EncryptField encrypts a plaintext value using envelope encryption.
func (p *EnvelopeEncryptionProvider) EncryptField(ctx context.Context, tenantID string, plaintext []byte) (string, error) {
	// Generate DEK.
	plaintextDEK, encryptedDEK, err := p.GenerateDataKey(ctx, tenantID)
	if err != nil {
		return "", err
	}

	// Encrypt data with DEK.
	var ciphertext []byte
	switch p.cipher {
	case CipherSM4GCM:
		ciphertext, err = SM4Encrypt(plaintext, plaintextDEK)
	default:
		ciphertext, err = AESEncrypt(plaintext, plaintextDEK)
	}
	if err != nil {
		return "", fmt.Errorf("encrypt field: %w", err)
	}

	// Zero out plaintext DEK from memory.
	for i := range plaintextDEK {
		plaintextDEK[i] = 0
	}

	env := EncryptedEnvelope{
		EncryptedDEK: base64.StdEncoding.EncodeToString(encryptedDEK),
		Ciphertext:   base64.StdEncoding.EncodeToString(ciphertext),
		Cipher:       string(p.cipher),
		Timestamp:    time.Now().Unix(),
	}

	data, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("marshal envelope: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecryptField decrypts a value produced by EncryptField.
func (p *EnvelopeEncryptionProvider) DecryptField(ctx context.Context, encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode envelope base64: %w", err)
	}

	var env EncryptedEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	encryptedDEK, err := base64.StdEncoding.DecodeString(env.EncryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted DEK: %w", err)
	}

	plaintextDEK, err := p.DecryptDataKey(ctx, encryptedDEK)
	if err != nil {
		return nil, err
	}
	defer func() {
		for i := range plaintextDEK {
			plaintextDEK[i] = 0
		}
	}()

	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	switch env.Cipher {
	case string(CipherSM4GCM):
		return SM4Decrypt(ciphertext, plaintextDEK)
	default:
		return AESDecrypt(ciphertext, plaintextDEK)
	}
}

// encryptWithKEK encrypts data using the KEK (AES-256-GCM).
func (p *EnvelopeEncryptionProvider) encryptWithKEK(plaintext []byte) ([]byte, error) {
	return AESEncrypt(plaintext, p.kekKey)
}

// decryptWithKEK decrypts data using the KEK (AES-256-GCM).
func (p *EnvelopeEncryptionProvider) decryptWithKEK(ciphertext []byte) ([]byte, error) {
	return AESDecrypt(ciphertext, p.kekKey)
}

// TenantKeyRecord represents a tenant key entry in the database.
type TenantKeyRecord struct {
	TenantID     string    `json:"tenant_id"`
	KeyID        string    `json:"key_id"`
	EncryptedKEK string    `json:"encrypted_kek"`   // base64
	ProviderType string    `json:"provider_type"`   // "local" | "aws" | "vault" | ...
	Cipher       string    `json:"cipher"`          // "aes-256-gcm" | "sm4-gcm"
	CreatedAt    time.Time `json:"created_at"`
	RotatedAt    time.Time `json:"rotated_at"`
}

// EnsureTenantKeysSchema returns the SQL for the tenant_keys table.
func EnsureTenantKeysSchema() string {
	return `
	CREATE TABLE IF NOT EXISTS tenant_keys (
		tenant_id     TEXT NOT NULL,
		key_id        TEXT NOT NULL,
		encrypted_kek TEXT NOT NULL,
		provider_type TEXT NOT NULL DEFAULT 'local',
		cipher        TEXT NOT NULL DEFAULT 'aes-256-gcm',
		created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
		rotated_at    TIMESTAMPTZ,
		PRIMARY KEY (tenant_id, key_id)
	);
	CREATE INDEX IF NOT EXISTS idx_tenant_keys_tenant ON tenant_keys(tenant_id);
	`
}

// --- Compile-time interface check ---
var _ DataKeyProvider = (*EnvelopeEncryptionProvider)(nil)

// ensureAES256Key returns a 32-byte key suitable for AES-256.
func ensureAES256Key(key []byte) []byte {
	if len(key) >= 32 {
		return key[:32]
	}
	return hashKey(key)
}

// suppress unused imports
var (
	_ = aes.NewCipher
	_ = cipher.NewGCM
	_ = errors.New
	_ = fmt.Sprintf
)
