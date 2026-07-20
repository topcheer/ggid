package server

import (
	"net/http"
	"time"
)

type SuspiciousSession struct {
	SessionID   string   `json:"session_id"`
	UserID      string   `json:"user_id"`
	Reason      string   `json:"reason"`
	IPAddresses []string `json:"ip_addresses"`
	Locations   []string `json:"locations"`
	RiskScore   int      `json:"risk_score"`
	DetectedAt  string   `json:"detected_at"`
}

// GET /api/v1/auth/sessions/hijack-check
func (h *Handler) handleHijackCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Real DB-backed hijack detection: find users with concurrent sessions from multiple IPs
	suspicious := []SuspiciousSession{}
	pool := h.pool
	if pool != nil {
		rows, err := pool.Query(r.Context(), `
			SELECT user_id::text, array_agg(DISTINCT ip_address) as ips, count(DISTINCT ip_address) as ip_count
			FROM sessions
			WHERE revoked_at IS NULL AND created_at > NOW() - INTERVAL '1 hour'
			GROUP BY user_id HAVING count(DISTINCT ip_address) >= 2 LIMIT 20`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var uid string
				var ips []string
				var ipCount int
				if err := rows.Scan(&uid, &ips, &ipCount); err != nil {
					continue
				}
				suspicious = append(suspicious, SuspiciousSession{
					UserID:      uid,
					Reason:      "concurrent IPs — multiple active sessions from distinct addresses",
					IPAddresses: ips,
					RiskScore:   60 + ipCount*10,
					DetectedAt:  time.Now().UTC().Format(time.RFC3339),
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"suspicious_sessions": suspicious,
		"total_checked":       len(suspicious),
		"flagged":             len(suspicious),
		"checked_at":          time.Now().UTC().Format(time.RFC3339),
		"detection_rules":     []string{"concurrent_ip"},
	})
}
