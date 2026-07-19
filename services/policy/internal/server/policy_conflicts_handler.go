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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid request body"); return }
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

	// Return empty list if no real conflicts found (no mock data)

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
