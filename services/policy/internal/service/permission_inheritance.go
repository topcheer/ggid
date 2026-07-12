package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type PermissionNode struct {
	RoleID         string   `json:"role_id"`
	ParentID       string   `json:"parent_id"`
	Permissions    []string `json:"permissions"`
	InheritedFrom  []string `json:"inherited_from"`
}

type PermissionInheritanceService struct {
	mu    sync.RWMutex
	nodes map[string]*PermissionNode
}

func NewPermissionInheritanceService() *PermissionInheritanceService {
	return &PermissionInheritanceService{nodes: make(map[string]*PermissionNode)}
}

func (s *PermissionInheritanceService) SetPermissionInheritance(roleID, parentRoleID string) error {
	if s.DetectInheritanceCycle(roleID, parentRoleID) {
		return fmt.Errorf("inheritance cycle detected: %s -> %s", roleID, parentRoleID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.nodes[roleID]
	if !ok {
		node = &PermissionNode{RoleID: roleID}
		s.nodes[roleID] = node
	}
	node.ParentID = parentRoleID
	return nil
}

func (s *PermissionInheritanceService) GetEffectivePermissions(roleID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := make(map[string]bool)
	var result []string
	s.collectPermissions(roleID, seen, &result, []string{})
	return result
}

func (s *PermissionInheritanceService) collectPermissions(roleID string, seen map[string]bool, result *[]string, chain []string) {
	node, ok := s.nodes[roleID]
	if !ok {
		return
	}
	for _, p := range node.Permissions {
		if !seen[p] {
			seen[p] = true
			*result = append(*result, p)
		}
	}
	if node.ParentID != "" {
		s.collectPermissions(node.ParentID, seen, result, append(chain, roleID))
	}
}

func (s *PermissionInheritanceService) CheckPermissionInheritance(roleID, permission string) bool {
	effective := s.GetEffectivePermissions(roleID)
	for _, p := range effective {
		if p == permission {
			return true
		}
	}
	return false
}

func (s *PermissionInheritanceService) DetectInheritanceCycle(roleID, parentRoleID string) bool {
	if roleID == parentRoleID {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	visited := make(map[string]bool)
	current := parentRoleID
	for current != "" {
		if current == roleID {
			return true
		}
		if visited[current] {
			return true
		}
		visited[current] = true
		node, ok := s.nodes[current]
		if !ok {
			return false
		}
		current = node.ParentID
	}
	return false
}

func (s *PermissionInheritanceService) SetPermissions(roleID string, permissions []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.nodes[roleID]
	if !ok {
		node = &PermissionNode{RoleID: roleID}
		s.nodes[roleID] = node
	}
	node.Permissions = permissions
}

// --- PermissionTree type for existing test compatibility ---

type treeNode struct {
	id       uuid.UUID
	parent   uuid.UUID
	perms    []string
}

type PermissionTree struct {
	mu    sync.RWMutex
	nodes map[uuid.UUID]*treeNode
}

func NewPermissionTree() *PermissionTree {
	return &PermissionTree{nodes: make(map[uuid.UUID]*treeNode)}
}

func (t *PermissionTree) AddNode(id, parent uuid.UUID, perms []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes[id] = &treeNode{id: id, parent: parent, perms: perms}
}

func (t *PermissionTree) GetEffectivePermissions(ctx context.Context, id uuid.UUID) ([]string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	seen := make(map[string]bool)
	visited := make(map[uuid.UUID]bool)
	var result []string
	current := id
	for current != uuid.Nil {
		if visited[current] {
			return nil, fmt.Errorf("inheritance cycle detected at node %s", current)
		}
		visited[current] = true
		node, ok := t.nodes[current]
		if !ok {
			break
		}
		for _, p := range node.perms {
			if !seen[p] {
				seen[p] = true
				result = append(result, p)
			}
		}
		current = node.parent
	}
	return result, nil
}

func (t *PermissionTree) HasPermission(ctx context.Context, id uuid.UUID, perm string) bool {
	perms, _ := t.GetEffectivePermissions(ctx, id)
	for _, p := range perms {
		if p == perm || p == "*" {
			return true
		}
	}
	return false
}

func (t *PermissionTree) DetectCycle(id, parentID uuid.UUID) bool {
	if id == parentID {
		return true
	}
	visited := make(map[uuid.UUID]bool)
	current := parentID
	for current != uuid.Nil {
		if current == id {
			return true
		}
		if visited[current] {
			return true
		}
		visited[current] = true
		node, ok := t.nodes[current]
		if !ok {
			return false
		}
		current = node.parent
	}
	return false
}

// GetTreeDepth returns the depth of a node from the root (root = 1).
func (t *PermissionTree) GetTreeDepth(id uuid.UUID) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	depth := 0
	current := id
	for current != uuid.Nil {
		node, ok := t.nodes[current]
		if !ok {
			return depth
		}
		depth++
		current = node.parent
	}
	return depth
}