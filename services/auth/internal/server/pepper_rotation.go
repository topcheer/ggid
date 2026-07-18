package server

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
)

// pepperRotationState tracks pepper rotation lifecycle.
type pepperRotationState struct {
	mu              sync.RWMutex
	activePepper    string
	pendingPepper   string // old pepper pending removal
	rotatedAt       time.Time
	usersRehashed   int
}

var pepperState = &pepperRotationState{}

// InitPepperState initializes the pepper state from the current crypto pepper.
func InitPepperState(pepper string) {
	pepperState.mu.Lock()
	defer pepperState.mu.Unlock()
	pepperState.activePepper = pepper
}

// POST /api/v1/auth/password-pepper/rotate — generate new pepper, mark old as pending-removal.
// Next user login re-hashes to the new pepper automatically.
func (h *Handler) handlePepperRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Generate a new random pepper (32 bytes)
	newPepperBytes := make([]byte, 32)
	if _, err := rand.Read(newPepperBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate pepper")
		return
	}
	newPepper := base64.RawURLEncoding.EncodeToString(newPepperBytes)

	pepperState.mu.Lock()
	oldPepper := pepperState.activePepper
	pepperState.pendingPepper = oldPepper
	pepperState.activePepper = newPepper
	pepperState.rotatedAt = time.Now().UTC()
	pepperState.usersRehashed = 0
	pepperState.mu.Unlock()

	// Apply new pepper to crypto package
	crypto.SetPepper(newPepper)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":                "rotated",
		"active_pepper_prefix":  newPepper[:8] + "...",
		"has_pending_removal":   oldPepper != "",
		"pending_pepper_prefix": func() string {
			if oldPepper == "" {
				return ""
			}
			return oldPepper[:8] + "..."
		}(),
		"rotated_at": pepperState.rotatedAt.Format(time.RFC3339),
		"note":       "Users will be re-hashed to new pepper on next login",
	})
}

// GET /api/v1/auth/password-pepper/status — returns active/pending pepper info.
func (h *Handler) handlePepperStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pepperState.mu.RLock()
	defer pepperState.mu.RUnlock()

	resp := map[string]any{
		"has_active_pepper":  pepperState.activePepper != "",
		"has_pending_pepper": pepperState.pendingPepper != "",
		"users_rehashed":     pepperState.usersRehashed,
	}

	if pepperState.activePepper != "" {
		resp["active_pepper_prefix"] = pepperState.activePepper[:8] + "..."
	}
	if pepperState.pendingPepper != "" {
		resp["pending_pepper_prefix"] = pepperState.pendingPepper[:8] + "..."
	}
	if !pepperState.rotatedAt.IsZero() {
		resp["rotated_at"] = pepperState.rotatedAt.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, resp)
}

// RecordPepperRehash increments the rehash counter (called on login when user is re-hashed).
func RecordPepperRehash() {
	pepperState.mu.Lock()
	pepperState.usersRehashed++
	pepperState.mu.Unlock()
}
