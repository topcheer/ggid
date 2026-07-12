package service

import (
	"fmt"
	"strings"
	"sync"
)

type OrgNode struct {
	OrgID    string `json:"org_id"`
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Path     string `json:"path"`
}

type OrgHierarchyService struct {
	mu   sync.RWMutex
	orgs map[string]*OrgNode
	seq  int
}

func NewOrgHierarchyService() *OrgHierarchyService {
	return &OrgHierarchyService{orgs: make(map[string]*OrgNode)}
}

func (s *OrgHierarchyService) CreateOrg(name, parentID string) (*OrgNode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := fmt.Sprintf("org_%d", s.seq)
	level := 0
	path := "/" + id
	if parentID != "" {
		parent, ok := s.orgs[parentID]
		if !ok {
			return nil, fmt.Errorf("parent org not found")
		}
		level = parent.Level + 1
		path = parent.Path + "/" + id
	}
	node := &OrgNode{
		OrgID:    id,
		ParentID: parentID,
		Name:     name,
		Level:    level,
		Path:     path,
	}
	s.orgs[id] = node
	return node, nil
}

func (s *OrgHierarchyService) GetOrg(id string) *OrgNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.orgs[id]
}

func (s *OrgHierarchyService) ListOrgs(filter string) []*OrgNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*OrgNode
	for _, n := range s.orgs {
		if filter != "" && !strings.Contains(strings.ToLower(n.Name), strings.ToLower(filter)) {
			continue
		}
		list = append(list, n)
	}
	return list
}

func (s *OrgHierarchyService) GetOrgTree(rootOrgID string) []*OrgNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	root, ok := s.orgs[rootOrgID]
	if !ok {
		return nil
	}
	var tree []*OrgNode
	tree = append(tree, root)
	for _, n := range s.orgs {
		if n.OrgID != rootOrgID && strings.HasPrefix(n.Path, root.Path+"/") {
			tree = append(tree, n)
		}
	}
	return tree
}

func (s *OrgHierarchyService) MoveOrg(orgID, newParentID string) (*OrgNode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.orgs[orgID]
	if !ok {
		return nil, fmt.Errorf("org not found")
	}
	if newParentID == orgID {
		return nil, fmt.Errorf("cannot move org into itself")
	}
	// Check for circular reference
	if newParentID != "" {
		p := s.orgs[newParentID]
		for p != nil {
			if p.OrgID == orgID {
				return nil, fmt.Errorf("circular reference detected")
			}
			p = s.orgs[p.ParentID]
		}
	}
	node.ParentID = newParentID
	if newParentID == "" {
		node.Level = 0
		node.Path = "/" + orgID
	} else {
		parent := s.orgs[newParentID]
		node.Level = parent.Level + 1
		node.Path = parent.Path + "/" + orgID
	}
	// Update children paths
	for _, n := range s.orgs {
		if strings.HasPrefix(n.Path, orgID+"/") || n.ParentID == orgID {
			s.rebuildPath(n)
		}
	}
	return node, nil
}

func (s *OrgHierarchyService) DeleteOrg(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orgs[id]; !ok {
		return fmt.Errorf("org not found")
	}
	// Check for children
	for _, n := range s.orgs {
		if n.ParentID == id {
			return fmt.Errorf("cannot delete org with children")
		}
	}
	delete(s.orgs, id)
	return nil
}

func (s *OrgHierarchyService) rebuildPath(n *OrgNode) {
	if n.ParentID == "" {
		n.Level = 0
		n.Path = "/" + n.OrgID
	} else if parent, ok := s.orgs[n.ParentID]; ok {
		n.Level = parent.Level + 1
		n.Path = parent.Path + "/" + n.OrgID
	}
}