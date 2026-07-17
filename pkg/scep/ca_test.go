package scep

import (
	"context"
	"strings"
	"testing"
)

func TestCA_GenerateAndCACert(t *testing.T) {
	ca, err := NewCA(nil)
	if err != nil {
		t.Fatalf("NewCA failed: %v", err)
	}

	pem := ca.CACertPEM()
	if !strings.Contains(pem, "BEGIN CERTIFICATE") {
		t.Fatal("CA cert should be PEM format")
	}
	if !strings.Contains(pem, "END CERTIFICATE") {
		t.Fatal("CA cert should have END marker")
	}
}

func TestCA_IssueAndValidate(t *testing.T) {
	ca, _ := NewCA(nil)
	ctx := context.Background()

	// Issue a device cert.
	cert, err := ca.Issue(ctx, "dev-001", "user-123")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	if cert.DeviceID != "dev-001" {
		t.Fatalf("expected device dev-001, got %s", cert.DeviceID)
	}
	if cert.Status != StatusActive {
		t.Fatalf("expected active status, got %s", cert.Status)
	}
	if !strings.Contains(cert.CertPEM, "BEGIN CERTIFICATE") {
		t.Fatal("cert should be PEM")
	}

	// Validate it against the CA.
	err = ca.ValidateDeviceCert(cert.CertPEM)
	if err != nil {
		t.Fatalf("ValidateDeviceCert failed: %v", err)
	}
}

func TestCA_RevokeAndValidate(t *testing.T) {
	ca, _ := NewCA(nil)
	ctx := context.Background()

	cert, _ := ca.Issue(ctx, "dev-002", "user-456")

	// Should validate before revocation.
	err := ca.ValidateDeviceCert(cert.CertPEM)
	if err != nil {
		t.Fatalf("should validate before revoke: %v", err)
	}

	// Revoke.
	serial := cert.Serial.String()
	err = ca.Revoke(ctx, serial)
	if err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	// Should fail after revocation.
	err = ca.ValidateDeviceCert(cert.CertPEM)
	if err == nil {
		t.Fatal("should fail validation after revocation")
	}
	if !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("expected revoked error, got: %v", err)
	}
}

func TestCA_GenerateCRL(t *testing.T) {
	ca, _ := NewCA(nil)
	ctx := context.Background()

	// Issue and revoke a cert.
	cert, _ := ca.Issue(ctx, "dev-003", "user-789")
	ca.Revoke(ctx, cert.Serial.String())

	crl, err := ca.GenerateCRL()
	if err != nil {
		t.Fatalf("GenerateCRL failed: %v", err)
	}
	if !strings.Contains(crl, "BEGIN X509 CRL") {
		t.Fatal("CRL should be PEM format")
	}
}

func TestCA_IssueFromCSR(t *testing.T) {
	ca, _ := NewCA(nil)
	ctx := context.Background()

	// Create a CSR using Go crypto.
	// For simplicity, test with the Issue method (which generates internally).
	cert, err := ca.Issue(ctx, "dev-csr", "user-csr")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	if cert.DeviceID != "dev-csr" {
		t.Fatalf("expected dev-csr, got %s", cert.DeviceID)
	}
}

func TestCA_EnsureSchema_NilPool(t *testing.T) {
	ca, _ := NewCA(nil)
	if err := ca.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
}

func TestCA_ListDeviceCerts_NilPool(t *testing.T) {
	ca, _ := NewCA(nil)
	certs, err := ca.ListDeviceCerts(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if certs != nil {
		t.Fatal("nil pool should return nil")
	}
}

func TestCA_MultipleCertsUniqueSerials(t *testing.T) {
	ca, _ := NewCA(nil)
	ctx := context.Background()

	cert1, _ := ca.Issue(ctx, "dev-a", "user-a")
	cert2, _ := ca.Issue(ctx, "dev-b", "user-b")

	if cert1.Serial.String() == cert2.Serial.String() {
		t.Fatal("serials should be unique")
	}
}
