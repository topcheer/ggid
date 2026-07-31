package server

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	ggidSAML "github.com/ggid/ggid/pkg/saml"
	"github.com/google/uuid"
)

// --- SAML IdP Endpoints ---

// publicURIFromRequest derives the public base URL from the request or PUBLIC_URL env.
func publicURIFromRequest(r *http.Request) string {
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL != "" {
		return strings.TrimRight(publicURL, "/")
	}
	scheme := "https"
	if r.TLS == nil && r.Host != "" && !strings.Contains(r.Host, "ggid.") {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

// GET /saml/metadata — returns SP metadata XML for IdP configuration
func (h *Handler) handleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	// Build SP metadata dynamically using the request's public URL
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		// Fallback: construct from request host
		scheme := "https"
		if r.TLS == nil && r.Host != "" && !strings.Contains(r.Host, "ggid.") {
			scheme = "http"
		}
		publicURL = scheme + "://" + r.Host
	}
	entityID := publicURL + "/saml/metadata"
	acsURL := publicURL + "/saml/acs"
	sloURL := publicURL + "/saml/slo"

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
	// SECURITY: Validate RelayState is a relative path (prevent open redirect + PII leak).
	if relayState == "" || !strings.HasPrefix(relayState, "/") || strings.HasPrefix(relayState, "//") {
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

	// Validate conditions (time window, audience, subject confirmation)
	// Expected audience = our SP entityID (derived from request)
	spEntityID := publicURIFromRequest(r) + "/saml/metadata"
	if err := assertion.ValidateConditionsWithAudience(spEntityID); err != nil {
		slog.Error("SAML ACS: conditions validation failed", "error", err)
		writeError(w, http.StatusForbidden, "SAML assertion conditions not met")
		return
	}

	// SECURITY (P1-5): Validate InResponseTo to prevent assertion replay.
	// If the assertion contains an InResponseTo attribute, it must match
	// a SAML AuthnRequest ID we initiated. Since this implementation only
	// supports IdP-initiated SSO (InResponseTo will be empty), any non-empty
	// InResponseTo that we didn't generate is rejected.
	inResponseTo := assertion.InResponseTo()
	if inResponseTo != "" {
		// We don't generate AuthnRequest IDs yet (no SP-initiated flow).
		// Reject unexpected InResponseTo to prevent replay of captured assertions.
		slog.Warn("SAML ACS: assertion has InResponseTo but no SP-initiated request was made",
			"in_response_to", inResponseTo)
		writeError(w, http.StatusForbidden, "unexpected SAML InResponseTo")
		return
	}
	// SECURITY (R-cron2 P1-4): one-time consumption of assertion IDs.
	// Without this, a captured validly-signed assertion could be replayed
	// indefinitely. SETNX is atomic across replicas; TTL covers the maximum
	// assertion validity window plus clock skew.
	if h.rdb != nil && assertion.ID != "" {
		ok, serr := h.rdb.SetNX(r.Context(), "saml_assertion:"+assertion.ID, 1, 15*time.Minute).Result()
		if serr != nil {
			slog.Error("SAML ACS: assertion replay cache error", "error", serr)
			writeError(w, http.StatusInternalServerError, "SAML processing error")
			return
		}
		if !ok {
			slog.Warn("SAML ACS: assertion ID replay rejected", "assertion_id", assertion.ID)
			writeError(w, http.StatusForbidden, "SAML assertion already used")
			return
		}
	} else if h.rdb == nil {
		// SECURITY: fail-closed — without Redis, replay protection cannot
		// operate. Reject the assertion rather than allowing potential replay.
		slog.Error("SAML ACS: redis unavailable, assertion replay protection required")
		writeError(w, http.StatusServiceUnavailable, "SAML replay protection unavailable")
		return
	}

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
		tenantID = os.Getenv("GGID_TENANT_ID")
	}

	// Generate session token (reuse auth token mechanism)
	sessionID := uuid.New().String()
	slog.Info("SAML ACS: assertion verified",
		"email", userEmail, "name", userName, "session", sessionID,
		"attributes", attrs, "relay_state", relayState)

	// Redirect back to relay state — session via cookie, NOT URL params (PII leak prevention)
	http.SetCookie(w, &http.Cookie{
		Name:     "saml_session",
		Value:    url.QueryEscape(sessionID),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})
	http.Redirect(w, r, relayState, http.StatusFound)
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
	// SECURITY: modifying SAML configuration requires admin scope
	if !hasAdminScope(r) {
		writeError(w, http.StatusForbidden, "admin scope required")
		return
	}
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
	block, _ := pemDecode(cfg.IDPCert)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM certificate")
	}

	return x509.ParseCertificate(block.Bytes)
}
