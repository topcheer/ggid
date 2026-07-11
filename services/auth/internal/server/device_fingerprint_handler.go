package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type DeviceFingerprint struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserAgent    string    `json:"user_agent"`
	Screen       string    `json:"screen"`
	Timezone     string    `json:"timezone"`
	PluginsHash  string    `json:"plugins_hash"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	SessionCount int       `json:"session_count"`
}

var (
	deviceFPRegistryMu sync.RWMutex
	deviceFPRegistry   = make(map[string]*DeviceFingerprint)
)

// POST /api/v1/auth/devices/register — register device fingerprint
// GET /api/v1/auth/devices/list?user_id=X — list user's devices
func (h *Handler) handleDeviceFingerprint(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			UserID      string `json:"user_id"`
			UserAgent   string `json:"user_agent"`
			Screen      string `json:"screen"`
			Timezone    string `json:"timezone"`
			PluginsHash string `json:"plugins_hash"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.UserID == "" {
			writeError(w, http.StatusBadRequest, "user_id required")
			return
		}

		// Check if this fingerprint already exists (same user + same hash)
		fpKey := req.UserID + ":" + req.PluginsHash + ":" + req.UserAgent
		now := time.Now().UTC()

		deviceFPRegistryMu.Lock()
		if fp, ok := deviceFPRegistry[fpKey]; ok {
			fp.LastSeen = now
			fp.SessionCount++
			deviceFPRegistryMu.Unlock()
			writeJSON(w, http.StatusOK, map[string]any{
				"status": "existing", "device": fp,
			})
			return
		}
		fp := &DeviceFingerprint{
			ID: uuid.New().String(), UserID: req.UserID, UserAgent: req.UserAgent,
			Screen: req.Screen, Timezone: req.Timezone, PluginsHash: req.PluginsHash,
			FirstSeen: now, LastSeen: now, SessionCount: 1,
		}
		deviceFPRegistry[fpKey] = fp
		deviceFPRegistryMu.Unlock()

		writeJSON(w, http.StatusCreated, map[string]any{
			"status": "registered", "device": fp,
		})
		return
	}

	if r.Method == http.MethodGet {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			writeError(w, http.StatusBadRequest, "user_id required")
			return
		}

		deviceFPRegistryMu.RLock()
		var devices []*DeviceFingerprint
		for _, fp := range deviceFPRegistry {
			if fp.UserID == userID {
				devices = append(devices, fp)
			}
		}
		deviceFPRegistryMu.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":      userID,
			"devices":      devices,
			"device_count": len(devices),
		})
		return
	}

	_ = strings.TrimSpace
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
