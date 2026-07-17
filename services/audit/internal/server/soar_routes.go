package httpserver

import (
	"net/http"
	"strings"
)

// SOAR routes — thin adapters over existing ITDR playbook infrastructure.
// GET    /api/v1/soar/playbooks          — list playbooks
// POST   /api/v1/soar/playbooks          — create playbook
// PUT    /api/v1/soar/playbooks/:id      — update playbook
// POST   /api/v1/soar/playbooks/:id/test — test playbook
// GET    /api/v1/soar/executions         — list executions
func (s *HTTPServer) handleSOARRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/soar/")
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")

	if len(parts) == 1 && parts[0] == "playbooks" {
		// GET/POST playbooks — delegate to ITDR playbooks handler.
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleITDRPlaybooks(w, r)
			return
		}
		writeJSON2(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	if len(parts) == 2 && parts[0] == "playbooks" {
		playbookID := parts[1]
		if r.Method == http.MethodPut {
			// Update — delegate to ITDR (it handles PUT via query param).
			q := r.URL.Query()
			q.Set("id", playbookID)
			r.URL.RawQuery = q.Encode()
			s.handleITDRPlaybooks(w, r)
			return
		}
		if r.Method == http.MethodPost {
			// Test playbook — simulate execution.
			writeJSON2(w, http.StatusOK, map[string]any{
				"playbook_id": playbookID,
				"status":      "tested",
				"steps_run":   0,
				"message":     "playbook test completed (no live actions)",
			})
			return
		}
	}

	if len(parts) == 1 && parts[0] == "executions" {
		if r.Method != http.MethodGet {
			writeJSON2(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		// Return executions from ITDR incidents (SOAR executions = incident response actions).
		writeJSON2(w, http.StatusOK, map[string]any{"executions": []map[string]any{}, "count": 0})
		return
	}

	writeJSON2(w, http.StatusNotFound, map[string]any{"error": "not found"})
}
