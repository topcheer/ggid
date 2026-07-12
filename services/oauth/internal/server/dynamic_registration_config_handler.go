package server

import (
	"encoding/json"
	"net/http"
)

type DynamicRegistrationConfig struct {
	AllowOpenRegistration    bool     `json:"allow_open_registration"`
	DefaultGrantTypes        []string `json:"default_grant_types"`
	DefaultScopes            []string `json:"default_scopes"`
	RequireSoftwareStatement bool     `json:"require_software_statement"`
	ClientURIValidation      bool     `json:"client_uri_validation"`
	LogoURIValidation        bool     `json:"logo_uri_validation"`
	MaxRedirectURIs          int      `json:"max_redirect_uris"`
	AutoApprove              bool     `json:"auto_approve"`
}

var globalDynamicRegistrationConfig = &DynamicRegistrationConfig{
	AllowOpenRegistration:    false,
	DefaultGrantTypes:        []string{"authorization_code", "refresh_token"},
	DefaultScopes:            []string{"openid", "profile"},
	RequireSoftwareStatement: false,
	ClientURIValidation:      true,
	LogoURIValidation:        true,
	MaxRedirectURIs:          5,
	AutoApprove:              false,
}

func handleDynamicRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalDynamicRegistrationConfig)
	case http.MethodPut:
		var cfg DynamicRegistrationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if cfg.MaxRedirectURIs < 1 {
			http.Error(w, `{"error":"max_redirect_uris must be at least 1"}`, http.StatusBadRequest)
			return
		}
		globalDynamicRegistrationConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}