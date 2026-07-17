package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// resourceCache is a read-through cache for resource allow lists (hot path).
var resourceCache sync.Map // clientID → map[string]bool

// SetAllowedResources configures allowed resources for a client.
func SetAllowedResources(clientID string, resources []string) {
	m := make(map[string]bool)
	for _, r := range resources {
		m[r] = true
	}
	resourceCache.Store(clientID, m)
	if mapRepoVar != nil {
		mapRepoVar.Store(nil, "oauth_resource_allow", clientID, map[string]any{"resources": resources})
	}
}

// isResourceAllowed checks if a resource is allowed for a client.
func isResourceAllowed(clientID, resource string) bool {
	if v, ok := resourceCache.Load(clientID); ok {
		return v.(map[string]bool)[resource]
	}
	// Cache miss — try PG.
	if mapRepoVar != nil {
		if row, _ := mapRepoVar.Get(nil, "oauth_resource_allow", clientID); row != nil {
			if resources, ok := row["resources"].([]any); ok {
				m := make(map[string]bool)
				for _, r := range resources {
					m[fmt.Sprintf("%v", r)] = true
				}
				resourceCache.Store(clientID, m)
				return m[resource]
			}
		}
	}
	return true // allow by default if no config
}

// POST /api/v1/oauth/resource-indicator — RFC 8707 resource indicator validation.
func handleResourceIndicator(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ClientID string `json:"client_id"`
		Resource string `json:"resource"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if req.ClientID == "" || req.Resource == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id and resource are required"})
		return
	}
	allowed := isResourceAllowed(req.ClientID, req.Resource)
	writeJSON(w, http.StatusOK, map[string]any{"client_id": req.ClientID, "resource": req.Resource, "allowed": allowed})
}

// POST /api/v1/oauth/resource-allowed — configure allowed resources for a client.
func handleResourceAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ClientID  string   `json:"client_id"`
		Resources []string `json:"resources"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if req.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id is required"})
		return
	}
	SetAllowedResources(req.ClientID, req.Resources)
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "client_id": req.ClientID, "resources": req.Resources, "updated_at": time.Now().UTC()})
}
