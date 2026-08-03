package httpserver

import (
	"encoding/json"
	"net/http"
)

type DelegationConfig struct {
	MaxDelegationDepth              int      `json:"max_delegation_depth"`
	AllowedDelegatorRoles           []string `json:"allowed_delegator_roles"`
	DelegationExpiryHours           int      `json:"delegation_expiry_hours"`
	RevocationByDelegator           bool     `json:"revocation_by_delegator"`
	RequireConsent                  bool     `json:"require_consent"`
	AuditAllDelegations             bool     `json:"audit_all_delegations"`
	CascadeRevokeOnDelegatorDisable bool     `json:"cascade_revoke_on_delegator_disable"`
}

var globalDelegationConfig = &DelegationConfig{
	MaxDelegationDepth:              3,
	AllowedDelegatorRoles:           []string{"admin", "manager", "security_admin"},
	DelegationExpiryHours:           24,
	RevocationByDelegator:           true,
	RequireConsent:                  true,
	AuditAllDelegations:             true,
	CascadeRevokeOnDelegatorDisable: true,
}

func (s *HTTPServer) handleDelegationConfig(w http.ResponseWriter, r *http.Request) {
	// SECURITY: PUT modifies a global singleton config affecting all tenants.
	// Require platform:admin scope, same as handleDefaultAction.
	if r.Method == http.MethodPut && !isAdminRequest(r) {
		writeJSONError(w, http.StatusForbidden, "platform:admin scope required to modify delegation config")
		return
	}
	// SECURITY: require valid tenant context.
	if callerTenant(r) == "" {
		writeJSONError(w, http.StatusForbidden, "valid X-Tenant-ID header required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalDelegationConfig)
	case http.MethodPut:
		var cfg DelegationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if cfg.MaxDelegationDepth < 1 {
			writeJSONError(w, http.StatusBadRequest, "max_delegation_depth must be at least 1")
			return
		}
		globalDelegationConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
