package server

import (
	"encoding/json"
	"net/http"
)

type AccountLinkingConfig struct {
	AllowedProviders         []string                `json:"allowed_providers"`
	AutoLinkOnEmailMatch     bool                    `json:"auto_link_on_email_match"`
	RequireVerificationLink  bool                    `json:"require_verification_for_link"`
	MaxLinkedAccounts        int                     `json:"max_linked_accounts"`
	UnlinkCooldownHours      int                     `json:"unlink_cooldown_hours"`
	PerProviderConfig        map[string]ProviderLink `json:"per_provider_config"`
}

type ProviderLink struct {
	DisplayName      string `json:"display_name"`
	Enabled          bool   `json:"enabled"`
	LinkOnMatch      bool   `json:"link_on_match"`
	VerifyBeforeLink bool   `json:"verify_before_link"`
}

var globalAccountLinkingConfig = &AccountLinkingConfig{
	AllowedProviders:        []string{"google", "github", "microsoft", "saml", "oidc"},
	AutoLinkOnEmailMatch:    false,
	RequireVerificationLink: true,
	MaxLinkedAccounts:       5,
	UnlinkCooldownHours:     24,
	PerProviderConfig: map[string]ProviderLink{
		"google":    {DisplayName: "Google", Enabled: true, LinkOnMatch: true, VerifyBeforeLink: true},
		"github":    {DisplayName: "GitHub", Enabled: true, LinkOnMatch: false, VerifyBeforeLink: true},
		"microsoft": {DisplayName: "Microsoft", Enabled: true, LinkOnMatch: true, VerifyBeforeLink: true},
		"saml":      {DisplayName: "SAML IdP", Enabled: false, LinkOnMatch: false, VerifyBeforeLink: true},
		"oidc":      {DisplayName: "OIDC Provider", Enabled: true, LinkOnMatch: true, VerifyBeforeLink: true},
	},
}

func (h *HTTPHandler) handleAccountLinkingConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalAccountLinkingConfig)
	case http.MethodPut:
		var cfg AccountLinkingConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if cfg.MaxLinkedAccounts < 1 {
			http.Error(w, `{"error":"max_linked_accounts must be at least 1"}`, http.StatusBadRequest)
			return
		}
		globalAccountLinkingConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
