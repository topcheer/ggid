// Package crypto provides key management abstractions supporting local PEM files,
// PKCS#11 HSMs, cloud KMS services, and HashiCorp Vault Transit.
package crypto

import (
	"context"
	"crypto"
	"errors"
)

// KeyAlgorithm represents the signing algorithm family exposed by a KeyProvider.
type KeyAlgorithm string

const (
	RS256 KeyAlgorithm = "RS256" // RSA PKCS#1 v1.5 + SHA-256
	RS384 KeyAlgorithm = "RS384"
	RS512 KeyAlgorithm = "RS512"
	ES256 KeyAlgorithm = "ES256" // ECDSA P-256 + SHA-256
	ES384 KeyAlgorithm = "ES384"
	ES512 KeyAlgorithm = "ES512"
	PS256 KeyAlgorithm = "PS256" // RSA-PSS + SHA-256
	PS384 KeyAlgorithm = "PS384"
	PS512 KeyAlgorithm = "PS512"
	EdDSA KeyAlgorithm = "EdDSA"
)

// KeyMetadata describes the key used by a provider.
type KeyMetadata struct {
	KeyID     string
	Algorithm KeyAlgorithm
	Use       string // "sig" or "enc"
}

// KeyProvider is the abstraction used by GGID to sign JWTs, SAML assertions,
// and other cryptographic material. It is intentionally narrow so that
// HSM, cloud KMS, and local PEM implementations all expose the same surface.
type KeyProvider interface {
	// Metadata returns the key identifier and algorithm.
	Metadata() KeyMetadata

	// Public returns the public key (used for JWKS, verification, etc.).
	Public() crypto.PublicKey

	// Signer returns a crypto.Signer that delegates signing to the provider.
	// For local keys this is the private key itself; for HSM/KMS it calls the
	// external service. The returned Signer is safe for concurrent use.
	Signer() crypto.Signer

	// Close releases any resources held by the provider (sessions, clients).
	Close() error
}

// KeyProviderFactory creates a KeyProvider from configuration.
type KeyProviderFactory func(ctx context.Context, cfg KeyProviderConfig) (KeyProvider, error)

// KeyProviderConfig selects the provider implementation and its parameters.
// Only one provider block should be populated at a time.
type KeyProviderConfig struct {
	Provider string `json:"provider" yaml:"provider"` // "local", "pkcs11", "aws", "gcp", "azure", "vault"

	// Local PEM file paths
	Local LocalKeyProviderConfig `json:"local,omitempty" yaml:"local,omitempty"`

	// PKCS#11 HSM
	PKCS11 PKCS11KeyProviderConfig `json:"pkcs11,omitempty" yaml:"pkcs11,omitempty"`

	// Cloud KMS configurations
	AWS   AWSKMSConfig   `json:"aws,omitempty" yaml:"aws,omitempty"`
	GCP   GCPKMSConfig   `json:"gcp,omitempty" yaml:"gcp,omitempty"`
	Azure AzureKMSConfig `json:"azure,omitempty" yaml:"azure,omitempty"`

	// HashiCorp Vault Transit
	Vault VaultTransitConfig `json:"vault,omitempty" yaml:"vault,omitempty"`
}

// LocalKeyProviderConfig loads a signing key from PEM files.
type LocalKeyProviderConfig struct {
	PrivateKeyPath string `json:"private_key_path" yaml:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path,omitempty" yaml:"public_key_path,omitempty"`
	KeyID          string `json:"key_id,omitempty" yaml:"key_id,omitempty"`
}

// PKCS11KeyProviderConfig describes a PKCS#11 token and key object.
type PKCS11KeyProviderConfig struct {
	LibPath   string `json:"lib_path" yaml:"lib_path"`
	SlotLabel string `json:"slot_label,omitempty" yaml:"slot_label,omitempty"`
	SlotID    uint   `json:"slot_id,omitempty" yaml:"slot_id,omitempty"`
	PIN       string `json:"pin" yaml:"pin"`
	KeyLabel  string `json:"key_label" yaml:"key_label"`
	KeyID     string `json:"key_id,omitempty" yaml:"key_id,omitempty"`
}

// AWSKMSConfig describes an AWS KMS asymmetric signing key.
type AWSKMSConfig struct {
	KeyID     string `json:"key_id" yaml:"key_id"` // ARN or key ID
	Region    string `json:"region,omitempty" yaml:"region,omitempty"`
	Algorithm string `json:"algorithm" yaml:"algorithm"` // RSASSA_PKCS1_V1_5_SHA_256, etc.
	KeyIDHint string `json:"key_id_hint,omitempty" yaml:"key_id_hint,omitempty"` // JWKS kid
}

// GCPKMSConfig describes a Google Cloud KMS asymmetric signing key.
type GCPKMSConfig struct {
	KeyVersion string `json:"key_version" yaml:"key_version"` // full resource name
	Algorithm  string `json:"algorithm" yaml:"algorithm"`      // RSA_SIGN_PKCS1_2048_SHA256, etc.
	KeyIDHint  string `json:"key_id_hint,omitempty" yaml:"key_id_hint,omitempty"`
}

// AzureKMSConfig describes an Azure Key Vault signing key.
type AzureKMSConfig struct {
	VaultURL   string `json:"vault_url" yaml:"vault_url"`
	KeyName    string `json:"key_name" yaml:"key_name"`
	KeyVersion string `json:"key_version,omitempty" yaml:"key_version,omitempty"`
	Algorithm  string `json:"algorithm" yaml:"algorithm"` // PS256, RS256, ES256, etc.
	KeyIDHint  string `json:"key_id_hint,omitempty" yaml:"key_id_hint,omitempty"`
}

// VaultTransitConfig describes a HashiCorp Vault Transit signing key.
type VaultTransitConfig struct {
	Address   string `json:"address" yaml:"address"`
	KeyName   string `json:"key_name" yaml:"key_name"`
	Algorithm string `json:"algorithm" yaml:"algorithm"` // sha2-256, sha2-384, etc.
	KeyIDHint string `json:"key_id_hint,omitempty" yaml:"key_id_hint,omitempty"`
	TokenPath string `json:"token_path,omitempty" yaml:"token_path,omitempty"` // file containing token
}

var (
	ErrKeyProviderNotSupported = errors.New("key provider type not supported")
	ErrKeyProviderConfig       = errors.New("invalid key provider configuration")
)

// NewKeyProvider creates a KeyProvider based on the supplied configuration.
// This is the single entry point used by auth and oauth services at startup.
func NewKeyProvider(ctx context.Context, cfg KeyProviderConfig) (KeyProvider, error) {
	switch cfg.Provider {
	case "", "local":
		return newLocalKeyProvider(cfg.Local)
	case "pkcs11":
		return newPKCS11KeyProvider(ctx, cfg.PKCS11)
	case "aws":
		return newAWSKMSKeyProvider(ctx, cfg.AWS)
	case "gcp":
		return newGCPKMSKeyProvider(ctx, cfg.GCP)
	case "azure":
		return newAzureKMSKeyProvider(ctx, cfg.Azure)
	case "vault":
		return newVaultTransitKeyProvider(ctx, cfg.Vault)
	default:
		return nil, ErrKeyProviderNotSupported
	}
}