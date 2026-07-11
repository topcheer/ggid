package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/auth/internal/service"
	"github.com/ggid/ggid/services/auth/internal/webauthn"
	"github.com/google/uuid"
)

func parseUUIDSafe(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

func (h *Handler) handleImpersonate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		ImpersonatorID string `json:"impersonator_id"`
		TargetUserID   string `json:"target_user_id"`
		TenantID       string `json:"tenant_id"`
		Reason         string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	tok, err := service.IssueImpersonationToken(
		parseUUIDSafe(req.ImpersonatorID), parseUUIDSafe(req.TargetUserID),
		parseUUIDSafe(req.TenantID), req.Reason,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, tok)
}

func (h *Handler) handleImpersonateRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		TokenID string `json:"token_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := service.RevokeImpersonationToken(parseUUIDSafe(req.TokenID)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) handleConditionalUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Challenge        string   `json:"challenge"`
		RPID             string   `json:"rp_id"`
		UserID           string   `json:"user_id"`
		UserVerification string   `json:"user_verification"`
		CredentialIDs    [][]byte `json:"credential_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	resp := webauthn.BeginConditionalUI(&webauthn.ConditionalUIRequest{
		Challenge:        req.Challenge,
		RPID:             req.RPID,
		UserID:           req.UserID,
		UserVerification: req.UserVerification,
	}, req.CredentialIDs)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleSendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Channel string `json:"channel"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "queued",
		"channel": req.Channel,
		"to":      req.To,
	})
}

func (h *Handler) handleExpiryStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id required"})
		return
	}
	notif := service.GetExpiryNotification(parseUUIDSafe(userID))
	if notif == nil {
		writeJSON(w, http.StatusOK, map[string]any{"notified": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notified":   true,
		"expires_at": notif.ExpiresAt.Format(time.RFC3339),
		"message":    notif.Message,
	})
}
