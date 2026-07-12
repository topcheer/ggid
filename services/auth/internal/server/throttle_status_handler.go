package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// throttleState tracks login attempt throttling per user.
type throttleState struct {
	UserID         string `json:"user_id"`
	IsThrottled    bool   `json:"is_throttled"`
	FailedAttempts int    `json:"failed_attempts"`
	MaxAttempts    int    `json:"max_attempts"`
	DelaySeconds   int    `json:"delay_seconds"`
	ResetAt        string `json:"reset_at"`
	LastAttempt    string `json:"last_attempt"`
}

var throttleTracker = struct {
	sync.RWMutex
	states map[string]*throttleState
}{states: make(map[string]*throttleState)}

// GET /api/v1/auth/throttle-status?user_id=X
// POST /api/v1/auth/throttle-status — record a failed attempt (for testing)
func (h *Handler) handleThrottleStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			writeError(w, http.StatusBadRequest, "user_id is required")
			return
		}

		throttleTracker.RLock()
		state, exists := throttleTracker.states[userID]
		throttleTracker.RUnlock()

		if !exists {
			writeJSON(w, http.StatusOK, &throttleState{
				UserID:         userID,
				IsThrottled:    false,
				FailedAttempts: 0,
				MaxAttempts:    5,
				DelaySeconds:   0,
				ResetAt:        "",
			})
			return
		}

		// Check if throttle has expired
		if state.ResetAt != "" {
			resetTime, _ := time.Parse(time.RFC3339, state.ResetAt)
			if time.Now().UTC().After(resetTime) {
				state.IsThrottled = false
				state.FailedAttempts = 0
				state.DelaySeconds = 0
				state.ResetAt = ""
			}
		}

		writeJSON(w, http.StatusOK, state)

	case http.MethodPost:
		var req struct {
			UserID string `json:"user_id"`
			Action string `json:"action"` // "fail" or "reset"
		}
		if err := readJSONBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.UserID == "" {
			req.UserID = uuid.New().String()
		}

		throttleTracker.Lock()
		defer throttleTracker.Unlock()

		state, exists := throttleTracker.states[req.UserID]
		if !exists {
			state = &throttleState{
				UserID:         req.UserID,
				FailedAttempts: 0,
				MaxAttempts:    5,
			}
			throttleTracker.states[req.UserID] = state
		}

		switch req.Action {
		case "reset":
			state.IsThrottled = false
			state.FailedAttempts = 0
			state.DelaySeconds = 0
			state.ResetAt = ""
		case "fail", "":
			state.FailedAttempts++
			state.LastAttempt = time.Now().UTC().Format(time.RFC3339)

			if state.FailedAttempts >= state.MaxAttempts {
				state.IsThrottled = true
				delay := state.FailedAttempts * state.FailedAttempts // exponential: 25, 36, 49...
				if delay > 3600 {
					delay = 3600
				}
				state.DelaySeconds = delay
				state.ResetAt = time.Now().UTC().Add(time.Duration(delay) * time.Second).Format(time.RFC3339)
			}
		}

		writeJSON(w, http.StatusOK, state)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// readJSONBody is a helper to decode JSON request bodies.
func readJSONBody(r *http.Request, v any) error {
	return jsonUnmarshalBody(r, v)
}
