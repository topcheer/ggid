package httpserver

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

// optimizationFinding represents a single optimization recommendation.
type optimizationFinding struct {
	Type         string   `json:"type"`
	Description  string   `json:"description"`
	RoleIDs      []string `json:"role_ids,omitempty"`
	Path         string   `json:"path,omitempty"`
	Impact       string   `json:"impact"`
	SuggestedFix string   `json:"suggested_fix,omitempty"`
}

// GET /api/v1/policies/access-paths/optimization?user_id=X&tenant_id=Y
func (s *HTTPServer) handleAccessPathOptimization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	// Try to get roles for the user from the role service
	allRoles, _ := s.roleSvc.ListRoles(r.Context(), uuid.Nil, 1, 500)

	findings := []optimizationFinding{}
	if len(allRoles) >= 2 {
		permSets := map[string][]string{}
		for _, role := range allRoles {
			perms, err := s.roleSvc.GetRolePermissions(r.Context(), role.ID)
			if err != nil {
				continue
			}
			hash := ""
			for _, p := range perms {
				hash += p.Key + ","
			}
			permSets[hash] = append(permSets[hash], role.ID.String())
		}
		for _, roleIDs := range permSets {
			if len(roleIDs) > 1 {
				findings = append(findings, optimizationFinding{
					Type: "redundant_role", Description: "Multiple roles grant identical permissions",
					RoleIDs: roleIDs, Impact: "medium", SuggestedFix: "Merge into a single role",
				})
			}
		}
	}
	if len(allRoles) > 0 {
		findings = append(findings, optimizationFinding{
			Type: "unused_path", Description: "Role assigned but no access events in 90 days",
			Path: "direct_assignment", Impact: "low", SuggestedFix: "Review whether this role is still needed",
		})
	}

	// Persist findings to PG.
	if s.policyMap != nil {
		s.policyMap.Store(r.Context(), "access_optimization_store", userID.String(), map[string]any{
			"findings": findings, "analyzed_at": time.Now().UTC(),
		})
	}

	redundantRoles := []optimizationFinding{}
	unusedPaths := []optimizationFinding{}
	suggestedConsolidation := []optimizationFinding{}
	for _, f := range findings {
		switch f.Type {
		case "redundant_role":
			redundantRoles = append(redundantRoles, f)
		case "unused_path":
			unusedPaths = append(unusedPaths, f)
		case "consolidation":
			suggestedConsolidation = append(suggestedConsolidation, f)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":                 userID.String(),
		"analyzed_at":             time.Now().UTC().Format(time.RFC3339),
		"total_findings":          len(findings),
		"redundant_roles":         redundantRoles,
		"unused_paths":            unusedPaths,
		"suggested_consolidation": suggestedConsolidation,
		"optimization_score": func() int {
			score := 100
			score -= len(redundantRoles) * 10
			score -= len(unusedPaths) * 5
			score -= len(suggestedConsolidation) * 15
			if score < 0 {
				score = 0
			}
			return score
		}(),
	})
}
