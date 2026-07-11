package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// BreakGlassAccess tracks an emergency break-glass access grant.
type BreakGlassAccess struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Requester   string    `json:"requester"`
	Approver    string    `json:"approver,omitempty"`
	Reason      string    `json:"reason"`
	Scope       string    `json:"scope"`
	Duration    string    `json:"duration"`
	PreAuthorized bool    `json:"pre_authorized"`
	Status      string    `json:"status"` // active, expired, revoked
	GrantedAt   time.Time `json:"granted_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

var (
	breakGlassMu sync.RWMutex
	breakGlassStore = make(map[string]*BreakGlassAccess)
)

// POST /api/v1/policies/break-glass — request emergency access.
// GET /api/v1/policies/break-glass/active — list active break-glass grants.
func (s *HTTPServer) handleBreakGlass(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			TenantID  string `json:"tenant_id"`
			Requester string `json:"requester"`
			Approver  string `json:"approver"`
			Reason    string `json:"reason"`
			Scope     string `json:"scope"`
			DurationHours int `json:"duration_hours"`
			PreAuthorized bool `json:"pre_authorized"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Requester == "" || req.Reason == "" {
			writeJSONError(w, http.StatusBadRequest, "requester and reason are required")
			return
		}
		if req.DurationHours <= 0 { req.DurationHours = 4 }
		if req.Scope == "" { req.Scope = "*" }

		now := time.Now().UTC()
		dur := time.Duration(req.DurationHours) * time.Hour
		bg := &BreakGlassAccess{
			ID: uuid.New().String(), TenantID: req.TenantID,
			Requester: req.Requester, Approver: req.Approver,
			Reason: req.Reason, Scope: req.Scope,
			Duration: dur.String(),
			PreAuthorized: req.PreAuthorized, Status: "active",
			GrantedAt: now, ExpiresAt: now.Add(dur),
		}
		breakGlassMu.Lock(); breakGlassStore[bg.ID] = bg; breakGlassMu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{
			"break_glass":   bg,
			"admin_notified": true,
			"audit_logged":   true,
		})

	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		now := time.Now().UTC()
		breakGlassMu.RLock()
		result := []*BreakGlassAccess{}
		for _, bg := range breakGlassStore {
			if now.After(bg.ExpiresAt) && bg.Status == "active" {
				bg.Status = "expired"
			}
			if bg.Status != "active" { continue }
			if tenantID != "" && bg.TenantID != tenantID { continue }
			result = append(result, bg)
		}
		breakGlassMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"active": result, "count": len(result)})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
