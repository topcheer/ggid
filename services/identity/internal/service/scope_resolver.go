package service

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// ScopeResolution holds the result of scope resolution.
type ScopeResolution struct {
	Granted []string `json:"granted"`
	Denied  []string `json:"denied"`
	Reason  string   `json:"reason,omitempty"`
}

// ScopeResolver resolves requested scopes against user permissions and client restrictions.
type ScopeResolver struct {
	mu                 sync.RWMutex
	scopeHierarchy     map[string][]string
	clientRestrictions map[string][]string
	userPermissions    map[uuid.UUID][]string
}

// NewScopeResolver creates a new ScopeResolver.
func NewScopeResolver() *ScopeResolver {
	return &ScopeResolver{
		scopeHierarchy:     make(map[string][]string),
		clientRestrictions: make(map[string][]string),
		userPermissions:    make(map[uuid.UUID][]string),
	}
}

// SetScopeHierarchy sets the scope hierarchy (parent scope to child scopes).
func (sr *ScopeResolver) SetScopeHierarchy(hierarchy map[string][]string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.scopeHierarchy = hierarchy
}

// SetClientRestrictions sets per-client scope restrictions.
func (sr *ScopeResolver) SetClientRestrictions(restrictions map[string][]string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.clientRestrictions = restrictions
}

// SetUserPermissions sets permissions for a user.
func (sr *ScopeResolver) SetUserPermissions(userID uuid.UUID, permissions []string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.userPermissions[userID] = permissions
}

// ResolveScopes resolves requested scopes against user permissions, client restrictions, and hierarchy.
func (sr *ScopeResolver) ResolveScopes(userID uuid.UUID, clientID string, requestedScopes []string) (*ScopeResolution, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	if clientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	sr.mu.RLock()
	defer sr.mu.RUnlock()

	resolution := &ScopeResolution{Granted: []string{}, Denied: []string{}}
	userPerms, hasPerms := sr.userPermissions[userID]
	permSet := make(map[string]bool)
	for _, p := range userPerms {
		permSet[p] = true
	}
	clientScopes, hasRestrictions := sr.clientRestrictions[clientID]
	clientSet := make(map[string]bool)
	for _, s := range clientScopes {
		clientSet[s] = true
	}

	for _, scope := range requestedScopes {
		if scope == "*" {
			if hasPerms {
				for p := range permSet {
					if !hasRestrictions || clientSet[p] || clientSet["*"] {
						resolution.Granted = append(resolution.Granted, p)
					}
				}
			}
			continue
		}

		granted := false
		if hasPerms {
			if permSet[scope] || permSet["*"] || sr.hasParentScope(scope, permSet) {
				granted = true
			}
		}
		if granted && hasRestrictions {
			if !clientSet[scope] && !clientSet["*"] {
				granted = false
			}
		}
		if granted {
			resolution.Granted = append(resolution.Granted, scope)
		} else {
			resolution.Denied = append(resolution.Denied, scope)
		}
	}

	if len(resolution.Denied) > 0 {
		resolution.Reason = fmt.Sprintf("%d scope(s) denied", len(resolution.Denied))
	}
	return resolution, nil
}

func (sr *ScopeResolver) hasParentScope(scope string, permSet map[string]bool) bool {
	for parent, children := range sr.scopeHierarchy {
		if permSet[parent] {
			for _, child := range children {
				if child == scope {
					return true
				}
			}
		}
	}
	return false
}

// ExpandScope expands a scope to include all child scopes based on hierarchy.
func (sr *ScopeResolver) ExpandScope(scope string) []string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	if children, ok := sr.scopeHierarchy[scope]; ok {
		result := []string{scope}
		result = append(result, children...)
		for _, child := range children {
			result = append(result, sr.expandScopeLocked(child)...)
		}
		return dedupScopeStrings(result)
	}
	return []string{scope}
}

func (sr *ScopeResolver) expandScopeLocked(scope string) []string {
	if children, ok := sr.scopeHierarchy[scope]; ok {
		result := children
		for _, child := range children {
			result = append(result, sr.expandScopeLocked(child)...)
		}
		return result
	}
	return nil
}

// GetEffectiveScopes returns the effective set of scopes for a user after hierarchy expansion and client restrictions.
func (sr *ScopeResolver) GetEffectiveScopes(userID uuid.UUID, clientID string) ([]string, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	userPerms, ok := sr.userPermissions[userID]
	if !ok {
		return []string{}, nil
	}
	var effective []string
	for _, p := range userPerms {
		expanded := []string{p}
		if children, ok := sr.scopeHierarchy[p]; ok {
			expanded = append(expanded, children...)
		}
		for _, s := range expanded {
			if clientScopes, hasRestrictions := sr.clientRestrictions[clientID]; hasRestrictions {
				allowed := false
				for _, cs := range clientScopes {
					if cs == s || cs == "*" {
						allowed = true
						break
					}
				}
				if !allowed {
					continue
				}
			}
			effective = append(effective, s)
		}
	}
	return dedupScopeStrings(effective), nil
}

// Reset clears all data.
func (sr *ScopeResolver) Reset() {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.scopeHierarchy = make(map[string][]string)
	sr.clientRestrictions = make(map[string][]string)
	sr.userPermissions = make(map[uuid.UUID][]string)
}

func dedupScopeStrings(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
