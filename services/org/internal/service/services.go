// Package service implements the Org Service business logic.
package service

import (
	"context"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/ggid/ggid/services/org/internal/repository"
	"github.com/google/uuid"
)

// TenantService handles tenant CRUD operations.
type TenantService struct {
	repo *repository.TenantRepository
}

func NewTenantService(repo *repository.TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

// Create creates a new tenant.
func (s *TenantService) Create(ctx context.Context, t *domain.Tenant) (*domain.Tenant, error) {
	if t.Slug == "" {
		return nil, errors.InvalidArgument("slug is required")
	}
	if t.Plan == "" {
		t.Plan = domain.PlanFree
	}
	if t.Status == "" {
		t.Status = domain.TenantActive
	}
	if t.MaxUsers == 0 {
		t.MaxUsers = 50
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "create tenant", err)
	}
	return t, nil
}

// Get retrieves a tenant by ID.
func (s *TenantService) Get(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySlug retrieves a tenant by slug.
func (s *TenantService) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return s.repo.GetBySlug(ctx, slug)
}

// Update modifies a tenant.
func (s *TenantService) Update(ctx context.Context, t *domain.Tenant) (*domain.Tenant, error) {
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete soft-deletes a tenant.
func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// --- Org Service ---

// OrgService handles organization tree operations.
type OrgService struct {
	repo *repository.OrgRepository
}

func NewOrgService(repo *repository.OrgRepository) *OrgService {
	return &OrgService{repo: repo}
}

func (s *OrgService) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	if org.Name == "" {
		return nil, errors.InvalidArgument("organization name is required")
	}
	if err := s.repo.Create(ctx, org); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "create organization", err)
	}
	return org, nil
}

func (s *OrgService) Get(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *OrgService) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*domain.Organization, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByTenant(ctx, tenantID, pageSize, offset)
}

func (s *OrgService) GetSubTree(ctx context.Context, tenantID, rootID uuid.UUID) ([]*domain.Organization, error) {
	return s.repo.GetSubTree(ctx, tenantID, rootID)
}

func (s *OrgService) Update(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	if err := s.repo.Update(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *OrgService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// --- Dept Service ---

// DeptService handles department operations.
type DeptService struct {
	repo *repository.DeptRepository
}

func NewDeptService(repo *repository.DeptRepository) *DeptService {
	return &DeptService{repo: repo}
}

func (s *DeptService) Create(ctx context.Context, dept *domain.Department) (*domain.Department, error) {
	if dept.Name == "" {
		return nil, errors.InvalidArgument("department name is required")
	}
	if err := s.repo.Create(ctx, dept); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "create department", err)
	}
	return dept, nil
}

func (s *DeptService) Get(ctx context.Context, id uuid.UUID) (*domain.Department, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DeptService) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Department, error) {
	return s.repo.ListByOrg(ctx, orgID)
}

func (s *DeptService) Update(ctx context.Context, dept *domain.Department) (*domain.Department, error) {
	if err := s.repo.Update(ctx, dept); err != nil {
		return nil, err
	}
	return dept, nil
}

func (s *DeptService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// --- Team Service ---

// TeamService handles team operations.
type TeamService struct {
	repo *repository.TeamRepository
}

func NewTeamService(repo *repository.TeamRepository) *TeamService {
	return &TeamService{repo: repo}
}

func (s *TeamService) Create(ctx context.Context, team *domain.Team) (*domain.Team, error) {
	if team.Name == "" {
		return nil, errors.InvalidArgument("team name is required")
	}
	if err := s.repo.Create(ctx, team); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "create team", err)
	}
	return team, nil
}

func (s *TeamService) Get(ctx context.Context, id uuid.UUID) (*domain.Team, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TeamService) List(ctx context.Context, orgID uuid.UUID, page, pageSize int) ([]*domain.Team, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByOrg(ctx, orgID, pageSize, offset)
}

func (s *TeamService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// --- Membership Service ---

// MembershipService handles member invitation and management.
type MembershipService struct {
	repo *repository.MembershipRepository
}

func NewMembershipService(repo *repository.MembershipRepository) *MembershipService {
	return &MembershipService{repo: repo}
}

// Invite creates a new membership with 'invited' status.
func (s *MembershipService) Invite(ctx context.Context, m *domain.Membership) (*domain.Membership, error) {
	if m.Status == "" {
		m.Status = domain.MembershipInvited
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "invite member", err)
	}
	return m, nil
}

// AcceptInvitation activates an invited membership.
func (s *MembershipService) AcceptInvitation(ctx context.Context, id uuid.UUID) error {
	return s.repo.Activate(ctx, id)
}

// Remove sets membership status to removed.
func (s *MembershipService) Remove(ctx context.Context, id uuid.UUID) error {
	return s.repo.Remove(ctx, id)
}

// List returns memberships matching the filter.
func (s *MembershipService) List(ctx context.Context, filter repository.ListMembersFilter, page, pageSize int) ([]*domain.Membership, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, filter, pageSize, offset)
}
