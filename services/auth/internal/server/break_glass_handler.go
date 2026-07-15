package server

import (
	"net/http"
	"sync"
	"time"
)

// BreakGlassRecord represents an emergency access event.
type BreakGlassRecord struct {
	ID                string    `json:"id"`
	Requester         string    `json:"requester"`
	RequesterName     string    `json:"requester_name"`
	Reason            string    `json:"reason"`
	Scope             string    `json:"scope"`
	DurationMinutes   int       `json:"duration_minutes"`
	ActivatedAt       time.Time `json:"activated_at"`
	DeactivatedAt     time.Time `json:"deactivated_at"`
	Status            string    `json:"status"` // active, expired
}

var (
	breakGlassMu      sync.RWMutex
	breakGlassRecords = []BreakGlassRecord{}
)

// GET /api/v1/auth/break-glass/history
func (h *Handler) handleBreakGlassHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	breakGlassMu.RLock()
	records := make([]BreakGlassRecord, len(breakGlassRecords))
	copy(records, breakGlassRecords)
	breakGlassMu.RUnlock()

	writeJSON(w, http.StatusOK, records)
}
