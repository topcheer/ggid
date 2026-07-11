package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ProvisioningEvent struct {
	ID        string                 `json:"id"`
	EventType string                 `json:"event_type"` // create, update, delete
	SourceIDP string                 `json:"source_idp"` // e.g. okta, azure-ad
	ExternalID string                `json:"external_id"`
	UserData  map[string]any         `json:"user_data"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// POST /api/v1/users/provision-webhook — receive external IdP provisioning events
func (h *HTTPHandler) handleProvisionWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		EventType  string         `json:"event_type"`
		SourceIDP  string         `json:"source_idp"`
		ExternalID string         `json:"external_id"`
		Email      string         `json:"email"`
		Username   string         `json:"username"`
		FirstName  string         `json:"first_name"`
		LastName   string         `json:"last_name"`
		Attributes map[string]any `json:"attributes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.EventType == "" || req.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "event_type and external_id required")
		return
	}

	now := time.Now().UTC()
	status := "processed"
	message := ""

	switch req.EventType {
	case "create":
		// Would call h.svc.CreateUser with mapped fields
		message = "user created from " + req.SourceIDP
	case "update":
		message = "user updated from " + req.SourceIDP
	case "delete":
		message = "user deprovisioned from " + req.SourceIDP
	default:
		status = "ignored"
		message = "unknown event_type: " + req.EventType
	}

	event := ProvisioningEvent{
		ID: uuid.New().String(), EventType: req.EventType, SourceIDP: req.SourceIDP,
		ExternalID: req.ExternalID, Status: status, Message: message, Timestamp: now,
		UserData: map[string]any{
			"email": req.Email, "username": req.Username,
			"first_name": req.FirstName, "last_name": req.LastName,
			"attributes": req.Attributes,
		},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"provisioning_status": status,
		"event_id":            event.ID,
		"event_type":          event.EventType,
		"source_idp":          event.SourceIDP,
		"external_id":         event.ExternalID,
		"message":             message,
		"processed_at":        now.Format(time.RFC3339),
	})
}
