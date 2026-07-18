package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// deprovisionSchedule holds a scheduled future deactivation.
type deprovisionSchedule struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	ScheduledAt      string   `json:"scheduled_at"`
	Reason           string   `json:"reason"`
	NotifyBeforeDays int      `json:"notify_before_days"`
	CascadeToApps    []string `json:"cascade_to_apps"`
	Status           string   `json:"status"` // scheduled, executed, cancelled
	CreatedAt        string   `json:"created_at"`
}

var deprovisionStore = struct {
	sync.RWMutex
	data map[string]*deprovisionSchedule
}{data: make(map[string]*deprovisionSchedule)}

// POST /api/v1/identity/users/{id}/deprovision-schedule
// GET  /api/v1/identity/users/{id}/deprovision-schedule — list schedules for user
func (h *HTTPHandler) handleDeprovisionSchedule(w http.ResponseWriter, r *http.Request) {
	// Extract user ID
	path := r.URL.Path
	userID := ""
	if idx := strings.Index(path, "/users/"); idx >= 0 {
		rest := path[idx+len("/users/"):]
		if dIdx := strings.Index(rest, "/deprovision-schedule"); dIdx >= 0 {
			userID = rest[:dIdx]
		}
	}
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user ID is required in path")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req struct {
			ScheduledAt      string   `json:"scheduled_at"`
			Reason           string   `json:"reason"`
			NotifyBeforeDays int      `json:"notify_before_days"`
			CascadeToApps    []string `json:"cascade_to_apps"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.ScheduledAt == "" {
			req.ScheduledAt = time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
		}
		if req.Reason == "" {
			req.Reason = "offboarding"
		}
		if req.NotifyBeforeDays == 0 {
			req.NotifyBeforeDays = 7
		}

		sched := &deprovisionSchedule{
			ID: uuid.New().String(), UserID: userID,
			ScheduledAt: req.ScheduledAt, Reason: req.Reason,
			NotifyBeforeDays: req.NotifyBeforeDays, CascadeToApps: req.CascadeToApps,
			Status: "scheduled", CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		deprovisionStore.Lock()
		deprovisionStore.data[sched.ID] = sched
		deprovisionStore.Unlock()

		writeJSON(w, http.StatusCreated, sched)

	case http.MethodGet:
		deprovisionStore.RLock()
		result := []*deprovisionSchedule{}
		for _, s := range deprovisionStore.data {
			if s.UserID == userID {
				result = append(result, s)
			}
		}
		deprovisionStore.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":    userID,
			"schedules":  result,
			"total":      len(result),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
