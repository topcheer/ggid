package server

import (
	"net/http"
	"sync"
	"time"
)

// syncHistoryEntry represents a single sync run result.
type syncHistoryEntry struct {
	ID         string    `json:"id"`
	StartedAt  time.Time `json:"started_at"`
	Status     string    `json:"status"`
	SyncedUsers int      `json:"synced_users"`
	Errors     int       `json:"errors"`
	Duration   string    `json:"duration"`
}

var (
	syncHistoryMu sync.RWMutex
	syncHistory   []syncHistoryEntry
)

// addSyncHistory records a completed sync run.
func addSyncHistory(entry syncHistoryEntry) {
	syncHistoryMu.Lock()
	defer syncHistoryMu.Unlock()
	syncHistory = append(syncHistory, entry)
	if len(syncHistory) > 50 {
		syncHistory = syncHistory[len(syncHistory)-50:]
	}
}

// GET /api/v1/identity/ldap/sync-history
func (h *HTTPHandler) handleLDAPSyncHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	syncHistoryMu.RLock()
	history := make([]syncHistoryEntry, len(syncHistory))
	copy(history, syncHistory)
	syncHistoryMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"history": history,
		"total":   len(history),
	})
}
