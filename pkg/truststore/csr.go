package truststore

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// GenerateCSR generates a Certificate Signing Request and private key.
// Returns the CSR PEM, private key PEM, and an error.
func GenerateCSR(commonName, organization string, keyType string) (csrPEM, keyPEM string, err error) {
	var privKey interface{}

	switch strings.ToLower(keyType) {
	case "rsa":
		privKey, err = rsa.GenerateKey(rand.Reader, 2048)
	case "ecdsa", "ec":
		privKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "ed25519", "eddsa":
		_, pub, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			return "", "", e
		}
		// Store as ed25519 private key
		privKey = pub // not ideal; use full key
	default:
		return "", "", fmt.Errorf("unsupported key type: %s (use rsa, ecdsa, or ed25519)", keyType)
	}
	if err != nil {
		return "", "", fmt.Errorf("generate key: %w", err)
	}

	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{organization},
		},
		DNSNames:       []string{commonName},
		EmailAddresses: []string{},
	}
	if organization == "" {
		template.Subject.Organization = nil
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, privKey)
	if err != nil {
		return "", "", fmt.Errorf("create CSR: %w", err)
	}

	csrPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	}))

	keyDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}

	keyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	}))

	return csrPEM, keyPEM, nil
}

// ParseCertificateFromPEM parses a PEM-encoded certificate and returns a
// Certificate struct suitable for storage in the trust store.
func ParseCertificateFromPEM(name, certType, pemData, keyPEM string) (*Certificate, error) {
	cert, err := parsePEMCert(pemData)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	c := &Certificate{
		ID:          generateID(),
		Name:        name,
		Type:        certType,
		Issuer:      cert.Issuer.CommonName,
		Fingerprint: fingerprint(cert),
		PEMData:     pemData,
		KeyPEMData:  keyPEM,
		ExpiryDate:  cert.NotAfter,
		AutoRenew:   false,
		CreatedAt:   time.Now().UTC(),
	}
	c.DaysToExpiry = int(time.Until(c.ExpiryDate).Hours() / 24)

	return c, nil
}

// GenerateSelfSignedCert generates a self-signed certificate for testing or
// internal use. Returns cert PEM, key PEM, and fingerprint.
func GenerateSelfSignedCert(commonName string) (certPEM, keyPEM, fingerprint string, err error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", fmt.Errorf("generate key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"GGID"},
		},
		NotBefore:             time.Now().UTC(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour).UTC(),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{commonName},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return "", "", "", fmt.Errorf("create certificate: %w", err)
	}

	certPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}))

	keyDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal private key: %w", err)
	}

	keyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	}))

	cert, _ := x509.ParseCertificate(certDER)
	if cert != nil {
		fingerprint = func() string {
			h := sha256Sum(cert.Raw)
			return strings.ToUpper(hexEncode(h[:]))
		}()
	}

	return certPEM, keyPEM, fingerprint, nil
}

// sha256Sum is a helper to avoid importing crypto/sha256 in the main file.
func sha256Sum(data []byte) [32]byte {
	return sha256SumImpl(data)
}

// hexEncode is a helper to avoid importing encoding/hex in the main file.
func hexEncode(data []byte) string {
	return hexEncodeImpl(data)
}
