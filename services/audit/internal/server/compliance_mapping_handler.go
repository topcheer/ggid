package httpserver

import (
	"net/http"
)

type ControlMapping struct {
	ControlID     string `json:"control_id"`
	Requirement   string `json:"requirement"`
	Status        string `json:"status"`
	EvidenceCount int    `json:"evidence_count"`
}

var frameworkMappings = map[string][]ControlMapping{
	"soc2": {
		{"CC6.1", "Logical and physical access controls", "covered", 12},
		{"CC6.2", "User authentication credentials", "covered", 8},
		{"CC6.3", "Authorization controls for access", "covered", 5},
		{"CC6.5", "Transmission and disposal of data", "partial", 3},
		{"CC7.1", "System performance monitoring", "covered", 7},
		{"CC7.2", "Anomaly detection", "covered", 4},
		{"CC8.1", "Change management controls", "partial", 2},
	},
	"iso27001": {
		{"A.9.2.1", "User registration and de-registration", "covered", 10},
		{"A.9.2.2", "User access provisioning", "covered", 6},
		{"A.9.2.3", "Management of privileged access rights", "covered", 4},
		{"A.9.2.4", "Management of secret authentication info", "partial", 3},
		{"A.9.2.5", "Review of user access rights", "partial", 2},
		{"A.9.4.1", "Information access restriction", "covered", 8},
		{"A.12.4.1", "Event logging", "covered", 15},
	},
	"gdpr": {
		{"Art.5", "Data minimization and retention", "partial", 3},
		{"Art.6", "Lawfulness of processing (consent)", "covered", 5},
		{"Art.15", "Right of access (data subject rights)", "covered", 2},
		{"Art.17", "Right to erasure", "partial", 1},
		{"Art.25", "Data protection by design", "covered", 7},
		{"Art.32", "Security of processing", "covered", 9},
		{"Art.33", "Breach notification", "partial", 2},
	},
	"hipaa": {
		{"164.312(a)(1)", "Access control", "covered", 6},
		{"164.312(b)", "Audit controls", "covered", 8},
		{"164.312(c)(1)", "Integrity controls", "covered", 3},
		{"164.312(d)", "Person or entity authentication", "covered", 5},
		{"164.312(e)(1)", "Transmission security", "partial", 2},
	},
}

// GET /api/v1/audit/compliance/mapping?framework=soc2
func (s *HTTPServer) handleComplianceMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	framework := r.URL.Query().Get("framework")
	if framework == "" {
		framework = "soc2"
	}
	mappings, ok := frameworkMappings[framework]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "unsupported framework: "+framework)
		return
	}
	covered, partial, gaps := 0, 0, 0
	for _, m := range mappings {
		switch m.Status {
		case "covered":
			covered++
		case "partial":
			partial++
		case "gap":
			gaps++
		}
	}
	totalEvidence := 0
	for _, m := range mappings {
		totalEvidence += m.EvidenceCount
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"framework": framework,
		"controls":  mappings,
		"summary": map[string]int{
			"total": len(mappings), "covered": covered, "partial": partial, "gap": gaps,
		},
		"total_evidence": totalEvidence,
	})
}
