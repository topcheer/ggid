package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// DefaultExpiryDuration is how long an access request stays pending before auto-expiring.
const DefaultExpiryDuration = 7 * 24 * time.Hour

// AccessRequestStore is the persistence interface for access requests.
// Implementations may use in-memory maps or SQL databases.
type AccessRequestStore interface {
	Create(ctx context.Context, req *domain.AccessRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AccessRequest, error)
	Update(ctx context.Context, req *domain.AccessRequest) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, status domain.AccessRequestStatus) ([]*domain.AccessRequest, error)
	ListByRequester(ctx context.Context, tenantID, requesterID uuid.UUID) ([]*domain.AccessRequest, error)
	ListExpired(ctx context.Context) ([]*domain.AccessRequest, error)
}

// AccessRequestService implements the IGA approval workflow.
type AccessRequestService struct {
	store AccessRequestStore
	mu    sync.Mutex //nolint:unused // for future concurrency support
}

// NewAccessRequestService creates a new IGA workflow service.
func NewAccessRequestService(store AccessRequestStore) *AccessRequestService {
	return &AccessRequestService{store: store}
}

// CreateAccessRequest creates a new pending access request with a 7-day expiry.
func (s *AccessRequestService) CreateAccessRequest(
	ctx context.Context,
	tenantID, requesterID uuid.UUID,
	resourceType domain.ResourceType,
	resourceID, reason string,
) (*domain.AccessRequest, error) {
	if tenantID == uuid.Nil {
		return nil, errors.InvalidArgument("tenant_id is required")
	}
	if requesterID == uuid.Nil {
		return nil, errors.InvalidArgument("requester_id is required")
	}
	if resourceID == "" {
		return nil, errors.InvalidArgument("resource_id is required")
	}

	now := time.Now()
	req := &domain.AccessRequest{
		ID:           uuid.New(),
		TenantID:     tenantID,
		RequesterID:  requesterID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Reason:       reason,
		Status:       domain.AccessRequestPending,
		CreatedAt:    now,
		ExpiresAt:    now.Add(DefaultExpiryDuration),
	}

	if !req.IsValid() {
		return nil, errors.InvalidArgument("invalid access request")
	}

	if err := s.store.Create(ctx, req); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "failed to create access request", err)
	}
	return req, nil
}

// ApproveAccessRequest approves a pending request.
// Returns error if the request is not found, already resolved, or expired.
func (s *AccessRequestService) ApproveAccessRequest(
	ctx context.Context,
	requestID, approverID uuid.UUID,
) (*domain.AccessRequest, error) {
	req, err := s.store.GetByID(ctx, requestID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrNotFound, "access request not found", err)
	}

	if req.Status != domain.AccessRequestPending {
		return nil, errors.New(errors.ErrFailedPrecondition,
			fmt.Sprintf("request is already %s", req.Status))
	}

	if req.IsExpired() {
		req.Status = domain.AccessRequestExpired
		_ = s.store.Update(ctx, req)
		return nil, errors.New(errors.ErrFailedPrecondition, "request has expired")
	}

	if approverID == uuid.Nil {
		return nil, errors.InvalidArgument("approver_id is required")
	}

	// Prevent self-approval (governance best practice).
	if req.RequesterID == approverID {
		return nil, errors.New(errors.ErrPermissionDenied, "requester cannot approve their own request")
	}

	now := time.Now()
	req.Status = domain.AccessRequestApproved
	req.ApproverID = &approverID
	req.ResolvedAt = &now

	if err := s.store.Update(ctx, req); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "failed to approve request", err)
	}
	return req, nil
}

// DenyAccessRequest denies a pending request with an optional reason.
func (s *AccessRequestService) DenyAccessRequest(
	ctx context.Context,
	requestID, approverID uuid.UUID,
	denialReason string,
) (*domain.AccessRequest, error) {
	req, err := s.store.GetByID(ctx, requestID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrNotFound, "access request not found", err)
	}

	if req.Status != domain.AccessRequestPending {
		return nil, errors.New(errors.ErrFailedPrecondition,
			fmt.Sprintf("request is already %s", req.Status))
	}

	if req.IsExpired() {
		req.Status = domain.AccessRequestExpired
		_ = s.store.Update(ctx, req)
		return nil, errors.New(errors.ErrFailedPrecondition, "request has expired")
	}

	if approverID == uuid.Nil {
		return nil, errors.InvalidArgument("approver_id is required")
	}

	if req.RequesterID == approverID {
		return nil, errors.New(errors.ErrPermissionDenied, "requester cannot deny their own request")
	}

	now := time.Now()
	req.Status = domain.AccessRequestDenied
	req.ApproverID = &approverID
	req.DenialReason = denialReason
	req.ResolvedAt = &now

	if err := s.store.Update(ctx, req); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "failed to deny request", err)
	}
	return req, nil
}

// ListPendingRequests returns all pending requests for a tenant.
func (s *AccessRequestService) ListPendingRequests(
	ctx context.Context,
	tenantID uuid.UUID,
) ([]*domain.AccessRequest, error) {
	return s.store.ListByTenant(ctx, tenantID, domain.AccessRequestPending)
}

// ListRequests returns requests for a tenant, optionally filtered by status.
// If status is empty, returns all requests.
func (s *AccessRequestService) ListRequests(
	ctx context.Context,
	tenantID uuid.UUID,
	status domain.AccessRequestStatus,
) ([]*domain.AccessRequest, error) {
	return s.store.ListByTenant(ctx, tenantID, status)
}

// ListUserRequests returns all requests submitted by a specific user.
func (s *AccessRequestService) ListUserRequests(
	ctx context.Context,
	tenantID, userID uuid.UUID,
) ([]*domain.AccessRequest, error) {
	return s.store.ListByRequester(ctx, tenantID, userID)
}

// CheckExpiredRequests scans for expired pending requests and marks them as expired.
// Returns the number of requests that were expired.
func (s *AccessRequestService) CheckExpiredRequests(ctx context.Context) (int, error) {
	expired, err := s.store.ListExpired(ctx)
	if err != nil {
		return 0, errors.Wrap(errors.ErrInternal, "failed to list expired requests", err)
	}

	now := time.Now()
	for _, req := range expired {
		req.Status = domain.AccessRequestExpired
		req.ResolvedAt = &now
		if err := s.store.Update(ctx, req); err != nil {
			return 0, errors.Wrap(errors.ErrInternal, "failed to expire request", err)
		}
	}
	return len(expired), nil
}

// --- In-memory store for testing/development ---

// MemoryAccessRequestStore is a thread-safe in-memory implementation.
type MemoryAccessRequestStore struct {
	mu      sync.RWMutex
	requests map[uuid.UUID]*domain.AccessRequest
}

// NewMemoryAccessRequestStore creates a new in-memory store.
func NewMemoryAccessRequestStore() *MemoryAccessRequestStore {
	return &MemoryAccessRequestStore{
		requests: make(map[uuid.UUID]*domain.AccessRequest),
	}
}

func (m *MemoryAccessRequestStore) Create(_ context.Context, req *domain.AccessRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[req.ID] = req
	return nil
}

func (m *MemoryAccessRequestStore) GetByID(_ context.Context, id uuid.UUID) (*domain.AccessRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	req, ok := m.requests[id]
	if !ok {
		return nil, fmt.Errorf("not found: %s", id)
	}
	return req, nil
}

func (m *MemoryAccessRequestStore) Update(_ context.Context, req *domain.AccessRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[req.ID] = req
	return nil
}

func (m *MemoryAccessRequestStore) ListByTenant(_ context.Context, tenantID uuid.UUID, status domain.AccessRequestStatus) ([]*domain.AccessRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.AccessRequest
	for _, req := range m.requests {
		if req.TenantID != tenantID {
			continue
		}
		if status == "" || req.Status == status {
			result = append(result, req)
		}
	}
	return result, nil
}

func (m *MemoryAccessRequestStore) ListByRequester(_ context.Context, tenantID, requesterID uuid.UUID) ([]*domain.AccessRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.AccessRequest
	for _, req := range m.requests {
		if req.TenantID == tenantID && req.RequesterID == requesterID {
			result = append(result, req)
		}
	}
	return result, nil
}

func (m *MemoryAccessRequestStore) ListExpired(_ context.Context) ([]*domain.AccessRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.AccessRequest
	for _, req := range m.requests {
		if req.IsExpired() {
			result = append(result, req)
		}
	}
	return result, nil
}
