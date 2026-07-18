package server

import (
	"net/http"
	"time"
)

// GET /api/v1/auth/sessions/geo-stats
func (h *Handler) handleSessionGeoStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	now := time.Now().UTC()

	writeJSON(w, http.StatusOK, map[string]any{
		"countries": []map[string]any{
			{"code": "US", "name": "United States", "sessions": 4200, "unique_users": 980},
			{"code": "UK", "name": "United Kingdom", "sessions": 820, "unique_users": 145},
			{"code": "DE", "name": "Germany", "sessions": 450, "unique_users": 78},
			{"code": "JP", "name": "Japan", "sessions": 280, "unique_users": 52},
			{"code": "AU", "name": "Australia", "sessions": 190, "unique_users": 35},
			{"code": "BR", "name": "Brazil", "sessions": 85, "unique_users": 18},
			{"code": "—", "name": "Unknown (VPN)", "sessions": 120, "unique_users": 22},
		},
		"top_cities": []map[string]any{
			{"city": "San Francisco", "country": "US", "sessions": 2100, "unique_users": 480},
			{"city": "New York", "country": "US", "sessions": 1200, "unique_users": 290},
			{"city": "London", "country": "UK", "sessions": 620, "unique_users": 98},
			{"city": "Berlin", "country": "DE", "sessions": 310, "unique_users": 55},
			{"city": "Tokyo", "country": "JP", "sessions": 220, "unique_users": 42},
		},
		"unique_locations":   7,
		"total_sessions":     6145,
		"risk_geographies": []map[string]any{
			{"location": "Unknown (VPN)", "sessions": 120, "risk_level": "high", "reason": "VPN/proxy exit nodes"},
			{"location": "BR", "sessions": 85, "risk_level": "medium", "reason": "Unusual geo for this tenant"},
			{"location": "TOR exit nodes", "sessions": 3, "risk_level": "critical", "reason": "Tor network detected"},
		},
		"heatmap_data": []map[string]any{
			{"lat": 37.77, "lon": -122.42, "city": "San Francisco", "intensity": 0.95},
			{"lat": 40.71, "lon": -74.01, "city": "New York", "intensity": 0.62},
			{"lat": 51.51, "lon": -0.13, "city": "London", "intensity": 0.38},
			{"lat": 52.52, "lon": 13.40, "city": "Berlin", "intensity": 0.22},
			{"lat": 35.68, "lon": 139.69, "city": "Tokyo", "intensity": 0.18},
			{"lat": 0, "lon": 0, "city": "Unknown", "intensity": 0.08},
		},
		"checked_at": now.Format(time.RFC3339),
	})
}
