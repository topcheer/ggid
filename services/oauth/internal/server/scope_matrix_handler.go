package server

import (
	"net/http"
	"sort"
	"strings"
	"sync"
)

// scopeMatrixEntry represents one client's scope grants in the matrix.
type scopeMatrixEntry struct {
	ClientID   string         `json:"client_id"`
	ClientName string         `json:"client_name"`
	Scopes     map[string]bool `json:"scopes"` // scope → granted
}

var scopeMatrixStore = struct {
	sync.RWMutex
	clients map[string]*scopeMatrixEntry
}{clients: map[string]*scopeMatrixEntry{
	"web-app": {
		ClientID: "web-app", ClientName: "Web Application",
		Scopes: map[string]bool{"openid": true, "profile": true, "email": true, "offline_access": true, "read:users": true, "read:audit": false, "admin": false, "write:users": false},
	},
	"mobile-ios": {
		ClientID: "mobile-ios", ClientName: "iOS Mobile App",
		Scopes: map[string]bool{"openid": true, "profile": true, "email": true, "offline_access": true, "read:users": false, "read:audit": false, "admin": false, "write:users": false},
	},
	"admin-cli": {
		ClientID: "admin-cli", ClientName: "Admin CLI",
		Scopes: map[string]bool{"openid": true, "profile": true, "email": true, "offline_access": false, "read:users": true, "read:audit": true, "admin": true, "write:users": true},
	},
	"service-backend": {
		ClientID: "service-backend", ClientName: "Backend Service",
		Scopes: map[string]bool{"openid": false, "profile": false, "email": false, "offline_access": false, "read:users": true, "read:audit": true, "admin": false, "write:users": true},
	},
}}

// GET /api/v1/oauth/clients/scope-matrix
// Returns a clients × scopes grid showing which scopes each client has.
func handleScopeMatrix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	scopeMatrixStore.RLock()
	defer scopeMatrixStore.RUnlock()

	// Collect all unique scopes across all clients
	allScopes := map[string]bool{}
	for _, c := range scopeMatrixStore.clients {
		for s := range c.Scopes {
			allScopes[s] = true
		}
	}

	// Sort scopes
	scopeList := make([]string, 0, len(allScopes))
	for s := range allScopes {
		scopeList = append(scopeList, s)
	}
	sort.Strings(scopeList)

	// Build matrix
	matrix := make([]map[string]any, 0, len(scopeMatrixStore.clients))
	for _, c := range scopeMatrixStore.clients {
		row := map[string]any{
			"client_id":   c.ClientID,
			"client_name": c.ClientName,
		}
		grantedCount := 0
		for _, scope := range scopeList {
			granted := c.Scopes[scope]
			if granted {
				grantedCount++
			}
			row[scope] = granted
		}
		row["_granted_count"] = grantedCount
		row["_total_scopes"] = len(scopeList)
		matrix = append(matrix, row)
	}

	// Compute summary stats
	scopeUsage := map[string]int{} // scope → number of clients with access
	for _, scope := range scopeList {
		for _, c := range scopeMatrixStore.clients {
			if c.Scopes[scope] {
				scopeUsage[scope]++
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"scopes":         scopeList,
		"matrix":         matrix,
		"total_clients":  len(matrix),
		"total_scopes":   len(scopeList),
		"scope_usage":    scopeUsage,
		"filter": func() string {
			if f := r.URL.Query().Get("filter"); f != "" {
				return f
			}
			return ""
		}(),
	})
}

// suppress unused import
var _ = strings.Contains
