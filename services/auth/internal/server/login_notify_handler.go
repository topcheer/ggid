package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type LoginNotifyConfig struct {
	UserID  string `json:"user_id"`
	Channel string `json:"channel"` // email, push, sms
	Enabled bool   `json:"enabled"`
	NewDeviceOnly bool `json:"new_device_only"`
}

var (
	loginNotifyMu sync.RWMutex
	loginNotifyConfigs = make(map[string]*LoginNotifyConfig)
)

// POST /api/v1/auth/login-notify — send login notification
// GET /api/v1/auth/login-notify/config — get/update notification preferences
func (h *Handler) handleLoginNotify(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/auth/login-notify/config" && r.Method == http.MethodGet {
		userID := r.URL.Query().Get("user_id")
		loginNotifyMu.RLock()
		cfg, ok := loginNotifyConfigs[userID]
		loginNotifyMu.RUnlock()
		if !ok {
			cfg = &LoginNotifyConfig{UserID: userID, Channel: "email", Enabled: true, NewDeviceOnly: false}
		}
		writeJSON(w, http.StatusOK, cfg)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			UserID   string `json:"user_id"`
			Device   string `json:"device"`
			Location string `json:"location"`
			IP       string `json:"ip"`
			Channel  string `json:"channel"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.UserID == "" {
			writeError(w, http.StatusBadRequest, "user_id required")
			return
		}
		channel := req.Channel
		if channel == "" {
			channel = "email"
		}

		// "Send" notification
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "sent",
			"user_id":   req.UserID,
			"channel":   channel,
			"device":    req.Device,
			"location":  req.Location,
			"ip":        req.IP,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}
