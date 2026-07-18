package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
)

// BreachNotification tracks a password breach notification sent to users.
type BreachNotification struct {
	ID         string    `json:"id"`
	BreachName string    `json:"breach_name"`
	UserIDs    []string  `json:"user_ids"`
	ResetTokens map[string]string `json:"reset_tokens"` // user_id → token
	Status     string    `json:"status"`
	NotifiedAt time.Time `json:"notified_at"`
}

var (
	breachNotifMu sync.Mutex
	breachNotifs  = make(map[string]*BreachNotification)
)

// POST /api/v1/auth/password-breach/notify — admin notifies affected users.
// Body: {"user_ids": ["..."], "breach_name": "..."}
func (h *Handler) handlePasswordBreachNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserIDs    []string `json:"user_ids"`
		BreachName string   `json:"breach_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.UserIDs) == 0 {
		writeError(w, http.StatusBadRequest, "user_ids is required")
		return
	}
	if req.BreachName == "" {
		req.BreachName = "unnamed_breach"
	}

	// Generate reset tokens for each user
	resetTokens := make(map[string]string)
	for _, uid := range req.UserIDs {
		token, _ := crypto.GenerateRandomToken(32)
		resetTokens[uid] = token
	}

	notif := &BreachNotification{
		ID:          uuid.New().String(),
		BreachName:  req.BreachName,
		UserIDs:     req.UserIDs,
		ResetTokens: resetTokens,
		Status:      "sent",
		NotifiedAt:  time.Now().UTC(),
	}

	breachNotifMu.Lock()
	breachNotifs[notif.ID] = notif
	breachNotifMu.Unlock()

	// PG write-through
	if h.memMapRepo != nil {
		h.memMapRepo.StoreJSON(r.Context(), "auth_breach_notifs_json", notif.ID, map[string]any{
			"id": notif.ID, "breach_name": notif.BreachName,
			"user_ids": notif.UserIDs, "reset_tokens": notif.ResetTokens,
			"status": notif.Status, "notified_at": notif.NotifiedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "notified",
		"notification_id":   notif.ID,
		"breach_name":       req.BreachName,
		"notified_count":    len(req.UserIDs),
		"reset_tokens":      resetTokens,
		"message":           "Password reset tokens generated and notifications queued",
	})
}
