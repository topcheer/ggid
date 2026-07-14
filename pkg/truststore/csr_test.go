package truststore

import (
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

func TestGenerateCSR_RSA_New(t *testing.T) {
	csrPEM, keyPEM, err := GenerateCSR("test.example.com", "Test Org", "rsa")
	if err != nil {
		t.Fatalf("GenerateCSR rsa: %v", err)
	}
	if !strings.HasPrefix(csrPEM, "-----BEGIN CERTIFICATE REQUEST-----") {
		t.Error("CSR PEM header missing")
	}
	if !strings.HasPrefix(keyPEM, "-----BEGIN") {
		t.Error("Key PEM header missing")
	}

	// Parse CSR
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil {
		t.Fatal("failed to decode CSR PEM")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		t.Fatalf("parse CSR: %v", err)
	}
	if csr.Subject.CommonName != "test.example.com" {
		t.Errorf("CN = %s, want test.example.com", csr.Subject.CommonName)
	}
	if len(csr.Subject.Organization) == 0 || csr.Subject.Organization[0] != "Test Org" {
		t.Errorf("Org = %v, want [Test Org]", csr.Subject.Organization)
	}
}

func TestGenerateCSR_ECDSA_New(t *testing.T) {
	csrPEM, keyPEM, err := GenerateCSR("ec.example.com", "EC Org", "ecdsa")
	if err != nil {
		t.Fatalf("GenerateCSR ecdsa: %v", err)
	}
	if csrPEM == "" || keyPEM == "" {
		t.Error("empty PEM returned")
	}

	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil {
		t.Fatal("failed to decode CSR PEM")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		t.Fatalf("parse CSR: %v", err)
	}
	if csr.Subject.CommonName != "ec.example.com" {
		t.Errorf("CN = %s, want ec.example.com", csr.Subject.CommonName)
	}
}

func TestGenerateCSR_Ed25519_New(t *testing.T) {
	csrPEM, keyPEM, err := GenerateCSR("ed.example.com", "ED Org", "ed25519")
	if err != nil {
		t.Fatalf("GenerateCSR ed25519: %v", err)
	}
	if csrPEM == "" || keyPEM == "" {
		t.Error("empty PEM returned")
	}
}

func TestGenerateCSR_Unsupported_New(t *testing.T) {
	_, _, err := GenerateCSR("test.example.com", "Org", "dsa")
	if err == nil {
		t.Error("expected error for unsupported key type dsa")
	}
	if !strings.Contains(err.Error(), "unsupported key type") {
		t.Errorf("error message = %s, want 'unsupported key type'", err.Error())
	}
}

func TestGenerateCSR_AltNames_New(t *testing.T) {
	csrPEM, _, err := GenerateCSR("alt.example.com", "Alt Org", "rsa")
	if err != nil {
		t.Fatalf("GenerateCSR: %v", err)
	}
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil {
		t.Fatal("failed to decode CSR PEM")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		t.Fatalf("parse CSR: %v", err)
	}
	// Verify signature
	if err := csr.CheckSignature(); err != nil {
		t.Errorf("CSR signature invalid: %v", err)
	}
}
