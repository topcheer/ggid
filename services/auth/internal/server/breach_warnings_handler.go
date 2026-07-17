package server

import (
	"context"
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
		// Try PG first
		if h.memMapRepo != nil {
			if row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_breach_warnings_json", userID); row != nil {
				warning, _ := row["warning"].(bool)
				breachCount, _ := row["breach_count"].(float64)
				forcedChange, _ := row["forced_password_change"].(bool)
				lastCheckStr, _ := row["last_check"].(string)
				writeJSON(w, http.StatusOK, map[string]any{
					"user_id": userID, "breach_warning": warning,
					"breach_count": int(breachCount), "forced_password_change": forcedChange,
					"last_check": lastCheckStr, "redirect_to_change": warning,
				})
				return
			}
		}
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
	bw := &BreachWarning{
		UserID: userID, Warning: breachCount > 0, BreachCount: breachCount,
		LastCheck: time.Now().UTC(), ForcedChange: breachCount > 0,
	}
	breachWarnMu.Lock()
	breachWarnings[userID] = bw
	breachWarnMu.Unlock()
	// PG write-through
	if globalMemMapRepo != nil {
		globalMemMapRepo.StoreJSON(context.Background(), "auth_breach_warnings_json", userID, map[string]any{
			"user_id": userID, "warning": bw.Warning,
			"breach_count": breachCount, "forced_password_change": bw.ForcedChange,
			"last_check": bw.LastCheck,
		})
	}
}
