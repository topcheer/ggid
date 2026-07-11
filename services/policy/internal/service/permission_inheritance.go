package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// OrgNode represents a node in the org hierarchy for permission inheritance.
type OrgNode struct {
	ID          uuid.UUID
	ParentID    uuid.UUID
	Permissions []string
}

// PermissionTree manages org tree permission inheritance.
type PermissionTree struct {
	mu    sync.RWMutex
	nodes map[uuid.UUID]*OrgNode
}

func NewPermissionTree() *PermissionTree {
	return &PermissionTree{nodes: make(map[uuid.UUID]*OrgNode)}
}

// AddNode adds an org node to the tree.
func (t *PermissionTree) AddNode(id, parentID uuid.UUID, perms []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes[id] = &OrgNode{ID: id, ParentID: parentID, Permissions: perms}
}

// GetEffectivePermissions returns all permissions for a node, including
// inherited permissions from all ancestors up the tree.
func (t *PermissionTree) GetEffectivePermissions(_ context.Context, nodeID uuid.UUID) ([]string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	seen := make(map[string]bool)
	var result []string
	visited := make(map[uuid.UUID]bool) // cycle detection

	current := nodeID
	for current != uuid.Nil {
		if visited[current] {
			return nil, fmt.Errorf("cycle detected in org tree at node %s", current)
		}
		visited[current] = true

		node, ok := t.nodes[current]
		if !ok {
			break
		}
		for _, p := range node.Permissions {
			if !seen[p] {
				seen[p] = true
				result = append(result, p)
			}
		}
		current = node.ParentID
	}
	return result, nil
}

// HasPermission checks if a node has a permission (direct or inherited).
func (t *PermissionTree) HasPermission(ctx context.Context, nodeID uuid.UUID, perm string) bool {
	perms, err := t.GetEffectivePermissions(ctx, nodeID)
	if err != nil {
		return false
	}
	for _, p := range perms {
		if p == perm || p == "*" {
			return true
		}
	}
	return false
}

// GetTreeDepth returns the depth of a node from root.
func (t *PermissionTree) GetTreeDepth(nodeID uuid.UUID) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	depth := 0
	visited := make(map[uuid.UUID]bool)
	current := nodeID
	for current != uuid.Nil {
		if visited[current] {
			return depth // cycle
		}
		visited[current] = true
		node, ok := t.nodes[current]
		if !ok {
			break
		}
		depth++
		current = node.ParentID
	}
	return depth
}

// Reset clears the tree (for testing).
func (t *PermissionTree) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes = make(map[uuid.UUID]*OrgNode)
}
