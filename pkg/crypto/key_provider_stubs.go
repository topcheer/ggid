//go:build !pkcs11

package crypto

import (
	"context"
	"fmt"
)

// Stubs for external providers when PKCS#11 support is not compiled in.
// Vault and AWS KMS providers are always available (pure Go, no cgo).

func newPKCS11KeyProvider(_ context.Context, _ PKCS11KeyProviderConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("pkcs11 provider: %w", ErrKeyProviderNotSupported)
}

func newGCPKMSKeyProvider(_ context.Context, _ GCPKMSConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("gcp kms provider: %w", ErrKeyProviderNotSupported)
}

func newAzureKMSKeyProvider(_ context.Context, _ AzureKMSConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("azure key vault provider: %w", ErrKeyProviderNotSupported)
}

// Vault Transit and AWS KMS are implemented in key_provider_vault.go and
// key_provider_aws.go (pure Go, no cgo dependency).
