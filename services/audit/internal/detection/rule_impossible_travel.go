package detection

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
)

// ImpossibleTravelRule: same user, successful logins from distant locations within short time.
// Uses haversine distance to compute speed; >900 km/h is flagged.
type ImpossibleTravelRule struct{}

func (r *ImpossibleTravelRule) ID() string         { return "impossible_travel" }
func (r *ImpossibleTravelRule) Name() string       { return "Impossible Travel" }
func (r *ImpossibleTravelRule) MITRE() string      { return "T1078" }
func (r *ImpossibleTravelRule) DefaultSeverity() domain.Severity { return domain.SeverityHigh }
func (r *ImpossibleTravelRule) Actions() []string  { return []string{"user.login"} }

func (r *ImpossibleTravelRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	// Only successful logins.
	if evt.Result != "success" || evt.ActorID == nil {
		return nil, nil
	}

	// Get geo from event metadata.
	lat, lon := getGeo(evt)
	if lat == 0 && lon == 0 {
		return nil, nil
	}

	maxSpeed := 900.0 // km/h
	if s, ok := cfg.Threshold["max_speed_kmh"].(float64); ok && s > 0 {
		maxSpeed = s
	}

	// Store latest location per user.
	key := "it:" + evt.ActorID.String()
	member := fmt.Sprintf("%.6f,%.6f,%d", lat, lon, evt.CreatedAt.Unix())
	state.AddEvent(ctx, key, evt.CreatedAt.Unix(), member, 24*time.Hour)

	// Check previous location.
	since := evt.CreatedAt.Add(-2 * time.Hour).Unix()
	events, err := state.EventsSince(ctx, key, since)
	if err != nil {
		return nil, err
	}

	for _, m := range events {
		var prevLat, prevLon float64
		var prevTs int64
		if _, err := fmt.Sscanf(m, "%f,%f,%d", &prevLat, &prevLon, &prevTs); err != nil {
			continue
		}
		if prevLat == lat && prevLon == lon {
			continue // same location, skip
		}

		// Compute speed.
		dist := haversine(prevLat, prevLon, lat, lon) // km
		timeDiff := math.Abs(float64(evt.CreatedAt.Unix() - prevTs)) / 3600.0 // hours
		if timeDiff < 0.01 {
			continue
		}
		speed := dist / timeDiff
		if speed > maxSpeed {
			actorID := *evt.ActorID
			det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(), "Impossible travel detected")
			det.ActorID = &actorID
			det.Detail = map[string]any{
				"speed_kmh":      speed,
				"distance_km":    dist,
				"max_speed_kmh":  maxSpeed,
				"from_lat":       prevLat,
				"from_lon":       prevLon,
				"to_lat":         lat,
				"to_lon":         lon,
				"ip_address":     evt.IPAddress,
			}
			return det, nil
		}
	}
	return nil, nil
}

func getGeo(evt *domain.AuditEvent) (float64, float64) {
	if evt.Metadata == nil {
		return 0, 0
	}
	lat, _ := evt.Metadata["latitude"].(float64)
	lon, _ := evt.Metadata["longitude"].(float64)
	return lat, lon
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0 // Earth radius in km
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}
