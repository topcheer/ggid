package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

var testTenantID = uuid.New()

func TestDelegatePermissions_Success(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	perms := []string{"read:users", "write:roles"}

	d, err := svc.DelegatePermissions(context.Background(), testTenantID, delegator, delegatee, perms, 1*time.Hour)
	if err != nil {
		t.Fatalf("DelegatePermissions: %v", err)
	}
	if d.DelegatorID != delegator || d.DelegateeID != delegatee {
		t.Error("delegation IDs mismatch")
	}
	if d.TenantID != testTenantID {
		t.Error("TenantID should be set")
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
	_, err := svc.DelegatePermissions(context.Background(), testTenantID, id, id, []string{"read"}, time.Hour)
	if err == nil {
		t.Error("should reject self-delegation")
	}
}

func TestDelegatePermissions_EmptyPermissions(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	_, err := svc.DelegatePermissions(context.Background(), testTenantID, uuid.New(), uuid.New(), []string{}, time.Hour)
	if err == nil {
		t.Error("should reject empty permissions")
	}
}

func TestDelegatePermissions_InvalidDuration(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	_, err := svc.DelegatePermissions(context.Background(), testTenantID, uuid.New(), uuid.New(), []string{"read"}, 0)
	if err == nil {
		t.Error("should reject zero duration")
	}
}

func TestDelegatePermissions_PlatformEscalation(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	_, err := svc.DelegatePermissions(context.Background(), testTenantID, uuid.New(), uuid.New(), []string{"platform:admin"}, time.Hour)
	if err == nil {
		t.Error("should reject platform-level permission delegation")
	}
}

func TestDelegatePermissions_NilTenant(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	_, err := svc.DelegatePermissions(context.Background(), uuid.Nil, uuid.New(), uuid.New(), []string{"read"}, time.Hour)
	if err == nil {
		t.Error("should reject nil tenant")
	}
}

func TestCheckDelegatedPermission_Granted(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	svc.DelegatePermissions(context.Background(), testTenantID, delegator, delegatee, []string{"read:users", "write:roles"}, 1*time.Hour)

	if !svc.CheckDelegatedPermission(context.Background(), testTenantID, delegator, delegatee, "read:users") {
		t.Error("should have delegated read:users permission")
	}
}

func TestCheckDelegatedPermission_NotGranted(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	svc.DelegatePermissions(context.Background(), testTenantID, delegator, delegatee, []string{"read:users"}, 1*time.Hour)

	if svc.CheckDelegatedPermission(context.Background(), testTenantID, delegator, delegatee, "delete:users") {
		t.Error("should NOT have delete:users permission")
	}
}

func TestCheckDelegatedPermission_CrossTenant(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	svc.DelegatePermissions(context.Background(), testTenantID, delegator, delegatee, []string{"read"}, 1*time.Hour)

	otherTenant := uuid.New()
	if svc.CheckDelegatedPermission(context.Background(), otherTenant, delegator, delegatee, "read") {
		t.Error("should NOT grant permission from different tenant")
	}
}

func TestCheckDelegatedPermission_Expired(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	d, _ := svc.DelegatePermissions(context.Background(), testTenantID, delegator, delegatee, []string{"read"}, 1*time.Millisecond)
	d.ExpiresAt = time.Now().UTC().Add(-1 * time.Second) // manually expire

	time.Sleep(10 * time.Millisecond)
	if svc.CheckDelegatedPermission(context.Background(), testTenantID, delegator, delegatee, "read") {
		t.Error("expired delegation should not grant permission")
	}
}

func TestCheckDelegatedPermission_Revoked(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	d, _ := svc.DelegatePermissions(context.Background(), testTenantID, delegator, delegatee, []string{"read"}, 1*time.Hour)

	svc.RevokeDelegation(context.Background(), d.ID)

	if svc.CheckDelegatedPermission(context.Background(), testTenantID, delegator, delegatee, "read") {
		t.Error("revoked delegation should not grant permission")
	}
}

func TestListDelegations(t *testing.T) {
	ResetDelegationStore()
	svc := NewPolicyService(nil)

	delegator := uuid.New()
	delegatee := uuid.New()
	svc.DelegatePermissions(context.Background(), testTenantID, delegator, delegatee, []string{"read"}, 1*time.Hour)
	svc.DelegatePermissions(context.Background(), testTenantID, delegator, uuid.New(), []string{"write"}, 1*time.Hour)

	// Should list 2 active delegations for delegator
	delegs, _ := svc.ListDelegations(context.Background(), testTenantID, delegator)
	if len(delegs) != 2 {
		t.Errorf("expected 2 delegations for delegator, got %d", len(delegs))
	}

	// Should list 1 for delegatee
	delegs, _ = svc.ListDelegations(context.Background(), testTenantID, delegatee)
	if len(delegs) != 1 {
		t.Errorf("expected 1 delegation for delegatee, got %d", len(delegs))
	}

	// Cross-tenant should return 0
	delegs, _ = svc.ListDelegations(context.Background(), uuid.New(), delegator)
	if len(delegs) != 0 {
		t.Errorf("expected 0 delegations for different tenant, got %d", len(delegs))
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

	d, _ := svc.DelegatePermissions(context.Background(), testTenantID, uuid.New(), uuid.New(), []string{"read"}, 1*time.Hour)
	found, err := svc.GetDelegation(context.Background(), d.ID)
	if err != nil {
		t.Fatalf("GetDelegation: %v", err)
	}
	if found.ID != d.ID {
		t.Error("delegation ID mismatch")
	}
}
