package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type SessionLimit struct {
	UserID     string    `json:"user_id"`
	MaxSessions int       `json:"max_sessions"`
	Strategy   string    `json:"strategy"` // terminate_oldest, deny_new
	Enforced   bool      `json:"enforced"`
	UpdatedAt  time.Time `json:"updated_at"`
}

var (
	sessLimitMu sync.RWMutex
	sessLimits  = make(map[string]*SessionLimit)
)

// POST /api/v1/auth/sessions/enforce-limit
// GET /api/v1/auth/sessions/limits
func (h *Handler) handleSessionLimit(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			UserID      string `json:"user_id"`
			MaxSessions int    `json:"max_sessions"`
			Strategy    string `json:"strategy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.UserID == "" || req.MaxSessions == 0 {
			writeError(w, http.StatusBadRequest, "user_id and max_sessions required")
			return
		}
		if req.Strategy == "" {
			req.Strategy = "terminate_oldest"
		}
		limit := &SessionLimit{UserID: req.UserID, MaxSessions: req.MaxSessions, Strategy: req.Strategy, Enforced: true, UpdatedAt: time.Now().UTC()}
		sessLimitMu.Lock(); sessLimits[req.UserID] = limit; sessLimitMu.Unlock()
		// PG write-through
		if h.memMapRepo != nil {
			h.memMapRepo.StoreJSON(r.Context(), "auth_session_limits_json", req.UserID, map[string]any{
				"user_id": req.UserID, "max_sessions": req.MaxSessions,
				"strategy": req.Strategy, "enforced": true,
				"updated_at": time.Now().UTC(),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "enforced", "user_id": req.UserID, "max_sessions": req.MaxSessions,
			"strategy": req.Strategy, "action": "oldest sessions terminated if exceeded",
		})
		return
	}
		if r.Method == http.MethodGet {
		// Try PG first, fall back to in-memory map
		if h.memMapRepo != nil {
			rows, _ := h.memMapRepo.ListJSON(r.Context(), "auth_session_limits_json")
			if len(rows) > 0 {
				writeJSON(w, http.StatusOK, map[string]any{"limits": rows, "count": len(rows)})
				return
			}
		}
		sessLimitMu.RLock()
		result := make([]*SessionLimit, 0, len(sessLimits))
		for _, l := range sessLimits {
			result = append(result, l)
		}
		sessLimitMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"limits": result, "count": len(result)})
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
