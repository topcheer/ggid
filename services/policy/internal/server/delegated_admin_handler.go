package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type DelegatedAdmin struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Delegator   string    `json:"delegator"`
	Delegate    string    `json:"delegate"`
	ScopeType   string    `json:"scope_type"`
	ScopeID     string    `json:"scope_id"`
	Permissions []string  `json:"permissions"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *HTTPServer) handleDelegatedAdmin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			TenantID    string   `json:"tenant_id"`
			Delegator   string   `json:"delegator"`
			Delegate    string   `json:"delegate"`
			ScopeType   string   `json:"scope_type"`
			ScopeID     string   `json:"scope_id"`
			Permissions []string `json:"permissions"`
			ExpiryHours int      `json:"expiry_hours"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Delegator == "" || req.Delegate == "" || req.ScopeType == "" {
			writeJSONError(w, http.StatusBadRequest, "delegator, delegate, scope_type required")
			return
		}
		if req.ExpiryHours <= 0 { req.ExpiryHours = 72 }
		now := time.Now().UTC()
		da := &DelegatedAdmin{
			ID: uuid.New().String(), TenantID: req.TenantID,
			Delegator: req.Delegator, Delegate: req.Delegate,
			ScopeType: req.ScopeType, ScopeID: req.ScopeID,
			Permissions: req.Permissions,
			ExpiresAt: now.Add(time.Duration(req.ExpiryHours) * time.Hour),
			CreatedAt: now,
		}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_delegated_admins", da.ID, map[string]any{
				"tenant_id": da.TenantID, "delegator": da.Delegator, "delegate": da.Delegate,
				"scope_type": da.ScopeType, "scope_id": da.ScopeID,
				"permissions": da.Permissions, "expires_at": da.ExpiresAt,
			})
		}
		writeJSON(w, http.StatusCreated, da)
	case http.MethodGet:
		var result []map[string]any
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_delegated_admins")
			result = rows
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"delegations": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
