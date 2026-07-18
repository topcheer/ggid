package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type UserPreferences struct {
	Locale         string `json:"locale"`
	Timezone       string `json:"timezone"`
	Theme          string `json:"theme"`
	EmailNotif     bool   `json:"email_notifications"`
	PushNotif      bool   `json:"push_notifications"`
	DateFormat     string `json:"date_format"`
	DefaultLanding string `json:"default_landing_page"`
}

func (h *HTTPHandler) handleUserPreferences(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	uid := userID.String()
	switch r.Method {
	case http.MethodGet:
		var prefs any
		if h.identityPolicyMap != nil {
			row, _ := h.identityPolicyMap.Get(r.Context(), "identity_user_preferences", uid)
			if row != nil {
				prefs = row
			}
		}
		if prefs == nil {
			prefs = &UserPreferences{Locale: "en", Timezone: "UTC", Theme: "system", EmailNotif: true, DateFormat: "YYYY-MM-DD", DefaultLanding: "/dashboard"}
		}
		writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "preferences": prefs})
	case http.MethodPut, http.MethodPost:
		var prefs UserPreferences
		if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if h.identityPolicyMap != nil {
			h.identityPolicyMap.Store(r.Context(), "identity_user_preferences", uid, map[string]any{
				"locale": prefs.Locale, "timezone": prefs.Timezone, "theme": prefs.Theme,
				"email_notifications": prefs.EmailNotif, "push_notifications": prefs.PushNotif,
				"date_format": prefs.DateFormat, "default_landing_page": prefs.DefaultLanding,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "user_id": userID, "preferences": prefs})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
