package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

func newTestAccessRequestService() *AccessRequestService {
	return NewAccessRequestService(NewMemoryAccessRequestStore())
}

// 1. TestCreateAccessRequest_Success
func TestCreateAccessRequest_Success(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()
	tenantID := uuid.New()
	requesterID := uuid.New()

	req, err := svc.CreateAccessRequest(ctx, tenantID, requesterID,
		domain.ResourceTypeRole, "role-admin", "need admin for project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ID == uuid.Nil {
		t.Fatal("expected non-nil ID")
	}
	if req.Status != domain.AccessRequestPending {
		t.Fatalf("expected pending, got %s", req.Status)
	}
	if req.TenantID != tenantID {
		t.Fatal("tenant mismatch")
	}
	if req.RequesterID != requesterID {
		t.Fatal("requester mismatch")
	}
	if req.ResourceID != "role-admin" {
		t.Fatal("resource mismatch")
	}
	// Expiry should be ~7 days from now
	dur := req.ExpiresAt.Sub(req.CreatedAt)
	if dur < 6*24*time.Hour || dur > 8*24*time.Hour {
		t.Fatalf("expected ~7 days expiry, got %v", dur)
	}
}

// 2. TestCreateAccessRequest_InvalidResourceType
func TestCreateAccessRequest_InvalidResourceType(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()

	_, err := svc.CreateAccessRequest(ctx, uuid.New(), uuid.New(),
		"invalid_type", "res-1", "reason")
	if err == nil {
		t.Fatal("expected error for invalid resource type")
	}
}

// 3. TestCreateAccessRequest_MissingFields
func TestCreateAccessRequest_MissingFields(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()

	// Missing tenant
	_, err := svc.CreateAccessRequest(ctx, uuid.Nil, uuid.New(),
		domain.ResourceTypeRole, "r1", "")
	if err == nil {
		t.Fatal("expected error for nil tenant")
	}

	// Missing requester
	_, err = svc.CreateAccessRequest(ctx, uuid.New(), uuid.Nil,
		domain.ResourceTypeRole, "r1", "")
	if err == nil {
		t.Fatal("expected error for nil requester")
	}

	// Missing resource ID
	_, err = svc.CreateAccessRequest(ctx, uuid.New(), uuid.New(),
		domain.ResourceTypeRole, "", "")
	if err == nil {
		t.Fatal("expected error for empty resource ID")
	}
}

// 4. TestApproveAccessRequest_Success
func TestApproveAccessRequest_Success(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()
	tenantID := uuid.New()
	requesterID := uuid.New()
	approverID := uuid.New()

	req, _ := svc.CreateAccessRequest(ctx, tenantID, requesterID,
		domain.ResourceTypePermission, "perm:read", "need read access")

	approved, err := svc.ApproveAccessRequest(ctx, req.ID, approverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved.Status != domain.AccessRequestApproved {
		t.Fatalf("expected approved, got %s", approved.Status)
	}
	if approved.ApproverID == nil || *approved.ApproverID != approverID {
		t.Fatal("approver mismatch")
	}
	if approved.ResolvedAt == nil {
		t.Fatal("expected non-nil resolved_at")
	}
}

// 5. TestApproveAccessRequest_SelfApprovalBlocked
func TestApproveAccessRequest_SelfApprovalBlocked(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()
	requesterID := uuid.New()

	req, _ := svc.CreateAccessRequest(ctx, uuid.New(), requesterID,
		domain.ResourceTypeRole, "admin", "self approve")

	_, err := svc.ApproveAccessRequest(ctx, req.ID, requesterID)
	if err == nil {
		t.Fatal("expected error for self-approval")
	}
	ge, ok := errors.AsGGIDError(err)
	if !ok || ge.Code != errors.ErrPermissionDenied {
		t.Fatalf("expected permission_denied, got %v", err)
	}
}

// 6. TestApproveAccessRequest_AlreadyResolved
func TestApproveAccessRequest_AlreadyResolved(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()
	requesterID := uuid.New()
	approverID := uuid.New()

	req, _ := svc.CreateAccessRequest(ctx, uuid.New(), requesterID,
		domain.ResourceTypeRole, "r1", "test")

	_, _ = svc.ApproveAccessRequest(ctx, req.ID, approverID)

	// Try to approve again
	_, err := svc.ApproveAccessRequest(ctx, req.ID, approverID)
	if err == nil {
		t.Fatal("expected error for double approval")
	}
	ge, ok := errors.AsGGIDError(err)
	if !ok || ge.Code != errors.ErrFailedPrecondition {
		t.Fatalf("expected failed_precondition, got %v", err)
	}
}

// 7. TestDenyAccessRequest_Success
func TestDenyAccessRequest_Success(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()
	requesterID := uuid.New()
	approverID := uuid.New()

	req, _ := svc.CreateAccessRequest(ctx, uuid.New(), requesterID,
		domain.ResourceTypeGroup, "group-dev", "need access")

	denied, err := svc.DenyAccessRequest(ctx, req.ID, approverID, "insufficient justification")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if denied.Status != domain.AccessRequestDenied {
		t.Fatalf("expected denied, got %s", denied.Status)
	}
	if denied.DenialReason != "insufficient justification" {
		t.Fatalf("expected denial reason, got %s", denied.DenialReason)
	}
	if denied.ApproverID == nil || *denied.ApproverID != approverID {
		t.Fatal("approver mismatch")
	}
}

// 8. TestListPendingRequests
func TestListPendingRequests(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()
	tenantID := uuid.New()

	// Create 3 requests
	_, _ = svc.CreateAccessRequest(ctx, tenantID, uuid.New(),
		domain.ResourceTypeRole, "r1", "")
	_, _ = svc.CreateAccessRequest(ctx, tenantID, uuid.New(),
		domain.ResourceTypeRole, "r2", "")
	req3, _ := svc.CreateAccessRequest(ctx, tenantID, uuid.New(),
		domain.ResourceTypeRole, "r3", "")

	// Approve one
	_, _ = svc.ApproveAccessRequest(ctx, req3.ID, uuid.New())

	// Should have 2 pending
	pending, err := svc.ListPendingRequests(ctx, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}

	// Other tenant should have 0
	other, _ := svc.ListPendingRequests(ctx, uuid.New())
	if len(other) != 0 {
		t.Fatalf("expected 0 for other tenant, got %d", len(other))
	}
}

// 9. TestListUserRequests
func TestListUserRequests(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	// User creates 2 requests
	_, _ = svc.CreateAccessRequest(ctx, tenantID, userID,
		domain.ResourceTypeRole, "r1", "")
	_, _ = svc.CreateAccessRequest(ctx, tenantID, userID,
		domain.ResourceTypeRole, "r2", "")

	// Another user creates 1
	_, _ = svc.CreateAccessRequest(ctx, tenantID, uuid.New(),
		domain.ResourceTypeRole, "r3", "")

	reqs, err := svc.ListUserRequests(ctx, tenantID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests for user, got %d", len(reqs))
	}
}

// 10. TestCheckExpiredRequests
func TestCheckExpiredRequests(t *testing.T) {
	store := NewMemoryAccessRequestStore()
	svc := NewAccessRequestService(store)
	ctx := context.Background()

	// Create an already-expired request by manipulating the store directly
	tenantID := uuid.New()
	expiredReq := &domain.AccessRequest{
		ID:           uuid.New(),
		TenantID:     tenantID,
		RequesterID:  uuid.New(),
		ResourceType: domain.ResourceTypeRole,
		ResourceID:   "expired-role",
		Status:       domain.AccessRequestPending,
		CreatedAt:    time.Now().Add(-8 * 24 * time.Hour),
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // expired 1 hour ago
	}
	_ = store.Create(ctx, expiredReq)

	// Create a valid pending request
	_, _ = svc.CreateAccessRequest(ctx, tenantID, uuid.New(),
		domain.ResourceTypeRole, "valid-role", "")

	count, err := svc.CheckExpiredRequests(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 expired, got %d", count)
	}

	// Verify the expired request status changed
	req, _ := store.GetByID(ctx, expiredReq.ID)
	if req.Status != domain.AccessRequestExpired {
		t.Fatalf("expected expired status, got %s", req.Status)
	}
	if req.ResolvedAt == nil {
		t.Fatal("expected non-nil resolved_at")
	}
}

// 11. TestApproveAccessRequest_NotFound
func TestApproveAccessRequest_NotFound(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()

	_, err := svc.ApproveAccessRequest(ctx, uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent request")
	}
	ge, ok := errors.AsGGIDError(err)
	if !ok || ge.Code != errors.ErrNotFound {
		t.Fatalf("expected not_found, got %v", err)
	}
}

// 12. TestDenyAccessRequest_SelfDenialBlocked
func TestDenyAccessRequest_SelfDenialBlocked(t *testing.T) {
	svc := newTestAccessRequestService()
	ctx := context.Background()
	requesterID := uuid.New()

	req, _ := svc.CreateAccessRequest(ctx, uuid.New(), requesterID,
		domain.ResourceTypeRole, "r1", "test")

	_, err := svc.DenyAccessRequest(ctx, req.ID, requesterID, "self deny")
	if err == nil {
		t.Fatal("expected error for self-denial")
	}
}

// 13. TestAccessRequest_IsValid
func TestAccessRequest_IsValid(t *testing.T) {
	// Valid request
	validReq := &domain.AccessRequest{
		TenantID:     uuid.New(),
		RequesterID:  uuid.New(),
		ResourceType: domain.ResourceTypePermission,
		ResourceID:   "read:users",
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	if !validReq.IsValid() {
		t.Fatal("expected valid request")
	}

	// Nil tenant
	invalidReq := &domain.AccessRequest{
		RequesterID:  uuid.New(),
		ResourceType: domain.ResourceTypeRole,
		ResourceID:   "r1",
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	if invalidReq.IsValid() {
		t.Fatal("expected invalid for nil tenant")
	}

	// Invalid resource type
	invalidReq2 := &domain.AccessRequest{
		TenantID:     uuid.New(),
		RequesterID:  uuid.New(),
		ResourceType: "bogus",
		ResourceID:   "r1",
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	if invalidReq2.IsValid() {
		t.Fatal("expected invalid for bogus resource type")
	}
}

// 14. TestAccessRequest_IsExpired
func TestAccessRequest_IsExpired(t *testing.T) {
	// Not expired
	req := &domain.AccessRequest{
		Status:    domain.AccessRequestPending,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if req.IsExpired() {
		t.Fatal("expected not expired")
	}

	// Expired
	req2 := &domain.AccessRequest{
		Status:    domain.AccessRequestPending,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if !req2.IsExpired() {
		t.Fatal("expected expired")
	}

	// Already approved — not expired (even if past ExpiresAt)
	req3 := &domain.AccessRequest{
		Status:    domain.AccessRequestApproved,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if req3.IsExpired() {
		t.Fatal("approved request should not be considered expired")
	}
}
