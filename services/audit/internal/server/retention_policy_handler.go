package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RetentionPolicy defines a data retention rule for audit events.
type RetentionPolicy struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	EventType     string    `json:"event_type"`     // e.g. "user.login", "admin.*", "*"
	RetentionDays int       `json:"retention_days"` // 0 = unlimited
	Action        string    `json:"action"`         // "delete" or "anonymize"
	Description   string    `json:"description"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// retentionPolicyStore holds retention policies in memory.
type retentionPolicyStore struct {
	mu       sync.RWMutex
	policies map[string]*RetentionPolicy
}

var retentionPolicies = &retentionPolicyStore{policies: make(map[string]*RetentionPolicy)}

// GET/POST/PUT/DELETE /api/v1/audit/retention-policies
func (s *HTTPServer) handleRetentionPolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		tenantID := r.URL.Query().Get("tenant_id")

		retentionPolicies.mu.RLock()
		defer retentionPolicies.mu.RUnlock()

		if id != "" {
			p, ok := retentionPolicies.policies[id]
			if !ok {
				writeJSONError(w, http.StatusNotFound, "retention policy not found")
				return
			}
			writeJSON(w, http.StatusOK, p)
			return
		}

		result := []*RetentionPolicy{}
		for _, p := range retentionPolicies.policies {
			if tenantID != "" && p.TenantID != tenantID {
				continue
			}
			result = append(result, p)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"policies": result,
			"count":    len(result),
		})

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
		if req.Action == "" {
			req.Action = "delete"
		}
		if req.Action != "delete" && req.Action != "anonymize" {
			writeJSONError(w, http.StatusBadRequest, "action must be 'delete' or 'anonymize'")
			return
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		now := time.Now().UTC()
		p := &RetentionPolicy{
			ID:            uuid.New().String(),
			TenantID:      req.TenantID,
			EventType:     req.EventType,
			RetentionDays: req.RetentionDays,
			Action:        req.Action,
			Description:   req.Description,
			Enabled:       enabled,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		retentionPolicies.mu.Lock()
		retentionPolicies.policies[p.ID] = p
		retentionPolicies.mu.Unlock()

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
		if req.ID == "" {
			req.ID = r.URL.Query().Get("id")
		}
		if req.ID == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}

		retentionPolicies.mu.Lock()
		defer retentionPolicies.mu.Unlock()

		p, ok := retentionPolicies.policies[req.ID]
		if !ok {
			writeJSONError(w, http.StatusNotFound, "retention policy not found")
			return
		}

		if req.EventType != "" {
			p.EventType = req.EventType
		}
		if req.RetentionDays > 0 {
			p.RetentionDays = req.RetentionDays
		}
		if req.Action == "delete" || req.Action == "anonymize" {
			p.Action = req.Action
		}
		if req.Description != "" {
			p.Description = req.Description
		}
		if req.Enabled != nil {
			p.Enabled = *req.Enabled
		}
		p.UpdatedAt = time.Now().UTC()

		writeJSON(w, http.StatusOK, p)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}

		retentionPolicies.mu.Lock()
		defer retentionPolicies.mu.Unlock()

		if _, ok := retentionPolicies.policies[id]; !ok {
			writeJSONError(w, http.StatusNotFound, "retention policy not found")
			return
		}
		delete(retentionPolicies.policies, id)
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
