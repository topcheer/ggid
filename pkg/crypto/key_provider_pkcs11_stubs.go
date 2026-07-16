//go:build pkcs11

package crypto

import (
	"context"
	"fmt"
)

// Stubs for providers not yet implemented when PKCS#11 support is compiled in.
// Vault and AWS KMS are always available (pure Go, no cgo).

func newGCPKMSKeyProvider(_ context.Context, _ GCPKMSConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("gcp kms provider: %w", ErrKeyProviderNotSupported)
}

func newAzureKMSKeyProvider(_ context.Context, _ AzureKMSConfig) (KeyProvider, error) {
	return nil, fmt.Errorf("azure key vault provider: %w", ErrKeyProviderNotSupported)
}
