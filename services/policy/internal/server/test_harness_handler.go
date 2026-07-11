package httpserver

import (
	"encoding/json"
	"net/http"
)

// POST /api/v1/policies/test-harness
func (s *HTTPServer) handleTestHarness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		PolicyID      string `json:"policy_id"`
		TestScenarios []struct {
			Subject  string         `json:"subject"`
			Resource string         `json:"resource"`
			Action   string         `json:"action"`
			Context  map[string]any `json:"context"`
			Expected string         `json:"expected"` // allow/deny
		} `json:"test_scenarios"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.PolicyID == "" || len(req.TestScenarios) == 0 {
		writeJSONError(w, http.StatusBadRequest, "policy_id and test_scenarios required")
		return
	}
	type Result struct {
		Scenario        string `json:"scenario"`
		Expected        string `json:"expected"`
		ActualDecision  string `json:"actual_decision"`
		IsPass          bool   `json:"is_pass"`
	}
	results := make([]Result, len(req.TestScenarios))
	passCount := 0
	for i, sc := range req.TestScenarios {
		// Simulate evaluation — default allow unless resource contains "admin"
		actual := "allow"
		if contains(sc.Resource, "admin") || contains(sc.Resource, "secret") {
			actual = "deny"
		}
		passed := actual == sc.Expected
		if passed { passCount++ }
		results[i] = Result{Scenario: sc.Subject + ":" + sc.Resource, Expected: sc.Expected, ActualDecision: actual, IsPass: passed}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"policy_id": req.PolicyID, "results": results,
		"total": len(results), "passed": passCount, "failed": len(results) - passCount,
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr { return true }
	}
	return false
}
