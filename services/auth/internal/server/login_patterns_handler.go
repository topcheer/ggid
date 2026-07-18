package server

import (
	"net/http"
	"strings"
	"time"
)

// GET /api/v1/auth/login-patterns/{user_id}
func (h *Handler) handleLoginPatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/login-patterns/")
	userID = strings.TrimSuffix(userID, "/")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	now := time.Now().UTC()

	// Time of day histogram (24 buckets)
	timeOfDay := make([]map[string]any, 24)
	for h2 := 0; h2 < 24; h2++ {
		count := 0
		// Peak hours 9-18
		if h2 >= 9 && h2 <= 18 {
			count = 20 + (h2-9)*5
		} else if h2 >= 7 && h2 <= 21 {
			count = 5 + h2%3
		} else {
			count = h2 % 2
		}
		timeOfDay[h2] = map[string]any{"hour": h2, "login_count": count}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"analysis_period_days": 30,
		"time_of_day_histogram": timeOfDay,
		"device_usage": []map[string]any{
			{"device": "MacBook Pro (work)", "fingerprint": "fp-a1b2", "login_count": 142, "last_seen": now.Add(-2 * time.Hour).Format(time.RFC3339)},
			{"device": "iPhone 15 (mobile)", "fingerprint": "fp-c3d4", "login_count": 87, "last_seen": now.Add(-30 * time.Minute).Format(time.RFC3339)},
			{"device": "Chrome (personal)", "fingerprint": "fp-e5f6", "login_count": 12, "last_seen": now.Add(-3 * 24 * time.Hour).Format(time.RFC3339)},
			{"device": "Unknown (VPN exit)", "fingerprint": "fp-g7h8", "login_count": 3, "last_seen": now.Add(-7 * 24 * time.Hour).Format(time.RFC3339)},
		},
		"geo_distribution": []map[string]any{
			{"city": "San Francisco", "country": "US", "login_count": 180, "primary": true},
			{"city": "New York", "country": "US", "login_count": 32},
			{"city": "London", "country": "UK", "login_count": 15},
			{"city": "Unknown (VPN)", "country": "—", "login_count": 8},
		},
		"frequency_trend": []map[string]any{
			{"week": "W-4", "logins": 42, "avg_per_day": 6.0},
			{"week": "W-3", "logins": 48, "avg_per_day": 6.9},
			{"week": "W-2", "logins": 55, "avg_per_day": 7.9},
			{"week": "W-1", "logins": 51, "avg_per_day": 7.3},
		},
		"anomaly_flags": []map[string]any{
			{"type": "off_hours_login", "count": 3, "detail": "Logins between 2-5 AM (unusual for this user)", "severity": "low"},
			{"type": "new_device", "count": 1, "detail": "Unknown device login via VPN exit node", "severity": "medium"},
			{"type": "new_geo", "count": 1, "detail": "First login from London (no prior history)", "severity": "low"},
		},
		"summary": map[string]any{
			"total_logins_30d":   196,
			"avg_per_day":        6.5,
			"unique_devices":     4,
			"unique_locations":   4,
			"anomaly_count":      5,
			"risk_level":         "low",
		},
		"analyzed_at": now.Format(time.RFC3339),
	})
}
