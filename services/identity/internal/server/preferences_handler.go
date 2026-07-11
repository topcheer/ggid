package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

type UserPreferences struct {
	Locale          string `json:"locale"`
	Timezone        string `json:"timezone"`
	Theme           string `json:"theme"`
	EmailNotif      bool   `json:"email_notifications"`
	PushNotif       bool   `json:"push_notifications"`
	DateFormat      string `json:"date_format"`
	DefaultLanding  string `json:"default_landing_page"`
}

var (
	userPrefMu sync.RWMutex
	userPrefs  = make(map[uuid.UUID]*UserPreferences)
)

// GET/PUT /api/v1/users/{id}/preferences
func (h *HTTPHandler) handleUserPreferences(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		userPrefMu.RLock()
		prefs, ok := userPrefs[userID]
		userPrefMu.RUnlock()
		if !ok {
			prefs = &UserPreferences{Locale: "en", Timezone: "UTC", Theme: "system", EmailNotif: true, PushNotif: false, DateFormat: "YYYY-MM-DD", DefaultLanding: "/dashboard"}
		}
		writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "preferences": prefs})
	case http.MethodPut, http.MethodPost:
		var prefs UserPreferences
		if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		userPrefMu.Lock(); userPrefs[userID] = &prefs; userPrefMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "user_id": userID, "preferences": prefs})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
