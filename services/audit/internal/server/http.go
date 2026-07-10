// Package httpserver provides REST API endpoints for the Audit Service.
// These endpoints allow the Admin Console to query audit logs via HTTP
// through the API Gateway.
package httpserver

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/service"
	"github.com/google/uuid"
)

// HTTPServer exposes the Audit Service as a REST API.
type HTTPServer struct {
	svc *service.AuditService
}

// NewHTTPServer creates a new Audit Service HTTP server.
func NewHTTPServer(svc *service.AuditService) *HTTPServer {
	return &HTTPServer{svc: svc}
}

// RegisterRoutes registers all Audit Service HTTP routes on the given mux.
func (s *HTTPServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/audit/events", s.handleEvents)
	mux.HandleFunc("/api/v1/audit/events/", s.handleEventByID)
	mux.HandleFunc("/api/v1/audit/stats", s.handleStats)
	mux.HandleFunc("/api/v1/audit/export", s.handleExport)
	mux.HandleFunc("/api/v1/audit/stream", s.handleStream)
	mux.HandleFunc("/api/v1/audit/retention", s.handleRetention)
	// Alias: Gateway may route /api/v1/audit without /events suffix
	mux.HandleFunc("/api/v1/audit", s.handleEvents)
}

// GET /api/v1/audit/events?tenant_id=X&action=Y&result=Z&page_size=N
func (s *HTTPServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	filter := domain.ListFilter{
		TenantID:   tenantID,
		Descending: true, // default: newest first
	}

	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = action
	}
	if result := r.URL.Query().Get("result"); result != "" {
		filter.Result = domain.EventResult(result)
	}
	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		filter.ResourceType = resourceType
	}
	if actorIDStr := r.URL.Query().Get("actor_id"); actorIDStr != "" {
		actorID, err := uuid.Parse(actorIDStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid actor_id")
			return
		}
		filter.ActorID = &actorID
	}
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &t
		}
	}

	pageSize := 50
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 && n <= 500 {
			pageSize = n
		}
	}

	events, total, err := s.svc.ListEvents(r.Context(), filter, 1, pageSize)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := make([]map[string]any, len(events))
	for i, e := range events {
		result[i] = eventToJSON(e)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events": result,
		"total":  total,
	})
}

// GET /api/v1/audit/events/{id}
func (s *HTTPServer) handleEventByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/events/")
	if idStr == "" {
		writeJSONError(w, http.StatusBadRequest, "event ID is required")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	event, err := s.svc.GetEvent(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, eventToJSON(event))
}

// GET /api/v1/audit/stats?tenant_id=X
func (s *HTTPServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	stats, err := s.svc.GetStats(r.Context(), tenantID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, statsToJSON(stats))
}

func statsToJSON(s *domain.Stats) map[string]any {
	actions := make(map[string]any, len(s.EventsByAction))
	for k, v := range s.EventsByAction {
		actions[k] = v
	}
	hourly := make([]map[string]any, len(s.HourlyDistribution))
	for i, h := range s.HourlyDistribution {
		hourly[i] = map[string]any{
			"hour":  h.Hour.Format(time.RFC3339),
			"count": h.Count,
		}
	}
	actors := make([]map[string]any, len(s.TopActors))
	for i, a := range s.TopActors {
		actors[i] = map[string]any{
			"actor_id":   a.ActorID.String(),
			"actor_name": a.ActorName,
			"count":      a.Count,
		}
	}
	return map[string]any{
		"total_events_24h":     s.TotalEvents24h,
		"events_by_action":     actions,
		"hourly_distribution":  hourly,
		"top_actors":           actors,
		"failed_logins_24h":    s.FailedLogins24h,
	}
}

// GET /api/v1/audit/export?tenant_id=X&format=csv|json&action=Y&result=Z
func (s *HTTPServer) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if format != "csv" && format != "json" {
		writeJSONError(w, http.StatusBadRequest, "format must be csv or json")
		return
	}

	filter := domain.ListFilter{
		TenantID:   tenantID,
		Descending: true,
	}
	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = action
	}
	if result := r.URL.Query().Get("result"); result != "" {
		filter.Result = domain.EventResult(result)
	}
	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		filter.ResourceType = resourceType
	}
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &t
		}
	}

	// Export up to 10,000 events
	events, _, err := s.svc.ListEvents(r.Context(), filter, 1, 10000)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="audit_export.csv"`)
		writeAuditCSV(w, events)
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="audit_export.json"`)
		result := make([]map[string]any, len(events))
		for i, e := range events {
			result[i] = eventToJSON(e)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"events": result,
			"total":  len(events),
		})
	}
}

// writeAuditCSV writes audit events as CSV to the given writer.
func writeAuditCSV(w http.ResponseWriter, events []*domain.AuditEvent) {
	wr := csv.NewWriter(w)
	wr.Write([]string{"id", "created_at", "actor_type", "actor_id", "actor_name",
		"action", "resource_type", "resource_id", "resource_name", "result",
		"ip_address", "user_agent"})

	for _, e := range events {
		actorID := ""
		if e.ActorID != nil {
			actorID = e.ActorID.String()
		}
		resourceID := ""
		if e.ResourceID != nil {
			resourceID = e.ResourceID.String()
		}
		wr.Write([]string{
			e.ID.String(),
			e.CreatedAt.Format(time.RFC3339),
			string(e.ActorType),
			actorID,
			e.ActorName,
			e.Action,
			e.ResourceType,
			resourceID,
			e.ResourceName,
			string(e.Result),
			e.IPAddress,
			e.UserAgent,
		})
	}
	wr.Flush()
}

// GET /api/v1/audit/stream?tenant_id=X — Server-Sent Events for real-time audit events
func (s *HTTPServer) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send initial connection confirmation
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\",\"tenant_id\":\"%s\"}\n\n", tenantID)
	flusher.Flush()

	// Poll for new events every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastCheck = time.Now().UTC()

	for {
		select {
		case <-r.Context().Done():
			return
		case t := <-ticker.C:
			// Query events since last check
			events, _, err := s.svc.ListEvents(r.Context(), domain.ListFilter{
				TenantID:   tenantID,
				StartTime:  &lastCheck,
				Descending: true,
			}, 1, 50)
			if err != nil {
				continue
			}

			for _, e := range events {
				data, _ := json.Marshal(eventToJSON(e))
				fmt.Fprintf(w, "event: audit_event\ndata: %s\n\n", data)
			}
			flusher.Flush()
			lastCheck = t.UTC()
		}
	}
}

// POST /api/v1/audit/retention?action=cleanup&days=90
func (s *HTTPServer) handleRetention(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 90 // default retention
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	deleted, err := s.svc.CleanupOldEvents(r.Context(), days)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "completed",
		"retention_days":     days,
		"deleted_count":      deleted,
		"cleanup_timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// --- Helpers ---

func eventToJSON(e *domain.AuditEvent) map[string]any {
	m := map[string]any{
		"id":            e.ID.String(),
		"tenant_id":     e.TenantID.String(),
		"actor_type":    string(e.ActorType),
		"actor_name":    e.ActorName,
		"action":        e.Action,
		"resource_type": e.ResourceType,
		"resource_name": e.ResourceName,
		"result":        string(e.Result),
		"ip_address":    e.IPAddress,
		"user_agent":    e.UserAgent,
		"request_id":    e.RequestID,
		"created_at":    e.CreatedAt,
	}
	if e.ActorID != nil {
		m["actor_id"] = e.ActorID.String()
	}
	if e.ResourceID != nil {
		m["resource_id"] = e.ResourceID.String()
	}
	if e.Metadata != nil {
		m["metadata"] = e.Metadata
	}
	return m
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeServiceError(w http.ResponseWriter, err error) {
	if ge, ok := errors.AsGGIDError(err); ok {
		switch ge.Code {
		case errors.ErrNotFound:
			writeJSONError(w, http.StatusNotFound, ge.Message)
		case errors.ErrInvalidArgument:
			writeJSONError(w, http.StatusBadRequest, ge.Message)
		default:
			writeJSONError(w, http.StatusInternalServerError, ge.Message)
		}
		return
	}
	writeJSONError(w, http.StatusInternalServerError, err.Error())
}
