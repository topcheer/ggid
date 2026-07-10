package service

import (
	"context"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// PolicyRepo provides ABAC policy persistence operations.
type PolicyRepo interface {
	Create(ctx context.Context, p *domain.Policy) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Policy, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Policy, error)
	Delete(ctx context.Context, id uuid.UUID) error
	AttachPolicy(ctx context.Context, att *domain.PolicyAttachment) error
	DetachPolicy(ctx context.Context, policyID uuid.UUID, pt domain.PrincipalType, principalID uuid.UUID) error
}

// PolicyService handles ABAC policy CRUD and attachment.
type PolicyService struct {
	policyRepo PolicyRepo
}

// NewPolicyService creates a new PolicyService.
func NewPolicyService(policyRepo PolicyRepo) *PolicyService {
	return &PolicyService{policyRepo: policyRepo}
}

// CreatePolicy creates a new ABAC policy.
func (s *PolicyService) CreatePolicy(ctx context.Context, p *domain.Policy) (*domain.Policy, error) {
	if p.Effect != domain.EffectAllow && p.Effect != domain.EffectDeny {
		return nil, errors.InvalidArgument("effect must be 'allow' or 'deny'")
	}
	if p.Priority == 0 {
		if p.Effect == domain.EffectDeny {
			p.Priority = 100 // deny defaults to higher priority
		}
	}
	if err := s.policyRepo.Create(ctx, p); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "create policy", err)
	}
	return p, nil
}

// GetPolicy retrieves a policy by ID.
func (s *PolicyService) GetPolicy(ctx context.Context, id uuid.UUID) (*domain.Policy, error) {
	return s.policyRepo.GetByID(ctx, id)
}

// ListPolicies lists policies for a tenant.
func (s *PolicyService) ListPolicies(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*domain.Policy, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return s.policyRepo.ListByTenant(ctx, tenantID, pageSize, offset)
}

// DeletePolicy deletes a policy and its attachments.
func (s *PolicyService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	return s.policyRepo.Delete(ctx, id)
}

// AttachPolicy attaches a policy to a principal (user, role, or group).
func (s *PolicyService) AttachPolicy(ctx context.Context, policyID uuid.UUID, principalType domain.PrincipalType, principalID uuid.UUID) error {
	// Verify policy exists.
	if _, err := s.policyRepo.GetByID(ctx, policyID); err != nil {
		return err
	}
	return s.policyRepo.AttachPolicy(ctx, &domain.PolicyAttachment{
		PolicyID:      policyID,
		PrincipalType: principalType,
		PrincipalID:   principalID,
	})
}

// DetachPolicy removes a policy attachment.
func (s *PolicyService) DetachPolicy(ctx context.Context, policyID uuid.UUID, principalType domain.PrincipalType, principalID uuid.UUID) error {
	return s.policyRepo.DetachPolicy(ctx, policyID, principalType, principalID)
}
