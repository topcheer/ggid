package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type AdminOverride struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	ClientID   string    `json:"client_id"`
	Scope      string    `json:"scope"`
	Action     string    `json:"action"` // grant, revoke
	AdminID    string    `json:"admin_id"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

// POST /api/v1/oauth/consent/admin-override
func handleConsentAdminOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req struct {
		UserID   string `json:"user_id"`
		ClientID string `json:"client_id"`
		Scope    string `json:"scope"`
		Action   string `json:"action"` // grant, revoke
		AdminID  string `json:"admin_id"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if req.UserID == "" || req.ClientID == "" || req.Scope == "" || req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "user_id, client_id, scope, action required"})
		return
	}

	override := &AdminOverride{
		ID: uuid.New().String(), UserID: req.UserID, ClientID: req.ClientID,
		Scope: req.Scope, Action: req.Action, AdminID: req.AdminID,
		Reason: req.Reason, CreatedAt: time.Now().UTC(),
	}
	if mapRepoVar != nil {
		b, _ := json.Marshal(override)
		var dataMap map[string]any
		json.Unmarshal(b, &dataMap)
		mapRepoVar.Store(r.Context(), "oauth_consent_overrides", override.ID, dataMap)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "applied",
		"override": override,
		"message":  "admin override applied — user consent flow bypassed for scope: " + req.Scope,
	})
}
