package server

import (
	"net/http"
)

// GET /api/v1/oauth/token-entropy
func handleTokenEntropy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"}); return }
	writeJSON(w, http.StatusOK, map[string]any{
		"avg_entropy_bits": 187.4,
		"min_entropy_bits": 64,
		"max_entropy_bits": 256,
		"weak_tokens": []map[string]any{
			{"token_id": "tok-w1", "entropy_bits": 52, "reason": "short JWT secret"},
			{"token_id": "tok-w2", "entropy_bits": 48, "reason": "predictable jti"},
		},
		"distribution": []map[string]any{
			{"range": "0-64", "count": 2}, {"range": "64-128", "count": 47}, {"range": "128-192", "count": 312}, {"range": "192-256", "count": 184},
		},
		"total_tokens_analyzed": 545,
		"weak_count": 2,
	})
}
