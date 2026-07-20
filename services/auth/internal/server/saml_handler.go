package server

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	ggidSAML "github.com/ggid/ggid/pkg/saml"
	"github.com/google/uuid"
)

// --- SAML IdP Endpoints ---

// GET /saml/metadata — returns SP metadata XML for IdP configuration
func (h *Handler) handleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	// Build SP metadata dynamically
	entityID := "https://ggid.iot2.win/saml/metadata"
	acsURL := "https://ggid.iot2.win/saml/acs"
	sloURL := "https://ggid.iot2.win/saml/slo"

	metadata := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService index="0" isDefault="true" Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s"/>
    <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="%s"/>
  </SPSSODescriptor>
</EntityDescriptor>`, entityID, acsURL, sloURL)

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(metadata))
}

// GET /saml/sso — SP-initiated SSO: redirect to GGID login
func (h *Handler) handleSAMLSSO(w http.ResponseWriter, r *http.Request) {
	relayState := r.URL.Query().Get("RelayState")
	if relayState == "" {
		relayState = r.URL.Query().Get("relay_state")
	}
	if relayState == "" {
		relayState = "/"
	}

	// Redirect to login page with SAML relay state
	loginURL := "/login?saml=true&relay_state=" + relayState
	http.Redirect(w, r, loginURL, http.StatusFound)
}

// POST /saml/acs — Assertion Consumer Service: receive SAML Response from IdP
func (h *Handler) handleSAMLACS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	samlResponseB64 := r.FormValue("SAMLResponse")
	relayState := r.FormValue("RelayState")
	if relayState == "" {
		relayState = "/"
	}

	if samlResponseB64 == "" {
		writeError(w, http.StatusBadRequest, "missing SAMLResponse")
		return
	}

	// Decode SAML response
	responseXML, err := base64.StdEncoding.DecodeString(samlResponseB64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid base64 encoding")
		return
	}

	// Get IdP certificate from sys_config
	idpCert, err := h.getIdPCertificate(r)
	if err != nil {
		slog.Error("SAML ACS: cert error", "error", err)
		writeError(w, http.StatusServiceUnavailable, "IdP certificate not configured")
		return
	}

	// Parse and verify SAML assertion
	assertion, err := ggidSAML.VerifySignedAssertion(responseXML, idpCert)
	if err != nil {
		slog.Error("SAML ACS: assertion verification failed", "error", err)
		writeError(w, http.StatusUnauthorized, "SAML assertion verification failed")
		return
	}

	// Validate conditions (time window, audience)
	if err := assertion.ValidateConditions(); err != nil {
		slog.Error("SAML ACS: conditions validation failed", "error", err)
		writeError(w, http.StatusForbidden, "SAML assertion conditions not met")
		return
	}

	// Extract user attributes
	attrs := ggidSAML.ExtractAttributes(assertion)
	userEmail := ggidSAML.GetAttribute(assertion, "email")
	if userEmail == "" {
		userEmail = ggidSAML.GetAttribute(assertion, "EmailAddress")
	}
	if userEmail == "" {
		// Subject is a struct, try to get NameID value
		subjectStr := fmt.Sprintf("%v", assertion.Subject)
		if subjectStr != "" && subjectStr != "{<nil>}" {
			userEmail = subjectStr
		}
	}

	userName := ggidSAML.GetAttribute(assertion, "name")
	if userName == "" {
		userName = ggidSAML.GetAttribute(assertion, "cn")
	}

	// Issue JWT for the authenticated user
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	// Generate session token (reuse auth token mechanism)
	sessionID := uuid.New().String()
	slog.Info("SAML ACS: assertion verified",
		"email", userEmail, "name", userName, "session", sessionID,
		"attributes", attrs, "relay_state", relayState)

	// Redirect back to relay state with session info
	// In production, this would issue a real JWT and set cookie
	redirectURL := relayState
	if strings.Contains(redirectURL, "?") {
		redirectURL += "&"
	} else {
		redirectURL += "?"
	}
	redirectURL += fmt.Sprintf("saml_session=%s&email=%s&name=%s",
		sessionID,
		userEmail,
		userName,
	)

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// GET /saml/config — get SAML configuration from sys_config
func (h *Handler) handleSAMLConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.getSAMLConfig(w, r)
		return
	}
	if r.Method == http.MethodPut {
		h.putSAMLConfig(w, r)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (h *Handler) getSAMLConfig(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		writeJSON(w, http.StatusOK, map[string]any{"saml_config": map[string]any{}})
		return
	}

	var configJSON string
	err := h.pool.QueryRow(r.Context(),
		`SELECT value::text FROM sys_config WHERE key = 'saml_config'`).Scan(&configJSON)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"saml_config": map[string]any{}})
		return
	}

	var cfg any
	json.Unmarshal([]byte(configJSON), &cfg)
	writeJSON(w, http.StatusOK, map[string]any{"saml_config": cfg})
}

func (h *Handler) putSAMLConfig(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	cfg, ok := req["saml_config"]
	if !ok {
		// Allow direct key-value config
		cfg = req
	}

	configJSON, _ := json.Marshal(cfg)
	updatedBy := r.Header.Get("X-User-ID")

	if h.pool != nil {
		var uid *uuid.UUID
		if u, err := uuid.Parse(updatedBy); err == nil {
			uid = &u
		}
		if uid != nil {
			_, err := h.pool.Exec(r.Context(), `
				INSERT INTO sys_config (key, value, updated_by)
				VALUES ('saml_config', $1, $2)
				ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW(), updated_by = $2`,
				configJSON, *uid)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to save config")
				return
			}
		} else {
			_, err := h.pool.Exec(r.Context(), `
				INSERT INTO sys_config (key, value)
				VALUES ('saml_config', $1)
				ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW()`,
				configJSON)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to save config")
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "saml_config": cfg})
}

// getIdPCertificate reads IdP certificate from sys_config
func (h *Handler) getIdPCertificate(r *http.Request) (*x509.Certificate, error) {
	if h.pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	var configJSON string
	err := h.pool.QueryRow(r.Context(),
		`SELECT value::text FROM sys_config WHERE key = 'saml_config'`).Scan(&configJSON)
	if err != nil {
		return nil, fmt.Errorf("saml_config not found in sys_config")
	}

	var cfg struct {
		IDPCert string `json:"idp_cert"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("invalid saml_config JSON")
	}

	if cfg.IDPCert == "" {
		return nil, fmt.Errorf("idp_cert not configured")
	}

	// Use existing pemDecode from helpers.go
	block := pemDecode(cfg.IDPCert)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM certificate")
	}

	return x509.ParseCertificate(block.Bytes)
}
