package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ComplianceEvidence tracks collected evidence for a compliance framework.
type ComplianceEvidence struct {
	ID         string    `json:"id"`
	Framework  string    `json:"framework"` // soc2, hipaa, gdpr
	ControlID  string    `json:"control_id"`
	Status     string    `json:"status"` // compliant, non_compliant, in_progress
	Artifacts  []string  `json:"artifacts"`
	Notes      string    `json:"notes,omitempty"`
	CollectedAt time.Time `json:"collected_at"`
	CollectedBy string   `json:"collected_by,omitempty"`
}

var (
	evidenceMu sync.RWMutex
	evidenceStore = make(map[string]*ComplianceEvidence)
)

// POST /api/v1/audit/compliance/evidence — collect evidence.
// GET /api/v1/audit/compliance/evidence?framework=soc2 — query evidence.
func (s *HTTPServer) handleComplianceEvidence(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Framework   string   `json:"framework"`
			ControlID   string   `json:"control_id"`
			Status      string   `json:"status"`
			Artifacts   []string `json:"artifacts"`
			Notes       string   `json:"notes"`
			CollectedBy string   `json:"collected_by"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Framework == "" || req.ControlID == "" {
			writeJSONError(w, http.StatusBadRequest, "framework and control_id are required")
			return
		}
		if req.Status == "" {
			req.Status = "compliant"
		}
		ev := &ComplianceEvidence{
			ID:          uuid.New().String(),
			Framework:   req.Framework,
			ControlID:   req.ControlID,
			Status:      req.Status,
			Artifacts:   req.Artifacts,
			Notes:       req.Notes,
			CollectedAt: time.Now().UTC(),
			CollectedBy: req.CollectedBy,
		}
		evidenceMu.Lock()
		evidenceStore[ev.ID] = ev
		evidenceMu.Unlock()
		writeJSON(w, http.StatusCreated, ev)

	case http.MethodGet:
		framework := r.URL.Query().Get("framework")
		controlID := r.URL.Query().Get("control_id")
		evidenceMu.RLock()
		result := []*ComplianceEvidence{}
		for _, ev := range evidenceStore {
			if framework != "" && ev.Framework != framework {
				continue
			}
			if controlID != "" && ev.ControlID != controlID {
				continue
			}
			result = append(result, ev)
		}
		evidenceMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"evidence": result,
			"count":    len(result),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
