package scep

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CertificateStatus represents the lifecycle state of a device cert.
type CertificateStatus string

const (
	StatusActive   CertificateStatus = "active"
	StatusRevoked  CertificateStatus = "revoked"
	StatusExpired  CertificateStatus = "expired"
)

// DeviceCertificate is a record of an issued device certificate.
type DeviceCertificate struct {
	ID         string            `json:"id"`
	DeviceID   string            `json:"device_id"`
	UserID     string            `json:"user_id"`
	Serial     *big.Int          `json:"serial"`
	CertPEM    string            `json:"cert_pem"`
	Status     CertificateStatus `json:"status"`
	IssuedAt   time.Time         `json:"issued_at"`
	ExpiresAt  time.Time         `json:"expires_at"`
	RevokedAt  *time.Time        `json:"revoked_at,omitempty"`
}

// CA is the internal Certificate Authority for device identity certs.
type CA struct {
	key       *ecdsa.PrivateKey
	cert      *x509.Certificate
	certDER   []byte
	certPEM   string
	pool      *pgxpool.Pool
	mu        sync.Mutex
	serial    int64
	revoked   map[string]bool // serial hex string → revoked
	crlMu     sync.RWMutex
}

// NewCA generates a new self-signed ECDSA P-256 root CA.
func NewCA(pool *pgxpool.Pool) (*CA, error) {
	// Generate ECDSA P-256 key pair.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate CA keypair: %w", err)
	}

	// Create self-signed root certificate.
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "GGID Device CA",
			Organization: []string{"GGID"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10-year CA
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0, // only sign leaf certs
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, fmt.Errorf("create CA cert: %w", err)
	}

	caCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	certPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}))

	return &CA{
		key:     privKey,
		cert:    caCert,
		certDER: certDER,
		certPEM: certPEM,
		pool:    pool,
		revoked: make(map[string]bool),
	}, nil
}

// EnsureSchema creates the device_certificates table.
func (ca *CA) EnsureSchema(ctx context.Context) error {
	if ca.pool == nil {
		return nil
	}
	_, err := ca.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS device_certificates (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			user_id TEXT,
			serial TEXT NOT NULL,
			cert_pem TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at TIMESTAMPTZ NOT NULL,
			revoked_at TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_device_certs_device ON device_certificates(device_id);
		CREATE INDEX IF NOT EXISTS idx_device_certs_serial ON device_certificates(serial);
		CREATE INDEX IF NOT EXISTS idx_device_certs_status ON device_certificates(status, expires_at);
	`)
	return err
}

// CACertPEM returns the CA certificate in PEM format.
func (ca *CA) CACertPEM() string {
	return ca.certPEM
}

// IssueFromCSR signs a CSR and issues a device certificate (30-day TTL).
func (ca *CA) IssueFromCSR(ctx context.Context, deviceID, userID string, csrPEM string) (*DeviceCertificate, error) {
	// Parse CSR.
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return nil, fmt.Errorf("invalid CSR PEM")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CSR: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("CSR signature invalid: %w", err)
	}

	// Issue cert.
	return ca.signCertificate(ctx, deviceID, userID, csr)
}

// Issue generates a keypair and issues a cert directly (for testing/auto-enrollment).
func (ca *CA) Issue(ctx context.Context, deviceID, userID string) (*DeviceCertificate, error) {
	// Generate device keypair.
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate device key: %w", err)
	}

	// Create a minimal CSR-like template.
	csr := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: deviceID,
		},
	}

	// Use the template directly for signing.
	return ca.signWithPublicKey(ctx, deviceID, userID, csr.Subject.CommonName, &deviceKey.PublicKey)
}

// signCertificate signs a CSR into a device certificate.
func (ca *CA) signCertificate(ctx context.Context, deviceID, userID string, csr *x509.CertificateRequest) (*DeviceCertificate, error) {
	return ca.signWithPublicKey(ctx, deviceID, userID, csr.Subject.CommonName, csr.PublicKey)
}

// signWithPublicKey creates and signs a leaf certificate.
func (ca *CA) signWithPublicKey(ctx context.Context, deviceID, userID, cn string, pubKey any) (*DeviceCertificate, error) {
	ca.mu.Lock()
	ca.serial++
	serialNum := big.NewInt(ca.serial)
	ca.mu.Unlock()

	now := time.Now()
	ttl := 30 * 24 * time.Hour // 30 days

	template := &x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:   now.Add(-5 * time.Minute),
		NotAfter:    now.Add(ttl),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		DNSNames:    []string{cn},
	}

	// Add SAN for user_id if provided.
	if userID != "" {
		template.EmailAddresses = []string{userID}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, pubKey, ca.key)
	if err != nil {
		return nil, fmt.Errorf("create device cert: %w", err)
	}

	certPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}))

	deviceCert := &DeviceCertificate{
		ID:        fmt.Sprintf("dc-%s-%d", deviceID, serialNum.Int64()),
		DeviceID:  deviceID,
		UserID:    userID,
		Serial:    serialNum,
		CertPEM:   certPEM,
		Status:    StatusActive,
		IssuedAt:  now,
		ExpiresAt: now.Add(ttl),
	}

	// Persist to PG.
	if ca.pool != nil {
		_, err := ca.pool.Exec(ctx,
			`INSERT INTO device_certificates (id, device_id, user_id, serial, cert_pem, status, issued_at, expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			deviceCert.ID, deviceCert.DeviceID, deviceCert.UserID, serialNum.String(), certPEM, string(StatusActive), now, deviceCert.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("persist device cert: %w", err)
		}
	}

	return deviceCert, nil
}

// Revoke marks a certificate as revoked.
func (ca *CA) Revoke(ctx context.Context, serial string) error {
	ca.crlMu.Lock()
	ca.revoked[serial] = true
	ca.crlMu.Unlock()

	if ca.pool != nil {
		_, err := ca.pool.Exec(ctx,
			`UPDATE device_certificates SET status = 'revoked', revoked_at = now() WHERE serial = $1`,
			serial)
		return err
	}
	return nil
}

// IsRevoked checks if a serial number is revoked.
func (ca *CA) IsRevoked(serial string) bool {
	ca.crlMu.RLock()
	defer ca.crlMu.RUnlock()
	return ca.revoked[serial]
}

// GenerateCRL creates a Certificate Revocation List in PEM format.
func (ca *CA) GenerateCRL() (string, error) {
	ca.crlMu.RLock()
	defer ca.crlMu.RUnlock()

	now := time.Now()
	template := &x509.RevocationList{
		Number:     big.NewInt(time.Now().Unix()),
		ThisUpdate: now,
		NextUpdate: now.Add(24 * time.Hour),
	}

	crlDER, err := x509.CreateRevocationList(rand.Reader, template, ca.cert, ca.key)
	if err != nil {
		return "", fmt.Errorf("create CRL: %w", err)
	}

	crlPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "X509 CRL",
		Bytes: crlDER,
	}))

	return crlPEM, nil
}

// ListDeviceCerts returns all certs for a device.
func (ca *CA) ListDeviceCerts(ctx context.Context, deviceID string) ([]DeviceCertificate, error) {
	if ca.pool == nil {
		return nil, nil
	}
	rows, err := ca.pool.Query(ctx,
		`SELECT id, device_id, user_id, serial, cert_pem, status, issued_at, expires_at, revoked_at
		FROM device_certificates WHERE device_id = $1 ORDER BY issued_at DESC`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []DeviceCertificate
	for rows.Next() {
		var c DeviceCertificate
		var serialStr string
		if err := rows.Scan(&c.ID, &c.DeviceID, &c.UserID, &serialStr, &c.CertPEM, &c.Status, &c.IssuedAt, &c.ExpiresAt, &c.RevokedAt); err != nil {
			continue
		}
		c.Serial, _ = new(big.Int).SetString(serialStr, 10)
		certs = append(certs, c)
	}
	return certs, nil
}

// ValidateDeviceCert verifies a device certificate against the CA.
func (ca *CA) ValidateDeviceCert(certPEM string) error {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return fmt.Errorf("invalid PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse cert: %w", err)
	}

	// Verify against CA.
	roots := x509.NewCertPool()
	roots.AddCert(ca.cert)

	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return fmt.Errorf("verify cert: %w", err)
	}

	// Check revocation.
	if ca.IsRevoked(cert.SerialNumber.String()) {
		return fmt.Errorf("certificate %s is revoked", cert.SerialNumber)
	}

	return nil
}
