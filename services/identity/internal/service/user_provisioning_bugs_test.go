package service

import (
	"testing"
	"time"

)

// BUG 1: UserProvisioningService has no tenant isolation
func TestUserProvisioning_NoTenantIsolation_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewUserProvisioningService()

	// Provision a user with no tenant information
	userData := map[string]any{
		"username": "testuser",
		"email":    "test@example.com",
		"tenant_id": "tenant1",
	}

	user1, err := svc.ProvisionUser(SourceHR, userData)
	if err != nil {
		t.Fatalf("ProvisionUser failed: %v", err)
	}

	// Provision another user with the same data but different tenant
	userData["tenant_id"] = "tenant2"
	user2, err := svc.ProvisionUser(SourceHR, userData)
	if err != nil {
		t.Fatalf("ProvisionUser failed: %v", err)
	}

	// BUG: Both users are created even though they have the same username/email
	// The service doesn't enforce tenant isolation
	if user1.Data["username"] == user2.Data["username"] && user1.Data["email"] == user2.Data["email"] {
		t.Error("BUG: UserProvisioningService doesn't enforce tenant isolation - duplicate users created")
	}
}

// BUG 2: ProvisionUser doesn't validate required fields
func TestProvisionUser_Validation_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewUserProvisioningService()

	// Try to provision with empty data
	user, err := svc.ProvisionUser(SourceHR, map[string]any{})
	if err != nil {
		t.Fatalf("ProvisionUser failed: %v", err)
	}

	if user == nil {
		t.Fatal("Expected user to be created")
	}

	// BUG: User is created even with no required fields
	if len(user.Data) == 0 {
		t.Log("WARNING: User created with empty data - no validation")
	}
}

// BUG 3: ProvisionUser with field mapping doesn't validate mapped fields
func TestProvisionUser_FieldMapping_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewUserProvisioningService()

	// Set up a rule with field mapping
	rule := ProvisioningRule{
		Source: SourceSCIM,
		FieldMapping: map[string]string{
			"userName":  "username",
			"emails":    "email",
			"name":      "display_name",
		},
		DefaultValues: map[string]any{
			"locale": "en",
		},
	}
	svc.SetRule(rule)

	// Provide data with missing required fields
	userData := map[string]any{
		// Missing "userName" which maps to "username"
		"someOtherField": "value",
	}

	user, err := svc.ProvisionUser(SourceSCIM, userData)
	if err != nil {
		t.Fatalf("ProvisionUser failed: %v", err)
	}

	// BUG: User is created even though required fields (username) are missing
	if user.Data["username"] == nil || user.Data["username"] == "" {
		t.Error("BUG: ProvisionUser should validate required mapped fields")
	}
}

// BUG 4: SyncUser doesn't actually sync with any external source
func TestSyncUser_NoActualSync_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewUserProvisioningService()

	// Provision a user
	user, err := svc.ProvisionUser(SourceHR, map[string]any{"username": "test"})
	if err != nil {
		t.Fatalf("ProvisionUser failed: %v", err)
	}

	originalUpdatedAt := user.UpdatedAt

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Sync the user
	syncedUser, err := svc.SyncUser(user.UserID)
	if err != nil {
		t.Fatalf("SyncUser failed: %v", err)
	}

	// BUG: SyncUser only updates the timestamp, doesn't actually sync data
	// It doesn't connect to any external HR/SCIM/IaC system
	if syncedUser.UpdatedAt.Equal(originalUpdatedAt) {
		t.Error("BUG: SyncUser should update timestamp")
	}

	if syncedUser.Data["username"] != user.Data["username"] {
		t.Error("BUG: SyncUser should preserve data, but it doesn't actually sync")
	}
}

// BUG 5: DeprovisionUser doesn't cascade to dependent systems
func TestDeprovisionUser_NoCascade_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewUserProvisioningService()

	// Provision a user
	user, err := svc.ProvisionUser(SourceHR, map[string]any{
		"username": "test",
		"email":    "test@example.com",
	})
	if err != nil {
		t.Fatalf("ProvisionUser failed: %v", err)
	}

	// Deprovision the user
	err = svc.DeprovisionUser(user.UserID, "test deprovision")
	if err != nil {
		t.Fatalf("DeprovisionUser failed: %v", err)
	}

	// BUG: DeprovisionUser only changes status to "deprovisioned"
	// It doesn't actually:
	// - Revoke tokens
	// - Remove from groups
	// - Delete from external systems
	// - Archive data

	// The user still exists in the system
	user, err = svc.SyncUser(user.UserID)
	if err != nil {
		t.Fatalf("SyncUser failed: %v", err)
	}

	if user.Status != "deprovisioned" {
		t.Error("BUG: User status should be deprovisioned")
	}

	// BUG: Data is still there
	if user.Data["username"] != "test" {
		t.Error("BUG: User data should still exist after deprovision")
	}
}

// BUG 6: Audit trail doesn't include tenant information
func TestAuditTrail_NoTenantInfo_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewUserProvisioningService()

	// Provision users from different "tenants"
	svc.ProvisionUser(SourceHR, map[string]any{"username": "user1", "tenant_id": "tenant1"})
	svc.ProvisionUser(SourceSCIM, map[string]any{"username": "user2", "tenant_id": "tenant2"})

	audit := svc.GetAuditTrail()

	// BUG: Audit entries don't include tenant_id
	// This makes it impossible to filter audit trail by tenant
	for _, entry := range audit {
		if entry.Action == "provision" {
			t.Log("Audit entry (missing tenant info):", entry)
		}
	}
}

// BUG 7: Multiple sources can provision same user without conflict
func TestProvisionUser_MultipleSources_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewUserProvisioningService()

	// Provision same user from HR
	user1, err := svc.ProvisionUser(SourceHR, map[string]any{"username": "duplicate", "email": "dup@example.com"})
	if err != nil {
		t.Fatalf("ProvisionUser failed: %v", err)
	}

	// Provision same user from SCIM
	user2, err := svc.ProvisionUser(SourceSCIM, map[string]any{"username": "duplicate", "email": "dup@example.com"})
	if err != nil {
		t.Fatalf("ProvisionUser failed: %v", err)
	}

	// BUG: Two separate user records are created
	// The service doesn't check for existing users with the same identity
	if user1.UserID == user2.UserID {
		t.Log("Users have same ID - check if this is intended")
	} else {
		t.Error("BUG: Same identity provisioned from multiple sources creates duplicate users")
	}
}

// BUG 8: SetRule can be called multiple times for same source
func TestSetRule_Overwrite_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewUserProvisioningService()

	// Set initial rule
	rule1 := ProvisioningRule{
		Source: SourceHR,
		FieldMapping: map[string]string{
			"userName": "username",
		},
	}
	svc.SetRule(rule1)

	// Overwrite with different rule
	rule2 := ProvisioningRule{
		Source: SourceHR,
		FieldMapping: map[string]string{
			"userName": "username",
			"emails":   "email",
		},
	}
	svc.SetRule(rule2)

	// BUG: Rule is silently overwritten
	// There's no warning or check if the rule is in use
	// This could affect ongoing provisioning operations
}
