package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
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

func (s *HTTPServer) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var result []map[string]any
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_snapshots")
			result = rows
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"snapshots": result, "count": len(result)})
		return
	}
	if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/rollback") {
		snapID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/policies/snapshots/"), "/rollback")
		if s.policyMap != nil {
			sn, _ := s.policyMap.Get(r.Context(), "policy_snapshots", snapID)
			if sn == nil {
				writeJSONError(w, http.StatusNotFound, "snapshot not found")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"status": "rolled_back", "snapshot_id": snapID,
				"restored_version": pmGetString(sn, "version"),
				"rolled_back_at": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "rolled_back", "snapshot_id": snapID})
		return
	}
	if r.Method == http.MethodPost {
		var req struct{ PolicyID string `json:"policy_id"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		sn := &PolicySnapshot{
			ID: uuid.New().String(), PolicyID: req.PolicyID, Version: 1,
			State: "captured", CreatedAt: time.Now().UTC(), CreatedBy: "system",
		}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_snapshots", sn.ID, map[string]any{
				"policy_id": sn.PolicyID, "version": sn.Version,
				"state": sn.State, "created_by": sn.CreatedBy,
			})
		}
		writeJSON(w, http.StatusCreated, sn)
		return
	}
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}
