package domain

import (
	"time"

	"github.com/google/uuid"
)

// Stats holds aggregated audit analytics data for a dashboard.
type Stats struct {
	TotalEvents24h    int               `json:"total_events_24h"`
	EventsByAction    map[string]int    `json:"events_by_action"`
	HourlyDistribution []HourlyCount     `json:"hourly_distribution"`
	TopActors         []ActorActivity   `json:"top_actors"`
	FailedLogins24h   int               `json:"failed_logins_24h"`
}

// HourlyCount represents a single hour bucket in the timeline.
type HourlyCount struct {
	Hour  time.Time `json:"hour"`
	Count int       `json:"count"`
}

// ActorActivity represents an actor and their event count.
type ActorActivity struct {
	ActorID   uuid.UUID `json:"actor_id"`
	ActorName string    `json:"actor_name"`
	Count     int       `json:"count"`
}
