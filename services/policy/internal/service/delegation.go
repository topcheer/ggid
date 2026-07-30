package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// Delegation represents a permission subset delegated from user A to user B.
type Delegation struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	DelegatorID uuid.UUID // user A (granting)
	DelegateeID uuid.UUID // user B (receiving)
	Permissions []string  // subset of permissions delegated
	ExpiresAt   time.Time // max_duration enforced here
	Revoked     bool
	CreatedAt   time.Time
}

// delegationStore is an in-memory store for delegations.
type delegationStore struct {
	mu          sync.RWMutex
	delegations map[uuid.UUID]*Delegation
}

var globalDelegationStore = &delegationStore{
	delegations: make(map[uuid.UUID]*Delegation),
}

// DelegatePermissions delegates a subset of permissions from delegator to delegatee.
// maxDuration controls how long the delegation is valid.
// tenantID is required for tenant isolation.
func (s *PolicyService) DelegatePermissions(ctx context.Context, tenantID, delegatorID, delegateeID uuid.UUID, permissions []string, maxDuration time.Duration) (*Delegation, error) {
	if tenantID == uuid.Nil {
		return nil, errors.InvalidArgument("tenant_id is required")
	}
	if delegatorID == uuid.Nil {
		return nil, errors.InvalidArgument("delegator_id is required")
	}
	if delegateeID == uuid.Nil {
		return nil, errors.InvalidArgument("delegatee_id is required")
	}
	if delegatorID == delegateeID {
		return nil, errors.InvalidArgument("cannot delegate to self")
	}
	if len(permissions) == 0 {
		return nil, errors.InvalidArgument("at least one permission is required")
	}
	if maxDuration <= 0 {
		return nil, errors.InvalidArgument("max_duration must be positive")
	}

	// SECURITY: Prevent privilege escalation — tenant admins must not delegate platform-level permissions.
	for _, p := range permissions {
		if strings.HasPrefix(p, "platform:") {
			return nil, errors.PermissionDenied("cannot delegate platform-level permissions")
		}
	}

	now := time.Now().UTC()
	d := &Delegation{
		ID:          uuid.New(),
		TenantID:    tenantID,
		DelegatorID: delegatorID,
		DelegateeID: delegateeID,
		Permissions: permissions,
		ExpiresAt:   now.Add(maxDuration),
		CreatedAt:   now,
	}

	globalDelegationStore.mu.Lock()
	globalDelegationStore.delegations[d.ID] = d
	globalDelegationStore.mu.Unlock()

	return d, nil
}

// GetDelegation retrieves a delegation by ID.
func (s *PolicyService) GetDelegation(ctx context.Context, id uuid.UUID) (*Delegation, error) {
	globalDelegationStore.mu.RLock()
	defer globalDelegationStore.mu.RUnlock()
	d, ok := globalDelegationStore.delegations[id]
	if !ok {
		return nil, fmt.Errorf("delegation not found")
	}
	return d, nil
}

// ListDelegations lists active delegations for a user (as delegator or delegatee), scoped to tenant.
func (s *PolicyService) ListDelegations(ctx context.Context, tenantID, userID uuid.UUID) ([]*Delegation, error) {
	globalDelegationStore.mu.RLock()
	defer globalDelegationStore.mu.RUnlock()

	var result []*Delegation
	for _, d := range globalDelegationStore.delegations {
		if d.TenantID != tenantID {
			continue
		}
		if (d.DelegatorID == userID || d.DelegateeID == userID) && !d.Revoked && time.Now().UTC().Before(d.ExpiresAt) {
			result = append(result, d)
		}
	}
	return result, nil
}

// CheckDelegatedPermission checks if a delegatee has a specific permission
// via an active delegation from a delegator, within the same tenant.
func (s *PolicyService) CheckDelegatedPermission(ctx context.Context, tenantID, delegatorID, delegateeID uuid.UUID, permission string) bool {
	globalDelegationStore.mu.RLock()
	defer globalDelegationStore.mu.RUnlock()

	now := time.Now().UTC()
	for _, d := range globalDelegationStore.delegations {
		if d.TenantID != tenantID {
			continue
		}
		if d.DelegatorID == delegatorID && d.DelegateeID == delegateeID && !d.Revoked && now.Before(d.ExpiresAt) {
			for _, p := range d.Permissions {
				if p == permission {
					return true
				}
			}
		}
	}
	return false
}

// RevokeDelegation revokes a delegation by ID.
func (s *PolicyService) RevokeDelegation(ctx context.Context, id uuid.UUID) error {
	globalDelegationStore.mu.Lock()
	defer globalDelegationStore.mu.Unlock()

	d, ok := globalDelegationStore.delegations[id]
	if !ok {
		return fmt.Errorf("delegation not found")
	}
	d.Revoked = true
	return nil
}

// ResetDelegationStore clears all delegations (for testing).
func ResetDelegationStore() {
	globalDelegationStore.mu.Lock()
	defer globalDelegationStore.mu.Unlock()
	globalDelegationStore.delegations = make(map[uuid.UUID]*Delegation)
}

// Ensure domain types are referenced to avoid unused import.
var _ = domain.EffectAllow
