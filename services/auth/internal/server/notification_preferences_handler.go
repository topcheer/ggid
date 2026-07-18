package server

import (
	"encoding/json"
	"net/http"
)

type NotificationPreferencesResult struct {
	EventChannelMatrix map[string]map[string]bool `json:"event_channel_matrix"`
	QuietHours         struct {
		Enabled  bool   `json:"enabled"`
		Start    string `json:"start"`
		End      string `json:"end"`
		Timezone string `json:"timezone"`
	} `json:"quiet_hours"`
	DigestFrequency string `json:"digest_frequency"`
}

func (h *Handler) handleNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := NotificationPreferencesResult{}
	result.EventChannelMatrix = map[string]map[string]bool{
		"security_alert":      {"email": true, "sms": true, "push": true, "webhook": true},
		"access_change":       {"email": true, "sms": false, "push": true, "webhook": false},
		"mfa_event":           {"email": true, "sms": true, "push": false, "webhook": false},
		"compliance_deadline": {"email": true, "sms": false, "push": false, "webhook": true},
	}
	result.QuietHours.Enabled = true
	result.QuietHours.Start = "22:00"
	result.QuietHours.End = "07:00"
	result.QuietHours.Timezone = "UTC"
	result.DigestFrequency = "daily"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
