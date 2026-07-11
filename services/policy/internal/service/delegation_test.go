package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDelegatePermissions_Success(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	perms := []string{"read:users", "write:roles"}

	d, err := svc.DelegatePermissions(context.Background(), delegator, delegatee, perms, 1*time.Hour)
	if err != nil {
		t.Fatalf("DelegatePermissions: %v", err)
	}
	if d.DelegatorID != delegator || d.DelegateeID != delegatee {
		t.Error("delegation IDs mismatch")
	}
	if len(d.Permissions) != 2 {
		t.Error("should have 2 permissions")
	}
	if !time.Now().UTC().Before(d.ExpiresAt) {
		t.Error("delegation should not be expired")
	}
}

func TestDelegatePermissions_SelfDelegation(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	id := uuid.New()
	_, err := svc.DelegatePermissions(context.Background(), id, id, []string{"read"}, time.Hour)
	if err == nil {
		t.Error("should reject self-delegation")
	}
}

func TestDelegatePermissions_EmptyPermissions(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	_, err := svc.DelegatePermissions(context.Background(), uuid.New(), uuid.New(), []string{}, time.Hour)
	if err == nil {
		t.Error("should reject empty permissions")
	}
}

func TestDelegatePermissions_InvalidDuration(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	_, err := svc.DelegatePermissions(context.Background(), uuid.New(), uuid.New(), []string{"read"}, 0)
	if err == nil {
		t.Error("should reject zero duration")
	}
}

func TestCheckDelegatedPermission_Granted(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	svc.DelegatePermissions(context.Background(), delegator, delegatee, []string{"read:users", "write:roles"}, 1*time.Hour)

	if !svc.CheckDelegatedPermission(context.Background(), delegator, delegatee, "read:users") {
		t.Error("should have delegated read:users permission")
	}
}

func TestCheckDelegatedPermission_NotGranted(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	svc.DelegatePermissions(context.Background(), delegator, delegatee, []string{"read:users"}, 1*time.Hour)

	if svc.CheckDelegatedPermission(context.Background(), delegator, delegatee, "delete:users") {
		t.Error("should NOT have delete:users permission")
	}
}

func TestCheckDelegatedPermission_Expired(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	d, _ := svc.DelegatePermissions(context.Background(), delegator, delegatee, []string{"read"}, 1*time.Millisecond)
	d.ExpiresAt = time.Now().UTC().Add(-1 * time.Second) // manually expire

	time.Sleep(10 * time.Millisecond)
	if svc.CheckDelegatedPermission(context.Background(), delegator, delegatee, "read") {
		t.Error("expired delegation should not grant permission")
	}
}

func TestCheckDelegatedPermission_Revoked(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	d, _ := svc.DelegatePermissions(context.Background(), delegator, delegatee, []string{"read"}, 1*time.Hour)

	svc.RevokeDelegation(context.Background(), d.ID)

	if svc.CheckDelegatedPermission(context.Background(), delegator, delegatee, "read") {
		t.Error("revoked delegation should not grant permission")
	}
}

func TestListDelegations(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	svc.DelegatePermissions(context.Background(), delegator, delegatee, []string{"read"}, 1*time.Hour)
	svc.DelegatePermissions(context.Background(), delegator, uuid.New(), []string{"write"}, 1*time.Hour)

	// Should list 2 active delegations for delegator
	delegs, _ := svc.ListDelegations(context.Background(), delegator)
	if len(delegs) != 2 {
		t.Errorf("expected 2 delegations for delegator, got %d", len(delegs))
	}

	// Should list 1 for delegatee
	delegs, _ = svc.ListDelegations(context.Background(), delegatee)
	if len(delegs) != 1 {
		t.Errorf("expected 1 delegation for delegatee, got %d", len(delegs))
	}
}

func TestRevokeDelegation_NotFound(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	err := svc.RevokeDelegation(context.Background(), uuid.New())
	if err == nil {
		t.Error("should error for nonexistent delegation")
	}
}

func TestGetDelegation_Success(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	d, _ := svc.DelegatePermissions(context.Background(), uuid.New(), uuid.New(), []string{"read"}, 1*time.Hour)
	found, err := svc.GetDelegation(context.Background(), d.ID)
	if err != nil {
		t.Fatalf("GetDelegation: %v", err)
	}
	if found.ID != d.ID {
		t.Error("delegation ID mismatch")
	}
}
