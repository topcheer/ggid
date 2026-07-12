package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
)

// secretRotationEntry holds one rotation event.
type secretRotationEntry struct {
	ID               string `json:"id"`
	RotatedAt        string `json:"rotated_at"`
	RotatedBy        string `json:"rotated_by"`
	PreviousThumbprint string `json:"previous_thumbprint"`
	CurrentThumbprint  string `json:"current_thumbprint"`
	AgeDays            int    `json:"age_days"`
	Reason           string `json:"reason"`
}

var secretHistoryStore = struct {
	sync.RWMutex
	data map[string][]secretRotationEntry
}{data: map[string][]secretRotationEntry{
	"web-app": {
		{ID: "r1", RotatedAt: time.Now().UTC().Add(-180*24*time.Hour).Format(time.RFC3339), RotatedBy: "admin", PreviousThumbprint: "a1b2****", CurrentThumbprint: "c3d4****", AgeDays: 180, Reason: "scheduled_rotation"},
		{ID: "r2", RotatedAt: time.Now().UTC().Add(-90*24*time.Hour).Format(time.RFC3339), RotatedBy: "ci-cd", PreviousThumbprint: "c3d4****", CurrentThumbprint: "e5f6****", AgeDays: 90, Reason: "scheduled_rotation"},
		{ID: "r3", RotatedAt: time.Now().UTC().Add(-12*24*time.Hour).Format(time.RFC3339), RotatedBy: "sec-admin", PreviousThumbprint: "e5f6****", CurrentThumbprint: "g7h8****", AgeDays: 12, Reason: "security_incident"},
	},
	"admin-cli": {
		{ID: "r4", RotatedAt: time.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339), RotatedBy: "admin", PreviousThumbprint: "i9j0****", CurrentThumbprint: "k1l2****", AgeDays: 30, Reason: "scheduled_rotation"},
	},
}}

// GET /api/v1/oauth/clients/{id}/secret-history
func handleSecretHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	clientID = strings.TrimSuffix(clientID, "/secret-history")
	clientID = strings.TrimSuffix(clientID, "/")
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
		return
	}

	secretHistoryStore.RLock()
	entries := secretHistoryStore.data[clientID]
	result := make([]secretRotationEntry, len(entries))
	copy(result, entries)
	secretHistoryStore.RUnlock()

	// Compute current secret age
	currentAgeDays := 0
	if len(result) > 0 {
		last := result[len(result)-1]
		if t, err := time.Parse(time.RFC3339, last.RotatedAt); err == nil {
			currentAgeDays = int(time.Now().UTC().Sub(t).Hours() / 24)
		}
	}

	// Generate current thumbprint for verification
	h := sha256.Sum256([]byte(clientID))
	currentThumbprint := hex.EncodeToString(h[:4]) + "****"

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":             clientID,
		"rotation_log":          result,
		"total_rotations":       len(result),
		"current_age_days":      currentAgeDays,
		"current_thumbprint":    currentThumbprint,
		"avg_rotation_interval": func() int {
			if len(result) < 2 {
				return 0
			}
			totalDays := 0
			for i := 1; i < len(result); i++ {
				t1, _ := time.Parse(time.RFC3339, result[i-1].RotatedAt)
				t2, _ := time.Parse(time.RFC3339, result[i].RotatedAt)
				totalDays += int(t2.Sub(t1).Hours() / 24)
			}
			return totalDays / (len(result) - 1)
		}(),
		"checked_at": time.Now().UTC().Format(time.RFC3339),
	})
}
