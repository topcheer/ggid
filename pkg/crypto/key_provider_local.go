package crypto

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// localKeyProvider implements KeyProvider using PEM files on disk.
// It is the default when no HSM/KMS is configured.
type localKeyProvider struct {
	metadata KeyMetadata
	pubKey   crypto.PublicKey
	signer   crypto.Signer
}

func newLocalKeyProvider(cfg LocalKeyProviderConfig) (*localKeyProvider, error) {
	if cfg.PrivateKeyPath == "" {
		return nil, fmt.Errorf("%w: local provider requires private_key_path", ErrKeyProviderConfig)
	}

	privPEM, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	block, _ := pem.Decode(privPEM)
	if block == nil {
		return nil, fmt.Errorf("%w: failed to decode PEM private key", ErrKeyProviderConfig)
	}

	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		priv, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			priv, err = x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("%w: unsupported private key format: %w", ErrKeyProviderConfig, err)
			}
		}
	}

	signer, ok := priv.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("%w: private key does not implement crypto.Signer", ErrKeyProviderConfig)
	}

	meta := KeyMetadata{
		KeyID:     cfg.KeyID,
		Algorithm: inferAlgorithm(signer.Public()),
		Use:       "sig",
	}
	if meta.KeyID == "" {
		meta.KeyID = "local-signing-key"
	}

	return &localKeyProvider{
		metadata: meta,
		pubKey:   signer.Public(),
		signer:   signer,
	}, nil
}

func (p *localKeyProvider) Metadata() KeyMetadata { return p.metadata }
func (p *localKeyProvider) Public() crypto.PublicKey { return p.pubKey }
func (p *localKeyProvider) Signer() crypto.Signer  { return p.signer }
func (p *localKeyProvider) Close() error           { return nil }

func inferAlgorithm(pub crypto.PublicKey) KeyAlgorithm {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		_ = k
		return RS256
	case *ecdsa.PublicKey:
		_ = k
		return ES256
	case ed25519.PublicKey:
		return EdDSA
	default:
		return RS256
	}
}

// Stubs for external providers — backend teams implement per-provider packages.

func newPKCS11KeyProvider(_ context.Context, _ PKCS11KeyProviderConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("pkcs11 provider: %w", ErrKeyProviderNotSupported)
}

func newAWSKMSKeyProvider(_ context.Context, _ AWSKMSConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("aws kms provider: %w", ErrKeyProviderNotSupported)
}

func newGCPKMSKeyProvider(_ context.Context, _ GCPKMSConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("gcp kms provider: %w", ErrKeyProviderNotSupported)
}

func newAzureKMSKeyProvider(_ context.Context, _ AzureKMSConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("azure key vault provider: %w", ErrKeyProviderNotSupported)
}

func newVaultTransitKeyProvider(_ context.Context, _ VaultTransitConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("vault transit provider: %w", ErrKeyProviderNotSupported)
}