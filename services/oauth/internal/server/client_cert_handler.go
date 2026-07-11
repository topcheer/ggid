package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ClientCert holds mTLS certificate rotation state for an OAuth client.
type ClientCert struct {
	ClientID     string    `json:"client_id"`
	CertPEM      string    `json:"cert_pem"`
	KeyPEM       string    `json:"key_pem,omitempty"`
	Fingerprint  string    `json:"fingerprint"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

var (
	clientCertMu sync.RWMutex
	clientCerts  = make(map[string]*ClientCert)
)

// POST /api/v1/oauth/clients/{id}/rotate-cert — generate new mTLS cert for client.
// GET /api/v1/oauth/clients/{id}/cert-status — check cert status.
func handleClientCert(w http.ResponseWriter, r *http.Request) {
	// Extract client_id from path
	clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	clientID = strings.TrimSuffix(clientID, "/rotate-cert")
	clientID = strings.TrimSuffix(clientID, "/cert-status")

	if clientID == "" || strings.Contains(clientID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id is required"})
		return
	}

	if strings.HasSuffix(r.URL.Path, "/rotate-cert") {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}

		// Generate ECDSA P-256 key pair for mTLS
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "key generation failed"})
			return
		}

		keyDER, _ := x509.MarshalECPrivateKey(privKey)
		keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))

		pubDER, _ := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
		pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

		fp := uuid.New().String()[:16]
		now := time.Now().UTC()
		expiry := now.Add(90 * 24 * time.Hour)

		cert := &ClientCert{
			ClientID:    clientID,
			CertPEM:     pubPEM,
			Fingerprint: fp,
			IssuedAt:    now,
			ExpiresAt:   expiry,
		}

		clientCertMu.Lock()
		clientCerts[clientID] = cert
		clientCertMu.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "rotated",
			"client_id":   clientID,
			"cert_pem":    pubPEM,
			"key_pem":     keyPEM,
			"fingerprint": fp,
			"issued_at":   now.Format(time.RFC3339),
			"expires_at":  expiry.Format(time.RFC3339),
		})
		return
	}

	// cert-status
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	clientCertMu.RLock()
	cert, ok := clientCerts[clientID]
	clientCertMu.RUnlock()
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"client_id":    clientID,
			"has_cert":     false,
		})
		return
	}

	daysLeft := int(time.Until(cert.ExpiresAt).Hours() / 24)
	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":    clientID,
		"has_cert":     true,
		"fingerprint":  cert.Fingerprint,
		"issued_at":    cert.IssuedAt.Format(time.RFC3339),
		"expires_at":   cert.ExpiresAt.Format(time.RFC3339),
		"days_left":    daysLeft,
		"needs_rotation": daysLeft < 30,
	})
}
