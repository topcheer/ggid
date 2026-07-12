package service

import (
	"fmt"
	"strings"
	"sync"
)

type Scope struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IsSystem          bool   `json:"is_system"`
	DefaultForClients bool   `json:"default_for_clients"`
}

type ScopeManager struct {
	mu     sync.RWMutex
	scopes map[string]*Scope
}

func NewScopeManager() *ScopeManager {
	sm := &ScopeManager{scopes: make(map[string]*Scope)}
	// Register system scopes
	for _, s := range []Scope{
		{Name: "openid", Description: "OpenID Connect scope", IsSystem: true, DefaultForClients: true},
		{Name: "profile", Description: "Profile information", IsSystem: true, DefaultForClients: true},
		{Name: "email", Description: "Email address", IsSystem: true, DefaultForClients: true},
		{Name: "offline_access", Description: "Offline access with refresh tokens", IsSystem: true, DefaultForClients: false},
		{Name: "admin", Description: "Administrative access", IsSystem: true, DefaultForClients: false},
	} {
		s := s
		sm.scopes[s.Name] = &s
	}
	return sm
}

func (sm *ScopeManager) CreateScope(name, description string) (*Scope, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if _, exists := sm.scopes[name]; exists {
		return nil, fmt.Errorf("scope %s already exists", name)
	}
	s := &Scope{Name: name, Description: description, IsSystem: false, DefaultForClients: false}
	sm.scopes[name] = s
	return s, nil
}

func (sm *ScopeManager) ListScopes() []*Scope {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	var list []*Scope
	for _, s := range sm.scopes {
		list = append(list, s)
	}
	return list
}

func (sm *ScopeManager) GetScope(name string) *Scope {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.scopes[name]
}

func (sm *ScopeManager) UpdateScope(name string, changes map[string]any) (*Scope, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.scopes[name]
	if !ok {
		return nil, fmt.Errorf("scope %s not found", name)
	}
	if s.IsSystem {
		return nil, fmt.Errorf("cannot modify system scope %s", name)
	}
	if desc, ok := changes["description"].(string); ok {
		s.Description = desc
	}
	if def, ok := changes["default_for_clients"].(bool); ok {
		s.DefaultForClients = def
	}
	return s, nil
}

func (sm *ScopeManager) DeleteScope(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.scopes[name]
	if !ok {
		return fmt.Errorf("scope %s not found", name)
	}
	if s.IsSystem {
		return fmt.Errorf("cannot delete system scope %s", name)
	}
	delete(sm.scopes, name)
	return nil
}

func (sm *ScopeManager) ValidateScopes(requested, allowed []string) ([]string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	allowedSet := make(map[string]bool)
	for _, a := range allowed {
		allowedSet[a] = true
	}
	var valid []string
	for _, req := range requested {
		// Check wildcard
		if strings.HasSuffix(req, ":*") {
			prefix := strings.TrimSuffix(req, ":*")
			matched := false
			for scope := range sm.scopes {
				if strings.HasPrefix(scope, prefix+":") {
					if !allowedSet[scope] && !allowedSet[prefix+":*"] && !allowedSet["*"] {
						continue
					}
					matched = true
				}
			}
			if !matched {
				return nil, fmt.Errorf("scope %s not allowed or not found", req)
			}
			valid = append(valid, req)
			continue
		}
		// Check scope exists
		if _, exists := sm.scopes[req]; !exists {
			return nil, fmt.Errorf("scope %s does not exist", req)
		}
		if !allowedSet[req] && !allowedSet["*"] {
			return nil, fmt.Errorf("scope %s not allowed", req)
		}
		valid = append(valid, req)
	}
	return valid, nil
}