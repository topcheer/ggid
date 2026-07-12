package server

import (
	"encoding/json"
	"net/http"
)

type LDAPSyncConfig struct {
	ServerURL        string            `json:"server_url"`
	BindDN           string            `json:"bind_dn"`
	BaseDN           string            `json:"base_dn"`
	UserFilter       string            `json:"user_filter"`
	GroupFilter      string            `json:"group_filter"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
	SyncIntervalMins int               `json:"sync_interval_minutes"`
	StartTLS         bool              `json:"start_tls"`
}

type LDAPSyncConfigResult struct {
	Config        LDAPSyncConfig `json:"config"`
	TestConnection struct {
		Status       string `json:"status"`
		LatencyMs    int    `json:"latency_ms"`
		UsersFound   int    `json:"users_found"`
		GroupsFound  int    `json:"groups_found"`
	} `json:"test_connection"`
	LastSync struct {
		Timestamp string `json:"timestamp"`
		Status    string `json:"status"`
		Errors    int    `json:"errors"`
	} `json:"last_sync"`
}

func (h *HTTPHandler) handleLDAPSyncConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := LDAPSyncConfigResult{}
		result.Config = LDAPSyncConfig{
			ServerURL:   "ldap://corp.example.com:389",
			BindDN:      "cn=service,ou=services,dc=example,dc=com",
			BaseDN:      "dc=example,dc=com",
			UserFilter:  "(objectClass=person)",
			GroupFilter: "(objectClass=groupOfNames)",
			AttributeMapping: map[string]string{
				"mail":    "email",
				"cn":      "full_name",
				"uid":     "username",
				"memberOf": "groups",
			},
			SyncIntervalMins: 15,
			StartTLS:         true,
		}
		result.TestConnection.Status = "ok"
		result.TestConnection.LatencyMs = 42
		result.TestConnection.UsersFound = 450
		result.TestConnection.GroupsFound = 32
		result.LastSync.Timestamp = "2025-01-15T08:00:00Z"
		result.LastSync.Status = "success"
		result.LastSync.Errors = 0
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req LDAPSyncConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
