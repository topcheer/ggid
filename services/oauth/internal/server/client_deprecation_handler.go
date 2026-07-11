package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ClientDeprecation struct {
	ClientID           string     `json:"client_id"`
	Deprecated         bool       `json:"deprecated"`
	SunsetDate         *time.Time `json:"sunset_date,omitempty"`
	MigrationGuideURL  string     `json:"migration_guide_url,omitempty"`
	DeprecationNotice  string     `json:"deprecation_notice,omitempty"`
	MarkedAt           time.Time  `json:"marked_at"`
}

var (
	deprecationMu sync.RWMutex
	deprecations  = make(map[string]*ClientDeprecation)
)

// PUT /api/v1/oauth/clients/{id}/deprecation
// GET /api/v1/oauth/clients/{id}/deprecation
func handleClientDeprecation(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/deprecation") {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	clientID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/"), "/deprecation")
	if clientID == "" || strings.Contains(clientID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid client_id"})
		return
	}

	switch r.Method {
	case http.MethodPut, http.MethodPost:
		var req struct {
			SunsetDate        string `json:"sunset_date"`
			MigrationGuideURL string `json:"migration_guide_url"`
			DeprecationNotice string `json:"deprecation_notice"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		dep := &ClientDeprecation{
			ClientID: clientID, Deprecated: true, MigrationGuideURL: req.MigrationGuideURL,
			DeprecationNotice: req.DeprecationNotice, MarkedAt: time.Now().UTC(),
		}
		if req.SunsetDate != "" {
			t, _ := time.Parse(time.RFC3339, req.SunsetDate)
			dep.SunsetDate = &t
		}
		deprecationMu.Lock()
		deprecations[clientID] = dep
		deprecationMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "deprecated", "deprecation": dep,
			"note": "token responses will include deprecation_warning header",
		})
	case http.MethodGet:
		deprecationMu.RLock()
		dep, ok := deprecations[clientID]
		deprecationMu.RUnlock()
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "deprecated": false})
			return
		}
		writeJSON(w, http.StatusOK, dep)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// GetClientDeprecation returns deprecation status for header injection (internal)
func GetClientDeprecation(clientID string) *ClientDeprecation {
	deprecationMu.RLock()
	defer deprecationMu.RUnlock()
	return deprecations[clientID]
}
