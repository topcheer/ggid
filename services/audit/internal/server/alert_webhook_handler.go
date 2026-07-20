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
func (s *HTTPServer) handleAlertWebhooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if s.pool != nil {
			rows, err := s.pool.Query(r.Context(), `
				SELECT id::text, url, COALESCE(secret, ''), active, created_at
				FROM audit_alert_webhooks ORDER BY created_at DESC`)
			if err == nil {
				defer rows.Close()
				webhooks := []map[string]any{}
				for rows.Next() {
					var id, url, secret string
					var active bool
					var created interface{}
					_ = rows.Scan(&id, &url, &secret, &active, &created)
					webhooks = append(webhooks, map[string]any{
						"id": id, "url": url, "secret": secret, "active": active, "created_at": created,
					})
				}
				writeJSON(w, http.StatusOK, map[string]any{"webhooks": webhooks})
				return
			}
		}
		globalAlertWebhooks.mu.RLock()
		defer globalAlertWebhooks.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"webhooks": globalAlertWebhooks.webhooks})

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
			"id":     hookID,
			"url":    req.URL,
			"secret": req.Secret,
			"active": true,
		}
		if s.pool != nil {
			_, err := s.pool.Exec(r.Context(), `
				INSERT INTO audit_alert_webhooks (id, url, secret, active)
				VALUES ($1, $2, $3, true)`, hookID, req.URL, req.Secret)
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
		if s.pool != nil {
			_, err := s.pool.Exec(r.Context(), `DELETE FROM audit_alert_webhooks WHERE id::text = $1`, id)
			if err == nil {
				writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
				return
			}
		}
		globalAlertWebhooks.mu.Lock()
		defer globalAlertWebhooks.mu.Unlock()
		for i, h := range globalAlertWebhooks.webhooks {
			if h["id"] == id {
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
