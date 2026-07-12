package server

import (
	"net/http"
	"strings"
	"sync"
)

// migrationData holds the data needed for migrating an OAuth client.
type migrationData struct {
	ClientID       string            `json:"client_id"`
	ClientName     string            `json:"client_name"`
	CurrentConfig  map[string]any    `json:"current_config"`
	Dependencies   []map[string]any  `json:"dependencies"`
	ImpactAnalysis map[string]any    `json:"impact_analysis"`
	CompatFlags    map[string]bool   `json:"compatibility_flags"`
}

var migrationDataStore = struct {
	sync.RWMutex
	data map[string]*migrationData
}{data: map[string]*migrationData{
	"web-app": {
		ClientID: "web-app", ClientName: "Web Application",
		CurrentConfig: map[string]any{
			"grant_types":       []string{"authorization_code", "refresh_token"},
			"response_types":    []string{"code"},
			"scopes":            []string{"openid", "profile", "email"},
			"redirect_uris":     []string{"https://app.example.com/callback"},
			"token_endpoint_auth": "client_secret_post",
			"subject_type":       "public",
		},
		Dependencies: []map[string]any{
			{"type": "idp", "name": "primary-okta", "required": true},
			{"type": "database", "name": "user-store-postgres", "required": true},
			{"type": "cache", "name": "session-redis", "required": true},
			{"type": "webhook", "name": "provisioning-hook", "required": false},
		},
		ImpactAnalysis: map[string]any{
			"active_users_affected":   15420,
			"active_tokens_affected":  320,
			"estimated_downtime_min":  5,
			"rollback_possible":       true,
			"breaking_changes":        []string{"token format change", "session invalidation required"},
		},
		CompatFlags: map[string]bool{
			"oidc_compliant":          true,
			"supports_pkce":           true,
			"supports_dpop":           false,
			"requires_mtls":           false,
		},
	},
}}

// GET /api/v1/oauth/clients/{id}/migration-data
// Returns migration data for an OAuth client: config, dependencies, impact analysis.
func handleClientMigrationData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract client ID from path
	path := r.URL.Path
	clientID := strings.TrimPrefix(path, "/api/v1/oauth/clients/")
	clientID = strings.TrimSuffix(clientID, "/migration-data")
	clientID = strings.TrimSuffix(clientID, "/")
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
		return
	}

	migrationDataStore.RLock()
	data, exists := migrationDataStore.data[clientID]
	migrationDataStore.RUnlock()

	if !exists {
		// Return generic data for unknown clients
		writeJSON(w, http.StatusOK, map[string]any{
			"client_id":   clientID,
			"client_name": clientID,
			"current_config": map[string]any{
				"grant_types":     []string{"client_credentials"},
				"scopes":          []string{"read", "write"},
				"token_endpoint_auth": "client_secret_basic",
			},
			"dependencies":    []map[string]any{},
			"impact_analysis": map[string]any{
				"active_users_affected":  0,
				"estimated_downtime_min": 0,
				"rollback_possible":      true,
			},
			"compatibility_flags": map[string]bool{
				"oidc_compliant": true,
				"supports_pkce":  false,
			},
			"note": "generic migration data for unregistered client",
		})
		return
	}

	writeJSON(w, http.StatusOK, data)
}
