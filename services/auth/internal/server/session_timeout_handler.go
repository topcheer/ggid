package server

import (
	"net/http"
	"strconv"
	"time"
)

// GET /api/v1/auth/session-timeout?risk_score=X
// Returns dynamic session timeout based on risk score:
// 0-29: 8h (low), 30-59: 4h (medium), 60-79: 1h (high), 80+: 15min (critical)
func (h *Handler) handleSessionTimeout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	riskStr := r.URL.Query().Get("risk_score")
	riskScore := 0
	if riskStr != "" {
		riskScore, _ = strconv.Atoi(riskStr)
	}

	var timeoutSeconds int
	var riskLevel string

	switch {
	case riskScore < 30:
		timeoutSeconds = 8 * 3600
		riskLevel = "low"
	case riskScore < 60:
		timeoutSeconds = 4 * 3600
		riskLevel = "medium"
	case riskScore < 80:
		timeoutSeconds = 1 * 3600
		riskLevel = "high"
	default:
		timeoutSeconds = 15 * 60
		riskLevel = "critical"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"risk_score":       riskScore,
		"risk_level":       riskLevel,
		"timeout_seconds":  timeoutSeconds,
		"timeout_human":    (time.Duration(timeoutSeconds) * time.Second).String(),
		"policy":           "risk-based",
		"requires_step_up": riskScore >= 60,
	})
}
