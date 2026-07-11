package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ClientVersion struct {
	Version   int                    `json:"version"`
	Config    map[string]any         `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
	Note      string                 `json:"note,omitempty"`
}

var (
	clientVerMu sync.RWMutex
	clientVersions = make(map[string][]ClientVersion) // clientID → versions
)

// POST /api/v1/oauth/clients/{id}/version
// GET /api/v1/oauth/clients/{id}/versions
func handleClientVersioning(w http.ResponseWriter, r *http.Request) {
	clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	if strings.HasSuffix(clientID, "/versions") {
		clientID = strings.TrimSuffix(clientID, "/versions")
	} else if strings.HasSuffix(clientID, "/version") {
		clientID = strings.TrimSuffix(clientID, "/version")
	}
	if clientID == "" || strings.Contains(clientID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid client_id"}); return
	}
	if strings.HasSuffix(r.URL.Path, "/versions") && r.Method == http.MethodGet {
		clientVerMu.RLock()
		versions := clientVersions[clientID]
		clientVerMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "versions": versions, "total": len(versions)})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/version") && r.Method == http.MethodPost {
		var req struct {
			Config map[string]any `json:"config"`
			Note   string         `json:"note"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		clientVerMu.Lock()
		ver := ClientVersion{Version: len(clientVersions[clientID]) + 1, Config: req.Config, CreatedAt: time.Now().UTC(), Note: req.Note}
		clientVersions[clientID] = append(clientVersions[clientID], ver)
		clientVerMu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{"status": "versioned", "client_id": clientID, "version": ver})
		return
	}
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
}
