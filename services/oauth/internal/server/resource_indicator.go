package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// resourceAllowList tracks allowed resource servers per OAuth client.
type resourceAllowList struct {
	mu        sync.RWMutex
	allowed   map[string]map[string]bool // client_id → set of resource URLs
}

var resourceAllow = &resourceAllowList{allowed: make(map[string]map[string]bool)}

// SetAllowedResources configures allowed resources for a client.
func SetAllowedResources(clientID string, resources []string) {
	resourceAllow.mu.Lock()
	defer resourceAllow.mu.Unlock()
	resourceAllow.allowed[clientID] = make(map[string]bool)
	for _, r := range resources {
		resourceAllow.allowed[clientID][r] = true
	}
}

// POST /api/v1/oauth/resource-indicator — RFC 8707 resource indicator validation.
// Body: {"client_id": "...", "resource": "https://api.example.com"}
// Validates that the resource is in the client's allowed list.
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
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
		return
	}
	if req.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id is required"})
		return
	}
	if req.Resource == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "resource is required"})
		return
	}

	// Check allowlist
	resourceAllow.mu.RLock()
	allowed, hasList := resourceAllow.allowed[req.ClientID]
	resourceAllow.mu.RUnlock()

	allowedFlag := true
	if hasList {
		allowedFlag = allowed[req.Resource]
	}

	if !allowedFlag {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"valid":    false,
			"error":    "invalid_target",
			"resource": req.Resource,
			"detail":   "resource not in client allowed list",
		})
		return
	}

	// Return token with resource binding
	writeJSON(w, http.StatusOK, map[string]any{
		"valid":         true,
		"client_id":     req.ClientID,
		"resource":      req.Resource,
		"token_binding": "audience-restricted",
		"expires_in":    3600,
		"issued_at":     time.Now().UTC().Format(time.RFC3339),
	})
}

// GET /api/v1/oauth/resource-allowed?client_id=X — list allowed resources for a client.
func handleResourceAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	clientID := r.URL.Query().Get("client_id")
	resourceAllow.mu.RLock()
	defer resourceAllow.mu.RUnlock()

	if clientID != "" {
		allowed := resourceAllow.allowed[clientID]
		result := make([]string, 0, len(allowed))
		for r := range allowed {
			result = append(result, r)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"client_id": clientID,
			"resources": result,
		})
		return
	}

	// List all
	allClients := make(map[string][]string)
	for cid, allowed := range resourceAllow.allowed {
		result := make([]string, 0, len(allowed))
		for r := range allowed {
			result = append(result, r)
		}
		allClients[cid] = result
	}
	writeJSON(w, http.StatusOK, map[string]any{"clients": allClients})
}
