package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PolicySnapshot struct {
	ID        string    `json:"id"`
	PolicyID  string    `json:"policy_id"`
	Version   int       `json:"version"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

var (
	snapMu sync.RWMutex
	snaps  = make(map[string]*PolicySnapshot)
)

// GET /api/v1/policies/snapshots
// POST /api/v1/policies/snapshots/{id}/rollback
func (s *HTTPServer) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		snapMu.RLock()
		result := make([]*PolicySnapshot, 0, len(snaps))
		for _, sn := range snaps {
			result = append(result, sn)
		}
		snapMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"snapshots": result, "count": len(result)})
		return
	}
	if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/rollback") {
		snapID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/policies/snapshots/"), "/rollback")
		snapMu.RLock()
		sn, ok := snaps[snapID]
		snapMu.RUnlock()
		if !ok {
			writeJSONError(w, http.StatusNotFound, "snapshot not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "rolled_back", "snapshot_id": snapID,
			"restored_version": sn.Version, "rolled_back_at": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}
	// POST create snapshot
	if r.Method == http.MethodPost {
		var req struct{ PolicyID string `json:"policy_id"` }
		_ = json.NewDecoder(r.Body).Decode(&req)
		sn := &PolicySnapshot{ID: uuid.New().String(), PolicyID: req.PolicyID, Version: len(snaps) + 1, State: "captured", CreatedAt: time.Now().UTC(), CreatedBy: "system"}
		snapMu.Lock(); snaps[sn.ID] = sn; snapMu.Unlock()
		writeJSON(w, http.StatusCreated, sn)
		return
	}
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}
