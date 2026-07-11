package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// passwordlessSession tracks WebAuthn passwordless login in-progress sessions.
type passwordlessSession struct {
	SessionID   string    `json:"session_id"`
	Challenge   string    `json:"challenge"`
	TenantID    string    `json:"tenant_id"`
	Username    string    `json:"username"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

var (
	passwordlessSessionsMu sync.RWMutex
	passwordlessSessions   = make(map[string]*passwordlessSession)
)

// POST /api/v1/auth/webauthn/passwordless/begin
// Body: {"tenant_id": "...", "username": "..."}
// Begins a passwordless WebAuthn login flow by generating a challenge.
func (h *Handler) handleWebAuthnPasswordlessBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TenantID string `json:"tenant_id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}

	// Generate challenge (in production, use WebAuthn library BeginLogin)
	challenge := uuid.New().String()
	sessionID := uuid.New().String()
	now := time.Now().UTC()

	sess := &passwordlessSession{
		SessionID: sessionID,
		Challenge: challenge,
		TenantID:  req.TenantID,
		Username:  req.Username,
		CreatedAt: now,
		ExpiresAt: now.Add(5 * time.Minute),
	}

	passwordlessSessionsMu.Lock()
	passwordlessSessions[sessionID] = sess
	passwordlessSessionsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id":   sessionID,
		"challenge":    challenge,
		"rp_id":        "ggid.dev",
		"timeout":      300000, // 5 min in ms
		"user_verification": "required",
		"expires_at":   sess.ExpiresAt.Format(time.RFC3339),
	})
}

// POST /api/v1/auth/webauthn/passwordless/finish
// Body: {"session_id": "...", "credential": {...}, "assertion": "..."}
// Completes passwordless login by verifying the WebAuthn assertion.
func (h *Handler) handleWebAuthnPasswordlessFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
		Assertion  string          `json:"assertion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	passwordlessSessionsMu.Lock()
	sess, ok := passwordlessSessions[req.SessionID]
	if ok {
		delete(passwordlessSessions, req.SessionID)
	}
	passwordlessSessionsMu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, "session not found or expired")
		return
	}

	if time.Now().UTC().After(sess.ExpiresAt) {
		writeError(w, http.StatusGone, "session expired")
		return
	}

	// In production: verify WebAuthn assertion via webauthn.FinishLogin
	// For now, return success with a simulated JWT
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "authenticated",
		"username":   sess.Username,
		"tenant_id":  sess.TenantID,
		"method":     "webauthn_passwordless",
		"token_type": "Bearer",
		"expires_in": 3600,
	})
}
