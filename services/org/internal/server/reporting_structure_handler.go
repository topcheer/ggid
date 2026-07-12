package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ReportingNode struct {
	UserID   string   `json:"user_id"`
	Name     string   `json:"name"`
	Title    string   `json:"title"`
	Manager  string   `json:"manager_id,omitempty"`
	Reports  []string `json:"direct_reports"`
}

type ReportingStructureResult struct {
	Tree              []ReportingNode `json:"tree"`
	SpanOfControl     float64         `json:"avg_span_of_control"`
	Layers            int             `json:"layers"`
	DotRepresentation string          `json:"dot_representation"`
	OrphanManagers    []string        `json:"orphan_managers"`
	CircularReporting []string        `json:"circular_reporting"`
}

func (s *HTTPServer) handleReportingStructure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result := ReportingStructureResult{
		Tree: []ReportingNode{
			{UserID: "u-001", Name: "Alice Chen", Title: "CEO", Reports: []string{"u-002", "u-005"}},
			{UserID: "u-002", Name: "Bob Lee", Title: "VP Engineering", Manager: "u-001", Reports: []string{"u-003", "u-004"}},
			{UserID: "u-003", Name: "Carol Wu", Title: "Eng Manager", Manager: "u-002", Reports: []string{"u-006", "u-007"}},
			{UserID: "u-004", Name: "Dan Kim", Title: "Sr Engineer", Manager: "u-002", Reports: []string{}},
			{UserID: "u-005", Name: "Eve Park", Title: "VP Sales", Manager: "u-001", Reports: []string{"u-008"}},
			{UserID: "u-006", Name: "Frank Liu", Title: "Engineer", Manager: "u-003", Reports: []string{}},
			{UserID: "u-007", Name: "Grace Zhao", Title: "Engineer", Manager: "u-003", Reports: []string{}},
			{UserID: "u-008", Name: "Henry Wang", Title: "Sales Rep", Manager: "u-005", Reports: []string{}},
		},
		SpanOfControl:  2.0,
		Layers:         4,
		OrphanManagers: []string{},
		CircularReporting: []string{},
	}

	// Build DOT representation
	var dotBuilder fmt.Stringer = dotBuilder(result.Tree)
	result.DotRepresentation = dotBuilder.String()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type dotStr string

func (d dotStr) String() string { return string(d) }

func dotBuilder(nodes []ReportingNode) fmt.Stringer {
	s := "digraph reporting {\n"
	for _, n := range nodes {
		if n.Manager != "" {
			s += fmt.Sprintf("  \"%s\" -> \"%s\";\n", n.Manager, n.UserID)
		}
	}
	s += "}"
	return dotStr(s)
}
