package server

import (
	"encoding/json"
	"net/http"
)

type ImpersonationConfig struct {
	AllowedImpersonators   []string `json:"allowed_impersonators"`
	RequireReason          bool     `json:"require_reason"`
	MaxDurationMinutes     int      `json:"max_duration_minutes"`
	AuditLevel             string   `json:"audit_level"`
	RequireTargetConsent   bool     `json:"require_target_consent"`
	AutoRevokeOnIdle       bool     `json:"auto_revoke_on_idle"`
	RestrictToRoles        []string `json:"restrict_to_roles"`
}

var globalImpersonationConfig = &ImpersonationConfig{
	AllowedImpersonators: []string{"admin", "support_admin", "security_admin"},
	RequireReason:        true,
	MaxDurationMinutes:   30,
	AuditLevel:           "full",
	RequireTargetConsent: true,
	AutoRevokeOnIdle:     true,
	RestrictToRoles:      []string{"user", "manager"},
}

func (h *Handler) handleImpersonationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalImpersonationConfig)
	case http.MethodPut:
		var cfg ImpersonationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if cfg.MaxDurationMinutes < 1 {
			writeJSONError(w, http.StatusBadRequest, "max_duration_minutes must be at least 1")
			return
		}
		globalImpersonationConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}