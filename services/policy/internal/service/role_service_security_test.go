package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// TestCreateRole_ReservedKeysAreBlocked tests that reserved system role keys cannot be created.
func TestCreateRole_ReservedKeysAreBlocked(t *testing.T) {
	tests := []struct {
		name          string
		roleKey       string
		expectError   bool
		errorContains string
	}{
		{
			name:          "platform:admin is reserved",
			roleKey:       "platform:admin",
			expectError:   true,
			errorContains: "role key is reserved",
		},
		{
			name:          "tenant:admin is reserved",
			roleKey:       "tenant:admin",
			expectError:   true,
			errorContains: "role key is reserved",
		},
		{
			name:          "tenant:auditor is reserved",
			roleKey:       "tenant:auditor",
			expectError:   true,
			errorContains: "role key is reserved",
		},
		{
			name:          "user:self is reserved",
			roleKey:       "user:self",
			expectError:   true,
			errorContains: "role key is reserved",
		},
		{
			name:        "custom role is allowed",
			roleKey:     "custom:manager",
			expectError: false,
		},
		{
			name:          "PLATFORM:ADMIN (uppercase) is also reserved",
			roleKey:       "PLATFORM:ADMIN",
			expectError:   true,
			errorContains: "role key is reserved",
		},
		{
			name:          "platform:admin with whitespace is also reserved",
			roleKey:       "  platform:admin  ",
			expectError:   true,
			errorContains: "role key is reserved",
		},
		{
			name:        "platform-admin is NOT reserved (different format)",
			roleKey:     "platform-admin",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRole := &mockRoleRepo{}
			mockPerm := &mockPermRepo{}
			mockUserRole := &mockUserRoleRepo{}

			svc := NewRoleService(mockRole, mockPerm, mockUserRole)

			tenantID := uuid.New()
			role, err := svc.CreateRole(context.Background(), tenantID, tt.roleKey, "Test Role", "A test role", nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for reserved role key %s, but got none", tt.roleKey)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got: %v", tt.errorContains, err)
				}
				if role != nil {
					t.Errorf("expected nil role on error, got: %v", role)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for role key %s: %v", tt.roleKey, err)
				}
				if role == nil {
					t.Errorf("expected role to be created for key %s", tt.roleKey)
				} else if role.Key != tt.roleKey {
					t.Errorf("role key mismatch: got %s, want %s", role.Key, tt.roleKey)
				}
			}
		})
	}
}

// TestAssignRole_NoAuthorizationCheck tests that AssignRole does NOT check if the user assigning the role has permission.
// This is a security vulnerability - a user could assign themselves platform:admin if they can get that role ID.
func TestAssignRole_NoAuthorizationCheck(t *testing.T) {
	mockRole := &mockRoleRepo{}
	mockPerm := &mockPermRepo{}
	mockUserRole := &mockUserRoleRepo{}

	svc := NewRoleService(mockRole, mockPerm, mockUserRole)

	// Create a role (simulating platform:admin even though it should be reserved)
	tenantID := uuid.New()
	platformAdminRole := &domain.Role{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Key:       "platform:admin", // This should be blocked, but let's simulate it bypassing the check
		Name:      "Platform Admin",
		CreatedAt: time.Now(),
	}
	mockRole.roles = map[uuid.UUID]*domain.Role{
		platformAdminRole.ID: platformAdminRole,
	}

	// Now try to assign this role to a regular user
	userID := uuid.New()
	grantedBy := uuid.New() // Same user granting to themselves!

	err := svc.AssignRole(context.Background(), userID, platformAdminRole.ID, domain.ScopeOrganization, tenantID, grantedBy, nil)

	// BUG: This succeeds without any authorization check!
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the assignment was created
	if len(mockUserRole.assignments) != 1 {
		t.Errorf("expected 1 assignment, got %d", len(mockUserRole.assignments))
	}

	// A real system should check if 'grantedBy' has permission to assign this role!
	t.Logf("SECURITY ISSUE: User %s was able to assign role %s to themselves without authorization check",
		userID, platformAdminRole.Key)
}

// TestAssignRole_AllowsSelfAssignment verifies that self-assignment is now blocked.
func TestAssignRole_AllowsSelfAssignment(t *testing.T) {
	mockRole := &mockRoleRepo{}
	mockPerm := &mockPermRepo{}
	mockUserRole := &mockUserRoleRepo{}

	svc := NewRoleService(mockRole, mockPerm, mockUserRole)

	// Create a regular role
	tenantID := uuid.New()
	role := &domain.Role{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Key:       "manager",
		Name:      "Manager",
		CreatedAt: time.Now(),
	}
	mockRole.roles = map[uuid.UUID]*domain.Role{
		role.ID: role,
	}

	// A user assigns the role to themselves
	userID := uuid.New()

	err := svc.AssignRole(context.Background(), userID, role.ID, domain.ScopeOrganization, tenantID, userID, nil)

	// FIX: This should now fail with permission denied
	if err == nil {
		t.Error("expected error for self-assignment, but got none")
	} else {
		t.Logf("Good: Self-assignment blocked with error: %v", err)
	}

	// Verify no assignment was created
	if len(mockUserRole.assignments) != 0 {
		t.Errorf("expected 0 assignments after blocked self-assignment, got %d", len(mockUserRole.assignments))
	}
}

// TestCreateRole_ReservedKeyCaseInsensitivity tests that the check is case-insensitive.
func TestCreateRole_ReservedKeyCaseInsensitivity(t *testing.T) {
	mockRole := &mockRoleRepo{}
	mockPerm := &mockPermRepo{}
	mockUserRole := &mockUserRoleRepo{}

	svc := NewRoleService(mockRole, mockPerm, mockUserRole)

	tenantID := uuid.New()

	// Try various case combinations
	variations := []string{
		"Platform:Admin",
		"PLATFORM:ADMIN",
		"PlAtFoRm:AdMiN",
		"Tenant:Admin",
		"TENANT:ADMIN",
	}

	for _, variant := range variations {
		t.Run(variant, func(t *testing.T) {
			_, err := svc.CreateRole(context.Background(), tenantID, variant, "Test", "Test", nil)
			if err == nil {
				t.Errorf("expected error for reserved role key variant %s", variant)
			}
		})
	}
}
