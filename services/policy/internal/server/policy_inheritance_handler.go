package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PolicyInheritance struct {
	PolicyID        string         `json:"policy_id"`
	ParentPolicy    string         `json:"parent_policy,omitempty"`
	ChildPolicies   []string       `json:"child_policies"`
	OverriddenRules []OverrideRule `json:"overridden_rules"`
}

type OverrideRule struct {
	ID        string `json:"id"`
	RuleRef   string `json:"rule_ref"`
	Effect    string `json:"effect"` // allow, deny
	Reason    string `json:"reason"`
}

var (
	inheritanceMu sync.RWMutex
	inheritance   = make(map[string]*PolicyInheritance)
)

// GET /api/v1/policies/{id}/inheritance
// POST /api/v1/policies/{id}/override
func (s *HTTPServer) handlePolicyInheritance(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// /api/v1/policies/{id}/inheritance or /override
	if len(parts) < 4 {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}
	policyID := parts[3]

	if strings.HasSuffix(r.URL.Path, "/inheritance") && r.Method == http.MethodGet {
		inheritanceMu.RLock()
		ih, ok := inheritance[policyID]
		inheritanceMu.RUnlock()
		if !ok {
			ih = &PolicyInheritance{PolicyID: policyID, ChildPolicies: []string{}, OverriddenRules: []OverrideRule{}}
		}
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
		inheritanceMu.Lock()
		if inheritance[policyID] == nil {
			inheritance[policyID] = &PolicyInheritance{PolicyID: policyID, ChildPolicies: []string{}, OverriddenRules: []OverrideRule{}}
		}
		inheritance[policyID].OverriddenRules = append(inheritance[policyID].OverriddenRules, override)
		inheritanceMu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{"status": "created", "override": override})
		return
	}

	_ = time.Now
	writeJSONError(w, http.StatusNotFound, "not found")
}
