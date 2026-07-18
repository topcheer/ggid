package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// provisioningWebhook configures outbound webhooks for IdP provisioning callbacks.
type provisioningWebhook struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`    // user.created, user.updated, user.deactivated, user.deleted
	Secret    string   `json:"secret,omitempty"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"created_at"`
}

var provisioningWebhookStore = struct {
	sync.RWMutex
	webhooks map[string]*provisioningWebhook
}{webhooks: make(map[string]*provisioningWebhook)}

// POST   /api/v1/users/provisioning-webhooks — create a webhook
// GET    /api/v1/users/provisioning-webhooks — list webhooks
// DELETE /api/v1/users/provisioning-webhooks?id=X — delete a webhook
func (h *HTTPHandler) handleProvisioningWebhooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			URL    string   `json:"url"`
			Events []string `json:"events"`
			Secret string   `json:"secret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.URL == "" {
			writeJSONError(w, http.StatusBadRequest, "url is required")
			return
		}
		if len(req.Events) == 0 {
			req.Events = []string{"user.created", "user.updated", "user.deactivated"}
		}

		validEvents := map[string]bool{
			"user.created": true, "user.updated": true, "user.deactivated": true,
			"user.deleted": true, "user.reactivated": true, "user.role_changed": true,
		}
		for _, e := range req.Events {
			if !validEvents[e] {
				writeJSONError(w, http.StatusBadRequest, "unsupported event type: "+e)
				return
			}
		}

		wh := &provisioningWebhook{
			ID:        uuid.New().String(),
			URL:       req.URL,
			Events:    req.Events,
			Secret:    req.Secret,
			Active:    true,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		provisioningWebhookStore.Lock()
		provisioningWebhookStore.webhooks[wh.ID] = wh
		provisioningWebhookStore.Unlock()

		writeJSON(w, http.StatusCreated, wh)

	case http.MethodGet:
		provisioningWebhookStore.RLock()
		result := []*provisioningWebhook{}
		for _, wh := range provisioningWebhookStore.webhooks {
			result = append(result, wh)
		}
		provisioningWebhookStore.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"webhooks": result,
			"total":    len(result),
		})

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id query parameter is required")
			return
		}

		provisioningWebhookStore.Lock()
		_, exists := provisioningWebhookStore.webhooks[id]
		if exists {
			delete(provisioningWebhookStore.webhooks, id)
		}
		provisioningWebhookStore.Unlock()

		if !exists {
			writeJSONError(w, http.StatusNotFound, "webhook not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
			"id":      id,
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
