package server

import (
	"encoding/json"
	"net/http"
)

type ProvisioningConfig struct {
	Endpoint              string            `json:"endpoint"`
	MappingRules          map[string]string `json:"mapping_rules"`
	Triggers              []string          `json:"triggers"`
	SyncDirection         string            `json:"sync_direction"`
	DeprovisionOnDisable  bool              `json:"deprovision_on_disable"`
	TestConnection        struct {
		Status     string `json:"status"`
		LatencyMs  int    `json:"latency_ms"`
		UsersSync  int    `json:"users_synced"`
	} `json:"test_connection"`
}

func (h *HTTPHandler) handleSCIMProvisioningConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := ProvisioningConfig{
			Endpoint:             "https://scim.example.com/v2",
			MappingRules:         map[string]string{"userName": "username", "emails": "email", "displayName": "full_name", "active": "status"},
			Triggers:             []string{"user_create", "user_update", "user_disable", "group_change"},
			SyncDirection:        "bidirectional",
			DeprovisionOnDisable: true,
		}
		result.TestConnection.Status = "ok"
		result.TestConnection.LatencyMs = 85
		result.TestConnection.UsersSync = 450
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req ProvisioningConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
