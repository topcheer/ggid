package server

import (
	"sync"
	"strings"


	"encoding/json"
	"net/http"
)

type ImpersonationConfig struct {
	AllowedImpersonators []string `json:"allowed_impersonators"`
	RequireReason        bool     `json:"require_reason"`
	MaxDurationMinutes   int      `json:"max_duration_minutes"`
	AuditLevel           string   `json:"audit_level"`
	RequireTargetConsent bool     `json:"require_target_consent"`
	AutoRevokeOnIdle     bool     `json:"auto_revoke_on_idle"`
	RestrictToRoles      []string `json:"restrict_to_roles"`
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

var impersonationConfigMu sync.RWMutex

func (h *Handler) handleImpersonationConfig(w http.ResponseWriter, r *http.Request) {
	// SECURITY: PUT modifies global config — require platform:admin scope.
	if r.Method == http.MethodPut {
		scopes := r.Header.Get("X-Scopes")
		// SECURITY: Exact scope matching — strings.Contains allows substring bypass.
		hasPlatformAdmin := false
		for _, s := range strings.FieldsFunc(scopes, func(r rune) bool { return r == ' ' || r == ',' }) {
			if s == "platform:admin" {
				hasPlatformAdmin = true
				break
			}
		}
		if !hasPlatformAdmin {
			writeError(w, http.StatusForbidden, "platform:admin scope required")
			return
		}
	}
	switch r.Method {
	case http.MethodGet:
		impersonationConfigMu.RLock()
		cfg := globalImpersonationConfig
		impersonationConfigMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	case http.MethodPut:
		var cfg ImpersonationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if cfg.MaxDurationMinutes < 1 {
			writeError(w, http.StatusBadRequest, "max_duration_minutes must be at least 1")
			return
		}
		impersonationConfigMu.Lock()
		globalImpersonationConfig = &cfg
		impersonationConfigMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
