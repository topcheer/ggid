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
			writeJSONError(w, http.StatusBadRequest, "user_id is required")
			return
		}

		throttleTracker.RLock()
		state, exists := throttleTracker.states[userID]
		throttleTracker.RUnlock()

		if !exists {
			// Try PG first
			if h.memMapRepo != nil {
				if row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_throttle_states_json", userID); row != nil {
					writeJSON(w, http.StatusOK, row)
					return
				}
			}
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
				// PG write-through (reset)
				if h.memMapRepo != nil {
					h.memMapRepo.StoreJSON(r.Context(), "auth_throttle_states_json", userID, map[string]any{
						"user_id": state.UserID, "is_throttled": false,
						"failed_attempts": 0, "max_attempts": state.MaxAttempts,
						"delay_seconds": 0, "reset_at": "",
						"last_attempt": state.LastAttempt,
					})
				}
			}
		}

		writeJSON(w, http.StatusOK, state)

	case http.MethodPost:
		var req struct {
			UserID string `json:"user_id"`
			Action string `json:"action"` // "fail" or "reset"
		}
		if err := readJSONBody(r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
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
				delay := state.FailedAttempts * state.FailedAttempts
				if delay > 3600 {
					delay = 3600
				}
				state.DelaySeconds = delay
				state.ResetAt = time.Now().UTC().Add(time.Duration(delay) * time.Second).Format(time.RFC3339)
			}
		}

		// PG write-through
		if h.memMapRepo != nil {
			h.memMapRepo.StoreJSON(r.Context(), "auth_throttle_states_json", req.UserID, map[string]any{
				"user_id": state.UserID, "is_throttled": state.IsThrottled,
				"failed_attempts": state.FailedAttempts, "max_attempts": state.MaxAttempts,
				"delay_seconds": state.DelaySeconds, "reset_at": state.ResetAt,
				"last_attempt": state.LastAttempt,
			})
		}

		writeJSON(w, http.StatusOK, state)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// readJSONBody is a helper to decode JSON request bodies.
func readJSONBody(r *http.Request, v any) error {
	return jsonUnmarshalBody(r, v)
}
