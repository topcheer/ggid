package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type PrivilegedAccount struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	Roles       []string  `json:"roles"`
	GrantedAt   time.Time `json:"granted_at"`
	Justification string  `json:"justification"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

var (
	paMu sync.RWMutex
	paAccounts = []PrivilegedAccount{
		{UserID: "u-001", Username: "admin", Roles: []string{"admin", "auditor"}, GrantedAt: time.Now().Add(-720 * time.Hour), Justification: "break-glass"},
		{UserID: "u-004", Username: "bwang", Roles: []string{"manager", "compliance"}, GrantedAt: time.Now().Add(-48 * time.Hour), Justification: "quarterly audit"},
	}
)

// GET /api/v1/policies/privileged-access
// POST /api/v1/policies/privileged-access/revoke
func (s *HTTPServer) handlePrivilegedAccess(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct{ UserIDs []string `json:"user_ids"` }
		_ = json.NewDecoder(r.Body).Decode(&req)
		paMu.Lock()
		if len(req.UserIDs) > 0 { paAccounts = []PrivilegedAccount{} }
		for _, pa := range paAccounts {
			for _, uid := range req.UserIDs { if pa.UserID == uid { pa.Roles = nil } }
		}
		paMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "revoked", "revoked_count": len(req.UserIDs)})
		return
	}
	paMu.RLock()
	result := make([]PrivilegedAccount, len(paAccounts))
	copy(result, paAccounts)
	paMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"privileged_accounts": result, "count": len(result)})
}
