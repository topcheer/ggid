package server

import (
	"encoding/json"
	"net/http"
)

type WebAuthnConfig struct {
	RPID                  string            `json:"rp_id"`
	RPName                string            `json:"rp_name"`
	Origin                string            `json:"origin"`
	AttestationRequirement string           `json:"attestation_requirement"`
	UserVerification      string            `json:"user_verification"`
	SupportedAlgorithms   []string          `json:"supported_algorithms"`
	TimeoutSeconds        int               `json:"timeout_seconds"`
	PerPlatformConfig     map[string]string `json:"per_platform_config"`
}

func (h *Handler) handleWebAuthnConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := WebAuthnConfig{
			RPID:                   "ggid.example.com",
			RPName:                 "GGID IAM",
			Origin:                 "https://ggid.example.com",
			AttestationRequirement: "preferred",
			UserVerification:       "preferred",
			SupportedAlgorithms:    []string{"ES256", "RS256", "EdDSA"},
			TimeoutSeconds:         60,
			PerPlatformConfig: map[string]string{
				"macos":    "touch_id_preferred",
				"ios":      "face_id_preferred",
				"windows":  "hello_preferred",
				"android":  "fingerprint_preferred",
				"linux":    "security_key_required",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req WebAuthnConfig
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
