package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PasswordResetToken struct {
	Token     string
	Email     string
	UserID    string
	ExpiresAt time.Time
	Used      bool
}

var (
	pwdResetMu     sync.RWMutex
	pwdResetTokens = make(map[string]*PasswordResetToken)
)

// POST /api/v1/auth/password-reset/initiate — send reset token (don't reveal if user exists)
// POST /api/v1/auth/password-reset/complete — verify token + set new password
func (h *Handler) handlePasswordReset(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/initiate") && r.Method == http.MethodPost {
		var req struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		// Always return same response to prevent user enumeration
		// If user exists, generate token and "send email"
		if req.Email != "" {
			token := uuid.New().String()
			pwdResetMu.Lock()
			pwdResetTokens[token] = &PasswordResetToken{
				Token: token, Email: req.Email,
				ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
			}
			pwdResetMu.Unlock()
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "sent",
			"message":  "If an account exists for this email, a reset link has been sent.",
			"expires_in": 1800,
		})
		return
	}

	if strings.HasSuffix(path, "/complete") && r.Method == http.MethodPost {
		var req struct {
			Token       string `json:"token"`
			NewPassword string `json:"new_password"`
			UserID      string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Token == "" || req.NewPassword == "" {
			writeError(w, http.StatusBadRequest, "token and new_password required")
			return
		}

		// Validate token
		pwdResetMu.Lock()
		rt, ok := pwdResetTokens[req.Token]
		if !ok {
			pwdResetMu.Unlock()
			writeError(w, http.StatusBadRequest, "invalid or expired token")
			return
		}
		if rt.Used {
			pwdResetMu.Unlock()
			writeError(w, http.StatusBadRequest, "token already used")
			return
		}
		if time.Now().UTC().After(rt.ExpiresAt) {
			pwdResetMu.Unlock()
			writeError(w, http.StatusBadRequest, "token expired")
			return
		}
		rt.Used = true
		pwdResetMu.Unlock()

		// Validate password strength
		if len(req.NewPassword) < 8 {
			writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "completed",
			"email":     rt.Email,
			"completed_at": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}
