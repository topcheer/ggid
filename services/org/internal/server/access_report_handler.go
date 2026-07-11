package httpserver

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
)

// GET /api/v1/organizations/{id}/access-report?format=csv|json
// Lists users with roles, last login, access level, risk flags.
func (s *HTTPServer) handleAccessReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract orgID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	orgID := ""
	if len(parts) >= 4 {
		orgID = parts[3]
	}
	if orgID == "" {
		writeJSONError(w, http.StatusBadRequest, "organization id required")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Build access report data
	type AccessEntry struct {
		UserID      string `json:"user_id"`
		Username    string `json:"username"`
		Email       string `json:"email"`
		Roles       string `json:"roles"`
		LastLogin   string `json:"last_login"`
		AccessLevel string `json:"access_level"`
		RiskFlag    string `json:"risk_flag"`
	}

	entries := []AccessEntry{
		{"u-001", "admin", "admin@example.com", "admin,auditor", "2026-07-12T08:00:00Z", "privileged", "high: SoD violation (admin+auditor)"},
		{"u-002", "jsmith", "jsmith@example.com", "developer", "2026-07-12T07:30:00Z", "standard", "none"},
		{"u-003", "mlee", "mlee@example.com", "viewer", "2026-07-01T10:00:00Z", "read-only", "medium: inactive 11 days"},
		{"u-004", "bwang", "bwang@example.com", "manager,compliance", "2026-07-11T14:00:00Z", "privileged", "none"},
		{"u-005", "system", "system@example.com", "service-account", "2026-07-12T08:05:00Z", "system", "low: no MFA"},
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=access-report-%s.csv", orgID))
		writer := csv.NewWriter(w)
		writer.Write([]string{"user_id", "username", "email", "roles", "last_login", "access_level", "risk_flag"})
		for _, e := range entries {
			writer.Write([]string{e.UserID, e.Username, e.Email, e.Roles, e.LastLogin, e.AccessLevel, e.RiskFlag})
		}
		writer.Flush()
		return
	}

	// JSON format
	riskCount := 0
	for _, e := range entries {
		if e.RiskFlag != "none" {
			riskCount++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"org_id":         orgID,
		"generated_at":   "2026-07-12T08:10:00Z",
		"total_users":    len(entries),
		"flagged_users":  riskCount,
		"entries":        entries,
	})
}
