package server

import (
	"net/http"
	"strings"
	"sync"
)

// scopeDriftData holds scope usage vs registration data.
type scopeDriftData struct {
	ClientID              string   `json:"client_id"`
	RegisteredScopes      []string `json:"registered_scopes"`
	ActiveScopes          []string `json:"active_scopes"`
	UnusedScopes          []string `json:"unused_scopes"`
	UnregisteredUsed      []string `json:"unregistered_scopes_used"`
}

var scopeDriftStore = struct {
	sync.RWMutex
	data map[string]*scopeDriftData
}{data: map[string]*scopeDriftData{
	"web-app": {
		ClientID: "web-app",
		RegisteredScopes: []string{"openid", "profile", "email", "offline_access", "read:users", "read:audit"},
		ActiveScopes:     []string{"openid", "profile", "email", "read:users"},
		UnusedScopes:     []string{"offline_access", "read:audit"},
		UnregisteredUsed: []string{},
	},
	"mobile-ios": {
		ClientID: "mobile-ios",
		RegisteredScopes: []string{"openid", "profile", "email", "offline_access"},
		ActiveScopes:     []string{"openid", "profile", "email", "offline_access", "write:users"},
		UnusedScopes:     []string{},
		UnregisteredUsed: []string{"write:users"},
	},
	"admin-cli": {
		ClientID: "admin-cli",
		RegisteredScopes: []string{"openid", "profile", "email", "admin", "read:users", "read:audit", "write:policies"},
		ActiveScopes:     []string{"openid", "admin", "read:users", "read:audit"},
		UnusedScopes:     []string{"profile", "email", "write:policies"},
		UnregisteredUsed: []string{},
	},
}}

// GET /api/v1/oauth/clients/{id}/scope-drift
func handleScopeDrift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	clientID = strings.TrimSuffix(clientID, "/scope-drift")
	clientID = strings.TrimSuffix(clientID, "/")
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
		return
	}

	scopeDriftStore.RLock()
	data, exists := scopeDriftStore.data[clientID]
	scopeDriftStore.RUnlock()

	if !exists {
		writeJSON(w, http.StatusOK, map[string]any{
			"client_id":   clientID,
			"message":     "no drift data available for this client",
			"unused_scopes":         []string{},
			"unregistered_scopes_used": []string{},
			"drift_severity": "none",
		})
		return
	}

	// Determine severity
	severity := "none"
	if len(data.UnregisteredUsed) > 0 {
		severity = "high"
	} else if len(data.UnusedScopes) > 2 {
		severity = "medium"
	} else if len(data.UnusedScopes) > 0 {
		severity = "low"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":                  data.ClientID,
		"registered_scopes":          data.RegisteredScopes,
		"active_scopes":              data.ActiveScopes,
		"unused_scopes":              data.UnusedScopes,
		"unregistered_scopes_used":   data.UnregisteredUsed,
		"unused_count":               len(data.UnusedScopes),
		"unregistered_count":         len(data.UnregisteredUsed),
		"drift_severity":             severity,
		"recommendation": func() string {
			if len(data.UnregisteredUsed) > 0 {
				return "revoke_unregistered_scope_usage_and_register_or_remove"
			}
			if len(data.UnusedScopes) > 0 {
				return "consider_removing_unused_scopes_from_registration"
			}
			return "no_action_needed"
		}(),
	})
}
