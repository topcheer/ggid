package httpserver

import (
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// GET /api/v1/policies/risk-score?user_id=X&ip=Y&device_trust=trusted&hour=14
// Returns composite risk score (0-100) with contributing factors.
func (s *HTTPServer) handleRiskScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	ipAddr := r.URL.Query().Get("ip")
	deviceTrust := r.URL.Query().Get("device_trust")
	hourStr := r.URL.Query().Get("hour")

	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Login velocity: how many logins in last hour (simulated)
	velocity := 1
	if _, err := uuid.Parse(userID); err != nil {
		velocity = 3
	}

	// Geo anomaly: IP distance from usual (simulated by IP prefix)
	geoAnomaly := 10
	if ipAddr != "" && len(ipAddr) > 0 && ipAddr[0] == '1' {
		geoAnomaly = 30 // unfamiliar IP range
	}

	// Device trust score (0=untrusted, 50=unknown, 10=trusted)
	deviceScore := 50
	switch deviceTrust {
	case "trusted":
		deviceScore = 10
	case "untrusted":
		deviceScore = 60
	}

	// Time-of-day risk (off-hours = higher risk)
	hour := time.Now().UTC().Hour()
	if hourStr != "" {
		var h int
		for _, c := range hourStr {
			if c >= '0' && c <= '9' {
				h = h*10 + int(c-'0')
			}
		}
		if h >= 0 && h < 24 {
			hour = h
		}
	}
	timeRisk := 5
	if hour < 6 || hour > 22 {
		timeRisk = 25 // off-hours
	}

	// IP reputation (simulated)
	ipRep := 5
	if ipAddr == "" || ipAddr == "0.0.0.0" {
		ipRep = 40
	}

	// Composite score (weighted)
	factors := []map[string]any{
		{"factor": "login_velocity", "score": velocity * 15, "weight": 0.15, "detail": "logins in last hour"},
		{"factor": "geo_anomaly", "score": geoAnomaly, "weight": 0.25, "detail": "IP distance from usual location"},
		{"factor": "device_trust", "score": deviceScore, "weight": 0.25, "detail": "device trust level"},
		{"factor": "ip_reputation", "score": ipRep, "weight": 0.20, "detail": "IP reputation score"},
		{"factor": "time_pattern", "score": timeRisk, "weight": 0.15, "detail": "off-hours access risk"},
	}

	totalScore := 0.0
	for _, f := range factors {
		raw, _ := f["score"].(int)
		w, _ := f["weight"].(float64)
		totalScore += float64(raw) * w
	}
	totalScore = math.Min(math.Round(totalScore), 100)

	// Recommendation
	recommendation := "allow"
	if totalScore >= 70 {
		recommendation = "deny"
	} else if totalScore >= 40 {
		recommendation = "require_mfa"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":         userID,
		"score":           int(totalScore),
		"level":           riskLevel(int(totalScore)),
		"recommendation":  recommendation,
		"factors":         factors,
		"evaluated_at":    time.Now().UTC().Format(time.RFC3339),
	})
}

func riskLevel(score int) string {
	switch {
	case score >= 70:
		return "critical"
	case score >= 40:
		return "high"
	case score >= 20:
		return "medium"
	default:
		return "low"
	}
}
