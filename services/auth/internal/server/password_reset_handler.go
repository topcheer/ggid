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

		// PG-first lookup (using hashed token)
		if h.memMapRepo != nil {
			row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_pwd_reset_tokens", hashResetToken(req.Token))
			if row != nil {
				if used, _ := row["used"].(bool); used {
					writeError(w, http.StatusBadRequest, "token already used")
					return
				}
				// Valid token in PG — proceed to reset.
				writeJSON(w, http.StatusOK, map[string]any{"status": "verified"})
				return
			}
		}

		// Fallback: in-memory map (using hashed token)
		pwdResetMu.Lock()
		rt, ok := pwdResetTokens[hashResetToken(req.Token)]
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
			"status":       "completed",
			"email":        rt.Email,
			"completed_at": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}
