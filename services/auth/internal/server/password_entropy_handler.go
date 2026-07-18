package server

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
)

// POST /api/v1/auth/password-entropy/check
func (h *Handler) handlePasswordEntropy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "password required")
		return
	}

	// Calculate Shannon entropy
	charFreq := make(map[rune]float64)
	for _, c := range req.Password {
		charFreq[c]++
	}
	length := float64(len(req.Password))
	entropy := 0.0
	for _, count := range charFreq {
		p := count / length
		entropy -= p * math.Log2(p)
	}
	entropyBits := entropy * length

	// Calculate character pool size
	poolSize := 0
	if strings.ContainsAny(req.Password, "abcdefghijklmnopqrstuvwxyz") {
		poolSize += 26
	}
	if strings.ContainsAny(req.Password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		poolSize += 26
	}
	if strings.ContainsAny(req.Password, "0123456789") {
		poolSize += 10
	}
	if strings.ContainsAny(req.Password, "!@#$%^&*()_+-=[]{}|;:',.<>?/`~") {
		poolSize += 32
	}

	// Strength assessment
	var level string
	var suggestions []string
	switch {
	case entropyBits < 28:
		level = "weak"
		suggestions = []string{"Use at least 12 characters", "Mix uppercase, lowercase, numbers, and symbols"}
	case entropyBits < 36:
		level = "fair"
		suggestions = []string{"Add more characters", "Include special characters"}
	case entropyBits < 60:
		level = "good"
		suggestions = []string{"Consider adding a passphrase for extra strength"}
	default:
		level = "strong"
		suggestions = []string{}
	}
	if poolSize < 62 {
		suggestions = append(suggestions, "Mix character types to increase pool size")
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entropy_bits":   int(entropyBits),
		"pool_size":      poolSize,
		"strength_level": level,
		"length":         len(req.Password),
		"suggestions":    suggestions,
	})
}
