package httpserver

import (
	"encoding/json"
	"net/http"
)

type CoverageCell struct {
	SubjectID    string  `json:"subject_id"`
	ResourceID   string  `json:"resource_id"`
	CoveragePct  float64 `json:"coverage_pct"`
	PolicyCount  int     `json:"policy_count"`
	Decision     string  `json:"decision"`
}

type CoverageMatrixResult struct {
	Subjects           []string       `json:"subjects"`
	Resources          []string       `json:"resources"`
	Grid               []CoverageCell `json:"grid"`
	UncoveredCombos    []string       `json:"uncovered_combinations"`
	RedundantPolicies  []string       `json:"redundant_policies"`
	GapsCount          int            `json:"gaps_count"`
	OverallCoveragePct float64        `json:"overall_coverage_pct"`
}

func (s *HTTPServer) handleCoverageMatrix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	subjects := []string{"user:alice", "user:bob", "role:admin", "role:viewer"}
	resources := []string{"doc:1", "folder:2", "system:settings", "billing:invoices"}

	var grid []CoverageCell
	var uncovered []string
	for _, subj := range subjects {
		for _, res := range resources {
			cell := CoverageCell{SubjectID: subj, ResourceID: res, CoveragePct: 100, PolicyCount: 1, Decision: "allow"}
			if subj == "role:viewer" && res == "system:settings" {
				cell.CoveragePct = 0
				cell.PolicyCount = 0
				cell.Decision = "no_policy"
				uncovered = append(uncovered, subj+":"+res)
			}
			if subj == "role:viewer" && res == "billing:invoices" {
				cell.CoveragePct = 0
				cell.PolicyCount = 0
				cell.Decision = "deny"
			}
			grid = append(grid, cell)
		}
	}

	result := CoverageMatrixResult{
		Subjects:          subjects,
		Resources:         resources,
		Grid:              grid,
		UncoveredCombos:   uncovered,
		RedundantPolicies: []string{"policy:legacy-doc-access overlaps policy:doc-rbac for doc:1"},
		GapsCount:         len(uncovered),
		OverallCoveragePct: 93.75,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
