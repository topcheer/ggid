package server

import (
	"encoding/json"
	"net/http"
)

type PasswordlessConfig struct {
	EnabledMethods       []string          `json:"enabled_methods"`
	MagicLinkExpiryMins  int               `json:"magic_link_expiry_minutes"`
	PasskeyRPID          string            `json:"passkey_rp_id"`
	WebAuthnTimeout      int               `json:"webauthn_timeout_seconds"`
	FallbackToPassword   bool              `json:"fallback_to_password"`
	PerRoleRequirement   map[string]string `json:"per_role_requirement"`
}

func (h *Handler) handlePasswordlessConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := PasswordlessConfig{
			EnabledMethods:      []string{"magic_link", "passkey", "webauthn", "biometric"},
			MagicLinkExpiryMins: 10,
			PasskeyRPID:         "ggid.example.com",
			WebAuthnTimeout:     60,
			FallbackToPassword:  true,
			PerRoleRequirement: map[string]string{
				"admin":       "webauthn",
				"developer":   "passkey",
				"viewer":      "magic_link",
				"service":     "passkey",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req PasswordlessConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
