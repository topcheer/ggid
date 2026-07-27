package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// --- Separation of Duties (SoD) tests ---

func TestCheckSoD_NoViolation(t *testing.T) {
	ResetSoDRules()
	violations := CheckSoD(context.Background(), uuid.New(), []string{"admin"})
	if len(violations) != 0 {
		t.Errorf("expected no violations for single role, got %d", len(violations))
	}
}

func TestCheckSoD_AdminAndAuditor(t *testing.T) {
	ResetSoDRules()
	userID := uuid.New()
	violations := CheckSoD(context.Background(), userID, []string{"admin", "auditor"})
	if len(violations) == 0 {
		t.Fatal("expected SoD violation for admin+auditor")
	}
	if violations[0].UserID != userID {
		t.Errorf("UserID mismatch: got %v, want %v", violations[0].UserID, userID)
	}
}

func TestCheckSoD_AdminAndCompliance(t *testing.T) {
	ResetSoDRules()
	violations := CheckSoD(context.Background(), uuid.New(), []string{"admin", "compliance"})
	if len(violations) == 0 {
		t.Fatal("expected SoD violation for admin+compliance")
	}
}

func TestCheckSoD_NoConflictRoles(t *testing.T) {
	ResetSoDRules()
	violations := CheckSoD(context.Background(), uuid.New(), []string{"viewer", "editor"})
	if len(violations) != 0 {
		t.Errorf("expected no violations for compatible roles, got %d", len(violations))
	}
}

func TestCheckSoD_EmptyRoles(t *testing.T) {
	ResetSoDRules()
	violations := CheckSoD(context.Background(), uuid.New(), []string{})
	if len(violations) != 0 {
		t.Errorf("expected no violations for empty roles, got %d", len(violations))
	}
}

func TestCheckSoD_AllThreeConflicting(t *testing.T) {
	ResetSoDRules()
	// admin + auditor + compliance triggers two rules
	violations := CheckSoD(context.Background(), uuid.New(), []string{"admin", "auditor", "compliance"})
	if len(violations) != 2 {
		t.Errorf("expected 2 SoD violations, got %d", len(violations))
	}
}

func TestCanAssignRole_Allowed(t *testing.T) {
	ResetSoDRules()
	err := CanAssignRole(context.Background(), []string{"viewer"}, "editor")
	if err != nil {
		t.Errorf("expected no error for compatible role, got %v", err)
	}
}

func TestCanAssignRole_BlockedBySoD(t *testing.T) {
	ResetSoDRules()
	err := CanAssignRole(context.Background(), []string{"admin"}, "auditor")
	if err == nil {
		t.Fatal("expected SoD violation error")
	}
}

func TestAddSoDRule_Custom(t *testing.T) {
	ResetSoDRules()
	AddSoDRule([]string{"deployer", "approver"}, "CI/CD separation")
	violations := CheckSoD(context.Background(), uuid.New(), []string{"deployer", "approver"})
	if len(violations) == 0 {
		t.Fatal("expected custom SoD rule violation")
	}
	found := false
	for _, v := range violations {
		if v.Reason == "CI/CD separation" {
			found = true
		}
	}
	if !found {
		t.Error("custom rule violation not found in results")
	}
	ResetSoDRules() // cleanup
}

func TestDelegationValidator_DefaultMaxDepth(t *testing.T) {
	dv := NewDelegationValidator(0)
	if dv.GetMaxDepth() != 3 {
		t.Errorf("expected default max depth 3, got %d", dv.GetMaxDepth())
	}
}

func TestDelegationValidator_ScopeNarrowing_NotSubset(t *testing.T) {
	dv := NewDelegationValidator(3)
	delegated := []string{"read", "write"}
	requested := []string{"delete"}
	ok, reason := dv.CheckScopeNarrowing(delegated, requested)
	if ok {
		t.Error("expected scope narrowing to fail — 'delete' not in delegated scopes")
	}
	if reason == "" {
		t.Error("expected non-empty reason for scope violation")
	}
}

func TestDelegationValidator_Circular_RepeatDelegatee(t *testing.T) {
	dv := NewDelegationValidator(3)
	userA := uuid.New()
	userB := uuid.New()
	userC := uuid.New()
	chain := []DelegationLink{
		{DelegatorID: userA, DelegateeID: userB},
		{DelegatorID: userB, DelegateeID: userC},
		{DelegatorID: userC, DelegateeID: userB}, // B appears twice
	}
	isCircular, _ := dv.CheckCircularDelegation(chain)
	if !isCircular {
		t.Error("expected circular delegation detected (B appears twice)")
	}
}

func TestDelegationValidator_Expiry_NilExpiry(t *testing.T) {
	dv := NewDelegationValidator(3)
	chain := []DelegationLink{
		{DelegatorID: uuid.New(), DelegateeID: uuid.New(), ExpiresAt: nil},
	}
	ok, _ := dv.CheckDelegationExpiry(chain)
	if !ok {
		t.Error("expected nil expiry to pass (no expiration)")
	}
}

func TestDelegationValidator_SetMaxDepth(t *testing.T) {
	dv := NewDelegationValidator(3)
	dv.SetMaxDepth(10)
	if dv.GetMaxDepth() != 10 {
		t.Errorf("MaxDepth mismatch: got %d, want 10", dv.GetMaxDepth())
	}
}

func TestDelegationValidator_EmptyChain(t *testing.T) {
	dv := NewDelegationValidator(3)
	result, _ := dv.CheckDelegationDepth(nil)
	if !result.Valid {
		t.Error("expected valid for empty chain")
	}
}
