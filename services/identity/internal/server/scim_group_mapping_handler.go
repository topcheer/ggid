package server

import (
	"encoding/json"
	"net/http"
)

type GroupMappingEntry struct {
	ExternalGroup  string            `json:"external_group"`
	LocalRole      string            `json:"local_role"`
	AutoProvision  bool              `json:"auto_provision"`
	PerAppOverride map[string]string `json:"per_app_override"`
	SyncDirection  string            `json:"sync_direction"`
	LastSyncStatus string            `json:"last_sync_status"`
}

type GroupMappingResult struct {
	Mappings      []GroupMappingEntry `json:"mappings"`
	TotalMappings int                 `json:"total_mappings"`
	LastSyncAt    string              `json:"last_sync_at"`
}

func (h *HTTPHandler) handleSCIMGroupMapping(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := GroupMappingResult{
			Mappings: []GroupMappingEntry{
				{ExternalGroup: "CN=admins,OU=groups", LocalRole: "admin", AutoProvision: true, PerAppOverride: map[string]string{"slack": "admin", "github": "owner"}, SyncDirection: "inbound", LastSyncStatus: "success"},
				{ExternalGroup: "CN=developers,OU=groups", LocalRole: "developer", AutoProvision: true, PerAppOverride: map[string]string{"slack": "member", "github": "write"}, SyncDirection: "bidirectional", LastSyncStatus: "success"},
				{ExternalGroup: "CN=viewers,OU=groups", LocalRole: "viewer", AutoProvision: false, PerAppOverride: map[string]string{}, SyncDirection: "inbound", LastSyncStatus: "pending"},
			},
			TotalMappings: 3,
			LastSyncAt:    "2025-01-15T08:00:00Z",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req struct{ Mappings []GroupMappingEntry `json:"mappings"` }
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "count": len(req.Mappings)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
