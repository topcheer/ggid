package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type PolicyInheritance struct {
	PolicyID        string         `json:"policy_id"`
	ParentPolicy    string         `json:"parent_policy,omitempty"`
	ChildPolicies   []string       `json:"child_policies"`
	OverriddenRules []OverrideRule `json:"overridden_rules"`
}

type OverrideRule struct {
	ID      string `json:"id"`
	RuleRef string `json:"rule_ref"`
	Effect  string `json:"effect"`
	Reason  string `json:"reason"`
}

func (s *HTTPServer) handlePolicyInheritance(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}
	policyID := parts[3]

	if strings.HasSuffix(r.URL.Path, "/inheritance") && r.Method == http.MethodGet {
		if s.policyMap != nil {
			row, _ := s.policyMap.Get(r.Context(), "policy_inheritance", policyID)
			if row != nil {
				writeJSON(w, http.StatusOK, row)
				return
			}
		}
		ih := &PolicyInheritance{PolicyID: policyID, ChildPolicies: []string{}, OverriddenRules: []OverrideRule{}}
		writeJSON(w, http.StatusOK, ih)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/override") && r.Method == http.MethodPost {
		var req struct {
			RuleRef string `json:"rule_ref"`
			Effect  string `json:"effect"`
			Reason  string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		override := OverrideRule{ID: "ov-" + uuid.New().String()[:8], RuleRef: req.RuleRef, Effect: req.Effect, Reason: req.Reason}
		if s.policyMap != nil {
			row, _ := s.policyMap.Get(r.Context(), "policy_inheritance", policyID)
			if row == nil {
				row = map[string]any{"policy_id": policyID, "child_policies": []string{}, "overridden_rules": []any{}}
			}
			rules, _ := row["overridden_rules"].([]any)
			rules = append(rules, map[string]any{"id": override.ID, "rule_ref": override.RuleRef, "effect": override.Effect, "reason": override.Reason})
			row["overridden_rules"] = rules
			s.policyMap.Store(r.Context(), "policy_inheritance", policyID, row)
		}
		writeJSON(w, http.StatusCreated, map[string]any{"status": "created", "override": override})
		return
	}

	writeJSONError(w, http.StatusNotFound, "not found")
}
