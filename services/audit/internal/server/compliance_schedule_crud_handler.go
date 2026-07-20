package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

type complianceScheduleStore struct{}

var globalSchedules = &complianceSchedule{
	schedules: []map[string]any{},
}

type complianceSchedule struct {
	mu        sync.RWMutex
	schedules []map[string]any
}

// POST/GET/DELETE /api/v1/audit/compliance-schedules
// DB-backed: uses compliance_schedules table. Falls back to in-memory when pool is nil.
func (s *HTTPServer) handleComplianceScheduleCRUD(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if s.pool != nil {
			rows, err := s.pool.Query(r.Context(), `
				SELECT id::text, data::text, created_at FROM compliance_schedules ORDER BY created_at DESC`)
			if err == nil {
				defer rows.Close()
				schedules := []map[string]any{}
				for rows.Next() {
					var id, dataStr string
					var createdAt interface{}
					_ = rows.Scan(&id, &dataStr, &createdAt)
					var m map[string]any
					_ = json.Unmarshal([]byte(dataStr), &m)
					m["id"] = id
					schedules = append(schedules, m)
				}
				writeJSON(w, http.StatusOK, map[string]any{"schedules": schedules})
				return
			}
		}
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
		schedID := uuid.New().String()
		sched := map[string]any{
			"id":          schedID,
			"report_type": req.ReportType,
			"frequency":   req.Frequency,
			"recipients":  req.Recipients,
			"active":      true,
		}
		if s.pool != nil {
			dataJSON, _ := json.Marshal(sched)
			_, err := s.pool.Exec(r.Context(), `
				INSERT INTO compliance_schedules (id, data) VALUES ($1, $2)`, schedID, dataJSON)
			if err != nil {
				// Table might not exist — create it
				_, _ = s.pool.Exec(r.Context(), `
					CREATE TABLE IF NOT EXISTS compliance_schedules (
						id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT NOW()
					)`)
				_, err = s.pool.Exec(r.Context(), `INSERT INTO compliance_schedules (id, data) VALUES ($1, $2)`, schedID, dataJSON)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
					return
				}
			}
		} else {
			globalSchedules.mu.Lock()
			globalSchedules.schedules = append(globalSchedules.schedules, sched)
			globalSchedules.mu.Unlock()
		}
		writeJSON(w, http.StatusCreated, sched)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if s.pool != nil {
			_, _ = s.pool.Exec(r.Context(), `DELETE FROM compliance_schedules WHERE id = $1`, id)
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
			return
		}
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
