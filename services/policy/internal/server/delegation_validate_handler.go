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
	var req struct {
		DelegatorID string `json:"delegator_id"`
		DelegateeID string `json:"delegatee_id"`
		Permissions []string `json:"permissions"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	writeJSON(w, http.StatusOK, map[string]any{
		"valid":          true,
		"chain_depth":    1,
		"cycle_detected": false,
		"permissions":    req.Permissions,
	})
}
