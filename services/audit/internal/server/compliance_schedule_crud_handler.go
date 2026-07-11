package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

type complianceSchedule struct {
	mu         sync.RWMutex
	schedules  []map[string]any
}

var globalSchedules = &complianceSchedule{}

// POST/GET/PUT/DELETE /api/v1/audit/compliance-schedules
func (s *HTTPServer) handleComplianceScheduleCRUD(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		globalSchedules.mu.RLock()
		defer globalSchedules.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"schedules": globalSchedules.schedules})

	case http.MethodPost:
		var req struct {
			ReportType string   `json:"report_type"`
			Frequency  string   `json:"frequency"`
			Recipients []string `json:"recipients"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		sched := map[string]any{
			"id":         uuid.New().String(),
			"report_type": req.ReportType,
			"frequency":   req.Frequency,
			"recipients":  req.Recipients,
			"next_run_at": "weekly",
			"active":      true,
		}
		globalSchedules.mu.Lock()
		globalSchedules.schedules = append(globalSchedules.schedules, sched)
		globalSchedules.mu.Unlock()
		writeJSON(w, http.StatusCreated, sched)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		globalSchedules.mu.Lock()
		defer globalSchedules.mu.Unlock()
		for i, sc := range globalSchedules.schedules {
			if sc["id"] == id {
				globalSchedules.schedules = append(globalSchedules.schedules[:i], globalSchedules.schedules[i+1:]...)
				writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
				return
			}
		}
		writeJSONError(w, http.StatusNotFound, "schedule not found")

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
