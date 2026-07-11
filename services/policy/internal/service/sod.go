package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// SoDRule defines mutually exclusive roles.
type SoDRule struct {
	ID      uuid.UUID
	Roles   []string // roles that are mutually exclusive
	Reason  string
}

// SoDViolation represents a detected conflict.
type SoDViolation struct {
	UserID  uuid.UUID
	Roles   []string
	RuleID  uuid.UUID
	Reason  string
}

var (
	sodMu       sync.RWMutex
	sodRules    = []SoDRule{
		{ID: uuid.New(), Roles: []string{"admin", "auditor"}, Reason: "admin + auditor mutually exclusive"},
		{ID: uuid.New(), Roles: []string{"admin", "compliance"}, Reason: "admin + compliance mutually exclusive"},
	}
)

// AddSoDRule adds a custom SoD rule.
func AddSoDRule(roles []string, reason string) SoDRule {
	sodMu.Lock()
	defer sodMu.Unlock()
	r := SoDRule{ID: uuid.New(), Roles: roles, Reason: reason}
	sodRules = append(sodRules, r)
	return r
}

// CheckSoD checks if a user's role set violates any SoD rule.
func CheckSoD(_ context.Context, userID uuid.UUID, userRoles []string) []SoDViolation {
	sodMu.RLock()
	defer sodMu.RUnlock()

	roleSet := make(map[string]bool)
	for _, r := range userRoles {
		roleSet[r] = true
	}

	var violations []SoDViolation
	for _, rule := range sodRules {
		heldCount := 0
		for _, r := range rule.Roles {
			if roleSet[r] {
				heldCount++
			}
		}
		if heldCount >= 2 {
			violations = append(violations, SoDViolation{
				UserID: userID,
				Roles:  rule.Roles,
				RuleID: rule.ID,
				Reason: rule.Reason,
			})
		}
	}
	return violations
}

// CanAssignRole checks if adding a new role would cause an SoD violation.
func CanAssignRole(_ context.Context, currentRoles []string, newRole string) error {
	testRoles := append(currentRoles, newRole)
	violations := CheckSoD(context.Background(), uuid.Nil, testRoles)
	if len(violations) > 0 {
		return fmt.Errorf("SoD violation: %s", violations[0].Reason)
	}
	return nil
}

// ResetSoDRules resets to default rules (for testing).
func ResetSoDRules() {
	sodMu.Lock()
	defer sodMu.Unlock()
	sodRules = []SoDRule{
		{ID: uuid.New(), Roles: []string{"admin", "auditor"}, Reason: "admin + auditor mutually exclusive"},
		{ID: uuid.New(), Roles: []string{"admin", "compliance"}, Reason: "admin + compliance mutually exclusive"},
	}
}
