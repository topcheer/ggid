package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type EventSubscription struct {
	ID          string         `json:"id"`
	Filter      map[string]any `json:"filter"`
	CallbackURL string         `json:"callback_url,omitempty"`
	Delivery    string         `json:"delivery"` // callback, sse
	Active      bool           `json:"active"`
	CreatedAt   time.Time      `json:"created_at"`
}

// POST /api/v1/audit/events/subscribe
// DELETE /api/v1/audit/events/subscribe/{id}
func (s *HTTPServer) handleEventSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Filter      map[string]any `json:"filter"`
			CallbackURL string         `json:"callback_url"`
			Delivery    string         `json:"delivery"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Delivery == "" {
			req.Delivery = "sse"
		}
		sub := &EventSubscription{ID: uuid.New().String(), Filter: req.Filter, CallbackURL: req.CallbackURL, Delivery: req.Delivery, Active: true, CreatedAt: time.Now().UTC()}
		if s.memMapRepo2 != nil {
			s.memMapRepo2.StoreJSON(r.Context(), "audit_event_subscriptions", sub.ID, map[string]any{
				"id":           sub.ID,
				"filter":       sub.Filter,
				"callback_url": sub.CallbackURL,
				"delivery":     sub.Delivery,
				"active":       sub.Active,
				"created_at":   sub.CreatedAt,
			})
		}
		writeJSON(w, http.StatusCreated, sub)
		return
	}
	if r.Method == http.MethodDelete {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/events/subscribe/")
		if s.memMapRepo2 != nil {
			s.memMapRepo2.DeleteJSON(r.Context(), "audit_event_subscriptions", id)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})
		return
	}
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}
