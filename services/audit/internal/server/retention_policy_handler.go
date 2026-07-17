package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// RetentionPolicy defines a data retention rule for audit events.
type RetentionPolicy struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	EventType     string    `json:"event_type"`
	RetentionDays int       `json:"retention_days"`
	Action        string    `json:"action"`
	Description   string    `json:"description"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (s *HTTPServer) handleRetentionPolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		tenantID := r.URL.Query().Get("tenant_id")
		var result []*RetentionPolicy
		if s.memMapRepo2 != nil {
			rows, _ := s.memMapRepo2.ListJSON(r.Context(), "audit_retention_policies")
			for _, row := range rows {
				if id != "" && amGetString(row, "id") != id { continue }
				if tenantID != "" && amGetString(row, "tenant_id") != tenantID { continue }
				result = append(result, &RetentionPolicy{
					ID: amGetString(row, "id"), TenantID: amGetString(row, "tenant_id"),
					EventType: amGetString(row, "event_type"), Action: amGetString(row, "action"),
					Description: amGetString(row, "description"),
				})
			}
		}
		if result == nil { result = []*RetentionPolicy{} }
		if id != "" && len(result) == 1 {
			writeJSON(w, http.StatusOK, result[0])
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"policies": result, "count": len(result)})

	case http.MethodPost:
		var req struct {
			TenantID      string `json:"tenant_id"`
			EventType     string `json:"event_type"`
			RetentionDays int    `json:"retention_days"`
			Action        string `json:"action"`
			Description   string `json:"description"`
			Enabled       *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.EventType == "" {
			writeJSONError(w, http.StatusBadRequest, "event_type is required")
			return
		}
		if req.Action == "" { req.Action = "delete" }
		enabled := true
		if req.Enabled != nil { enabled = *req.Enabled }
		now := time.Now().UTC()
		p := &RetentionPolicy{
			ID: uuid.New().String(), TenantID: req.TenantID, EventType: req.EventType,
			RetentionDays: req.RetentionDays, Action: req.Action,
			Description: req.Description, Enabled: enabled, CreatedAt: now, UpdatedAt: now,
		}
		if s.memMapRepo2 != nil {
			s.memMapRepo2.StoreJSON(r.Context(), "audit_retention_policies", p.ID, map[string]any{
				"tenant_id": p.TenantID, "event_type": p.EventType,
				"retention_days": p.RetentionDays, "action": p.Action,
				"description": p.Description, "enabled": p.Enabled,
			})
		}
		writeJSON(w, http.StatusCreated, p)

	case http.MethodPut:
		var req struct {
			ID            string `json:"id"`
			EventType     string `json:"event_type"`
			RetentionDays int    `json:"retention_days"`
			Action        string `json:"action"`
			Description   string `json:"description"`
			Enabled       *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.ID == "" { req.ID = r.URL.Query().Get("id") }
		if req.ID == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		if s.memMapRepo2 != nil {
			update := map[string]any{"event_type": req.EventType, "retention_days": req.RetentionDays,
				"action": req.Action, "description": req.Description}
			if req.Enabled != nil { update["enabled"] = *req.Enabled }
			s.memMapRepo2.StoreJSON(r.Context(), "audit_retention_policies", req.ID, update)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "id": req.ID})

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		if s.memMapRepo2 != nil {
			s.memMapRepo2.DeleteJSON(r.Context(), "audit_retention_policies", id)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

var _ = fmt.Sprintf
