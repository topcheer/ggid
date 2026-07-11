package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SecretStatus struct {
	ClientID      string     `json:"client_id"`
	CurrentSecret string     `json:"current_secret_preview"`
	OldSecret     string     `json:"old_secret_preview,omitempty"`
	GracePeriod   bool       `json:"grace_period"`
	GraceExpires  *time.Time `json:"grace_expires_at,omitempty"`
	RotatedAt     time.Time  `json:"rotated_at"`
}

var (
	secretStatusMu sync.RWMutex
	secretStatuses = make(map[string]*SecretStatus)
)

// POST /api/v1/oauth/clients/{id}/rotate-secret
// GET /api/v1/oauth/clients/{id}/secret-status
func handleClientSecretRotation(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.URL.Path, "/rotate-secret") && !strings.Contains(r.URL.Path, "/secret-status") {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	clientID := extractIDFromPath(r.URL.Path)

	if strings.HasSuffix(r.URL.Path, "/secret-status") && r.Method == http.MethodGet {
		secretStatusMu.RLock()
		status, ok := secretStatuses[clientID]
		secretStatusMu.RUnlock()
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{
				"client_id": clientID, "rotated_at": time.Time{}, "grace_period": false,
			})
			return
		}
		writeJSON(w, http.StatusOK, status)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/rotate-secret") && r.Method == http.MethodPost {
		newSecret := uuid.New().String() + uuid.New().String()
		graceExpiry := time.Now().UTC().Add(24 * time.Hour)
		status := &SecretStatus{
			ClientID:      clientID,
			CurrentSecret: newSecret[:8] + "****",
			OldSecret:     "prev_****",
			GracePeriod:   true,
			GraceExpires:  &graceExpiry,
			RotatedAt:     time.Now().UTC(),
		}
		secretStatusMu.Lock()
		secretStatuses[clientID] = status
		secretStatusMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":          "rotated",
			"client_id":       clientID,
			"new_secret":      newSecret,
			"grace_period":    "24h",
			"grace_expires":   graceExpiry.Format(time.RFC3339),
			"old_secret_valid_until": graceExpiry.Format(time.RFC3339),
			"rotated_at":      time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
}

func extractIDFromPath(path string) string {
	path = strings.TrimPrefix(path, "/api/v1/oauth/clients/")
	path = strings.TrimSuffix(path, "/rotate-secret")
	path = strings.TrimSuffix(path, "/secret-status")
	return strings.TrimSuffix(path, "/")
}
