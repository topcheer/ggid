package server

import (
	"encoding/json"
	"net/http"
)

type SessionBindingConfig struct {
	BindingMethod          string            `json:"binding_method"`
	PerApplicationBinding  map[string]string `json:"per_application_binding"`
	BindingRotationPolicy  string            `json:"binding_rotation_policy"`
	SessionHijackProtection bool             `json:"session_hijack_protection"`
	FallbackMethod         string            `json:"fallback_method"`
}

func (h *Handler) handleSessionBindingConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := SessionBindingConfig{
			BindingMethod:          "DPoP",
			PerApplicationBinding:  map[string]string{"admin-console": "mTLS", "mobile-app": "cookie", "api-client": "DPoP"},
			BindingRotationPolicy:  "rotate_on_reauth",
			SessionHijackProtection: true,
			FallbackMethod:         "cookie",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req SessionBindingConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
