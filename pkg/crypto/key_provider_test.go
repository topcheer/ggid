package crypto

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestNewKeyProvider_Local(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "test.pem")

	// Generate an RSA key for testing
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privBytes := x509.MarshalPKCS1PrivateKey(key)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	provider, err := NewKeyProvider(context.Background(), KeyProviderConfig{
		Provider: "local",
		Local: LocalKeyProviderConfig{
			PrivateKeyPath: privPath,
			KeyID:          "test-key",
		},
	})
	if err != nil {
		t.Fatalf("NewKeyProvider: %v", err)
	}
	defer provider.Close()

	meta := provider.Metadata()
	if meta.KeyID != "test-key" {
		t.Errorf("expected key id test-key, got %s", meta.KeyID)
	}
	if meta.Algorithm != RS256 {
		t.Errorf("expected RS256, got %s", meta.Algorithm)
	}
	if provider.Public() == nil {
		t.Error("expected public key")
	}
	if provider.Signer() == nil {
		t.Error("expected signer")
	}
}

func TestNewKeyProvider_Unsupported(t *testing.T) {
	_, err := NewKeyProvider(context.Background(), KeyProviderConfig{Provider: "unknown"})
	if err != ErrKeyProviderNotSupported {
		t.Errorf("expected ErrKeyProviderNotSupported, got %v", err)
	}
}

func TestNewKeyProvider_Local_MissingPath(t *testing.T) {
	_, err := NewKeyProvider(context.Background(), KeyProviderConfig{Provider: "local"})
	if err == nil {
		t.Error("expected error for missing private key path")
	}
}