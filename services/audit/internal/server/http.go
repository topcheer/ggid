// Package httpserver provides REST API endpoints for the Audit Service.
// These endpoints allow the Admin Console to query audit logs via HTTP
// through the API Gateway.
package httpserver

import (
	"encoding/json"
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
