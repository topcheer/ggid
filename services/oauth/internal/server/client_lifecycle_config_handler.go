package server

import (
	"encoding/json"
	"net/http"
)

type ClientLifecycleConfig struct {
	AllowedGrantTypesPerEnv       map[string][]string `json:"allowed_grant_types_per_environment"`
	DefaultTokenTTL               int                 `json:"default_token_ttl_seconds"`
	RefreshTokenRotation          string              `json:"refresh_token_rotation"`
	ConsentRequiredScopes         []string            `json:"consent_required_scopes"`
	ClientSecretRotationDays      int                 `json:"client_secret_rotation_days"`
	InactiveClientDeactivateDays  int                 `json:"inactive_client_deactivate_days"`
}

var globalClientLifecycleConfig = &ClientLifecycleConfig{
	AllowedGrantTypesPerEnv: map[string][]string{
		"production":  {"authorization_code", "refresh_token", "client_credentials"},
		"staging":     {"authorization_code", "refresh_token", "client_credentials", "device_code"},
		"development": {"authorization_code", "refresh_token", "client_credentials", "password", "device_code"},
	},
	DefaultTokenTTL:              3600,
	RefreshTokenRotation:         "rotating",
	ConsentRequiredScopes:        []string{"openid", "profile", "email"},
	ClientSecretRotationDays:     90,
	InactiveClientDeactivateDays: 180,
}

func handleClientLifecycleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalClientLifecycleConfig)
	case http.MethodPut:
		var cfg ClientLifecycleConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if cfg.DefaultTokenTTL < 60 {
			writeJSONError(w, http.StatusBadRequest, "default_token_ttl_seconds must be at least 60")
			return
		}
		globalClientLifecycleConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
