package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

type alertWebhookConfig struct {
	mu       sync.RWMutex
	webhooks []map[string]any
}

var globalAlertWebhooks = &alertWebhookConfig{}

// POST/GET/DELETE /api/v1/audit/alert-webhooks
// DB-backed: uses audit_alert_webhooks table. Falls back to in-memory when pool is nil.
// P1 fix: All operations now require and enforce tenant_id isolation.
func (s *HTTPServer) handleAlertWebhooks(w http.ResponseWriter, r *http.Request) {
	// Require tenant context for all operations
	tid := r.Header.Get("X-Tenant-ID")
	if tid == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "missing tenant context"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		if s.pool != nil {
			rows, err := s.pool.Query(r.Context(), `
				SELECT id::text, url, COALESCE(secret, ''), active, created_at
				FROM audit_alert_webhooks WHERE tenant_id::text = $1 ORDER BY created_at DESC`, tid)
			if err == nil {
				defer rows.Close()
				webhooks := []map[string]any{}
				for rows.Next() {
					var id, url, secret string
					var active bool
					var created interface{}
					_ = rows.Scan(&id, &url, &secret, &active, &created)
					webhooks = append(webhooks, map[string]any{
						"id": id, "url": url, "secret": maskSecret(secret), "active": active, "created_at": created,
					})
				}
				writeJSON(w, http.StatusOK, map[string]any{"webhooks": webhooks})
				return
			}
		}
		// In-memory fallback: filter by tenant
		globalAlertWebhooks.mu.RLock()
		defer globalAlertWebhooks.mu.RUnlock()
		filtered := []map[string]any{}
		for _, h := range globalAlertWebhooks.webhooks {
			if h["tenant_id"] == tid {
				filtered = append(filtered, h)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"webhooks": filtered})

	case http.MethodPost:
		var req struct {
			URL    string `json:"url"`
			Secret string `json:"secret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if req.URL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url required"})
			return
		}
		hookID := uuid.New().String()
		hook := map[string]any{
			"id":        hookID,
			"url":       req.URL,
			"secret":    req.Secret,
			"active":    true,
			"tenant_id": tid,
		}
		if s.pool != nil {
			_, err := s.pool.Exec(r.Context(), `
				INSERT INTO audit_alert_webhooks (id, tenant_id, url, secret, active)
				VALUES ($1, $2, $3, $4, true)`, hookID, tid, req.URL, req.Secret)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save webhook"})
				return
			}
		} else {
			globalAlertWebhooks.mu.Lock()
			globalAlertWebhooks.webhooks = append(globalAlertWebhooks.webhooks, hook)
			globalAlertWebhooks.mu.Unlock()
		}
		writeJSON(w, http.StatusCreated, hook)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
			return
		}
		if s.pool != nil {
			_, err := s.pool.Exec(r.Context(), `DELETE FROM audit_alert_webhooks WHERE id::text = $1 AND tenant_id::text = $2`, id, tid)
			if err == nil {
				writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
				return
			}
		}
		// In-memory fallback: filter by id AND tenant
		globalAlertWebhooks.mu.Lock()
		defer globalAlertWebhooks.mu.Unlock()
		for i, h := range globalAlertWebhooks.webhooks {
			if h["id"] == id && h["tenant_id"] == tid {
				globalAlertWebhooks.webhooks = append(globalAlertWebhooks.webhooks[:i], globalAlertWebhooks.webhooks[i+1:]...)
				writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "webhook not found"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
