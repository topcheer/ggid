package tlsconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServerTLS_MissingFile(t *testing.T) {
	_, err := LoadServerTLS("nonexistent-cert.pem", "nonexistent-key.pem")
	if err == nil {
		t.Fatal("expected error for missing files")
	}
}

func TestLoadClientTLS_MissingFile(t *testing.T) {
	_, err := LoadClientTLS("nonexistent-ca.pem")
	if err == nil {
		t.Fatal("expected error for missing CA file")
	}
}

func TestLoadClientTLS_InvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()
	caFile := filepath.Join(tmpDir, "ca.pem")
	if err := os.WriteFile(caFile, []byte("not a certificate"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadClientTLS(caFile)
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestLoadMutualTLS_MissingCert(t *testing.T) {
	_, err := LoadMutualTLS("missing.pem", "missing-key.pem", "missing-ca.pem")
	if err == nil {
		t.Fatal("expected error for missing files")
	}
}
