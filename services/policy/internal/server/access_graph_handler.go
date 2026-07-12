package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

type AccessGraphNode struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type AccessGraphEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"`
	Label  string `json:"label,omitempty"`
}

type AccessGraphResult struct {
	SubjectID            string             `json:"subject_id"`
	DirectPermissions    []string           `json:"direct_permissions"`
	InheritedPermissions []string           `json:"inherited_permissions"`
	ViaGroups            []string           `json:"via_groups"`
	ViaRoles             []string           `json:"via_roles"`
	EffectivePermissions []string           `json:"effective_permissions"`
	GraphDepth           int                `json:"graph_depth"`
	VisualData           AccessGraphVisual  `json:"visual_data"`
}

type AccessGraphVisual struct {
	Nodes []AccessGraphNode `json:"nodes"`
	Edges []AccessGraphEdge `json:"edges"`
}

func (s *HTTPServer) handleAccessGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/policy/access-graph/"), "/")
	subjectID := parts[0]
	if subjectID == "" {
		subjectID = "unknown"
	}

	result := AccessGraphResult{
		SubjectID:            subjectID,
		DirectPermissions:    []string{"doc:read", "doc:write"},
		InheritedPermissions: []string{"folder:read", "system:settings:view"},
		ViaGroups:            []string{"group:eng-team", "group:oncall"},
		ViaRoles:             []string{"role:engineer", "role:viewer"},
		EffectivePermissions: []string{"doc:read", "doc:write", "folder:read", "system:settings:view"},
		GraphDepth:           3,
		VisualData: AccessGraphVisual{
			Nodes: []AccessGraphNode{
				{ID: subjectID, Type: "subject", Label: subjectID},
				{ID: "role:engineer", Type: "role", Label: "Engineer"},
				{ID: "role:viewer", Type: "role", Label: "Viewer"},
				{ID: "group:eng-team", Type: "group", Label: "Eng Team"},
				{ID: "doc:read", Type: "permission", Label: "doc:read"},
				{ID: "doc:write", Type: "permission", Label: "doc:write"},
				{ID: "folder:read", Type: "permission", Label: "folder:read"},
				{ID: "system:settings:view", Type: "permission", Label: "system:settings:view"},
			},
			Edges: []AccessGraphEdge{
				{From: subjectID, To: "role:engineer", Type: "has_role"},
				{From: subjectID, To: "role:viewer", Type: "has_role"},
				{From: subjectID, To: "group:eng-team", Type: "member_of"},
				{From: "role:engineer", To: "doc:write", Type: "grants"},
				{From: "role:viewer", To: "doc:read", Type: "grants"},
				{From: "group:eng-team", To: "folder:read", Type: "grants"},
				{From: "role:engineer", To: "system:settings:view", Type: "grants"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
