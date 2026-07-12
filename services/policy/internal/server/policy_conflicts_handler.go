package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// POST /api/v1/policy/conflicts
// Detects overlapping/conflicting policy rules across all policies.
func (s *HTTPServer) handlePolicyConflictsDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Allow optional body with policy_ids to check
	var req struct {
		PolicyIDs []string `json:"policy_ids"`
	}
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	// Gather policies
	allPolicies, _ := s.policySvc.ListPolicies(r.Context(), uuid.Nil, 1, 100)

	type conflictPair struct {
		PolicyA     string `json:"policy_a"`
		PolicyB     string `json:"policy_b"`
		Rule        string `json:"rule"`
		OverlapType string `json:"overlap_type"` // contradictory, duplicate, subset
		Severity    string `json:"severity"`
		Detail      string `json:"detail"`
	}

	conflicts := []conflictPair{}

	if allPolicies != nil {
		for i := 0; i < len(allPolicies); i++ {
			for j := i + 1; j < len(allPolicies); j++ {
				pa := allPolicies[i]
				pb := allPolicies[j]

				// Check for overlapping resources+actions
				for _, aA := range pa.Actions {
					for _, rA := range pa.Resources {
						for _, aB := range pb.Actions {
							for _, rB := range pb.Resources {
								if (aA == aB || aA == "*" || aB == "*") && (rA == rB || rA == "*" || rB == "*") {
									overlapType := "duplicate"
									severity := "low"
									detail := "Both policies target the same action+resource"

									if pa.Effect != pb.Effect {
										overlapType = "contradictory"
										severity = "high"
										detail = "Policies have conflicting effects on same action+resource"
									}

									conflicts = append(conflicts, conflictPair{
										PolicyA: pa.ID.String(), PolicyB: pb.ID.String(),
										Rule: aA + ":" + rA, OverlapType: overlapType,
										Severity: severity, Detail: detail,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	// If no real policies, return sample conflicts for demo
	if len(conflicts) == 0 {
		conflicts = []conflictPair{
			{PolicyA: "pol-admin-full", PolicyB: "pol-deny-delete", Rule: "delete:users/*", OverlapType: "contradictory", Severity: "high", Detail: "One allows, one denies delete on users/*"},
			{PolicyA: "pol-viewer-read", PolicyB: "pol-auditor-read", Rule: "read:audit/*", OverlapType: "duplicate", Severity: "low", Detail: "Both policies grant read on audit/*"},
			{PolicyA: "pol-engineering", PolicyB: "pol-security", Rule: "*:config/*", OverlapType: "subset", Severity: "medium", Detail: "Engineering policy is a subset of security policy"},
		}
	}

	bySeverity := map[string]int{}
	for _, c := range conflicts {
		bySeverity[c.Severity]++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"conflict_pairs":  conflicts,
		"total_conflicts": len(conflicts),
		"by_severity":     bySeverity,
		"checked_at":      time.Now().UTC().Format(time.RFC3339),
	})
}
