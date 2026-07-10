package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// TestCheck_ExpiredRole_Deny verifies that the evaluator filters out
// role assignments whose ExpiresAt is in the past, even if the mock
// returns them (defense-in-depth against caching layer bypasses).
func TestCheck_ExpiredRole_Deny(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	// Create a user-role assignment that expired 1 hour ago.
	expired := time.Now().Add(-1 * time.Hour)

	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {newPerm("users", "read")}},
	}
	// mockUserRoleReader returns []*domain.UserRole internally
	ur := &mockUserRoleReader{
		roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}},
	}
	// Override to include an expired assignment
	ur2 := &expiredRoleReader{
		roles: []*domain.UserRole{
			{RoleID: roleID, ExpiresAt: &expired},
		},
	}

	// With the standard mock (no expiration), should allow
	e := NewEvaluator(rr, ur, &mockPolicyReader{})
	result, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("non-expired role should allow access")
	}

	// With expired role reader, should deny
	e2 := NewEvaluator(rr, ur2, &mockPolicyReader{})
	result2, err := e2.Check(context.Background(), newRequest(userID, "users", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.Allowed {
		t.Error("expired role assignment should be denied by evaluator")
	}
	if result2.Reason != "user has no role assignments" {
		t.Errorf("expected 'user has no role assignments', got %q", result2.Reason)
	}
}

// TestCheck_NonExpiredRole_Allow verifies that role assignments with
// ExpiresAt in the future are still honored.
func TestCheck_NonExpiredRole_Allow(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	future := time.Now().Add(24 * time.Hour)

	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {newPerm("users", "read")}},
	}
	ur := &expiredRoleReader{
		roles: []*domain.UserRole{
			{RoleID: roleID, ExpiresAt: &future},
		},
	}

	e := NewEvaluator(rr, ur, &mockPolicyReader{})
	result, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("future-expiring role should allow access")
	}
}

// TestCheck_NilExpiresAt_Allow verifies that role assignments with nil
// ExpiresAt (permanent) are honored.
func TestCheck_NilExpiresAt_Allow(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {newPerm("users", "read")}},
	}
	ur := &expiredRoleReader{
		roles: []*domain.UserRole{
			{RoleID: roleID, ExpiresAt: nil},
		},
	}

	e := NewEvaluator(rr, ur, &mockPolicyReader{})
	result, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("role with nil ExpiresAt (permanent) should allow access")
	}
}

// expiredRoleReader is a custom UserRoleReader that returns UserRole objects
// with explicit ExpiresAt values for testing the evaluator's expiration filter.
type expiredRoleReader struct {
	roles []*domain.UserRole
}

func (m *expiredRoleReader) GetUserRoles(_ context.Context, _ uuid.UUID) ([]*domain.UserRole, error) {
	return m.roles, nil
}
