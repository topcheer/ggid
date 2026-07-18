package server

import (
	"encoding/json"
	"net/http"
)

type ConsentConfig struct {
	ConsentTemplatePerScope     map[string]string `json:"consent_template_per_scope"`
	ConsentExpiryDays           int               `json:"consent_expiry_days"`
	AllowRememberConsent        bool              `json:"allow_remember_consent"`
	RequireExplicitForScopes    []string          `json:"require_explicit_for_scopes"`
	PerClientConsentOverride    bool              `json:"per_client_consent_override"`
	DynamicConsentRegistration  bool              `json:"dynamic_consent_registration"`
	ConsentRevocationEndpoint   string            `json:"consent_revocation_endpoint"`
}

var globalConsentConfig = &ConsentConfig{
	ConsentTemplatePerScope: map[string]string{
		"openid":         "We need to verify your identity.",
		"profile":        "Access your basic profile information.",
		"email":          "Access your email address.",
		"offline_access": "Access your data even when you are offline.",
	},
	ConsentExpiryDays:          90,
	AllowRememberConsent:       true,
	RequireExplicitForScopes:   []string{"offline_access", "admin"},
	PerClientConsentOverride:   true,
	DynamicConsentRegistration: false,
	ConsentRevocationEndpoint:  "/api/v1/oauth/consent/revoke",
}

func handleConsentConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalConsentConfig)
	case http.MethodPut:
		var cfg ConsentConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if cfg.ConsentExpiryDays < 1 {
			writeJSONError(w, http.StatusBadRequest, "consent_expiry_days must be at least 1")
			return
		}
		globalConsentConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
