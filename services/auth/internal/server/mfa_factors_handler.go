package server

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type MFAFactor struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"` // totp, webauthn, sms, backup
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
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
