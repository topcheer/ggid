package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// Certificate represents a certificate record for the frontend.
type Certificate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // SAML, OAuth, JWT, TLS
	Issuer      string `json:"issuer"`
	Subject     string `json:"subject"`
	Domain      string `json:"domain,omitempty"`
	Expiry      string `json:"expiry"`
	Fingerprint string `json:"fingerprint"`
	Status      string `json:"status"` // active, expiring, expired
}

// GET /api/v1/certificates — list certificates
// POST /api/v1/certificates/sign — sign/create a new certificate
func (h *Handler) handleCertificatesV2(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/v1/certificates" && r.Method == http.MethodGet:
		certs := []Certificate{
			{ID: "cert-1", Name: "SAML Signing", Type: "SAML", Issuer: "ggid-ca", Subject: "ggid-auth", Expiry: "2026-01-15", Fingerprint: "AB:CD:EF:01", Status: "active"},
			{ID: "cert-2", Name: "JWT Signing", Type: "JWT", Issuer: "ggid-ca", Subject: "ggid-auth", Expiry: "2025-12-01", Fingerprint: "12:34:56:78", Status: "active"},
		}
		writeJSON(w, http.StatusOK, certs)

	case r.URL.Path == "/api/v1/certificates/sign" && r.Method == http.MethodPost:
		var req struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			Domain string `json:"domain"`
			CN     string `json:"cn"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Type == "" {
			req.Type = "TLS"
		}
		if req.CN == "" {
			req.CN = req.Name
		}

		// Generate a self-signed certificate
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to generate key")
			return
		}

		template := x509.Certificate{
			SerialNumber: big.NewInt(time.Now().UnixNano()),
			Subject: pkix.Name{
				CommonName: req.CN,
			},
			NotBefore: time.Now(),
			NotAfter:  time.Now().Add(365 * 24 * time.Hour),
			KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage: []x509.ExtKeyUsage{
				x509.ExtKeyUsageServerAuth,
				x509.ExtKeyUsageClientAuth,
			},
			DNSNames: []string{req.Domain},
		}

		certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create certificate")
			return
		}

		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

		cert := Certificate{
			ID:          fmt.Sprintf("cert-%d", time.Now().UnixNano()),
			Name:        req.Name,
			Type:        req.Type,
			Issuer:      "ggid-self-signed",
			Subject:     req.CN,
			Domain:      req.Domain,
			Expiry:      template.NotAfter.Format("2006-01-02"),
			Fingerprint: fmt.Sprintf("%02X", certBytes[len(certBytes)-4:]),
			Status:      "active",
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"certificate":   cert,
			"cert_pem":      string(certPEM),
			"private_key":   string(keyPEM),
		})

	case strings.HasPrefix(r.URL.Path, "/api/v1/certificates/") && r.Method == http.MethodDelete:
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
