package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type MFAFactor struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Secret    string     `json:"-"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
}

var (
	mfaFactorMu sync.RWMutex
	mfaFactors  = []MFAFactor{
		{ID: "mf-001", UserID: "u-042", Type: "totp", Name: "Authenticator App", Enabled: true, CreatedAt: time.Now().UTC().Add(-720 * time.Hour)},
		{ID: "mf-002", UserID: "u-042", Type: "webauthn", Name: "YubiKey", Enabled: true, CreatedAt: time.Now().UTC().Add(-360 * time.Hour)},
		{ID: "mf-003", UserID: "u-042", Type: "backup", Name: "Backup Codes", Enabled: true, CreatedAt: time.Now().UTC().Add(-720 * time.Hour)},
	}
)

// GET /api/v1/auth/mfa/factors?user_id=X
// DELETE /api/v1/auth/mfa/factors/{id}
func (h *Handler) handleMFAFactors(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		userID := r.URL.Query().Get("user_id")
		mfaFactorMu.RLock()
		var result []MFAFactor
		for _, f := range mfaFactors {
			if userID == "" || f.UserID == userID { result = append(result, f) }
		}
		mfaFactorMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"factors": result, "count": len(result)})
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Type     string `json:"type"`
			Friendly string `json:"friendly_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Type == "" {
			req.Type = "totp"
		}

		// Extract user_id from JWT
		authHeader := r.Header.Get("Authorization")
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		_, parseErr := jwt.ParseWithClaims(tokenStr, claims, func(tok *jwt.Token) (any, error) {
			return h.authSvc.PublicKey(), nil
		})
		userID, _ := claims["sub"].(string)
		if parseErr != nil || userID == "" {
			userID = r.URL.Query().Get("user_id")
		}

		factorID := uuid.NewString()
		// For TOTP, generate a placeholder secret (production would use otp.NewTOTP)
		secret := "JBSWY3DPEHPK3PXP" // example TOTP secret
		factor := MFAFactor{
			ID:        factorID,
			UserID:    userID,
			Type:      req.Type,
			Secret:    secret,
			Enabled:   false,
			CreatedAt: time.Now().UTC(),
		}
		if req.Friendly != "" {
			factor.Name = req.Friendly
		}
		mfaFactorMu.Lock()
		mfaFactors = append(mfaFactors, factor)
		mfaFactorMu.Unlock()

		writeJSON(w, http.StatusCreated, map[string]any{
			"factor_id":      factorID,
			"type":           req.Type,
			"secret":         secret,
			"otpauth_uri":    fmt.Sprintf("otpauth://totp/GGID:%s?secret=%s&issuer=GGID", userID, secret),
			"status":         "pending_verification",
		})
		return
	}
	if r.Method == http.MethodDelete {
		factorID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/mfa/factors/")
		mfaFactorMu.Lock()
		for i, f := range mfaFactors {
			if f.ID == factorID {
				mfaFactors = append(mfaFactors[:i], mfaFactors[i+1:]...)
				break
			}
		}
		mfaFactorMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "factor_id": factorID})
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
