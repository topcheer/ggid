package server

import (
	"encoding/json"
	"net/http"
	"sync"
)

// LoginPolicy defines the login security policy for a tenant.
type LoginPolicy struct {
	MaxAttempts            int `json:"max_attempts"`
	LockoutDurationMinutes int `json:"lockout_duration_minutes"`
}

var (
	loginPolicyMu sync.RWMutex
	loginPolicy   = LoginPolicy{
		MaxAttempts:            5,
		LockoutDurationMinutes: 30,
	}
)

// GET/PUT /api/v1/auth/login-policy
func (h *Handler) handleLoginPolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		loginPolicyMu.RLock()
		policy := loginPolicy
		loginPolicyMu.RUnlock()
		writeJSON(w, http.StatusOK, policy)

	case http.MethodPut:
		var req LoginPolicy
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.MaxAttempts < 1 || req.MaxAttempts > 100 {
			writeJSONError(w, http.StatusBadRequest, "max_attempts must be between 1 and 100")
			return
		}
		if req.LockoutDurationMinutes < 1 || req.LockoutDurationMinutes > 1440 {
			writeJSONError(w, http.StatusBadRequest, "lockout_duration_minutes must be between 1 and 1440")
			return
		}

		loginPolicyMu.Lock()
		loginPolicy = req
		loginPolicyMu.Unlock()

		writeJSON(w, http.StatusOK, req)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
