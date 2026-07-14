package server

import (
	"crypto/x509"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/truststore"
)

// TrustStoreHandler handles trust store and certificate management endpoints.
type TrustStoreHandler struct {
	store *truststore.MemoryStore
	mtls  *truststore.MTLSConfig
}

// NewTrustStoreHandler creates a new handler with an in-memory store.
func NewTrustStoreHandler() *TrustStoreHandler {
	return &TrustStoreHandler{
		store: truststore.NewMemoryStore(),
		mtls: &truststore.MTLSConfig{
			RevocationCheck:  "none",
			FallbackToBearer: true,
		},
	}
}

// GetStore returns the underlying trust store for use by other components.
func (h *TrustStoreHandler) GetStore() *truststore.MemoryStore {
	return h.store
}

// --- Trust Store CA endpoints ---

// POST /api/v1/auth/trust-store/cas — upload a trusted CA certificate
// GET  /api/v1/auth/trust-store/cas — list all trusted CAs
// DELETE /api/v1/auth/trust-store/cas/{id} — remove a trusted CA
func (h *TrustStoreHandler) HandleTrustStoreCAs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.addCA(w, r)
	case http.MethodGet:
		h.listCAs(w, r)
	case http.MethodDelete:
		h.removeCA(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *TrustStoreHandler) addCA(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		PEMData string `json:"pem_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	uploadedBy := "admin"
		ca, err := h.store.AddCA(req.Name, req.PEMData, uploadedBy)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid CA certificate"})
			return
		}

		writeJSON(w, http.StatusCreated, ca)
	}

func (h *TrustStoreHandler) listCAs(w http.ResponseWriter, r *http.Request) {
	cas, err := h.store.ListCAs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list trust store CAs"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"trusted_cas": cas})
}

func (h *TrustStoreHandler) removeCA(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/trust-store/cas/")
	if id == "" || id == r.URL.Path {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "CA id is required"})
		return
	}

	if err := h.store.RemoveCA(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// --- Certificate Management endpoints ---

// POST   /api/v1/auth/certificates — upload a certificate
// GET    /api/v1/auth/certificates — list all certificates
// DELETE /api/v1/auth/certificates/{id} — revoke/remove a certificate
// POST   /api/v1/auth/certificates/csr — generate a CSR
// POST   /api/v1/auth/certificates/{id}/renew — renew a certificate
func (h *TrustStoreHandler) HandleCertificates(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// CSR generation
	if strings.HasSuffix(path, "/csr") && r.Method == http.MethodPost {
		h.generateCSR(w, r)
		return
	}

	// Renew certificate
	if strings.Contains(path, "/renew") && r.Method == http.MethodPost {
		h.renewCertificate(w, r)
		return
	}

	// Expiry tracker
	if strings.HasSuffix(path, "/expiry") && r.Method == http.MethodGet {
		h.certExpiryTracker(w, r)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.uploadCertificate(w, r)
	case http.MethodGet:
		if strings.TrimPrefix(path, "/api/v1/auth/certificates") != "" {
			h.getCertificate(w, r)
		} else {
			h.listCertificates(w, r)
		}
	case http.MethodDelete:
		h.removeCertificate(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *TrustStoreHandler) uploadCertificate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		PEMData    string `json:"pem_data"`
		KeyPEMData string `json:"key_pem_data"`
		AutoRenew  bool   `json:"auto_renew"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Type == "" {
		req.Type = "TLS"
	}

	cert, err := truststore.ParseCertificateFromPEM(req.Name, req.Type, req.PEMData, req.KeyPEMData)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	cert.AutoRenew = req.AutoRenew

	if err := h.store.AddCertificate(cert); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, cert)
}

func (h *TrustStoreHandler) listCertificates(w http.ResponseWriter, r *http.Request) {
	certs := h.store.ListCertificates()
	writeJSON(w, http.StatusOK, map[string]any{"certificates": certs})
}

func (h *TrustStoreHandler) getCertificate(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/certificates/")
	id = strings.TrimSuffix(id, "/")

	cert, err := h.store.GetCertificate(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cert)
}

func (h *TrustStoreHandler) removeCertificate(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/certificates/")

	if err := h.store.RemoveCertificate(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *TrustStoreHandler) renewCertificate(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/certificates/")
	id = strings.TrimSuffix(id, "/renew")

	cert, err := h.store.GetCertificate(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	// Generate a new self-signed cert as renewal (in production, this would
	// submit a CSR to an internal CA or ACME endpoint)
	newPEM, newKey, fp, err := truststore.GenerateSelfSignedCert(cert.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Update the certificate
	cert.PEMData = newPEM
	cert.KeyPEMData = newKey
	cert.Fingerprint = fp
	cert.ExpiryDate = time.Now().Add(365 * 24 * time.Hour).UTC()
	cert.DaysToExpiry = 365

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "renewed",
		"id":          cert.ID,
		"fingerprint": fp,
		"expiry_date": cert.ExpiryDate.Format(time.RFC3339),
	})
}

func (h *TrustStoreHandler) generateCSR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CommonName   string `json:"cn"`
		Organization string `json:"organization"`
		KeyType      string `json:"key_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.KeyType == "" {
		req.KeyType = "rsa"
	}

	csrPEM, keyPEM, err := truststore.GenerateCSR(req.CommonName, req.Organization, req.KeyType)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"csr":      csrPEM,
		"key_pem":  keyPEM,
	})
}

// --- Certificate Expiry Tracker ---

// GET /api/v1/auth/certificates/expiry — returns cert expiry summary
func (h *TrustStoreHandler) certExpiryTracker(w http.ResponseWriter, r *http.Request) {
	certs := h.store.ListCertificates()

	healthy := 0    // >90 days
	expiring := 0   // 30-90 days
	critical := 0   // 1-30 days
	expired := 0    // <=0 days

	for _, c := range certs {
		if c.DaysToExpiry <= 0 {
			expired++
		} else if c.DaysToExpiry <= 30 {
			critical++
		} else if c.DaysToExpiry <= 90 {
			expiring++
		} else {
			healthy++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"summary": map[string]int{
			"healthy":   healthy,
			"expiring":  expiring,
			"critical":  critical,
			"expired":   expired,
			"total":     len(certs),
		},
		"certs": certs,
	})
}

// --- mTLS Config endpoints ---

// GET /api/v1/auth/mtls/config — get mTLS configuration
// PUT /api/v1/auth/mtls/config — update mTLS configuration
func (h *TrustStoreHandler) HandleMTLSConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cas, _ := h.store.ListCAs()
		result := map[string]any{
			"require_mtls":             h.mtls.RequireMTLS,
			"trusted_ca_certs":         cas,
			"per_client_cert_binding":  h.mtls.PerClientCertBinding,
			"revocation_check":         h.mtls.RevocationCheck,
			"allow_self_signed":        h.mtls.AllowSelfSigned,
			"fallback_to_bearer":       h.mtls.FallbackToBearer,
		}
		writeJSON(w, http.StatusOK, result)

	case http.MethodPut, http.MethodPost:
		var req struct {
			RequireMTLS          *bool   `json:"require_mtls"`
			PerClientCertBinding *bool   `json:"per_client_cert_binding"`
			RevocationCheck      *string `json:"revocation_check"`
			AllowSelfSigned      *bool   `json:"allow_self_signed"`
			FallbackToBearer     *bool   `json:"fallback_to_bearer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if req.RequireMTLS != nil {
			h.mtls.RequireMTLS = *req.RequireMTLS
		}
		if req.PerClientCertBinding != nil {
			h.mtls.PerClientCertBinding = *req.PerClientCertBinding
		}
		if req.RevocationCheck != nil {
			h.mtls.RevocationCheck = *req.RevocationCheck
		}
		if req.AllowSelfSigned != nil {
			h.mtls.AllowSelfSigned = *req.AllowSelfSigned
		}
		if req.FallbackToBearer != nil {
			h.mtls.FallbackToBearer = *req.FallbackToBearer
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// --- Cert Verification endpoint ---

// POST /api/v1/auth/trust-store/verify — verify a certificate against the trust store
func (h *TrustStoreHandler) HandleVerifyCert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		PEMData string `json:"pem_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Parse the certificate
	block := pemDecodeSimple(req.PEMData)
	if block == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid PEM data"})
		return
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse certificate: " + err.Error()})
		return
	}

	// Get the trust pool
	pool, err := h.store.CertPool()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build cert pool"})
		return
	}

	// Verify
	opts := x509.VerifyOptions{
		Roots: pool,
	}
	chains, err := cert.Verify(opts)

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":       err == nil,
		"subject":     cert.Subject.CommonName,
		"issuer":      cert.Issuer.CommonName,
		"expiry":      cert.NotAfter.Format(time.RFC3339),
		"chain_count": len(chains),
		"error":       errToString(err),
	})
}
