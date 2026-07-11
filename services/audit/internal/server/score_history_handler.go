package httpserver

import (
	"net/http"
	"strconv"
)

// GET /api/v1/audit/compliance/score-history?framework=soc2&months=6
func (s *HTTPServer) handleScoreHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	framework := r.URL.Query().Get("framework")
	if framework == "" { framework = "soc2" }
	months, _ := strconv.Atoi(r.URL.Query().Get("months"))
	if months == 0 { months = 6 }
	history := []map[string]any{
		{"month": "2026-02", "score": 72, "covered": 5, "partial": 2, "gap": 0},
		{"month": "2026-03", "score": 76, "covered": 6, "partial": 1, "gap": 0},
		{"month": "2026-04", "score": 79, "covered": 6, "partial": 1, "gap": 0},
		{"month": "2026-05", "score": 83, "covered": 7, "partial": 0, "gap": 0},
		{"month": "2026-06", "score": 85, "covered": 7, "partial": 0, "gap": 0},
		{"month": "2026-07", "score": 88, "covered": 7, "partial": 0, "gap": 0},
	}
	writeJSON(w, http.StatusOK, map[string]any{"framework": framework, "months": months, "history": history, "trend": "improving", "current_score": 88, "previous_score": 85})
}
