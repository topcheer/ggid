package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ComplianceGap struct {
	ID               string     `json:"id"`
	Framework        string     `json:"framework"`
	ControlID        string     `json:"control_id"`
	GapDescription   string     `json:"gap_description"`
	RemediationPlan  string     `json:"remediation_plan"`
	Owner            string     `json:"owner"`
	DueDate          *time.Time `json:"due_date,omitempty"`
	Status           string     `json:"status"` // open, in_progress, resolved
}

var (
	gapMu sync.RWMutex
	gaps  = []ComplianceGap{
		{ID: "gap-001", Framework: "soc2", ControlID: "CC6.5", GapDescription: "Data at rest encryption not enabled for all databases", RemediationPlan: "Enable AES-256 on all PostgreSQL instances", Owner: "devops", Status: "in_progress"},
		{ID: "gap-002", Framework: "soc2", ControlID: "CC8.1", GapDescription: "Change management lacks formal approval workflow", RemediationPlan: "Implement GitOps approval gates", Owner: "engineering", Status: "open"},
		{ID: "gap-003", Framework: "gdpr", ControlID: "Art.17", GapDescription: "Right to erasure automation incomplete", RemediationPlan: "Complete GDPR forget endpoint integration", Owner: "backend", Status: "resolved"},
	}
)

// GET /api/v1/audit/compliance/gaps?framework=soc2
// POST /api/v1/audit/compliance/gaps/{id}/update
func (s *HTTPServer) handleComplianceGaps(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		framework := r.URL.Query().Get("framework")
		gapMu.RLock()
		result := []ComplianceGap{}
		for _, g := range gaps {
			if framework != "" && g.Framework != framework {
				continue
			}
			result = append(result, g)
		}
		gapMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"gaps": result, "count": len(result)})
		return
	}

	if r.Method == http.MethodPost {
		// Update gap status
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		gapID := ""
		if len(parts) >= 5 {
			gapID = parts[4]
		}
		var req struct{ Status string `json:"status"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid request body"); return }
		gapMu.Lock()
		for i := range gaps {
			if gaps[i].ID == gapID {
				gaps[i].Status = req.Status
				break
			}
		}
		gapMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "gap_id": gapID, "new_status": req.Status})
		return
	}

	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}
