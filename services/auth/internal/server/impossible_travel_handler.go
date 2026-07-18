package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// travelLoginEvent records a login event with geo data.
type travelLoginEvent struct {
	UserID    string  `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	IPAddress string  `json:"ip_address"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Timestamp time.Time `json:"timestamp"`
}

var travelEventStore = struct {
	sync.RWMutex
	data []travelLoginEvent
}{data: []travelLoginEvent{}}

// POST /api/v1/auth/detect-impossible-travel
// Body: {"user_id": "...", "login_events": [{"latitude": ..., "longitude": ..., "timestamp": "...", "city": "..."}]}
// Returns is_detected + details about the impossible travel pattern.
func (h *Handler) handleDetectImpossibleTravel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID      string            `json:"user_id"`
		LoginEvents []map[string]any  `json:"login_events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if len(req.LoginEvents) < 2 {
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":     req.UserID,
			"is_detected": false,
			"reason":      "insufficient login events for analysis (need >= 2)",
		})
		return
	}

	// Parse and sort events by timestamp
	type parsedEvent struct {
		Latitude  float64
		Longitude float64
		City      string
		Country   string
		IPAddress string
		Timestamp time.Time
	}

	var events []parsedEvent
	for _, e := range req.LoginEvents {
		lat, _ := e["latitude"].(float64)
		lon, _ := e["longitude"].(float64)
		city, _ := e["city"].(string)
		country, _ := e["country"].(string)
		ip, _ := e["ip_address"].(string)

		var ts time.Time
		switch v := e["timestamp"].(type) {
		case string:
			ts, _ = time.Parse(time.RFC3339, v)
		case float64:
			ts = time.Unix(int64(v), 0)
		}
		events = append(events, parsedEvent{
			Latitude: lat, Longitude: lon,
			City: city, Country: country, IPAddress: ip,
			Timestamp: ts,
		})
	}

	// Sort by timestamp
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].Timestamp.Before(events[i].Timestamp) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	// Check consecutive pairs for impossible travel
	const maxSpeedKmh = 900.0 // ~speed of commercial jet
	isDetected := false
	var details []map[string]any

	for i := 1; i < len(events); i++ {
		prev := events[i-1]
		curr := events[i]
		timeDiff := curr.Timestamp.Sub(prev.Timestamp).Hours()
		if timeDiff <= 0 {
			continue
		}

		distKm := haversine(prev.Latitude, prev.Longitude, curr.Latitude, curr.Longitude)
		speedKmh := distKm / timeDiff

		if speedKmh > maxSpeedKmh {
			isDetected = true
			details = append(details, map[string]any{
				"from_city":     prev.City,
				"from_country":  prev.Country,
				"to_city":       curr.City,
				"to_country":    curr.Country,
				"distance_km":   distKm,
				"time_hours":    timeDiff,
				"required_speed_kmh": speedKmh,
				"max_feasible_kmh":   maxSpeedKmh,
				"from_timestamp":     prev.Timestamp.Format(time.RFC3339),
				"to_timestamp":       curr.Timestamp.Format(time.RFC3339),
			})
		}
	}

	// Record for tracking
	travelEventStore.Lock()
	travelEventStore.data = append(travelEventStore.data, travelLoginEvent{
		UserID: req.UserID, Timestamp: time.Now().UTC(),
	})
	if len(travelEventStore.data) > 500 {
		travelEventStore.data = travelEventStore.data[len(travelEventStore.data)-500:]
	}
	travelEventStore.Unlock()

	riskLevel := "low"
	if isDetected {
		riskLevel = "critical"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":        req.UserID,
		"is_detected":    isDetected,
		"details":        details,
		"risk_level":     riskLevel,
		"events_analyzed": len(events),
		"checked_at":     time.Now().UTC().Format(time.RFC3339),
		"detection_id":   uuid.New().String(),
		"recommended_action": func() string {
			if isDetected {
				return "block_session_require_step_up_auth"
			}
			return "allow"
		}(),
	})
}

// haversine computes great-circle distance in km.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0
	dLat := (lat2 - lat1) * 3.141592653589793 / 180
	dLon := (lon2 - lon1) * 3.141592653589793 / 180
	lat1Rad := lat1 * 3.141592653589793 / 180
	lat2Rad := lat2 * 3.141592653589793 / 180

	a := sin(dLat/2)*sin(dLat/2) + cos(lat1Rad)*cos(lat2Rad)*sin(dLon/2)*sin(dLon/2)
	c := 2 * r * atan2Sqrt(a)
	return c
}

func sin(x float64) float64     { return x - x*x*x/6 + x*x*x*x*x/120 }
func cos(x float64) float64     { return 1 - x*x/2 + x*x*x*x/24 }
func atan2Sqrt(a float64) float64 {
	if a > 1 {
		a = 1
	}
	return a + a*a*a*0.16667 + a*a*a*a*a*0.075
}
