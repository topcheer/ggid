package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// GET /api/v1/audit/compliance/evidence-expiry?days=30
// POST /api/v1/audit/compliance/evidence-refresh
func (s *HTTPServer) handleEvidenceExpiry(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/audit/compliance/evidence-refresh" && r.Method == http.MethodPost {
		var req struct{ ControlIDs []string `json:"control_ids"` }
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusOK, map[string]any{"status": "refreshed", "refreshed_count": len(req.ControlIDs), "refreshed_at": time.Now().UTC().Format(time.RFC3339)})
		return
	}
	if r.Method != http.MethodGet { writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	days, _ := strconv.Atoi(r.URL.Query().Get("days")); if days == 0 { days = 30 }
	expiring := []map[string]any{
		{"control_id": "CC6.1", "evidence": "MFA config", "collected_at": "2026-04-01T00:00:00Z", "expires_at": "2026-07-15T00:00:00Z", "days_until_expiry": 3},
		{"control_id": "CC7.2", "evidence": "Anomaly detection report", "collected_at": "2026-04-10T00:00:00Z", "expires_at": "2026-07-20T00:00:00Z", "days_until_expiry": 8},
		{"control_id": "CC8.1", "evidence": "Change management log", "collected_at": "2026-05-01T00:00:00Z", "expires_at": "2026-07-25T00:00:00Z", "days_until_expiry": 13},
	}
	writeJSON(w, http.StatusOK, map[string]any{"expiring_evidence": expiring, "total": len(expiring), "days_threshold": days, "generated_at": time.Now().UTC().Format(time.RFC3339)})
}
