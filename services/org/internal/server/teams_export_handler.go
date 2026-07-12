package httpserver

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
)

// GET /api/v1/organizations/{id}/teams/export?format=csv
func (s *HTTPServer) handleTeamsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	orgID := ""; if len(parts) >= 4 { orgID = parts[3] }
	if orgID == "" { writeJSONError(w, http.StatusBadRequest, "org_id required"); return }
	teams := [][]string{{"team_name", "lead", "member_count", "budget", "cost_center"},
		{"Platform Team", "jane@example.com", "12", "500000", "ENG-001"},
		{"Security Team", "bob@example.com", "8", "350000", "SEC-001"},
		{"Data Team", "sarah@example.com", "15", "750000", "DATA-001"},
		{"Mobile Team", "mike@example.com", "6", "280000", "MOB-001"},
	}
	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=teams-%s.csv", orgID))
		wr := csv.NewWriter(w)
		for _, row := range teams { wr.Write(row) }
		wr.Flush()
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"org_id": orgID, "teams": teams[1:], "total": len(teams) - 1})
}
