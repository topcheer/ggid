package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/errors"
)

// passwordStrengthRequest is the DTO for the strength check endpoint.
type passwordStrengthRequest struct {
	Password string `json:"password"`
}

// handlePasswordStrength handles POST /api/v1/auth/password/strength.
// Returns score (0-4), crack time, detected patterns, and suggestions.
func (h *Handler) handlePasswordStrength(w http.ResponseWriter, r *http.Request) {
	// GET returns password policy info
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"min_length":     8,
			"require_upper":  true,
			"require_lower":  true,
			"require_digit":  true,
			"require_symbol": false,
			"score_range":    []string{"weak", "fair", "good", "strong"},
		})
		return
	}

	if r.Method != http.MethodPost {
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req passwordStrengthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Password == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "password is required")
		return
	}

	result := EstimateStrength(req.Password)

	// Check if password is in HIBP breach database (simulated via dictWords overlap).
	// In production, this would call the HIBP API.
	if h.isBreachedPassword(req.Password) {
		result.Score = 0
		result.Warning = "This password has been found in known data breaches"
		result.Suggestions = append([]string{"This password is compromised — do not use it"}, result.Suggestions...)
	}

	writeJSON(w, http.StatusOK, result)
}

// isBreachedPassword checks if a password appears in known breach databases.
// Currently checks the dictionary; in production, this would call HIBP API.
func (h *Handler) isBreachedPassword(password string) bool {
	// Top breached passwords overlap with our dictionary.
	// In production: call HIBP k-anonymity API.
	if _, found := dictWords[strings.ToLower(password)]; found {
		return true
	}
	// Common breached passwords not in dict.
	breached := []string{"123456", "123456789", "12345678", "12345", "1234567", "1234567890"}
	for _, b := range breached {
		if password == b {
			return true
		}
	}
	return false
}

// checkPasswordStrengthGate validates that a password meets the minimum
// strength requirement (score >= 2). Returns an error message if rejected.
func checkPasswordStrengthGate(password string) (bool, string) {
	result := EstimateStrength(password)
	if result.Score < 2 {
		return false, "Password is too weak (score " + itoa(result.Score) + "/4). " + result.Warning
	}
	return true, ""
}

// itoa is a simple int to string converter to avoid strconv import.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
