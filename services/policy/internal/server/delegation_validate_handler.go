package httpserver

import (
	"encoding/json"
	"net/http"
)

func (s *HTTPServer) handleDelegationValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// SECURITY: Require valid tenant header.
	tid := callerTenant(r)
	if tid == "" {
		writeJSONError(w, http.StatusForbidden, "valid X-Tenant-ID header required")
		return
	}
	var req struct {
		DelegatorID string   `json:"delegator_id"`
		DelegateeID string   `json:"delegatee_id"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	// SECURITY: Verify delegator has an active delegation to delegatee within
	// the caller's tenant. Cross-tenant delegator/delegatee pairs are rejected.
	reviewDelegationStore.RLock()
	valid := false
	for _, d := range reviewDelegationStore.data {
		if d.TenantID != tid {
			continue
		}
		if d.OriginalReviewer == req.DelegatorID && d.DelegatedTo == req.DelegateeID && d.Status == "active" {
			valid = true
			break
		}
	}
	reviewDelegationStore.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":          valid,
		"chain_depth":    1,
		"cycle_detected": false,
		"permissions":    req.Permissions,
	})
}
