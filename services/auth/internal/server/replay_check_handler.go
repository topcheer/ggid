package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// replayEntry stores a login request hash for replay detection.
type replayEntry struct {
	RequestHash string    `json:"request_hash"`
	UserID      string    `json:"user_id"`
	Timestamp   time.Time `json:"timestamp"`
	IPAddress   string    `json:"ip_address"`
}

var replayStore = struct {
	sync.RWMutex
	entries map[string]*replayEntry // hash → entry
}{entries: make(map[string]*replayEntry)}

// POST /api/v1/auth/replay-check
// Body: {"request_data": "...", "user_id": "...", "ip_address": "...", "timestamp_window_seconds": 30}
func (h *Handler) handleReplayCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		RequestData           string `json:"request_data"`
		UserID                string `json:"user_id"`
		IPAddress             string `json:"ip_address"`
		TimestampWindowSeconds int   `json:"timestamp_window_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RequestData == "" {
		writeJSONError(w, http.StatusBadRequest, "request_data is required")
		return
	}
	if req.TimestampWindowSeconds <= 0 {
		req.TimestampWindowSeconds = 30
	}

	// Compute hash
	hsum := sha256.Sum256([]byte(req.RequestData))
	requestHash := hex.EncodeToString(hsum[:])

	now := time.Now().UTC()

	replayStore.Lock()
	defer replayStore.Unlock()

	// Check for existing entry within window
	if existing, found := replayStore.entries[requestHash]; found {
		windowEnd := existing.Timestamp.Add(time.Duration(req.TimestampWindowSeconds) * time.Second)
		if now.Before(windowEnd) {
			// This is a replay
			writeJSON(w, http.StatusOK, map[string]any{
				"is_replay":         true,
				"request_hash":      requestHash,
				"original_request": map[string]any{
					"user_id":    existing.UserID,
					"ip_address": existing.IPAddress,
					"timestamp":  existing.Timestamp.Format(time.RFC3339),
				},
				"time_since_original_s": int(now.Sub(existing.Timestamp).Seconds()),
				"window_seconds":        req.TimestampWindowSeconds,
				"recommended_action":    "reject_replay",
				"checked_at":            now.Format(time.RFC3339),
			})
			return
		}
	}

	// Not a replay — record this request
	entry := &replayEntry{
		RequestHash: requestHash,
		UserID:      req.UserID,
		Timestamp:   now,
		IPAddress:   req.IPAddress,
	}
	replayStore.entries[requestHash] = entry

	// Cleanup old entries (keep last 500)
	if len(replayStore.entries) > 500 {
		count := 0
		for hash, e := range replayStore.entries {
			if now.Sub(e.Timestamp) > time.Duration(req.TimestampWindowSeconds*2)*time.Second {
				delete(replayStore.entries, hash)
				count++
			}
			if count > 100 {
				break
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"is_replay":       false,
		"request_hash":    requestHash,
		"recorded":        true,
		"window_seconds":  req.TimestampWindowSeconds,
		"checked_at":      now.Format(time.RFC3339),
		"check_id":        uuid.New().String(),
	})
}
