package httpserver

import (
	"net/http"
	"strings"
	"sync"
)

// RoleNode represents a role in the hierarchy tree.
type RoleNode struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Permissions []string    `json:"permissions"`
	Parents     []*RoleNode `json:"parent_roles,omitempty"`
	Children    []*RoleNode `json:"child_roles,omitempty"`
}

// roleHierarchyStore holds role parent/child relationships and permissions.
type roleHierarchyStore struct {
	mu         sync.RWMutex
	parents    map[string][]string // role_id → parent_role_ids
	children   map[string][]string // role_id → child_role_ids
	perms      map[string][]string // role_id → permissions
	names      map[string]string   // role_id → name
}

var roleHierarchy = &roleHierarchyStore{
	parents:  make(map[string][]string),
	children: make(map[string][]string),
	perms:    make(map[string][]string),
	names:    make(map[string]string),
}

// RegisterRole adds a role to the hierarchy store.
func RegisterRole(id, name string, permissions, parentIDs []string) {
	roleHierarchy.mu.Lock()
	defer roleHierarchy.mu.Unlock()
	roleHierarchy.names[id] = name
	roleHierarchy.perms[id] = permissions
	roleHierarchy.parents[id] = parentIDs
	for _, pid := range parentIDs {
		roleHierarchy.children[pid] = append(roleHierarchy.children[pid], id)
	}
}

// GET /api/v1/policies/roles/{id}/hierarchy?recursive=true
func (s *HTTPServer) handleRoleHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// Expected: api/v1/policies/roles/{id}/hierarchy
	if len(parts) < 5 {
		writeJSONError(w, http.StatusBadRequest, "role ID is required")
		return
	}
	roleID := parts[4]
	recursive := r.URL.Query().Get("recursive") == "true"

	roleHierarchy.mu.RLock()
	defer roleHierarchy.mu.RUnlock()

	node := buildRoleNode(roleID, recursive, 0, 5)
	if node == nil {
		writeJSONError(w, http.StatusNotFound, "role not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"role":     node,
		"recursive": recursive,
	})
}

func buildRoleNode(roleID string, recursive bool, depth, maxDepth int) *RoleNode {
	name, ok := roleHierarchy.names[roleID]
	if !ok {
		return nil
	}
	node := &RoleNode{
		ID:          roleID,
		Name:        name,
		Permissions: roleHierarchy.perms[roleID],
	}
	if !recursive || depth >= maxDepth {
		return node
	}
	for _, pid := range roleHierarchy.parents[roleID] {
		if p := buildRoleNode(pid, recursive, depth+1, maxDepth); p != nil {
			node.Parents = append(node.Parents, p)
		}
	}
	for _, cid := range roleHierarchy.children[roleID] {
		if c := buildRoleNode(cid, recursive, depth+1, maxDepth); c != nil {
			node.Children = append(node.Children, c)
		}
	}
	return node
}
