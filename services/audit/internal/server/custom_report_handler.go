package httpserver

import (
	"encoding/json"
	"net/http"
)

// POST /api/v1/audit/reports/custom
func (s *HTTPServer) handleCustomReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	var spec struct {
		Fields  []string         `json:"fields"`
		Filters map[string]any   `json:"filters"`
		GroupBy string           `json:"group_by"`
		Sort    string           `json:"sort"`
		Format  string           `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid JSON"); return }
	if spec.Format == "" { spec.Format = "json" }
	rows := []map[string]any{
		{"timestamp": "2026-07-12T08:00:00Z", "action": "login", "user": "admin", "result": "success"},
		{"timestamp": "2026-07-12T07:30:00Z", "action": "api_call", "user": "jsmith", "result": "success"},
		{"timestamp": "2026-07-12T07:15:00Z", "action": "login", "user": "mlee", "result": "failed"},
	}
	if spec.Format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte("timestamp,action,user,result\n"))
		for _, row := range rows {
			w.Write([]byte(row["timestamp"].(string) + "," + row["action"].(string) + "," + row["user"].(string) + "," + row["result"].(string) + "\n"))
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"query_spec": spec, "rows": rows, "total": len(rows)})
}
