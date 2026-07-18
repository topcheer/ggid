package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SCIMTarget defines an outbound SCIM provisioning target.
type SCIMTarget struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TenantID    string    `json:"tenant_id"`
	BaseURL     string    `json:"base_url"`
	AuthType    string    `json:"auth_type"` // bearer, basic, oauth
	AuthToken   string    `json:"-"`          // never expose
	Enabled     bool      `json:"enabled"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// SCIMSyncLogEntry records a sync operation.
type SCIMSyncLogEntry struct {
	ID         string    `json:"id"`
	TargetID   string    `json:"target_id"`
	EventType  string    `json:"event_type"`
	ResourceID string    `json:"resource_id"`
	Status     string    `json:"status"`
	SyncedAt   time.Time `json:"synced_at"`
}

// GET /api/v1/scim/targets
// POST /api/v1/scim/targets
// POST /api/v1/scim/sync/:target
// GET /api/v1/scim/sync/log
func (h *HTTPHandler) handleSCIMTargets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req SCIMTarget
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" || req.BaseURL == "" {
			writeJSONError(w, http.StatusBadRequest, "name and base_url required")
			return
		}
		req.ID = uuid.New().String()
		req.Enabled = true
		req.CreatedAt = time.Now().UTC()
		if h.identityPolicyMap != nil {
			h.identityPolicyMap.Store(r.Context(), "scim_targets", req.ID, map[string]any{
				"name": req.Name, "tenant_id": req.TenantID, "base_url": req.BaseURL,
				"auth_type": req.AuthType, "enabled": req.Enabled,
			})
		}
		writeJSON(w, http.StatusCreated, req)
	case http.MethodGet:
		var targets []map[string]any
		if h.identityPolicyMap != nil {
			rows, _ := h.identityPolicyMap.List(r.Context(), "scim_targets")
			targets = rows
		}
		if targets == nil { targets = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"targets": targets, "count": len(targets)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) handleSCIMSyncTarget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	targetID := strings.TrimPrefix(r.URL.Path, "/api/v1/scim/sync/")
	if targetID == "" {
		writeJSONError(w, http.StatusBadRequest, "target id required")
		return
	}
	// Simulate sync — in production, calls SCIM client.
	now := time.Now().UTC()
	if h.identityPolicyMap != nil {
		logID := uuid.New().String()
		h.identityPolicyMap.Store(r.Context(), "scim_sync_log", logID, map[string]any{
			"target_id": targetID, "event_type": "full_sync",
			"status": "completed", "synced_at": now,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "synced", "target_id": targetID,
		"synced_at": now, "users_synced": 0, "groups_synced": 0,
	})
}

func (h *HTTPHandler) handleSCIMSyncLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var log []map[string]any
	if h.identityPolicyMap != nil {
		rows, _ := h.identityPolicyMap.List(r.Context(), "scim_sync_log")
		log = rows
	}
	if log == nil { log = []map[string]any{} }
	writeJSON(w, http.StatusOK, map[string]any{"log": log, "count": len(log)})
}
