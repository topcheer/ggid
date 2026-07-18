package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/service"
)

// handleJWKSRotateWithKP handles POST /api/v1/oauth/jwks/rotate.
func handleJWKSRotateWithKP(w http.ResponseWriter, r *http.Request, kp *service.RotatingKeyProvider) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	if kp == nil {
		writeJSON(w, http.StatusOK, map[string]any{"keys": []any{}, "rotation_enabled": false})
		return
	}

	oldKID := kp.KeyID()
	if err := kp.RotateKey(); err != nil {
		slog.Error("jwks rotation error", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
		return
	}
	newKID := kp.KeyID()
	prevKID := kp.PreviousKeyID()

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "rotated",
		"active_kid":   newKID,
		"previous_kid": prevKID,
		"old_kid":      oldKID,
		"rotated_at":   time.Now().UTC().Format(time.RFC3339),
		"grace_period": "24h",
	})
}

// handleJWKSRotationStatusWithKP handles GET /api/v1/oauth/jwks/rotation-status.
func handleJWKSRotationStatusWithKP(w http.ResponseWriter, r *http.Request, kp *service.RotatingKeyProvider) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	if kp == nil {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false})
		return
	}

	activeKID := kp.KeyID()
	prevKID := kp.PreviousKeyID()
	graceExpired := kp.IsGracePeriodExpired()

	resp := map[string]any{
		"configured":           true,
		"active_kid":           activeKID,
		"has_previous_key":     prevKID != "",
		"grace_period_expired": graceExpired,
	}
	if prevKID != "" {
		resp["previous_kid"] = prevKID
	}

	writeJSON(w, http.StatusOK, resp)
}
