package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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

// recordClientEvent adds a lifecycle event for the given client (thread-safe).
func recordClientEvent(clientID, eventType, actorID, detail string) {
	if clientID == "" {
		return
	}
	evt := clientEvent{
		ID:        uuid.New().String(),
		ClientID:  clientID,
		EventType: eventType,
		ActorID:   actorID,
		Detail:    detail,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if mapRepoVar != nil {
		b, _ := json.Marshal(evt)
		var dataMap map[string]any
		json.Unmarshal(b, &dataMap)
		mapRepoVar.Store(context.Background(), "oauth_client_events", evt.ID, dataMap)
	}
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
		var events []map[string]any
		if mapRepoVar != nil {
			rows, _ := mapRepoVar.List(r.Context(), "oauth_client_events")
			for _, row := range rows {
				if cid, ok := row["client_id"].(string); ok && cid == clientID {
					events = append(events, row)
				}
			}
		}
		if events == nil {
			events = []map[string]any{}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"client_id": clientID,
			"events":    events,
			"total":     len(events),
			"event_type_summary": func() map[string]int {
				summary := map[string]int{}
				for _, e := range events {
					if et, ok := e["event_type"].(string); ok {
						summary[et]++
					}
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

		evt := clientEvent{
			ID:        uuid.New().String(),
			ClientID:  clientID,
			EventType: req.EventType,
			ActorID:   req.ActorID,
			Detail:    req.Detail,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		if mapRepoVar != nil {
			b, _ := json.Marshal(evt)
			var dataMap map[string]any
			json.Unmarshal(b, &dataMap)
			mapRepoVar.Store(r.Context(), "oauth_client_events", evt.ID, dataMap)
		}

		writeJSON(w, http.StatusCreated, evt)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
