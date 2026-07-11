package server

import (
	"net/http"
	"sync"
	"time"
)

type BreachWarning struct {
	UserID    string    `json:"user_id"`
	Warning   bool      `json:"warning"`
	BreachCount int     `json:"breach_count"`
	LastCheck time.Time `json:"last_check"`
	ForcedChange bool   `json:"forced_password_change"`
}

var (
	breachWarnMu sync.RWMutex
	breachWarnings = make(map[string]*BreachWarning)
)

// GET /api/v1/auth/breach-warnings?user_id=X
func (h *Handler) handleBreachWarnings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id required")
		return
	}

	breachWarnMu.RLock()
	warn, ok := breachWarnings[userID]
	breachWarnMu.RUnlock()
	if !ok {
		warn = &BreachWarning{UserID: userID, Warning: false, BreachCount: 0}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":              userID,
		"breach_warning":       warn.Warning,
		"breach_count":         warn.BreachCount,
		"forced_password_change": warn.ForcedChange,
		"last_check":           warn.LastCheck.Format(time.RFC3339),
		"redirect_to_change":   warn.Warning,
	})
}

// SetBreachWarning is called by login handler after HIBP check
func SetBreachWarning(userID string, breachCount int) {
	breachWarnMu.Lock()
	breachWarnings[userID] = &BreachWarning{
		UserID: userID, Warning: breachCount > 0, BreachCount: breachCount,
		LastCheck: time.Now().UTC(), ForcedChange: breachCount > 0,
	}
	breachWarnMu.Unlock()
}
