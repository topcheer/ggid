package tlsconfig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadServerTLS_MissingFile(t *testing.T) {
	_, err := LoadServerTLS("nonexistent-cert.pem", "nonexistent-key.pem")
	if err == nil {
		t.Fatal("expected error for missing files")
	}
}

func TestLoadServerTLS_Success(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	cfg, err := LoadServerTLS(certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadServerTLS failed: %v", err)
	}
	if cfg == nil || len(cfg.Certificates) != 1 {
		t.Error("expected valid config with 1 certificate")
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
	_ = os.WriteFile(caFile, []byte("not a certificate"), 0644)
	_, err := LoadClientTLS(caFile)
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestLoadClientTLS_Success(t *testing.T) {
	caFile := generateTestCA(t)
	cfg, err := LoadClientTLS(caFile)
	if err != nil {
		t.Fatalf("LoadClientTLS failed: %v", err)
	}
	if cfg == nil || cfg.RootCAs == nil {
		t.Error("expected non-nil config and RootCAs")
	}
}

func TestLoadMutualTLS_MissingCert(t *testing.T) {
	_, err := LoadMutualTLS("missing.pem", "missing-key.pem", "missing-ca.pem")
	if err == nil {
		t.Fatal("expected error for missing files")
	}
}

func TestLoadMutualTLS_InvalidCA(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	tmpDir := t.TempDir()
	badCA := filepath.Join(tmpDir, "bad-ca.pem")
	_ = os.WriteFile(badCA, []byte("not pem"), 0644)
	_, err := LoadMutualTLS(certFile, keyFile, badCA)
	if err == nil {
		t.Fatal("expected error for invalid CA PEM")
	}
}

func TestLoadMutualTLS_Success(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	caFile := generateTestCA(t)
	cfg, err := LoadMutualTLS(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("LoadMutualTLS failed: %v", err)
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("expected RequireAndVerifyClientCert, got %d", cfg.ClientAuth)
	}
	if cfg.RootCAs == nil || cfg.ClientCAs == nil {
		t.Error("expected non-nil RootCAs and ClientCAs")
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
}

func generateTestCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()
	tmpDir := t.TempDir()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certFile = writePEM(t, tmpDir, "cert.pem", "CERTIFICATE", certDER)
	keyFile = writePEM(t, tmpDir, "key.pem", "EC PRIVATE KEY", keyDER)
	return
}

func generateTestCA(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return writePEM(t, tmpDir, "ca.pem", "CERTIFICATE", certDER)
}

func writePEM(t *testing.T, dir, name, blockType string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_ = pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
	return path
}
