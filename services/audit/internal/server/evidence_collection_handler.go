package httpserver

import (
	"net/http"
	"strings"
)

// EvidenceItem represents a compliance evidence record.
type EvidenceItem struct {
	ControlID   string `json:"control_id"`
	Framework   string `json:"framework"`
	Description string `json:"description"`
	Status      string `json:"status"` // collected, pending, missing
	CollectedAt string `json:"collected_at"`
}

// GET /api/v1/audit/evidence-collection?framework=X
// POST /api/v1/audit/evidence-collection/{id}/upload
func (s *HTTPServer) handleEvidenceCollection(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/v1/audit/evidence-collection/") && r.Method == http.MethodPost {
		// Upload evidence for a control
		controlID := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/evidence-collection/")
		controlID = strings.TrimSuffix(controlID, "/upload")
		writeJSON(w, http.StatusOK, map[string]any{
			"control_id": controlID,
			"status":     "uploaded",
		})
		return
	}

	if r.Method == http.MethodGet {
		framework := r.URL.Query().Get("framework")
		evidence := []EvidenceItem{
			{ControlID: "CC1.1", Framework: framework, Description: "Access control policy", Status: "collected", CollectedAt: "2025-01-15"},
			{ControlID: "CC1.2", Framework: framework, Description: "User provisioning", Status: "pending", CollectedAt: ""},
			{ControlID: "CC6.1", Framework: framework, Description: "Logical access", Status: "collected", CollectedAt: "2025-01-14"},
		}
		writeJSON(w, http.StatusOK, evidence)
		return
	}

	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}
