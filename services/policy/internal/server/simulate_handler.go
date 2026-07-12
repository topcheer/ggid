package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// simulationResult captures the impact of proposed policy rules.
type simulationResult struct {
	ID           string         `json:"id"`
	ProposedRules []map[string]any `json:"proposed_rules"`
	WouldAllow   []map[string]any `json:"would_allow"`
	WouldDeny    []map[string]any `json:"would_deny"`
	Unchanged    []map[string]any `json:"unchanged"`
	Summary      map[string]int `json:"summary"`
	SimulatedAt  string         `json:"simulated_at"`
}

var simulationStore = struct {
	sync.RWMutex
	results map[string]*simulationResult
}{results: make(map[string]*simulationResult)}

// POST /api/v1/policies/simulate
// Body: {"proposed_rules": [{"effect": "allow", "action": "read", "resource": "users/*", "condition": "..."}], "test_cases": [...]}
// Read-only simulation — evaluates proposed rules against test cases without applying them.
func (s *HTTPServer) handlePolicySimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ProposedRules []map[string]any `json:"proposed_rules"`
		TestCases     []map[string]any `json:"test_cases"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.ProposedRules) == 0 {
		writeJSONError(w, http.StatusBadRequest, "proposed_rules must not be empty")
		return
	}

	// Generate default test cases if none provided
	if len(req.TestCases) == 0 {
		req.TestCases = []map[string]any{
			{"user_id": "user-001", "action": "read", "resource": "users/*", "role": "viewer"},
			{"user_id": "user-002", "action": "write", "resource": "users/*", "role": "editor"},
			{"user_id": "user-003", "action": "delete", "resource": "users/*", "role": "admin"},
			{"user_id": "user-004", "action": "read", "resource": "audit/*", "role": "viewer"},
			{"user_id": "user-005", "action": "admin", "resource": "policies/*", "role": "admin"},
		}
	}

	// Evaluate each test case against proposed rules
	wouldAllow := []map[string]any{}
	wouldDeny := []map[string]any{}
	unchanged := []map[string]any{}

	for _, tc := range req.TestCases {
		matched := false
		for _, rule := range req.ProposedRules {
			effect, _ := rule["effect"].(string)
			ruleAction, _ := rule["action"].(string)
			ruleResource, _ := rule["resource"].(string)
			tcAction, _ := tc["action"].(string)
			tcResource, _ := tc["resource"].(string)

			// Simple matching: "*" matches all, prefix match on resource
			actionMatch := ruleAction == "*" || ruleAction == tcAction
			resourceMatch := ruleResource == "*" || ruleResource == tcResource ||
				(len(ruleResource) > 1 && ruleResource[len(ruleResource)-2:] == "/*" &&
					tcResource[:len(tcResource)/2] == ruleResource[:len(ruleResource)/2])

			if actionMatch && resourceMatch {
				matched = true
				entry := map[string]any{
					"test_case": tc,
					"matched_rule": rule,
					"effect": effect,
				}
				if effect == "allow" {
					wouldAllow = append(wouldAllow, entry)
				} else {
					wouldDeny = append(wouldDeny, entry)
				}
				break
			}
		}
		if !matched {
			unchanged = append(unchanged, map[string]any{
				"test_case":  tc,
				"effect":     "no_match",
				"note":       "no proposed rule matched this test case",
			})
		}
	}

	simID := uuid.New().String()
	result := &simulationResult{
		ID:            simID,
		ProposedRules: req.ProposedRules,
		WouldAllow:    wouldAllow,
		WouldDeny:     wouldDeny,
		Unchanged:     unchanged,
		Summary: map[string]int{
			"total_test_cases": len(req.TestCases),
			"would_allow":      len(wouldAllow),
			"would_deny":       len(wouldDeny),
			"unchanged":        len(unchanged),
		},
		SimulatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	simulationStore.Lock()
	simulationStore.results[simID] = result
	simulationStore.Unlock()

	writeJSON(w, http.StatusOK, result)
}
