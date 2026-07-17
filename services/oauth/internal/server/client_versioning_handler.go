package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// versionCache provides test/dev fallback when no DB is available.
var versionCache sync.Map // clientID → []ClientVersion

type ClientVersion struct {
	Version   int            `json:"version"`
	Config    map[string]any `json:"config"`
	CreatedAt time.Time      `json:"created_at"`
	Note      string         `json:"note,omitempty"`
}

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
		var versions []map[string]any
		if mapRepoVar != nil {
			rows, _ := mapRepoVar.List(r.Context(), "oauth_client_versions")
			for _, row := range rows {
				if cid, ok := row["client_id"].(string); ok && cid == clientID {
					versions = append(versions, row)
				}
			}
		}
		if versions == nil {
			// Cache fallback for tests without DB.
			if v, ok := versionCache.Load(clientID); ok {
				versions = v.([]map[string]any)
			}
		}
		if versions == nil { versions = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "versions": versions, "total": len(versions)})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/version") && r.Method == http.MethodPost {
		var req struct {
			Config map[string]any `json:"config"`
			Note   string         `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"}); return }

		versionNum := 1
		if mapRepoVar != nil {
			rows, _ := mapRepoVar.List(r.Context(), "oauth_client_versions")
			count := 0
			for _, row := range rows {
				if cid, ok := row["client_id"].(string); ok && cid == clientID {
					count++
				}
			}
			versionNum = count + 1
		}
		ver := ClientVersion{Version: versionNum, Config: req.Config, CreatedAt: time.Now().UTC(), Note: req.Note}
		if mapRepoVar != nil {
			verID := uuid.New().String()
			b, _ := json.Marshal(ver)
			var dataMap map[string]any
			json.Unmarshal(b, &dataMap)
			dataMap["client_id"] = clientID
			mapRepoVar.Store(r.Context(), "oauth_client_versions", verID, dataMap)
		}
		// Cache fallback.
		var existing []map[string]any
		if v, ok := versionCache.Load(clientID); ok {
			existing = v.([]map[string]any)
		}
		existing = append(existing, map[string]any{"version": ver.Version, "config": ver.Config, "note": ver.Note, "client_id": clientID})
		versionCache.Store(clientID, existing)
		writeJSON(w, http.StatusCreated, map[string]any{"status": "versioned", "client_id": clientID, "version": ver})
		return
	}
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
}
