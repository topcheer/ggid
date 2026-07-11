package server

import (
	"encoding/json"
	"net/http"
)

// POST /api/v1/auth/password-policy/check
func (h *Handler) handlePasswordPolicyCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Password  string   `json:"password"`
		UserID    string   `json:"user_id"`
		History   []string `json:"history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password required")
		return
	}

	var violations []map[string]string
	if len(req.Password) < 8 {
		violations = append(violations, map[string]string{"rule": "min_length", "message": "password must be at least 8 characters"})
	}
	hasUpper, hasLower, hasDigit, hasSpecial := false, false, false, false
	for _, c := range req.Password {
		switch {
			case c >= 'A' && c <= 'Z': hasUpper = true
		case c >= 'a' && c <= 'z': hasLower = true
		case c >= '0' && c <= '9': hasDigit = true
		default: hasSpecial = true
		}
	}
	if !hasUpper { violations = append(violations, map[string]string{"rule": "complexity", "message": "must contain uppercase"}) }
	if !hasLower { violations = append(violations, map[string]string{"rule": "complexity", "message": "must contain lowercase"}) }
	if !hasDigit { violations = append(violations, map[string]string{"rule": "complexity", "message": "must contain digit"}) }
	if !hasSpecial { violations = append(violations, map[string]string{"rule": "complexity", "message": "must contain special character"}) }
	for _, h := range req.History {
		if h == req.Password {
			violations = append(violations, map[string]string{"rule": "history", "message": "password matches a recent password"})
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"is_valid":    len(violations) == 0,
		"violations":  violations,
		"violation_count": len(violations),
	})
}
