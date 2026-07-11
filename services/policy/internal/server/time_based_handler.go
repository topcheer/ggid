package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TimeAccessRule defines a time-based access control policy.
type TimeAccessRule struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	Name         string   `json:"name"`
	Schedule     string   `json:"schedule"` // cron expression
	Timezone     string   `json:"timezone"`
	AllowedRoles []string `json:"allowed_roles"`
	ResourcePattern string `json:"resource_pattern"`
	Action       string   `json:"action"` // allow, deny
	Enabled      bool     `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

var (
	timeAccessMu sync.RWMutex
	timeAccessRules = make(map[string]*TimeAccessRule)
)

// POST /api/v1/policies/time-based — create time-based access rule.
// GET /api/v1/policies/time-based — list rules.
// POST /api/v1/policies/time-based/{id}/check — check if access is currently allowed.
func (s *HTTPServer) handleTimeBased(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			TenantID     string   `json:"tenant_id"`
			Name         string   `json:"name"`
			Schedule     string   `json:"schedule"`
			Timezone     string   `json:"timezone"`
			AllowedRoles []string `json:"allowed_roles"`
			ResourcePattern string `json:"resource_pattern"`
			Action       string   `json:"action"`
			Enabled      *bool    `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Name == "" || req.Schedule == "" {
			writeJSONError(w, http.StatusBadRequest, "name and schedule are required")
			return
		}
		if req.Timezone == "" {
			req.Timezone = "UTC"
		}
		if req.Action == "" {
			req.Action = "allow"
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		rule := &TimeAccessRule{
			ID: uuid.New().String(),
			TenantID: req.TenantID, Name: req.Name,
			Schedule: req.Schedule, Timezone: req.Timezone,
			AllowedRoles: req.AllowedRoles, ResourcePattern: req.ResourcePattern,
			Action: req.Action, Enabled: enabled, CreatedAt: time.Now().UTC(),
		}
		timeAccessMu.Lock()
		timeAccessRules[rule.ID] = rule
		timeAccessMu.Unlock()
		writeJSON(w, http.StatusCreated, rule)

	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		timeAccessMu.RLock()
		result := []*TimeAccessRule{}
		for _, rule := range timeAccessRules {
			if tenantID != "" && rule.TenantID != tenantID {
				continue
			}
			result = append(result, rule)
		}
		timeAccessMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"rules": result, "count": len(result)})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
