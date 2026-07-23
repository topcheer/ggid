package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// SSEThreatEvent is the real-time threat event pushed via SSE to the console.
// Field names match the frontend ThreatEvent interface exactly.
type SSEThreatEvent struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	SourceIP    string   `json:"source_ip"`
	Indicators  []string `json:"indicators"`
	Target      string   `json:"target"`
	Source      string   `json:"source"`
	CreatedAt   string   `json:"created_at"`
}

// handleThreatFeedStream is the SSE endpoint for real-time threat feed.
// GET /api/v1/audit/threat-feed/stream?token=X&tenant_id=Y
//
// EventSource (browser API) cannot set Authorization headers, so the JWT
// token is passed as a query parameter. The gateway validates it before
// proxying. The handler polls for new suspicious audit events and pushes
// them as SSE messages in the format:
//
//	data: {"id":"...","severity":"high","type":"brute_force",...}
func (s *HTTPServer) handleThreatFeedStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// CORS: same controlled-origin policy as the audit SSE stream.
	if origin := r.Header.Get("Origin"); origin != "" {
		if isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
	}

	// Send initial connection confirmation so the frontend knows we're live.
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
	flusher.Flush()

	// Parse tenant ID from query (EventSource can't send headers)
	tenantIDStr := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		// Without a valid tenant, just keep the connection alive with no data.
		tenantID = uuid.Nil
	}

	// Track the last poll time so we only push new events.
	lastCheck := time.Now().UTC()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			events := s.pollThreatEvents(r, tenantID, lastCheck)
			lastCheck = time.Now().UTC()
			for _, evt := range events {
				data, _ := json.Marshal(evt)
				fmt.Fprintf(w, "data: %s\n\n", data)
			}
			flusher.Flush()
		}
	}
}

// pollThreatEvents queries recent suspicious audit events since the last check.
// Returns SSE-formatted threat events derived from failed/denied audit events.
func (s *HTTPServer) pollThreatEvents(r *http.Request, tenantID uuid.UUID, since time.Time) []SSEThreatEvent {
	if s.svc == nil || tenantID == uuid.Nil {
		return nil
	}

	// Query failed/denied events from the audit trail as threat indicators.
	filter := domain.ListFilter{
		TenantID:   tenantID,
		StartTime:  &since,
		Descending: true,
	}

	auditEvents, _, err := s.svc.ListEvents(r.Context(), filter, 1, 20)
	if err != nil || len(auditEvents) == 0 {
		return nil
	}

	var events []SSEThreatEvent
	for _, e := range auditEvents {
		// Skip successful events — only surface failures and denials as threats.
		if e.Result == domain.ResultSuccess {
			continue
		}

		severity := "medium"
		if e.Result == "denied" || e.Action == "login" && e.Result == "failure" {
			severity = "high"
		}

		description := fmt.Sprintf("%s — %s by %s", e.Action, e.Result, e.ActorName)
		if e.IPAddress != "" {
			description += fmt.Sprintf(" from %s", stripCIDR(e.IPAddress))
		}

		events = append(events, SSEThreatEvent{
			ID:          e.ID.String(),
			Severity:    severity,
			Type:        e.Action,
			Description: description,
			SourceIP:    stripCIDR(e.IPAddress),
			Indicators:  []string{},
			Target:      e.ResourceName,
			Source:      "audit_engine",
			CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		})
	}

	return events
}
