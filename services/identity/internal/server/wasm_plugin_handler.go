package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// WASMPlugin represents a WebAssembly plugin.
type WASMPlugin struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Status    string    `json:"status"` // enabled, disabled
	Size      int64     `json:"size_bytes"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

// GET /api/v1/plugins — list plugins
// POST /api/v1/plugins/upload — upload plugin
// POST /api/v1/plugins/:name/enable
// POST /api/v1/plugins/:name/disable
// DELETE /api/v1/plugins/:name
func (h *HTTPHandler) handlePluginsRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// POST /api/v1/plugins/upload
	if strings.HasSuffix(path, "/upload") && r.Method == http.MethodPost {
		// Parse multipart or JSON metadata.
		var meta struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&meta); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if meta.Name == "" {
			writeError(w, http.StatusBadRequest, "name required")
			return
		}
		plugin := WASMPlugin{
			ID: uuid.New().String(), Name: meta.Name, Version: meta.Version,
			Status: "enabled", CreatedAt: time.Now().UTC(),
		}
		writeJSON(w, http.StatusCreated, plugin)
		return
	}

	// GET /api/v1/plugins — list
	if path == "/api/v1/plugins" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"plugins": []WASMPlugin{}, "count": 0})
		return
	}

	// POST /api/v1/plugins/:name/enable or /disable
	if (strings.HasSuffix(path, "/enable") || strings.HasSuffix(path, "/disable")) && r.Method == http.MethodPost {
		action := "enabled"
		if strings.HasSuffix(path, "/disable") {
			action = "disabled"
		}
		name := strings.TrimSuffix(strings.TrimSuffix(path, "/enable"), "/disable")
		name = strings.TrimPrefix(name, "/api/v1/plugins/")
		writeJSON(w, http.StatusOK, map[string]any{"status": action, "plugin": name})
		return
	}

	// DELETE /api/v1/plugins/:name
	if r.Method == http.MethodDelete {
		name := strings.TrimPrefix(path, "/api/v1/plugins/")
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "plugin": name})
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
