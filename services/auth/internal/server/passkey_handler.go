package server

import (
	"net/http"
	"time"
)

type Passkey struct {
	ID        string    `json:"id"`
	Device    string    `json:"device"`
	Platform  string    `json:"platform"` // apple, google, microsoft
	CreatedAt time.Time `json:"created_at"`
	LastUsed  *time.Time `json:"last_used"`
	Synced    bool      `json:"synced"`
	SyncProvider string `json:"sync_provider,omitempty"`
}

// GET /api/v1/auth/passkeys/status?user_id=X
func (h *Handler) handlePasskeyStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id required")
		return
	}

	now := time.Now().UTC()
	lastUsed := now.Add(-2 * time.Hour)
	passkeys := []Passkey{
		{ID: "pk-001", Device: "iPhone 15 Pro", Platform: "apple", CreatedAt: now.Add(-30 * 24 * time.Hour), LastUsed: &lastUsed, Synced: true, SyncProvider: "iCloud Keychain"},
		{ID: "pk-002", Device: "MacBook Pro", Platform: "apple", CreatedAt: now.Add(-30 * 24 * time.Hour), LastUsed: &lastUsed, Synced: true, SyncProvider: "iCloud Keychain"},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":           userID,
		"passkeys":          passkeys,
		"total":             len(passkeys),
		"multi_device_sync":  true,
		"sync_provider":     "iCloud Keychain",
	})
}
