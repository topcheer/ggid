package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestSoDRegression_NoViolation verifies no false positives when roles don't conflict.
func TestSoDRegression_NoViolation(t *testing.T) {
	ResetSoDRules()
	defer ResetSoDRules()

	violations := CheckSoD(context.Background(), uuid.New(), []string{"viewer", "editor"})
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for non-conflicting roles, got %d", len(violations))
	}
}

// TestSoDRegression_AdminAuditorConflict verifies the default rule.
func TestSoDRegression_AdminAuditorConflict(t *testing.T) {
	ResetSoDRules()
	defer ResetSoDRules()

	userID := uuid.New()
	violations := CheckSoD(context.Background(), userID, []string{"admin", "auditor"})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].UserID != userID {
		t.Error("violation should carry the user ID")
	}
}

// TestSoDRegression_AdminComplianceConflict verifies second default rule.
func TestSoDRegression_AdminComplianceConflict(t *testing.T) {
	ResetSoDRules()
	defer ResetSoDRules()

	violations := CheckSoD(context.Background(), uuid.New(), []string{"admin", "compliance"})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

// TestSoDRegression_CustomRuleAddedAndChecked.
func TestSoDRegression_CustomRule(t *testing.T) {
	ResetSoDRules()
	defer ResetSoDRules()

	// Add custom rule: developer + deployer mutually exclusive
	AddSoDRule([]string{"developer", "deployer"}, "dev + deploy separation")

	violations := CheckSoD(context.Background(), uuid.New(), []string{"developer", "deployer"})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for custom rule, got %d", len(violations))
	}
}

// TestSoDRegression_CanAssignRole_AllowsSafe verifies preemptive check allows non-conflicting role.
func TestSoDRegression_CanAssignRole_AllowsSafe(t *testing.T) {
	ResetSoDRules()
	defer ResetSoDRules()

	err := CanAssignRole(context.Background(), []string{"viewer"}, "editor")
	if err != nil {
		t.Errorf("expected no error for safe assignment, got %v", err)
	}
}

// TestSoDRegression_CanAssignRole_BlocksConflict verifies preemptive check blocks.
func TestSoDRegression_CanAssignRole_BlocksConflict(t *testing.T) {
	ResetSoDRules()
	defer ResetSoDRules()

	err := CanAssignRole(context.Background(), []string{"admin"}, "auditor")
	if err == nil {
		t.Error("expected SoD violation error when assigning auditor to admin")
	}
}

// TestSoDRegression_MultipleViolations verifies multiple rules triggered simultaneously.
func TestSoDRegression_MultipleViolations(t *testing.T) {
	ResetSoDRules()
	defer ResetSoDRules()

	// User holds admin + auditor + compliance → 2 violations
	violations := CheckSoD(context.Background(), uuid.New(), []string{"admin", "auditor", "compliance"})
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations (admin+auditor + admin+compliance), got %d", len(violations))
	}
}

// TestSoDRegression_EmptyRoles verifies no panic with empty role list.
func TestSoDRegression_EmptyRoles(t *testing.T) {
	ResetSoDRules()
	defer ResetSoDRules()

	violations := CheckSoD(context.Background(), uuid.New(), []string{})
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for empty roles, got %d", len(violations))
	}
}

// TestSoDRegression_SingleRoleNoViolation verifies single role never violates.
func TestSoDRegression_SingleRoleNoViolation(t *testing.T) {
	ResetSoDRules()
	defer ResetSoDRules()

	for _, role := range []string{"admin", "auditor", "compliance", "viewer"} {
		violations := CheckSoD(context.Background(), uuid.New(), []string{role})
		if len(violations) != 0 {
			t.Errorf("expected 0 violations for single role %s, got %d", role, len(violations))
		}
	}
}
