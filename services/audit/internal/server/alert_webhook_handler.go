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
func (s *HTTPServer) handleAlertWebhooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
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
		hook := map[string]any{
			"id":     uuid.New().String(),
			"url":    req.URL,
			"secret": req.Secret,
			"active": true,
		}
		globalAlertWebhooks.mu.Lock()
		globalAlertWebhooks.webhooks = append(globalAlertWebhooks.webhooks, hook)
		globalAlertWebhooks.mu.Unlock()
		writeJSON(w, http.StatusCreated, hook)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
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
