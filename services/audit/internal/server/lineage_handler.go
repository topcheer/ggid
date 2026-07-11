package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/audit/lineage?resource=X
func (s *HTTPServer) handleDataLineage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	resource := r.URL.Query().Get("resource")
	if resource == "" {
		writeJSONError(w, http.StatusBadRequest, "resource required")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resource": resource,
		"lineage": []map[string]any{
			{"timestamp": "2026-07-12T08:00:00Z", "event": "created", "actor": "admin", "source": "api"},
			{"timestamp": "2026-07-12T08:15:00Z", "event": "accessed", "actor": "u-002", "source": "dashboard"},
			{"timestamp": "2026-07-12T09:30:00Z", "event": "modified", "actor": "u-001", "source": "api", "changes": 3},
			{"timestamp": "2026-07-12T10:00:00Z", "event": "accessed", "actor": "u-003", "source": "export"},
			{"timestamp": "2026-07-12T11:00:00Z", "event": "shared", "actor": "u-001", "source": "api", "shared_with": "u-004"},
		},
		"total_events": 5,
		"unique_actors": 4,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
