package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/policy/internal/service"
	"github.com/google/uuid"
)

// POST /api/v1/policies/delegate — delegate permissions to another user
func (s *HTTPServer) handleDelegate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// SECURITY: delegation requires admin scope (verifies delegator is authorized)
	scopes := r.Header.Get("X-Scopes")
	if !hasRole(scopes, "platform:admin") && !hasRole(scopes, "tenant:admin") {
		writeJSONError(w, http.StatusForbidden, "admin scope required for delegation")
		return
	}
	var req struct {
		DelegatorID  string   `json:"delegator_id"`
		DelegateeID  string   `json:"delegatee_id"`
		Permissions  []string `json:"permissions"`
		MaxDurationH int      `json:"max_duration_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	hours := time.Duration(req.MaxDurationH) * time.Hour
	if hours <= 0 {
		hours = 24 * time.Hour // default 24h
	}
	// SECURITY: Use caller's identity as delegator, not request body value.
	delegatorIDStr := r.Header.Get("X-User-ID")
	if delegatorIDStr == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	delegatorID := parseUUIDSafe(delegatorIDStr)
	d, err := s.policySvc.DelegatePermissions(
		r.Context(),
		delegatorID,
		parseUUIDSafe(req.DelegateeID),
		req.Permissions, hours,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

// GET /api/v1/policies/delegations?user_id=X — list delegations for a user
func (s *HTTPServer) handleListDelegations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// SECURITY: require tenant context
	tid, ok := requireTenantHeader(w, r)
	if !ok {
		return
	}
	userID := parseUUIDSafe(r.URL.Query().Get("user_id"))
	delegations, err := s.policySvc.ListDelegations(context.Background(), userID)
	if err != nil {
		log.Printf("delegation list error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	// SECURITY: filter to caller's tenant
	filtered := []*service.Delegation{}
	for _, d := range delegations {
		if d.TenantID.String() == tid {
			filtered = append(filtered, d)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"delegations": filtered})
}

func parseUUIDSafe(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
