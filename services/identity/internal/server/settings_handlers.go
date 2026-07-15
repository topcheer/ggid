package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

// GET/PUT /api/v1/settings/branding
func (h *HTTPHandler) handleSettingsBranding(w http.ResponseWriter, r *http.Request) {
	defaultBranding := map[string]any{
		"logo_url": "", "primary_color": "#3b82f6", "app_name": "GGID",
		"login_bg_url": "", "footer_text": "", "custom_css": "",
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, defaultBranding)
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET /api/v1/settings/certificates
func (h *HTTPHandler) handleSettingsCertificates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"certificates": []map[string]any{
			{"id": "cert-1", "name": "SAML Signing", "type": "SAML", "expiry": "2026-01-15", "status": "active"},
			{"id": "cert-2", "name": "JWT Signing", "type": "JWT", "expiry": "2025-12-01", "status": "active"},
		},
	})
}

// GET/PUT /api/v1/settings/data-retention
func (h *HTTPHandler) handleSettingsDataRetention(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"audit_events_days": 365, "sessions_days": 90, "logs_days": 30,
			"gdpr_auto_delete": false, "archive_after_days": 730,
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET/PUT /api/v1/settings/delegations
func (h *HTTPHandler) handleSettingsDelegations(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"delegations": []map[string]any{}})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET/PUT /api/v1/settings/geo-fencing
func (h *HTTPHandler) handleSettingsGeoFencing(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false, "allowed_countries": []string{}, "blocked_countries": []string{},
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET /api/v1/settings/idp
func (h *HTTPHandler) handleSettingsIDP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": []map[string]any{
			{"id": "saml-1", "type": "saml", "name": "Corporate SAML", "status": "active"},
			{"id": "oidc-1", "type": "oidc", "name": "Google OIDC", "status": "active"},
		},
	})
}

// GET/PUT /api/v1/settings/jit-provisioning
func (h *HTTPHandler) handleSettingsJIT(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": true, "default_role": "user", "attribute_mapping": map[string]string{"email": "email", "name": "name"},
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET /api/v1/settings/jwks
func (h *HTTPHandler) handleSettingsJWKS(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"keys": []map[string]any{
			{"kid": "key-1", "kty": "RSA", "alg": "RS256", "use": "sig", "status": "active"},
		},
	})
}

// POST /api/v1/settings/jwks/rotate
func (h *HTTPHandler) handleSettingsJWKSRotate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "rotated", "new_kid": "key-2"})
}

// GET /api/v1/settings/login-flows
func (h *HTTPHandler) handleSettingsLoginFlows(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"flows": []map[string]any{
			{"id": "password", "name": "Password", "enabled": true, "priority": 1},
			{"id": "webauthn", "name": "Passkey", "enabled": true, "priority": 2},
			{"id": "saml", "name": "SAML SSO", "enabled": false, "priority": 3},
		},
	})
}

// GET/PUT /api/v1/settings/notifications
func (h *HTTPHandler) handleSettingsNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"channels": []map[string]any{
				{"type": "email", "enabled": true, "config": map[string]any{"from": "noreply@ggid.local"}},
				{"type": "webhook", "enabled": false},
			},
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET /api/v1/settings/notifications/templates
func (h *HTTPHandler) handleSettingsNotificationTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"templates": []map[string]any{
			{"id": "welcome", "name": "Welcome Email", "channel": "email", "subject": "Welcome to GGID"},
			{"id": "mfa-setup", "name": "MFA Setup", "channel": "email", "subject": "MFA Setup Required"},
		},
	})
}

// GET/PUT /api/v1/settings/siem
func (h *HTTPHandler) handleSettingsSIEM(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false, "protocol": "syslog", "endpoint": "", "format": "cef",
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// POST /api/v1/settings/siem/test
func (h *HTTPHandler) handleSettingsSIEMTest(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "latency_ms": 45})
}

// GET/PUT /api/v1/settings/sso
func (h *HTTPHandler) handleSettingsSSO(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"saml": map[string]any{"enabled": false, "entity_id": "", "acs_url": ""},
			"oidc": map[string]any{"enabled": false, "issuer": "", "client_id": ""},
			"social": map[string]any{"google": false, "github": false, "microsoft": false},
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// POST /api/v1/settings/sso/metadata/import
func (h *HTTPHandler) handleSettingsSSOMetadataImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MetadataXML string `json:"metadata_xml"`
		MetadataURL string `json:"metadata_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "imported", "entity_id": "imported-idp", "name": "Imported IDP",
	})
}

// GET /api/v1/settings/sso/providers
func (h *HTTPHandler) handleSettingsSSOProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": []map[string]any{
			{"id": "google", "type": "oidc", "name": "Google", "enabled": true},
			{"id": "github", "type": "oauth2", "name": "GitHub", "enabled": false},
		},
	})
}

// GET/PUT /api/v1/settings/sso/saml
func (h *HTTPHandler) handleSettingsSSOSAML(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false, "entity_id": "", "acs_url": "", "idp_metadata_url": "",
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET/PUT /api/v1/settings/sso/oidc
func (h *HTTPHandler) handleSettingsSSOIDC(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false, "issuer": "", "client_id": "", "scopes": []string{"openid", "email", "profile"},
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET/PUT /api/v1/settings/sso/social
func (h *HTTPHandler) handleSettingsSSOSocial(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"google": map[string]any{"enabled": true, "client_id": ""},
			"github": map[string]any{"enabled": false, "client_id": ""},
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET/PUT /api/v1/settings/webauthn
func (h *HTTPHandler) handleSettingsWebAuthN(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": true, "rp_id": "", "rp_name": "GGID", "require_verification": false,
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET/PUT /api/v1/settings/alerting/rules
func (h *HTTPHandler) handleSettingsAlertingRules(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"rules": []map[string]any{
				{"id": "alert-1", "name": "Failed Login Spike", "metric": "failed_logins", "threshold": 10, "window": "5m", "enabled": true},
			},
		})
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	writeJSON(w, http.StatusOK, req)
}

// GET /api/v1/settings/webhooks/delivery-log
func (h *HTTPHandler) handleSettingsWebhookDeliveryLog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"deliveries": []map[string]any{},
		"total":      0,
	})
}

// POST /api/v1/settings/webhooks/deliveries/retry-all
func (h *HTTPHandler) handleSettingsWebhookRetryAll(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "retrying", "count": 0})
}

// registerSettingsRoutes registers all /api/v1/settings/* routes.
func (h *HTTPHandler) registerSettingsRoutes() {
	mux := h.mux
	settings := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/v1/settings/branding", h.handleSettingsBranding},
		{"/api/v1/settings/certificates", h.handleSettingsCertificates},
		{"/api/v1/settings/data-retention", h.handleSettingsDataRetention},
		{"/api/v1/settings/delegations", h.handleSettingsDelegations},
		{"/api/v1/settings/geo-fencing", h.handleSettingsGeoFencing},
		{"/api/v1/settings/idp", h.handleSettingsIDP},
		{"/api/v1/settings/jit-provisioning", h.handleSettingsJIT},
		{"/api/v1/settings/jwks", h.handleSettingsJWKS},
		{"/api/v1/settings/jwks/rotate", h.handleSettingsJWKSRotate},
		{"/api/v1/settings/login-flows", h.handleSettingsLoginFlows},
		{"/api/v1/settings/notifications", h.handleSettingsNotifications},
		{"/api/v1/settings/notifications/templates", h.handleSettingsNotificationTemplates},
		{"/api/v1/settings/siem", h.handleSettingsSIEM},
		{"/api/v1/settings/siem/test", h.handleSettingsSIEMTest},
		{"/api/v1/settings/sso", h.handleSettingsSSO},
		{"/api/v1/settings/sso/metadata/import", h.handleSettingsSSOMetadataImport},
		{"/api/v1/settings/sso/providers", h.handleSettingsSSOProviders},
		{"/api/v1/settings/sso/saml", h.handleSettingsSSOSAML},
		{"/api/v1/settings/sso/oidc", h.handleSettingsSSOIDC},
		{"/api/v1/settings/sso/social", h.handleSettingsSSOSocial},
		{"/api/v1/settings/webauthn", h.handleSettingsWebAuthN},
		{"/api/v1/settings/alerting/rules", h.handleSettingsAlertingRules},
		{"/api/v1/settings/webhooks/delivery-log", h.handleSettingsWebhookDeliveryLog},
		{"/api/v1/settings/webhooks/deliveries/retry-all", h.handleSettingsWebhookRetryAll},
	}
	for _, s := range settings {
		mux.HandleFunc(s.path, s.handler)
	}
	_ = strings.TrimSpace // keep import
}
