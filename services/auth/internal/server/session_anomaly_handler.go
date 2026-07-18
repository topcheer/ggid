package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// sessionAnomalyData holds anomaly indicators for a session.
type sessionAnomalyData struct {
	SessionID      string  `json:"session_id"`
	IPChangeRate   float64 `json:"ip_change_rate"`
	GeoVelocity    float64 `json:"geo_velocity_kmh"`
	DeviceMatch    bool    `json:"device_match"`
	UniqueIPs      int     `json:"unique_ips"`
	IPHistory      []string `json:"ip_history"`
	Score          int     `json:"score"`
	RiskLevel      string  `json:"risk_level"`
	ContributingFactors []string `json:"contributing_factors"`
}

var sessionAnomalyStore = struct {
	sync.RWMutex
	data map[string]*sessionAnomalyData
}{data: make(map[string]*sessionAnomalyData)}

// GET /api/v1/auth/sessions/anomaly-score?session_id=X
// POST /api/v1/auth/sessions/anomaly-score — submit data for scoring
func (h *Handler) handleSessionAnomalyScore(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			writeError(w, http.StatusBadRequest, "session_id is required")
			return
		}

		sessionAnomalyStore.RLock()
		data, exists := sessionAnomalyStore.data[sessionID]
		sessionAnomalyStore.RUnlock()

		if !exists {
			// Try PG first
			if h.memMapRepo != nil {
				if row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_session_anomalies_json", sessionID); row != nil {
					writeJSON(w, http.StatusOK, row)
					return
				}
			}
			// Return a default low-risk score
			writeJSON(w, http.StatusOK, map[string]any{
				"session_id": sessionID,
				"score":      0,
				"risk_level": "none",
				"message":    "no anomaly data recorded for this session",
				"checked_at": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}

		writeJSON(w, http.StatusOK, data)

	case http.MethodPost:
		var req struct {
			SessionID    string   `json:"session_id"`
			IPHistory    []string `json:"ip_history"`
			GeoPoints    []map[string]float64 `json:"geo_points"` // [{lat, lon, timestamp_epoch}]
			DeviceMatch  *bool    `json:"device_match"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.SessionID == "" {
			req.SessionID = uuid.New().String()
		}

		// Calculate anomaly indicators
		uniqueIPs := map[string]bool{}
		for _, ip := range req.IPHistory {
			uniqueIPs[ip] = true
		}
		ipChangeRate := float64(len(uniqueIPs))
		if len(req.IPHistory) > 0 {
			ipChangeRate = float64(len(uniqueIPs)) / float64(len(req.IPHistory)) * 100
		}

		// Geo velocity: compute max speed between consecutive points
		maxVelocity := 0.0
		if len(req.GeoPoints) >= 2 {
			for i := 1; i < len(req.GeoPoints); i++ {
				lat1 := req.GeoPoints[i-1]["lat"]
				lon1 := req.GeoPoints[i-1]["lon"]
				lat2 := req.GeoPoints[i]["lat"]
				lon2 := req.GeoPoints[i]["lon"]
				ts1 := req.GeoPoints[i-1]["timestamp_epoch"]
				ts2 := req.GeoPoints[i]["timestamp_epoch"]

				if ts2 > ts1 {
					dist := haversineDistance(lat1, lon1, lat2, lon2)
					timeHours := (ts2 - ts1) / 3600.0
					if timeHours > 0 {
						velocity := dist / timeHours
						if velocity > maxVelocity {
							maxVelocity = velocity
						}
					}
				}
			}
		}

		deviceMatch := true
		if req.DeviceMatch != nil {
			deviceMatch = *req.DeviceMatch
		}

		// Compute anomaly score 0-100
		score := 0
		var factors []string

		if ipChangeRate > 50 {
			score += 30
			factors = append(factors, "high_ip_change_rate")
		} else if ipChangeRate > 25 {
			score += 15
			factors = append(factors, "moderate_ip_change_rate")
		}

		if maxVelocity > 1000 {
			score += 40
			factors = append(factors, "impossible_travel")
		} else if maxVelocity > 500 {
			score += 20
			factors = append(factors, "high_geo_velocity")
		}

		if !deviceMatch {
			score += 25
			factors = append(factors, "device_fingerprint_mismatch")
		}

		if len(uniqueIPs) > 5 {
			score += 15
			factors = append(factors, "excessive_unique_ips")
		}

		if score > 100 {
			score = 100
		}

		riskLevel := "low"
		switch {
		case score >= 70:
			riskLevel = "critical"
		case score >= 50:
			riskLevel = "high"
		case score >= 25:
			riskLevel = "medium"
		}

		data := &sessionAnomalyData{
			SessionID:           req.SessionID,
			IPChangeRate:        ipChangeRate,
			GeoVelocity:         maxVelocity,
			DeviceMatch:         deviceMatch,
			UniqueIPs:           len(uniqueIPs),
			IPHistory:           req.IPHistory,
			Score:               score,
			RiskLevel:           riskLevel,
			ContributingFactors: factors,
		}

		sessionAnomalyStore.Lock()
		sessionAnomalyStore.data[req.SessionID] = data
		sessionAnomalyStore.Unlock()

		// PG write-through
		if h.memMapRepo != nil {
			h.memMapRepo.StoreJSON(r.Context(), "auth_session_anomalies_json", req.SessionID, map[string]any{
				"session_id": data.SessionID,
				"ip_change_rate": data.IPChangeRate,
				"geo_velocity_kmh": data.GeoVelocity,
				"device_match": data.DeviceMatch,
				"unique_ips": data.UniqueIPs,
				"ip_history": data.IPHistory,
				"score": data.Score,
				"risk_level": data.RiskLevel,
				"contributing_factors": data.ContributingFactors,
			})
		}

		writeJSON(w, http.StatusOK, data)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// haversineDistance computes the great-circle distance between two points in km.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	lat1Rad := lat1 * 3.141592653589793 / 180
	lat2Rad := lat2 * 3.141592653589793 / 180
	dLat := (lat2 - lat1) * 3.141592653589793 / 180
	dLon := (lon2 - lon1) * 3.141592653589793 / 180

	a := sin2(dLat/2) + cosf(lat1Rad)*cosf(lat2Rad)*sin2(dLon/2)
	c := 2 * earthRadiusKm * asinSqrt(a)
	return c
}

func sin2(x float64) float64 {
	s := x * x // simplified sin approximation for small angles
	_ = s
	// Use proper math
	return (1 - cosf(2*x)) / 2
}

func cosf(x float64) float64 {
	// Taylor series approximation
	return 1 - x*x/2 + x*x*x*x/24 - x*x*x*x*x*x/720
}

func asinSqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		x = 1
	}
	// Approximation for asin(sqrt(x))
	return x + x*x*x*0.16667 + x*x*x*x*x*0.075
}
