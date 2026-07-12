package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// clientEvent represents a lifecycle event for an OAuth client.
type clientEvent struct {
	ID         string `json:"id"`
	ClientID   string `json:"client_id"`
	EventType  string `json:"event_type"` // created, updated, rotated, suspended, reinstated
	ActorID    string `json:"actor_id"`
	Detail     string `json:"detail"`
	Timestamp  string `json:"timestamp"`
}

var clientEventStore = struct {
	sync.RWMutex
	events map[string][]clientEvent // clientID → events
}{events: make(map[string][]clientEvent)}

// recordClientEvent adds a lifecycle event for the given client (thread-safe).
func recordClientEvent(clientID, eventType, actorID, detail string) {
	if clientID == "" {
		return
	}
	clientEventStore.Lock()
	defer clientEventStore.Unlock()
	clientEventStore.events[clientID] = append(clientEventStore.events[clientID], clientEvent{
		ID:        uuid.New().String(),
		ClientID:  clientID,
		EventType: eventType,
		ActorID:   actorID,
		Detail:    detail,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// GET /api/v1/oauth/clients/{id}/events — list lifecycle events for a client
// POST /api/v1/oauth/clients/{id}/events — record a manual event
func handleClientEvents(w http.ResponseWriter, r *http.Request) {
	// Extract client ID from path /api/v1/oauth/clients/{id}/events
	clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	clientID = strings.TrimSuffix(clientID, "/events")
	clientID = strings.TrimSuffix(clientID, "/")
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		clientEventStore.RLock()
		events := clientEventStore.events[clientID]
		result := make([]clientEvent, len(events))
		copy(result, events)
		clientEventStore.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"client_id": clientID,
			"events":    result,
			"total":     len(result),
			"event_type_summary": func() map[string]int {
				summary := map[string]int{}
				for _, e := range result {
					summary[e.EventType]++
				}
				return summary
			}(),
		})

	case http.MethodPost:
		var req struct {
			EventType string `json:"event_type"`
			ActorID   string `json:"actor_id"`
			Detail    string `json:"detail"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		validTypes := map[string]bool{
			"created": true, "updated": true, "rotated": true,
			"suspended": true, "reinstated": true,
		}
		if !validTypes[req.EventType] {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "event_type must be one of: created, updated, rotated, suspended, reinstated",
			})
			return
		}

		if req.ActorID == "" {
			req.ActorID = "system"
		}

		recordClientEvent(clientID, req.EventType, req.ActorID, req.Detail)

		clientEventStore.RLock()
		events := clientEventStore.events[clientID]
		lastEvent := events[len(events)-1]
		clientEventStore.RUnlock()

		writeJSON(w, http.StatusCreated, lastEvent)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
