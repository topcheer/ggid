package httpserver

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// deptTreeNode represents a department in the tree with budget info.
type deptTreeNode struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	ParentID   *string       `json:"parent_id,omitempty"`
	ManagerID  *string       `json:"manager_id,omitempty"`
	Budget     *float64      `json:"budget,omitempty"`
	Headcount  int           `json:"headcount"`
	CostCenter string        `json:"cost_center,omitempty"`
	Active     bool          `json:"active"`
	Children   []*deptTreeNode `json:"children,omitempty"`
}

// GET /api/v1/organizations/{id}/departments/tree?include_inactive=true
// Returns department tree with budget, headcount, cost_center for each node.
func (s *HTTPServer) handleDeptTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract org ID from path: /api/v1/organizations/{id}/departments/tree
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// Expected: ["api", "v1", "organizations", "{id}", "departments", "tree"]
	if len(pathParts) < 4 {
		writeJSONError(w, http.StatusBadRequest, "organization ID is required")
		return
	}
	orgIDStr := pathParts[3]
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	departments, err := s.deptSvc.ListByOrg(r.Context(), orgID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load departments")
		return
	}

	// Build node map
	nodeMap := make(map[uuid.UUID]*deptTreeNode)
	var roots []*deptTreeNode

	for _, dept := range departments {
		active := true
		if dept.Metadata != nil {
			if v, ok := dept.Metadata["active"].(bool); ok {
				active = v
			}
		}
		if !active && !includeInactive {
			continue
		}

		node := &deptTreeNode{
			ID:        dept.ID.String(),
			Name:      dept.Name,
			Active:    active,
			Children:  []*deptTreeNode{},
		}

		if dept.ParentID != nil {
			pid := dept.ParentID.String()
			node.ParentID = &pid
		}
		if dept.ManagerID != nil {
			mid := dept.ManagerID.String()
			node.ManagerID = &mid
		}

		// Extract budget/headcount/cost_center from metadata
		if dept.Metadata != nil {
			if v, ok := dept.Metadata["budget"].(float64); ok {
				node.Budget = &v
			}
			if v, ok := dept.Metadata["headcount"].(float64); ok {
				node.Headcount = int(v)
			}
			if v, ok := dept.Metadata["cost_center"].(string); ok {
				node.CostCenter = v
			}
		}

		nodeMap[dept.ID] = node
	}

	// Link children to parents
	for _, dept := range departments {
		node, ok := nodeMap[dept.ID]
		if !ok {
			continue // filtered out
		}
		if dept.ParentID == nil {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[*dept.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Parent filtered out → treat as root
			roots = append(roots, node)
		}
	}

	// Compute aggregate headcount/budget up the tree
	for _, root := range roots {
		aggregateDeptNode(root)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"organization_id": orgIDStr,
		"total_departments": len(nodeMap),
		"tree":              roots,
	})
}

// aggregateDeptNode recursively computes total headcount and budget for each node
// by summing children values into the parent.
func aggregateDeptNode(node *deptTreeNode) (int, float64) {
	totalHeadcount := node.Headcount
	totalBudget := 0.0
	if node.Budget != nil {
		totalBudget = *node.Budget
	}

	for _, child := range node.Children {
		childHC, childBudget := aggregateDeptNode(child)
		totalHeadcount += childHC
		totalBudget += childBudget
	}

	node.Headcount = totalHeadcount
	if totalBudget > 0 {
		budgetCopy := totalBudget
		node.Budget = &budgetCopy
	}

	return totalHeadcount, totalBudget
}
