package server

import (
	"encoding/json"
	"net/http"
)

type TokenRotationClientConfig struct {
	ClientID           string `json:"client_id"`
	RotationIntervalDays int  `json:"rotation_interval_days"`
	MaxAgeDays         int    `json:"max_age_days"`
	NotifyBeforeDays   int    `json:"notify_before_days"`
	AutoRotate         bool   `json:"auto_rotate"`
	GracePeriodHours   int    `json:"grace_period_hours"`
}

type TokenRotationConfig struct {
	PerClient    []TokenRotationClientConfig `json:"per_client"`
	GlobalDefaults struct {
		DefaultRotationDays int  `json:"default_rotation_days"`
		DefaultMaxAgeDays   int  `json:"default_max_age_days"`
		DefaultNotifyBefore int  `json:"default_notify_before_days"`
		DefaultAutoRotate   bool `json:"default_auto_rotate"`
	} `json:"global_defaults"`
}

func handleTokenRotationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := TokenRotationConfig{
			PerClient: []TokenRotationClientConfig{
				{ClientID: "web-console", RotationIntervalDays: 30, MaxAgeDays: 90, NotifyBeforeDays: 7, AutoRotate: true, GracePeriodHours: 24},
				{ClientID: "mobile-app", RotationIntervalDays: 60, MaxAgeDays: 180, NotifyBeforeDays: 14, AutoRotate: true, GracePeriodHours: 48},
				{ClientID: "service-agent-01", RotationIntervalDays: 7, MaxAgeDays: 30, NotifyBeforeDays: 2, AutoRotate: true, GracePeriodHours: 4},
				{ClientID: "analytics-dashboard", RotationIntervalDays: 90, MaxAgeDays: 365, NotifyBeforeDays: 30, AutoRotate: false, GracePeriodHours: 72},
			},
		}
		result.GlobalDefaults.DefaultRotationDays = 30
		result.GlobalDefaults.DefaultMaxAgeDays = 90
		result.GlobalDefaults.DefaultNotifyBefore = 7
		result.GlobalDefaults.DefaultAutoRotate = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req TokenRotationConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
