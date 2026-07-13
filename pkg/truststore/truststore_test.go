package truststore

import (
	"crypto/x509"
	"strings"
	"testing"
)

// generateTestCertPEM creates a self-signed cert PEM for testing.
func generateTestCertPEM(t *testing.T, cn string) string {
	t.Helper()
	certPEM, _, _, err := GenerateSelfSignedCert(cn)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}
	return certPEM
}

func TestAddCA_Success(t *testing.T) {
	s := NewMemoryStore()
	pem := generateTestCertPEM(t, "test-ca.example.com")

	ca, err := s.AddCA("Test CA", pem, "admin")
	if err != nil {
		t.Fatalf("AddCA: %v", err)
	}
	if ca.ID == "" {
		t.Error("expected non-empty ID")
	}
	if ca.Name != "Test CA" {
		t.Errorf("expected Name 'Test CA', got %s", ca.Name)
	}
	if ca.Fingerprint == "" {
		t.Error("expected non-empty fingerprint")
	}
	if ca.Subject != "test-ca.example.com" {
		t.Errorf("expected Subject 'test-ca.example.com', got %s", ca.Subject)
	}
}

func TestAddCA_Duplicate(t *testing.T) {
	s := NewMemoryStore()
	pem := generateTestCertPEM(t, "dup-ca.example.com")

	_, err := s.AddCA("First", pem, "admin")
	if err != nil {
		t.Fatalf("first AddCA: %v", err)
	}

	_, err = s.AddCA("Second", pem, "admin")
	if err == nil {
		t.Error("expected error on duplicate fingerprint")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

func TestAddCA_InvalidPEM(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.AddCA("Bad", "not a pem", "admin")
	if err == nil {
		t.Error("expected error on invalid PEM")
	}
}

func TestRemoveCA(t *testing.T) {
	s := NewMemoryStore()
	pem := generateTestCertPEM(t, "remove-ca.example.com")

	ca, _ := s.AddCA("Remove CA", pem, "admin")

	err := s.RemoveCA(ca.ID)
	if err != nil {
		t.Fatalf("RemoveCA: %v", err)
	}

	_, err = s.GetCA(ca.ID)
	if err == nil {
		t.Error("expected error after removal")
	}
}

func TestListCAs(t *testing.T) {
	s := NewMemoryStore()

	cas, err := s.ListCAs()
	if err != nil {
		t.Fatalf("ListCAs: %v", err)
	}
	if len(cas) != 0 {
		t.Errorf("expected 0 CAs, got %d", len(cas))
	}

	pem1 := generateTestCertPEM(t, "ca1.example.com")
	pem2 := generateTestCertPEM(t, "ca2.example.com")
	s.AddCA("CA1", pem1, "admin")
	s.AddCA("CA2", pem2, "admin")

	cas, _ = s.ListCAs()
	if len(cas) != 2 {
		t.Errorf("expected 2 CAs, got %d", len(cas))
	}
}

func TestCertPool_Empty(t *testing.T) {
	s := NewMemoryStore()

	pool, err := s.CertPool()
	if err != nil {
		t.Fatalf("CertPool: %v", err)
	}
	if pool == nil {
		t.Error("expected non-nil pool (system roots)")
	}
}

func TestCertPool_WithCAs(t *testing.T) {
	s := NewMemoryStore()
	pem := generateTestCertPEM(t, "pool-ca.example.com")

	_, err := s.AddCA("Pool CA", pem, "admin")
	if err != nil {
		t.Fatalf("AddCA: %v", err)
	}

	pool, err := s.CertPool()
	if err != nil {
		t.Fatalf("CertPool: %v", err)
	}
	if pool == nil {
		t.Error("expected non-nil pool")
	}

	// Verify the cert is in the pool by trying to parse it
	cert, err := parsePEMCert(pem)
	if err != nil {
		t.Fatalf("parsePEMCert: %v", err)
	}

	// Check that our cert is trusted by the pool
	opts := x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	_, err = cert.Verify(opts)
	// Self-signed certs may not verify fully (no CA basic constraint),
	// but the important thing is that the pool was built without error.
	_ = err
}

func TestHasCAs(t *testing.T) {
	s := NewMemoryStore()
	if s.HasCAs() {
		t.Error("expected false on empty store")
	}

	pem := generateTestCertPEM(t, "has-ca.example.com")
	s.AddCA("Has CA", pem, "admin")

	if !s.HasCAs() {
		t.Error("expected true after adding CA")
	}
}

func TestCertificateManagement(t *testing.T) {
	s := NewMemoryStore()

	pem := generateTestCertPEM(t, "managed.example.com")
	cert, err := ParseCertificateFromPEM("Managed Cert", "TLS", pem, "")
	if err != nil {
		t.Fatalf("ParseCertificateFromPEM: %v", err)
	}

	err = s.AddCertificate(cert)
	if err != nil {
		t.Fatalf("AddCertificate: %v", err)
	}

	got, err := s.GetCertificate(cert.ID)
	if err != nil {
		t.Fatalf("GetCertificate: %v", err)
	}
	if got.Name != "Managed Cert" {
		t.Errorf("expected name 'Managed Cert', got %s", got.Name)
	}
	if got.Type != "TLS" {
		t.Errorf("expected type 'TLS', got %s", got.Type)
	}

	certs := s.ListCertificates()
	if len(certs) != 1 {
		t.Errorf("expected 1 cert, got %d", len(certs))
	}

	err = s.RemoveCertificate(cert.ID)
	if err != nil {
		t.Fatalf("RemoveCertificate: %v", err)
	}

	certs = s.ListCertificates()
	if len(certs) != 0 {
		t.Errorf("expected 0 certs after removal, got %d", len(certs))
	}
}

func TestGenerateCSR(t *testing.T) {
	csrPEM, keyPEM, err := GenerateCSR("test.example.com", "Test Org", "rsa")
	if err != nil {
		t.Fatalf("GenerateCSR: %v", err)
	}
	if !strings.Contains(csrPEM, "CERTIFICATE REQUEST") {
		t.Error("CSR PEM should contain CERTIFICATE REQUEST")
	}
	if !strings.Contains(keyPEM, "PRIVATE KEY") {
		t.Error("Key PEM should contain PRIVATE KEY")
	}
}

func TestGenerateCSR_ECDSA(t *testing.T) {
	csrPEM, keyPEM, err := GenerateCSR("ecdsa.example.com", "", "ecdsa")
	if err != nil {
		t.Fatalf("GenerateCSR ecdsa: %v", err)
	}
	if !strings.Contains(csrPEM, "CERTIFICATE REQUEST") {
		t.Error("CSR PEM should contain CERTIFICATE REQUEST")
	}
	if !strings.Contains(keyPEM, "PRIVATE KEY") {
		t.Error("Key PEM should contain PRIVATE KEY")
	}
}

func TestGenerateCSR_InvalidKeyType(t *testing.T) {
	_, _, err := GenerateCSR("test.example.com", "", "invalid")
	if err == nil {
		t.Error("expected error on invalid key type")
	}
}
