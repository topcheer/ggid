//go:build !pkcs11

package crypto

import (
	"context"
	"fmt"
)

// Stubs for external providers when PKCS#11 support is not compiled in.

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
