package httpserver

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// collectSchedule defines a recurring evidence collection schedule.
type collectSchedule struct {
	ID          string   `json:"id"`
	Framework   string   `json:"framework"`
	Frequency   string   `json:"frequency"` // daily, weekly, monthly
	ControlIDs  []string `json:"control_ids"`
	Status      string   `json:"status"` // active, paused, completed
	NextRun     string   `json:"next_run"`
	LastRun     string   `json:"last_run,omitempty"`
	CreatedAt   string   `json:"created_at"`
	AutoUpload  bool     `json:"auto_upload"`
}

var collectScheduleStore = struct {
	sync.RWMutex
	schedules map[string]*collectSchedule
}{schedules: make(map[string]*collectSchedule)}

// POST /api/v1/audit/compliance/schedule-collect — create a collection schedule
// GET  /api/v1/audit/compliance/schedule-collect — list schedules
// DELETE /api/v1/audit/compliance/schedule-collect?id=X — delete a schedule
func (s *HTTPServer) handleScheduleCollect(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Framework  string   `json:"framework"`
			Frequency  string   `json:"frequency"`
			ControlIDs []string `json:"control_ids"`
			AutoUpload bool     `json:"auto_upload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Framework == "" {
			req.Framework = "soc2"
		}

		validFreqs := map[string]bool{"daily": true, "weekly": true, "monthly": true}
		if !validFreqs[req.Frequency] {
			req.Frequency = "weekly"
		}
		if len(req.ControlIDs) == 0 {
			req.ControlIDs = []string{"CC1.1", "CC2.1", "CC6.1", "CC7.1"}
		}

		// Compute next run time
		now := time.Now().UTC()
		var nextRun time.Time
		switch req.Frequency {
		case "daily":
			nextRun = now.Add(24 * time.Hour)
		case "weekly":
			nextRun = now.Add(7 * 24 * time.Hour)
		case "monthly":
			nextRun = now.AddDate(0, 1, 0)
		}

		sched := &collectSchedule{
			ID:         uuid.New().String(),
			Framework:  req.Framework,
			Frequency:  req.Frequency,
			ControlIDs: req.ControlIDs,
			Status:     "active",
			NextRun:    nextRun.Format(time.RFC3339),
			AutoUpload: req.AutoUpload,
			CreatedAt:  now.Format(time.RFC3339),
		}

		collectScheduleStore.Lock()
		collectScheduleStore.schedules[sched.ID] = sched
		collectScheduleStore.Unlock()

		writeJSON(w, http.StatusCreated, sched)

	case http.MethodGet:
		collectScheduleStore.RLock()
		result := []*collectSchedule{}
		for _, sched := range collectScheduleStore.schedules {
			result = append(result, sched)
		}
		collectScheduleStore.RUnlock()

		// Sort by next run
		sort.Slice(result, func(i, j int) bool {
			return result[i].NextRun < result[j].NextRun
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"schedules":     result,
			"total":         len(result),
			"active_count":  countActive(result),
		})

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id query parameter is required")
			return
		}

		collectScheduleStore.Lock()
		_, exists := collectScheduleStore.schedules[id]
		if exists {
			delete(collectScheduleStore.schedules, id)
		}
		collectScheduleStore.Unlock()

		if !exists {
			writeJSONError(w, http.StatusNotFound, "schedule not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
			"id":      id,
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func countActive(schedules []*collectSchedule) int {
	count := 0
	for _, s := range schedules {
		if s.Status == "active" {
			count++
		}
	}
	return count
}
