package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DelegationValidationResult holds the result of validating a delegation.
type DelegationValidationResult struct {
	Valid    bool   `json:"valid"`
	Reason   string `json:"reason"`
	MaxDepth int    `json:"max_depth"`
}

// DelegationLink represents a single link in a delegation chain.
type DelegationLink struct {
	DelegatorID uuid.UUID  `json:"delegator_id"`
	DelegateeID uuid.UUID  `json:"delegatee_id"`
	Scopes      []string   `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// DelegationValidator validates delegation chains.
type DelegationValidator struct {
	mu       sync.RWMutex
	maxDepth int
}

// NewDelegationValidator creates a new DelegationValidator.
func NewDelegationValidator(maxDepth int) *DelegationValidator {
	if maxDepth <= 0 {
		maxDepth = 3
	}
	return &DelegationValidator{maxDepth: maxDepth}
}

// ValidateDelegation validates a single delegation request.
func (dv *DelegationValidator) ValidateDelegation(delegatorID, delegateeID uuid.UUID, scopes []string, maxDepth int) (*DelegationValidationResult, error) {
	if delegatorID == uuid.Nil || delegateeID == uuid.Nil {
		return &DelegationValidationResult{Valid: false, Reason: "delegator and delegatee IDs are required"}, nil
	}
	if delegatorID == delegateeID {
		return &DelegationValidationResult{Valid: false, Reason: "self-delegation is not allowed"}, nil
	}
	if len(scopes) == 0 {
		return &DelegationValidationResult{Valid: false, Reason: "at least one scope is required"}, nil
	}
	if maxDepth <= 0 {
		maxDepth = dv.maxDepth
	}
	return &DelegationValidationResult{Valid: true, Reason: "delegation is valid", MaxDepth: maxDepth}, nil
}

// CheckDelegationDepth checks that a chain does not exceed maxDepth.
func (dv *DelegationValidator) CheckDelegationDepth(chain []DelegationLink) (*DelegationValidationResult, error) {
	if len(chain) == 0 {
		return &DelegationValidationResult{Valid: true, Reason: "empty chain"}, nil
	}
	if len(chain) > dv.maxDepth {
		return &DelegationValidationResult{Valid: false, Reason: fmt.Sprintf("chain depth %d exceeds max %d", len(chain), dv.maxDepth), MaxDepth: dv.maxDepth}, nil
	}
	return &DelegationValidationResult{Valid: true, Reason: "depth within limit", MaxDepth: dv.maxDepth}, nil
}

// CheckScopeNarrowing ensures requested scopes are a subset of delegated scopes.
func (dv *DelegationValidator) CheckScopeNarrowing(delegatedScopes, requestedScopes []string) (bool, string) {
	delegated := make(map[string]bool)
	for _, s := range delegatedScopes {
		delegated[s] = true
	}
	for _, req := range requestedScopes {
		if !delegated[req] && !delegated["*"] {
			return false, fmt.Sprintf("scope '%s' not in delegated scopes", req)
		}
	}
	return true, ""
}

// CheckDelegationExpiry checks if any link in the chain has expired.
func (dv *DelegationValidator) CheckDelegationExpiry(chain []DelegationLink) (bool, string) {
	now := time.Now()
	for i, link := range chain {
		if link.ExpiresAt != nil && now.After(*link.ExpiresAt) {
			return false, fmt.Sprintf("link %d expired", i)
		}
	}
	return true, ""
}

// CheckCircularDelegation detects circular delegation using a visited-set.
// A cycle exists when a delegatee appears more than once in the chain.
func (dv *DelegationValidator) CheckCircularDelegation(chain []DelegationLink) (bool, string) {
	visited := make(map[uuid.UUID]bool)
	for _, link := range chain {
		if visited[link.DelegateeID] {
			return true, fmt.Sprintf("circular delegation at user %s", link.DelegateeID)
		}
		visited[link.DelegateeID] = true
	}
	// Also check if the last delegatee points back to the first delegator.
	if len(chain) >= 2 {
		firstDelegator := chain[0].DelegatorID
		lastDelegatee := chain[len(chain)-1].DelegateeID
		if firstDelegator == lastDelegatee {
			return true, fmt.Sprintf("circular delegation at user %s", lastDelegatee)
		}
	}
	return false, ""
}

// ValidateChain performs all checks on a delegation chain.
func (dv *DelegationValidator) ValidateChain(chain []DelegationLink) (*DelegationValidationResult, error) {
	depthResult, _ := dv.CheckDelegationDepth(chain)
	if !depthResult.Valid {
		return depthResult, nil
	}
	if ok, reason := dv.CheckDelegationExpiry(chain); !ok {
		return &DelegationValidationResult{Valid: false, Reason: reason}, nil
	}
	if isCircular, reason := dv.CheckCircularDelegation(chain); isCircular {
		return &DelegationValidationResult{Valid: false, Reason: reason}, nil
	}
	for _, link := range chain {
		if link.DelegatorID == link.DelegateeID {
			return &DelegationValidationResult{Valid: false, Reason: "self-delegation is not allowed"}, nil
		}
	}
	return &DelegationValidationResult{Valid: true, Reason: "chain is valid", MaxDepth: dv.maxDepth}, nil
}

// SetMaxDepth updates the max delegation depth.
func (dv *DelegationValidator) SetMaxDepth(depth int) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.maxDepth = depth
}

// GetMaxDepth returns the current max delegation depth.
func (dv *DelegationValidator) GetMaxDepth() int {
	dv.mu.RLock()
	defer dv.mu.RUnlock()
	return dv.maxDepth
}
