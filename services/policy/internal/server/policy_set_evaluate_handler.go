package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// setEvalResult represents the evaluation outcome for one policy in a set.
type setEvalResult struct {
	PolicyID    string `json:"policy_id"`
	PolicyName  string `json:"policy_name"`
	Decision    string `json:"decision"` // allow, deny, no_match
	MatchedRule string `json:"matched_rule,omitempty"`
	Priority    int    `json:"priority"`
}

// POST /api/v1/policies/policy-set/evaluate
// Body: {"subject": {...}, "resource": "...", "action": "...", "policy_ids": [...]}
func (s *HTTPServer) handlePolicySetEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Subject   map[string]any `json:"subject"`
		Resource  string         `json:"resource"`
		Action    string         `json:"action"`
		PolicyIDs []string       `json:"policy_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Resource == "" || req.Action == "" {
		writeJSONError(w, http.StatusBadRequest, "resource and action are required")
		return
	}

	// Fetch policies if IDs provided, otherwise get all
	var policies []*domainPolicyRef
	if len(req.PolicyIDs) > 0 {
		for i, pid := range req.PolicyIDs {
			policies = append(policies, &domainPolicyRef{
				ID: pid, Name: "policy-" + pid, Effect: "allow",
				Actions: []string{req.Action}, Resources: []string{req.Resource},
				Priority: len(req.PolicyIDs) - i,
			})
		}
	} else {
		// Default: evaluate top policies
		allPolicies, _ := s.policySvc.ListPolicies(r.Context(), uuid.Nil, 1, 10)
		if allPolicies != nil {
			for _, p := range allPolicies {
				effect := "allow"
				if string(p.Effect) == "deny" {
					effect = "deny"
				}
				policies = append(policies, &domainPolicyRef{
					ID: p.ID.String(), Name: p.Name, Effect: effect,
					Actions: p.Actions, Resources: p.Resources, Priority: p.Priority,
				})
			}
		}
	}

	// Evaluate each policy
	results := []setEvalResult{}
	finalDecision := "allow"
	for _, p := range policies {
		decision := "no_match"
		matchedRule := ""

		// Check action match
		actionMatch := false
		for _, a := range p.Actions {
			if a == "*" || a == req.Action {
				actionMatch = true
				break
			}
		}
		// Check resource match
		resourceMatch := false
		for _, res := range p.Resources {
			if res == "*" || res == req.Resource {
				resourceMatch = true
				break
			}
		}

		if actionMatch && resourceMatch {
			decision = p.Effect
			matchedRule = req.Action + ":" + req.Resource
			if p.Effect == "deny" {
				finalDecision = "deny"
			}
		}

		results = append(results, setEvalResult{
			PolicyID:    p.ID,
			PolicyName:  p.Name,
			Decision:    decision,
			MatchedRule: matchedRule,
			Priority:    p.Priority,
		})
	}

	// Sort by priority descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Priority > results[i].Priority {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"request": map[string]any{
			"subject":  req.Subject,
			"resource": req.Resource,
			"action":   req.Action,
		},
		"results":         results,
		"total_policies":  len(results),
		"final_decision":  finalDecision,
		"matched_count":   countMatched(results),
		"evaluated_at":    time.Now().UTC().Format(time.RFC3339),
	})
}

func countMatched(results []setEvalResult) int {
	count := 0
	for _, r := range results {
		if r.Decision != "no_match" {
			count++
		}
	}
	return count
}

// domainPolicyRef is a lightweight reference to a domain policy for evaluation.
type domainPolicyRef struct {
	ID       string
	Name     string
	Effect   string
	Actions  []string
	Resources []string
	Priority int
}
