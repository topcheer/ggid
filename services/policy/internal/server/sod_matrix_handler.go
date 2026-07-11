package httpserver

import (
	"net/http"

	"github.com/ggid/ggid/services/policy/internal/service"
)

type SoDMatrixEntry struct {
	RuleID     string   `json:"rule_id"`
	RoleA      string   `json:"role_a"`
	RoleB      string   `json:"role_b"`
	IsExclusive bool    `json:"is_exclusive"`
	Roles      []string `json:"roles"`
	Description string  `json:"description"`
}

// GET /api/v1/policies/sod/matrix — return role-pair mutual exclusion matrix.
func (s *HTTPServer) handleSoDMatrix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	rules := service.GetSoDRules()
	matrix := make([]SoDMatrixEntry, 0, len(rules))
	for _, rule := range rules {
		// Expand role list into pairwise entries
		for i := 0; i < len(rule.Roles); i++ {
			for j := i + 1; j < len(rule.Roles); j++ {
				matrix = append(matrix, SoDMatrixEntry{
					RuleID:      rule.ID,
					RoleA:       rule.Roles[i],
					RoleB:       rule.Roles[j],
					IsExclusive: true,
					Roles:       rule.Roles,
					Description: rule.Reason,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"matrix":      matrix,
		"entry_count": len(matrix),
		"rule_count":  len(rules),
	})
}
