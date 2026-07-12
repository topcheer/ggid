package server

import (
	"encoding/json"
	"net/http"
)

type RedirectURIValidationConfig struct {
	HTTPSOnly             bool              `json:"https_only"`
	ExactMatchOnly        bool              `json:"exact_match_only"`
	LocalhostAllowlist    []string          `json:"localhost_allowlist"`
	CustomSchemeAllowlist []string          `json:"custom_scheme_allowlist"`
	PerClientPatterns     map[string][]string `json:"per_client_allowed_patterns"`
}

func handleRedirectURIValidationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := RedirectURIValidationConfig{
			HTTPSOnly:          true,
			ExactMatchOnly:     false,
			LocalhostAllowlist: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
			CustomSchemeAllowlist: []string{"myapp://callback", "com.example.app://oauth"},
			PerClientPatterns: map[string][]string{
				"web-console":   {"https://*.GGID.example.com/callback"},
				"mobile-app":    {"myapp://callback"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req RedirectURIValidationConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
