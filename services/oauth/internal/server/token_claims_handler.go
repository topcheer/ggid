package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// GET /api/v1/oauth/token/claims?token=X
func handleTokenClaims(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "token required"})
		return
	}
	// Decode JWT without signature verification (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JWT format"})
		return
	}
	// Header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JWT header"})
		return
	}
	var header map[string]any
	_ = json.Unmarshal(headerBytes, &header)
	// Payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JWT payload"})
		return
	}
	var claims map[string]any
	_ = json.Unmarshal(payloadBytes, &claims)
	writeJSON(w, http.StatusOK, map[string]any{
		"header":           header,
		"claims":           claims,
		"signature_verified": false,
		"note":             "decoded without signature verification — debugging only",
		"has_signature":     len(parts) == 3,
	})
}
