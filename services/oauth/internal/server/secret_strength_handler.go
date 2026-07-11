package server

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
)

// POST /api/v1/oauth/clients/{id}/validate-secret
func handleValidateClientSecret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"}); return
	}
	clientID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/"), "/validate-secret")
	if clientID == "" || strings.Contains(clientID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid client_id"}); return
	}
	var req struct{ Secret string `json:"secret"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"}); return
	}
	if req.Secret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "secret required"}); return
	}
	// Entropy
	charFreq := make(map[rune]float64)
	for _, c := range req.Secret { charFreq[c]++ }
	length := float64(len(req.Secret))
	entropy := 0.0
	for _, count := range charFreq { p := count / length; entropy -= p * math.Log2(p) }
	entropyBits := entropy * length
	// Pool size
	poolSize := 0
	if strings.ContainsAny(req.Secret, "abcdefghijklmnopqrstuvwxyz") { poolSize += 26 }
	if strings.ContainsAny(req.Secret, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") { poolSize += 26 }
	if strings.ContainsAny(req.Secret, "0123456789") { poolSize += 10 }
	if strings.ContainsAny(req.Secret, "!@#$%^&*()_+-=[]{}|;:',.<>?/") { poolSize += 32 }
	// Crack time (simplified: poolSize^length / 1e10 guesses/sec)
	combinations := math.Pow(float64(poolSize), length)
	crackSeconds := combinations / 1e10
	isStrong := entropyBits >= 60 && len(req.Secret) >= 32
	var suggestions []string
	if len(req.Secret) < 32 { suggestions = append(suggestions, "use at least 32 characters") }
	if poolSize < 62 { suggestions = append(suggestions, "mix uppercase, lowercase, digits, and special characters") }
	writeJSON(w, http.StatusOK, map[string]any{
		"client_id": clientID, "is_strong": isStrong,
		"length": len(req.Secret), "entropy_bits": int(entropyBits),
		"pool_size": poolSize, "diversity": poolSize,
		"crack_time_seconds": int64(crackSeconds),
		"crack_time_human": formatDuration(int64(crackSeconds)),
		"suggestions": suggestions,
	})
}

func formatDuration(seconds int64) string {
	if seconds < 60 { return "<1 minute" }
	if seconds < 3600 { return "<1 hour" }
	if seconds < 86400 { return "<1 day" }
	if seconds < 86400*365 { return "<1 year" }
	return ">1 year"
}
