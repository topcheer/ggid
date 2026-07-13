// Package truststore provides a centralized certificate authority (CA) trust
// store for the GGID IAM platform. It allows administrators to upload, list,
// and revoke trusted CA certificates that are used by all outbound TLS
// connections (SMTP, LDAP, SIEM forwarder, SAML IdP, OAuth provider connections).
//
// The store maintains an in-memory cache of parsed certificates and exposes
// a CertPool() method that returns a *x509.CertPool suitable for use in
// crypto/tls.Config.RootCAs.
package truststore

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TrustedCA represents a trusted certificate authority.
type TrustedCA struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Fingerprint string    `json:"fingerprint"` // SHA-256 of the DER-encoded certificate
	Subject     string    `json:"subject"`     // CN from the certificate subject
	Issuer      string    `json:"issuer"`      // CN from the certificate issuer
	PEMData     string    `json:"pem_data"`    // PEM-encoded certificate
	ExpiryDate  time.Time `json:"expiry_date"`
	UploadedAt  time.Time `json:"uploaded_at"`
	UploadedBy  string    `json:"uploaded_by"`
}

// Certificate represents a managed certificate (TLS, signing, JWT).
type Certificate struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"` // "TLS", "signing", "JWT"
	Issuer       string    `json:"issuer"`
	Fingerprint  string    `json:"fingerprint"`
	PEMData      string    `json:"pem_data"`      // certificate PEM
	KeyPEMData   string    `json:"-"`              // private key PEM (never exposed in JSON)
	ExpiryDate   time.Time `json:"expiry_date"`
	AutoRenew    bool      `json:"auto_renew"`
	DaysToExpiry int       `json:"days_to_expiry"`
	CreatedAt    time.Time `json:"created_at"`
}

// MTLSConfig holds the mTLS authentication configuration.
type MTLSConfig struct {
	RequireMTLS          bool         `json:"require_mtls"`
	TrustedCACerts       []TrustedCA  `json:"trusted_ca_certs"`
	PerClientCertBinding bool         `json:"per_client_cert_binding"`
	RevocationCheck      string       `json:"revocation_check"` // "none", "CRL", "OCSP", "both"
	AllowSelfSigned      bool         `json:"allow_self_signed"`
	FallbackToBearer     bool         `json:"fallback_to_bearer"`
}

// Store is the interface for a CA trust store.
type Store interface {
	// AddCA adds a trusted CA certificate from PEM data.
	AddCA(name, pemData, uploadedBy string) (*TrustedCA, error)
	// RemoveCA removes a trusted CA by ID.
	RemoveCA(id string) error
	// ListCAs returns all trusted CAs.
	ListCAs() ([]*TrustedCA, error)
	// GetCA returns a specific trusted CA by ID.
	GetCA(id string) (*TrustedCA, error)
	// CertPool returns an x509.CertPool containing all trusted CAs.
	// Returns the system pool if no custom CAs are configured.
	CertPool() (*x509.CertPool, error)
	// HasCAs returns true if the store contains any custom CAs.
	HasCAs() bool
}

// MemoryStore is an in-memory implementation of Store.
// It is safe for concurrent use.
type MemoryStore struct {
	mu   sync.RWMutex
	cas  map[string]*TrustedCA
	cert map[string]*Certificate // managed certificates
	mu2  sync.RWMutex
}

// NewMemoryStore creates a new in-memory trust store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		cas:  make(map[string]*TrustedCA),
		cert: make(map[string]*Certificate),
	}
}

// AddCA parses PEM-encoded certificate data and stores it as a trusted CA.
func (s *MemoryStore) AddCA(name, pemData, uploadedBy string) (*TrustedCA, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if pemData == "" {
		return nil, fmt.Errorf("PEM data is required")
	}

	cert, err := parsePEMCert(pemData)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	fp := fingerprint(cert)
	id := generateID()

	// Check for duplicate fingerprints
	s.mu.RLock()
	for _, existing := range s.cas {
		if existing.Fingerprint == fp {
			s.mu.RUnlock()
			return nil, fmt.Errorf("CA certificate with fingerprint %s already exists (id: %s, name: %s)",
				fp, existing.ID, existing.Name)
		}
	}
	s.mu.RUnlock()

	ca := &TrustedCA{
		ID:          id,
		Name:        name,
		Fingerprint: fp,
		Subject:     cert.Subject.CommonName,
		Issuer:      cert.Issuer.CommonName,
		PEMData:     pemData,
		ExpiryDate:  cert.NotAfter,
		UploadedAt:  time.Now().UTC(),
		UploadedBy:  uploadedBy,
	}

	s.mu.Lock()
	s.cas[id] = ca
	s.mu.Unlock()

	return ca, nil
}

// RemoveCA removes a trusted CA by ID.
func (s *MemoryStore) RemoveCA(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cas[id]; !ok {
		return fmt.Errorf("CA with id %s not found", id)
	}
	delete(s.cas, id)
	return nil
}

// ListCAs returns all trusted CAs.
func (s *MemoryStore) ListCAs() ([]*TrustedCA, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*TrustedCA, 0, len(s.cas))
	for _, ca := range s.cas {
		result = append(result, ca)
	}
	return result, nil
}

// GetCA returns a specific trusted CA by ID.
func (s *MemoryStore) GetCA(id string) (*TrustedCA, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ca, ok := s.cas[id]
	if !ok {
		return nil, fmt.Errorf("CA with id %s not found", id)
	}
	return ca, nil
}

// CertPool returns an x509.CertPool containing all trusted CAs.
// If no custom CAs are configured, returns the system root pool.
func (s *MemoryStore) CertPool() (*x509.CertPool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.cas) == 0 {
		return x509.SystemCertPool()
	}

	// Start with system roots, then add custom CAs
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}

	for _, ca := range s.cas {
		if !pool.AppendCertsFromPEM([]byte(ca.PEMData)) {
			return nil, fmt.Errorf("failed to add CA %s to pool", ca.Name)
		}
	}

	return pool, nil
}

// HasCAs returns true if the store contains any custom CAs.
func (s *MemoryStore) HasCAs() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cas) > 0
}

// --- Certificate management (managed certs, not CAs) ---

// AddCertificate stores a managed certificate.
func (s *MemoryStore) AddCertificate(cert *Certificate) error {
	if cert.ID == "" {
		cert.ID = generateID()
	}
	if cert.CreatedAt.IsZero() {
		cert.CreatedAt = time.Now().UTC()
	}
	s.mu2.Lock()
	defer s.mu2.Unlock()
	s.cert[cert.ID] = cert
	return nil
}

// RemoveCertificate removes a managed certificate by ID.
func (s *MemoryStore) RemoveCertificate(id string) error {
	s.mu2.Lock()
	defer s.mu2.Unlock()
	if _, ok := s.cert[id]; !ok {
		return fmt.Errorf("certificate with id %s not found", id)
	}
	delete(s.cert, id)
	return nil
}

// ListCertificates returns all managed certificates.
func (s *MemoryStore) ListCertificates() []*Certificate {
	s.mu2.RLock()
	defer s.mu2.RUnlock()
	result := make([]*Certificate, 0, len(s.cert))
	for _, c := range s.cert {
		c.DaysToExpiry = int(time.Until(c.ExpiryDate).Hours() / 24)
		result = append(result, c)
	}
	return result
}

// GetCertificate returns a specific managed certificate by ID.
func (s *MemoryStore) GetCertificate(id string) (*Certificate, error) {
	s.mu2.RLock()
	defer s.mu2.RUnlock()
	c, ok := s.cert[id]
	if !ok {
		return nil, fmt.Errorf("certificate with id %s not found", id)
	}
	c.DaysToExpiry = int(time.Until(c.ExpiryDate).Hours() / 24)
	return c, nil
}

// --- Helpers ---

// parsePEMCert parses a PEM-encoded certificate and returns the first cert.
func parsePEMCert(pemData string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM data — ensure input is a valid PEM certificate")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("expected PEM block type CERTIFICATE, got %s", block.Type)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse X.509 certificate: %w", err)
	}
	return cert, nil
}

// fingerprint returns the SHA-256 fingerprint of a certificate.
func fingerprint(cert *x509.Certificate) string {
	h := sha256.Sum256(cert.Raw)
	return strings.ToUpper(hex.EncodeToString(h[:]))
}

// generateID generates a short unique ID.
func generateID() string {
	return fmt.Sprintf("ca-%s", time.Now().UTC().Format("20060102") + "-" +
		hex.EncodeToString(randomBytes(4)))
}

// randomBytes returns n random bytes.
func randomBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> uint(i*8))
	}
	return b
}
