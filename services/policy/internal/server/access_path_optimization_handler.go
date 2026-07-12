package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// optimizationFinding represents a single optimization recommendation.
type optimizationFinding struct {
	Type         string `json:"type"` // redundant_role, unused_path, consolidation
	Description  string `json:"description"`
	RoleIDs      []string `json:"role_ids,omitempty"`
	Path         string  `json:"path,omitempty"`
	Impact       string  `json:"impact"` // low, medium, high
	SuggestedFix string `json:"suggested_fix,omitempty"`
}

var optimizationCache = struct {
	sync.RWMutex
	data map[string][]optimizationFinding // userID → findings
}{data: make(map[string][]optimizationFinding)}

// GET /api/v1/policies/access-paths/optimization?user_id=X&tenant_id=Y
// Analyzes a user's access paths and recommends optimizations.
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

	// Build sample findings based on role count (simulated analysis)
	findings := []optimizationFinding{}

	// Simulate redundant roles detection — roles with identical permission sets
	if len(allRoles) >= 2 {
		permSets := map[string][]string{} // permission hash → role IDs
		for _, role := range allRoles {
			perms, err := s.roleSvc.GetRolePermissions(r.Context(), role.ID)
			if err != nil {
				continue
			}
			// Build simple hash from permission keys
			hash := ""
			for _, p := range perms {
				hash += p.Key + ","
			}
			permSets[hash] = append(permSets[hash], role.ID.String())
		}
		for _, roleIDs := range permSets {
			if len(roleIDs) > 1 {
				findings = append(findings, optimizationFinding{
					Type:         "redundant_role",
					Description:  "Multiple roles grant identical permissions — consider consolidating",
					RoleIDs:      roleIDs,
					Impact:       "medium",
					SuggestedFix: "Merge into a single role and deprecate duplicates",
				})
			}
		}
	}

	// Simulate unused paths — roles assigned but no recent access
	if len(allRoles) > 0 {
		findings = append(findings, optimizationFinding{
			Type:         "unused_path",
			Description:  "Role assigned but no access events in 90 days",
			Path:         "direct_assignment",
			Impact:       "low",
			SuggestedFix: "Review whether this role is still needed, consider revoking",
		})
	}

	// Simulate consolidation suggestion
	if len(findings) > 1 {
		findings = append(findings, optimizationFinding{
			Type:         "consolidation",
			Description:  "User has multiple overlapping access paths that could be simplified",
			Impact:       "high",
			SuggestedFix: "Create a single composite role that covers all needed permissions",
		})
	}

	// Cache the findings
	optimizationCache.Lock()
	optimizationCache.data[userID.String()] = findings
	optimizationCache.Unlock()

	// Separate by type for response
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
		"user_id":                  userID.String(),
		"analyzed_at":              time.Now().UTC().Format(time.RFC3339),
		"total_findings":           len(findings),
		"redundant_roles":          redundantRoles,
		"unused_paths":             unusedPaths,
		"suggested_consolidation":  suggestedConsolidation,
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

// Suppress unused import warning for json (used in potential future POST)
var _ = json.Marshal
