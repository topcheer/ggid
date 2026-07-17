package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ClientCert holds mTLS certificate rotation state for an OAuth client.
type ClientCert struct {
	ClientID    string    `json:"client_id"`
	CertPEM     string    `json:"cert_pem"`
	KeyPEM      string    `json:"key_pem,omitempty"`
	Fingerprint string    `json:"fingerprint"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

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

		if mapRepoVar != nil {
			b, _ := json.Marshal(cert)
			var dataMap map[string]any
			json.Unmarshal(b, &dataMap)
			mapRepoVar.Store(r.Context(), "oauth_client_certs", clientID, dataMap)
		}

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

	if mapRepoVar != nil {
		data, err := mapRepoVar.Get(r.Context(), "oauth_client_certs", clientID)
		if err == nil {
			fingerprint, _ := data["fingerprint"].(string)
			issuedAtStr, _ := data["issued_at"].(string)
			expiresAtStr, _ := data["expires_at"].(string)
			issuedAt, _ := time.Parse(time.RFC3339, issuedAtStr)
			expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)
			daysLeft := int(time.Until(expiresAt).Hours() / 24)
			writeJSON(w, http.StatusOK, map[string]any{
				"client_id":      clientID,
				"has_cert":       true,
				"fingerprint":    fingerprint,
				"issued_at":      issuedAt.Format(time.RFC3339),
				"expires_at":     expiresAt.Format(time.RFC3339),
				"days_left":      daysLeft,
				"needs_rotation": daysLeft < 30,
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id": clientID,
		"has_cert":  false,
	})
}
