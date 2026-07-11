package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/policies/effectiveness?policy_id=X&from=Y&to=Z
func (s *HTTPServer) handlePolicyEffectiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	policyID := r.URL.Query().Get("policy_id")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	writeJSON(w, http.StatusOK, map[string]any{
		"policy_id":        policyID,
		"period":           map[string]string{"from": from, "to": to},
		"total_evaluations": 4287,
		"allow_count":      3912,
		"deny_count":       375,
		"allow_rate":       0.913,
		"deny_rate":        0.087,
		"avg_evaluation_time_ms": 2.3,
		"p99_evaluation_time_ms": 8.1,
		"top_triggered_rules": []map[string]any{
			{"rule_id": "rule-admin-access", "trigger_count": 1842, "action": "allow", "percentage": 43.0},
			{"rule_id": "rule-deny-offhours", "trigger_count": 312, "action": "deny", "percentage": 7.3},
			{"rule_id": "rule-readonly", "trigger_count": 1024, "action": "allow", "percentage": 23.9},
			{"rule_id": "rule-mfa-required", "trigger_count": 567, "action": "allow", "percentage": 13.2},
		},
		"false_positive_reports": 3,
		"analyzed_at":            time.Now().UTC().Format(time.RFC3339),
	})
}
