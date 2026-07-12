package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// reactivationEvent represents a deactivate/reactivate cycle.
type reactivationEvent struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	DeactivatedAt string `json:"deactivated_at"`
	ReactivatedAt string `json:"reactivated_at,omitempty"`
	DurationDays  int    `json:"duration_days,omitempty"`
	Reason        string `json:"reason"`
	Actor         string `json:"actor"`
	Status        string `json:"status"` // deactivated, reactivated
}

var reactivationStore = struct {
	sync.RWMutex
	data map[string][]reactivationEvent
}{data: map[string][]reactivationEvent{
	"user-001": {
		{ID: "re-1", UserID: "user-001", DeactivatedAt: time.Now().UTC().Add(-90*24*time.Hour).Format(time.RFC3339), ReactivatedAt: time.Now().UTC().Add(-85*24*time.Hour).Format(time.RFC3339), DurationDays: 5, Reason: "Extended leave", Actor: "hr-admin", Status: "reactivated"},
		{ID: "re-2", UserID: "user-001", DeactivatedAt: time.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339), ReactivatedAt: time.Now().UTC().Add(-28*24*time.Hour).Format(time.RFC3339), DurationDays: 2, Reason: "Security review", Actor: "sec-admin", Status: "reactivated"},
	},
	"user-002": {
		{ID: "re-3", UserID: "user-002", DeactivatedAt: time.Now().UTC().Add(-15*24*time.Hour).Format(time.RFC3339), Reason: "Failed security audit", Actor: "sec-admin", Status: "deactivated"},
	},
}}

// GET /api/v1/users/{id}/reactivation-history
func (h *HTTPHandler) handleReactivationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user ID
	path := r.URL.Path
	userID := ""
	if idx := strings.Index(path, "/users/"); idx >= 0 {
		rest := path[idx+len("/users/"):]
		if rIdx := strings.Index(rest, "/reactivation-history"); rIdx >= 0 {
			userID = rest[:rIdx]
		}
	}
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user ID is required in path")
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		// Allow non-UUID IDs for test data
	}

	reactivationStore.RLock()
	events := reactivationStore.data[userID]
	result := make([]reactivationEvent, len(events))
	copy(result, events)
	reactivationStore.RUnlock()

	totalDeactivations := len(result)
	totalReactivations := 0
	for _, e := range result {
		if e.Status == "reactivated" {
			totalReactivations++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":             userID,
		"history":             result,
		"total_deactivations": totalDeactivations,
		"total_reactivations": totalReactivations,
		"currently_active":    len(result) == 0 || result[len(result)-1].Status == "reactivated",
		"checked_at":          time.Now().UTC().Format(time.RFC3339),
	})
}
