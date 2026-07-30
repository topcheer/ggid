package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// hashResetToken returns SHA-256 hex of a reset token for at-rest storage.
func hashResetToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

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
			tokenHash := hashResetToken(token)
			pwdResetMu.Lock()
			pwdResetTokens[tokenHash] = &PasswordResetToken{
				Token: tokenHash, Email: req.Email,
				ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
			}
			pwdResetMu.Unlock()

			// PG write-through (store hash, not plaintext)
			if h.memMapRepo != nil {
				h.memMapRepo.StoreJSON(r.Context(), "auth_pwd_reset_tokens", tokenHash, map[string]any{
					"token": tokenHash, "email": req.Email,
					"expires_at": time.Now().UTC().Add(30 * time.Minute), "used": false,
				})
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "sent",
			"message":    "If an account exists for this email, a reset link has been sent.",
			"expires_in": 1800,
		})
		return
	}

	if strings.HasSuffix(path, "/complete") && r.Method == http.MethodPost {
		// SECURITY (R16 P1): Delegate to the service-layer resetPassword
		// which uses GETDEL atomic token consumption + SetPassword +
		// RevokeAllForUser. The previous stub did not check expiry, did
		// not mark tokens as used, and never actually changed the password.
		var req struct {
			Token       string `json:"token"`
			NewPassword string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Token == "" || req.NewPassword == "" {
			writeError(w, http.StatusBadRequest, "token and new_password required")
			return
		}
		if err := h.authSvc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "completed",
			"completed_at": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}
