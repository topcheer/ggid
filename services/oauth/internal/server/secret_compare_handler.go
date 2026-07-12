package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// POST /api/v1/oauth/clients/{id}/secret-compare
// Body: {"secret_hash_a": "...", "secret_hash_b": "..."}
// Uses timing-safe comparison. Returns is_match without leaking secret content.
func handleSecretCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract client ID from path
	clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	clientID = strings.TrimSuffix(clientID, "/secret-compare")
	clientID = strings.TrimSuffix(clientID, "/")
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
		return
	}

	var req struct {
		SecretHashA string `json:"secret_hash_a"`
		SecretHashB string `json:"secret_hash_b"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.SecretHashA == "" || req.SecretHashB == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "secret_hash_a and secret_hash_b are required"})
		return
	}

	// Timing-safe comparison
	a := []byte(req.SecretHashA)
	b := []byte(req.SecretHashB)

	isMatch := false
	if len(a) == len(b) {
		isMatch = subtle.ConstantTimeCompare(a, b) == 1
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":     clientID,
		"is_match":      isMatch,
		"hash_length_a": len(a),
		"hash_length_b": len(b),
		"compared_at":   "timing-safe",
	})
}
