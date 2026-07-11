package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DelegatedAdmin holds admin permissions delegated to another user.
type DelegatedAdmin struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Delegator  string    `json:"delegator"`
	Delegate   string    `json:"delegate"`
	ScopeType  string    `json:"scope_type"`  // org, role, department
	ScopeID    string    `json:"scope_id"`
	Permissions []string `json:"permissions"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

var (
	delegatedAdminMu sync.RWMutex
	delegatedAdmins  = make(map[string]*DelegatedAdmin)
)

// POST /api/v1/policies/delegated-admin — delegate admin scope.
// GET /api/v1/policies/delegated-admin/list — list delegations.
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
		delegatedAdminMu.Lock(); delegatedAdmins[da.ID] = da; delegatedAdminMu.Unlock()
		writeJSON(w, http.StatusCreated, da)

	case http.MethodGet:
		delegate := r.URL.Query().Get("delegate")
		scopeType := r.URL.Query().Get("scope_type")
		delegatedAdminMu.RLock()
		result := []*DelegatedAdmin{}
		for _, da := range delegatedAdmins {
			if delegate != "" && da.Delegate != delegate { continue }
			if scopeType != "" && da.ScopeType != scopeType { continue }
			result = append(result, da)
		}
		delegatedAdminMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"delegations": result, "count": len(result)})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
