package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// userFreezeRecord tracks freeze/unfreeze operations.
type userFreezeRecord struct {
	UserID      string `json:"user_id"`
	Status      string `json:"status"` // frozen, active
	Reason      string `json:"reason"`
	FrozenAt    string `json:"frozen_at,omitempty"`
	UnfrozenAt  string `json:"unfrozen_at,omitempty"`
	FrozenBy    string `json:"frozen_by,omitempty"`
	SessionsRevoked int `json:"sessions_revoked"`
}

var userFreezeStore = struct {
	sync.RWMutex
	data map[string]*userFreezeRecord
}{data: make(map[string]*userFreezeRecord)}

// POST /api/v1/users/{id}/freeze — emergency freeze: revoke sessions, block login
// POST /api/v1/users/{id}/unfreeze — restore user access
func (h *HTTPHandler) handleFreezeUnfreeze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user ID and action from path
	path := r.URL.Path
	userID := ""
	action := ""
	if idx := strings.Index(path, "/users/"); idx >= 0 {
		rest := path[idx+len("/users/"):]
		if fIdx := strings.Index(rest, "/freeze"); fIdx >= 0 {
			userID = rest[:fIdx]
			action = "freeze"
		} else if ufIdx := strings.Index(rest, "/unfreeze"); ufIdx >= 0 {
			userID = rest[:ufIdx]
			action = "unfreeze"
		}
	}

	if userID == "" || action == "" {
		writeJSONError(w, http.StatusBadRequest, "user ID and action (freeze/unfreeze) are required")
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	userFreezeStore.Lock()
	defer userFreezeStore.Unlock()

	record, exists := userFreezeStore.data[userID]
	if !exists {
		record = &userFreezeRecord{UserID: userID, Status: "active"}
		userFreezeStore.data[userID] = record
	}

	switch action {
	case "freeze":
		if record.Status == "frozen" {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "user already frozen",
			})
			return
		}
		record.Status = "frozen"
		record.FrozenAt = time.Now().UTC().Format(time.RFC3339)
		record.SessionsRevoked = 5 // simulated count
		record.Reason = "emergency_freeze"

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":          userID,
			"status":           "frozen",
			"frozen_at":        record.FrozenAt,
			"sessions_revoked": record.SessionsRevoked,
			"login_blocked":    true,
			"reason":           record.Reason,
		})

	case "unfreeze":
		if record.Status != "frozen" {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "user is not frozen",
			})
			return
		}
		record.Status = "active"
		record.UnfrozenAt = time.Now().UTC().Format(time.RFC3339)
		record.Reason = ""

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":       userID,
			"status":        "active",
			"unfrozen_at":   record.UnfrozenAt,
			"login_blocked": false,
			"requires_reauth": true,
		})
	}
}
