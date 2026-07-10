package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// --- Mocks ---

type mockRoleRepo struct {
	roles     map[uuid.UUID]*domain.Role
	createErr error
	getErr    error
	updateErr error
	deleteErr error
	grantErr  error
	revokeErr error
}

func (m *mockRoleRepo) Create(_ context.Context, r *domain.Role) error {
	if m.createErr != nil { return m.createErr }
	if m.roles == nil { m.roles = map[uuid.UUID]*domain.Role{} }
	r.ID = uuid.New()
	r.CreatedAt = time.Now()
	m.roles[r.ID] = r
	return nil
}
func (m *mockRoleRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Role, error) {
	if m.getErr != nil { return nil, m.getErr }
	if r, ok := m.roles[id]; ok { return r, nil }
	return nil, errors.NotFound("role", id.String())
}
func (m *mockRoleRepo) ListByTenant(_ context.Context, tid uuid.UUID, limit, offset int) ([]*domain.Role, error) {
	var res []*domain.Role
	n := 0
	for _, r := range m.roles {
		if r.TenantID == tid {
			if n >= offset && len(res) < limit { res = append(res, r) }
			n++
		}
	}
	return res, nil
}
func (m *mockRoleRepo) Update(_ context.Context, r *domain.Role) error {
	if m.updateErr != nil { return m.updateErr }
	m.roles[r.ID] = r
	return nil
}
func (m *mockRoleRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil { return m.deleteErr }
	if _, ok := m.roles[id]; !ok { return errors.NotFound("role", id.String()) }
	delete(m.roles, id)
	return nil
}
func (m *mockRoleRepo) GrantPermissions(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ map[string]any) error {
	return m.grantErr
}
func (m *mockRoleRepo) RevokePermissions(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return m.revokeErr
}

type mockPermRepo struct {
	perms     map[uuid.UUID]*domain.Permission
	createErr error
}
func (m *mockPermRepo) Create(_ context.Context, p *domain.Permission) error {
	if m.createErr != nil { return m.createErr }
	if m.perms == nil { m.perms = map[uuid.UUID]*domain.Permission{} }
	p.ID = uuid.New()
	m.perms[p.ID] = p
	return nil
}
func (m *mockPermRepo) ListByTenant(_ context.Context, tid uuid.UUID, limit, offset int) ([]*domain.Permission, error) {
	var res []*domain.Permission
	n := 0
	for _, p := range m.perms {
		if p.TenantID == tid {
			if n >= offset && len(res) < limit { res = append(res, p) }
			n++
		}
	}
	return res, nil
}

type mockUserRoleRepo struct {
	assignments []*domain.UserRole
	assignErr   error
	revokeErr   error
}
func (m *mockUserRoleRepo) Assign(_ context.Context, ur *domain.UserRole) error {
	if m.assignErr != nil { return m.assignErr }
	m.assignments = append(m.assignments, ur)
	return nil
}
func (m *mockUserRoleRepo) Revoke(_ context.Context, _, _ uuid.UUID, _ domain.ScopeType, _ uuid.UUID) error {
	return m.revokeErr
}
func (m *mockUserRoleRepo) ListByUser(_ context.Context, uid uuid.UUID) ([]*domain.UserRole, error) {
	var res []*domain.UserRole
	for _, a := range m.assignments {
		if a.UserID == uid { res = append(res, a) }
	}
	return res, nil
}

type mockPolicyRepo struct {
	policies  map[uuid.UUID]*domain.Policy
	createErr error
	getErr    error
	deleteErr error
	attachErr error
	detachErr error
}
func (m *mockPolicyRepo) Create(_ context.Context, p *domain.Policy) error {
	if m.createErr != nil { return m.createErr }
	if m.policies == nil { m.policies = map[uuid.UUID]*domain.Policy{} }
	p.ID = uuid.New()
	m.policies[p.ID] = p
	return nil
}
func (m *mockPolicyRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Policy, error) {
	if m.getErr != nil { return nil, m.getErr }
	if p, ok := m.policies[id]; ok { return p, nil }
	return nil, errors.NotFound("policy", id.String())
}
func (m *mockPolicyRepo) ListByTenant(_ context.Context, tid uuid.UUID, limit, offset int) ([]*domain.Policy, error) {
	var res []*domain.Policy
	n := 0
	for _, p := range m.policies {
		if p.TenantID == tid {
			if n >= offset && len(res) < limit { res = append(res, p) }
			n++
		}
	}
	return res, nil
}
func (m *mockPolicyRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil { return m.deleteErr }
	delete(m.policies, id)
	return nil
}
func (m *mockPolicyRepo) AttachPolicy(_ context.Context, _ *domain.PolicyAttachment) error { return m.attachErr }
func (m *mockPolicyRepo) DetachPolicy(_ context.Context, _ uuid.UUID, _ domain.PrincipalType, _ uuid.UUID) error { return m.detachErr }

// ===== RoleService tests =====

func TestRoleService_CreateRole(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	pid := uuid.New()
	r, err := svc.CreateRole(context.Background(), uuid.New(), "admin", "Admin", "desc", &pid)
	if err != nil { t.Fatal(err) }
	if r.ID == uuid.Nil { t.Error("nil ID") }
	if *r.ParentRoleID != pid { t.Error("parent mismatch") }
}

func TestRoleService_CreateRole_NoParent(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	r, err := svc.CreateRole(context.Background(), uuid.New(), "v", "V", "", nil)
	if err != nil { t.Fatal(err) }
	if r.ParentRoleID != nil { t.Error("expected nil parent") }
}

func TestRoleService_CreateRole_Error(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{createErr: errors.New(errors.ErrInternal, "x")}, &mockPermRepo{}, &mockUserRoleRepo{})
	_, err := svc.CreateRole(context.Background(), uuid.New(), "k", "n", "", nil)
	if err == nil { t.Fatal("expected error") }
}

func TestRoleService_GetRole(t *testing.T) {
	repo := &mockRoleRepo{}
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	c, _ := svc.CreateRole(context.Background(), uuid.New(), "editor", "Editor", "", nil)
	g, err := svc.GetRole(context.Background(), c.ID)
	if err != nil { t.Fatal(err) }
	if g.Key != "editor" { t.Error("key mismatch") }
}

func TestRoleService_GetRole_NotFound(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	_, err := svc.GetRole(context.Background(), uuid.New())
	if err == nil { t.Fatal("expected error") }
}

func TestRoleService_ListRoles(t *testing.T) {
	repo := &mockRoleRepo{}
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	tid := uuid.New()
	for i := 0; i < 3; i++ { svc.CreateRole(context.Background(), tid, "r", "R", "", nil) }
	roles, err := svc.ListRoles(context.Background(), tid, 1, 10)
	if err != nil { t.Fatal(err) }
	if len(roles) != 3 { t.Errorf("got %d", len(roles)) }
}

func TestRoleService_ListRoles_DefaultPageSize(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	r, err := svc.ListRoles(context.Background(), uuid.New(), 1, 0)
	if err != nil { t.Fatal(err) }
	if len(r) > 50 { t.Error("should cap at 50") }
}

func TestRoleService_UpdateRole(t *testing.T) {
	repo := &mockRoleRepo{}
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	c, _ := svc.CreateRole(context.Background(), uuid.New(), "old", "Old", "", nil)
	n := "New"
	u, err := svc.UpdateRole(context.Background(), c.ID, &n, nil, nil)
	if err != nil { t.Fatal(err) }
	if u.Name != "New" { t.Error("name not updated") }
}

func TestRoleService_UpdateRole_NotFound(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	n := "x"
	_, err := svc.UpdateRole(context.Background(), uuid.New(), &n, nil, nil)
	if err == nil { t.Fatal("expected error") }
}

func TestRoleService_DeleteRole(t *testing.T) {
	repo := &mockRoleRepo{}
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	c, _ := svc.CreateRole(context.Background(), uuid.New(), "t", "T", "", nil)
	if err := svc.DeleteRole(context.Background(), c.ID); err != nil { t.Fatal(err) }
}

func TestRoleService_DeleteRole_SystemRole(t *testing.T) {
	repo := &mockRoleRepo{roles: map[uuid.UUID]*domain.Role{}}
	sr := &domain.Role{ID: uuid.New(), SystemRole: true}
	repo.roles[sr.ID] = sr
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	if err := svc.DeleteRole(context.Background(), sr.ID); err == nil { t.Fatal("expected error") }
}

func TestRoleService_DeleteRole_NotFound(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	if err := svc.DeleteRole(context.Background(), uuid.New()); err == nil { t.Fatal("expected error") }
}

func TestRoleService_AssignRole(t *testing.T) {
	repo := &mockRoleRepo{}
	ur := &mockUserRoleRepo{}
	svc := NewRoleService(repo, &mockPermRepo{}, ur)
	r, _ := svc.CreateRole(context.Background(), uuid.New(), "e", "E", "", nil)
	err := svc.AssignRole(context.Background(), uuid.New(), r.ID, domain.ScopeOrganization, uuid.New(), uuid.New(), nil)
	if err != nil { t.Fatal(err) }
}

func TestRoleService_AssignRole_RoleNotFound(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	err := svc.AssignRole(context.Background(), uuid.New(), uuid.New(), domain.ScopeGlobal, uuid.Nil, uuid.New(), nil)
	if err == nil { t.Fatal("expected error") }
}

func TestRoleService_RevokeRole(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	if err := svc.RevokeRole(context.Background(), uuid.New(), uuid.New(), domain.ScopeOrganization, uuid.New()); err != nil { t.Fatal(err) }
}

func TestRoleService_ListUserRoles(t *testing.T) {
	repo := &mockRoleRepo{}
	ur := &mockUserRoleRepo{}
	svc := NewRoleService(repo, &mockPermRepo{}, ur)
	r, _ := svc.CreateRole(context.Background(), uuid.New(), "r", "R", "", nil)
	uid := uuid.New()
	svc.AssignRole(context.Background(), uid, r.ID, domain.ScopeOrganization, uuid.New(), uuid.New(), nil)
	list, err := svc.ListUserRoles(context.Background(), uid)
	if err != nil { t.Fatal(err) }
	if len(list) != 1 { t.Errorf("got %d", len(list)) }
}

func TestRoleService_CreatePermission(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	p, err := svc.CreatePermission(context.Background(), &domain.Permission{TenantID: uuid.New(), Key: "k"})
	if err != nil { t.Fatal(err) }
	if p.ID == uuid.Nil { t.Error("nil ID") }
}

func TestRoleService_CreatePermission_Error(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{createErr: errors.New(errors.ErrInternal, "x")}, &mockUserRoleRepo{})
	_, err := svc.CreatePermission(context.Background(), &domain.Permission{Key: "k"})
	if err == nil { t.Fatal("expected error") }
}

func TestRoleService_ListPermissions(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	tid := uuid.New()
	for i := 0; i < 3; i++ { svc.CreatePermission(context.Background(), &domain.Permission{TenantID: tid, Key: "k"}) }
	p, err := svc.ListPermissions(context.Background(), tid, 1, 10)
	if err != nil { t.Fatal(err) }
	if len(p) != 3 { t.Errorf("got %d", len(p)) }
}

func TestRoleService_GrantPermissions(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	if err := svc.GrantPermissionsToRole(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}); err != nil { t.Fatal(err) }
}

func TestRoleService_RevokePermissions(t *testing.T) {
	repo := &mockRoleRepo{revokeErr: errors.New(errors.ErrInternal, "fail")}
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	if err := svc.RevokePermissionsFromRole(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}); err == nil { t.Fatal("expected error") }
}

// ===== PolicyService tests =====

func TestPolicyService_CreatePolicy(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	p, err := svc.CreatePolicy(context.Background(), &domain.Policy{TenantID: uuid.New(), Effect: domain.EffectAllow})
	if err != nil { t.Fatal(err) }
	if p.ID == uuid.Nil { t.Error("nil ID") }
}

func TestPolicyService_CreatePolicy_DenyPriority(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	p, err := svc.CreatePolicy(context.Background(), &domain.Policy{TenantID: uuid.New(), Effect: domain.EffectDeny})
	if err != nil { t.Fatal(err) }
	if p.Priority != 100 { t.Errorf("got %d", p.Priority) }
}

func TestPolicyService_CreatePolicy_InvalidEffect(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	_, err := svc.CreatePolicy(context.Background(), &domain.Policy{Effect: "bad"})
	if err == nil { t.Fatal("expected error") }
}

func TestPolicyService_CreatePolicy_Error(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{createErr: errors.New(errors.ErrInternal, "x")})
	_, err := svc.CreatePolicy(context.Background(), &domain.Policy{Effect: domain.EffectAllow})
	if err == nil { t.Fatal("expected error") }
}

func TestPolicyService_GetPolicy(t *testing.T) {
	repo := &mockPolicyRepo{}
	svc := NewPolicyService(repo)
	c, _ := svc.CreatePolicy(context.Background(), &domain.Policy{Effect: domain.EffectAllow})
	_, err := svc.GetPolicy(context.Background(), c.ID)
	if err != nil { t.Fatal(err) }
}

func TestPolicyService_GetPolicy_NotFound(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	_, err := svc.GetPolicy(context.Background(), uuid.New())
	if err == nil { t.Fatal("expected error") }
}

func TestPolicyService_ListPolicies(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	tid := uuid.New()
	for i := 0; i < 3; i++ { svc.CreatePolicy(context.Background(), &domain.Policy{TenantID: tid, Effect: domain.EffectAllow}) }
	p, err := svc.ListPolicies(context.Background(), tid, 1, 10)
	if err != nil { t.Fatal(err) }
	if len(p) != 3 { t.Errorf("got %d", len(p)) }
}

func TestPolicyService_ListPolicies_DefaultPageSize(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	p, err := svc.ListPolicies(context.Background(), uuid.New(), 1, 0)
	if err != nil { t.Fatal(err) }
	if len(p) > 50 { t.Error("should cap at 50") }
}

func TestPolicyService_DeletePolicy(t *testing.T) {
	repo := &mockPolicyRepo{}
	svc := NewPolicyService(repo)
	c, _ := svc.CreatePolicy(context.Background(), &domain.Policy{Effect: domain.EffectAllow})
	if err := svc.DeletePolicy(context.Background(), c.ID); err != nil { t.Fatal(err) }
}

func TestPolicyService_AttachPolicy(t *testing.T) {
	repo := &mockPolicyRepo{}
	svc := NewPolicyService(repo)
	p, _ := svc.CreatePolicy(context.Background(), &domain.Policy{Effect: domain.EffectAllow})
	if err := svc.AttachPolicy(context.Background(), p.ID, domain.PrincipalUser, uuid.New()); err != nil { t.Fatal(err) }
}

func TestPolicyService_AttachPolicy_NotFound(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	if err := svc.AttachPolicy(context.Background(), uuid.New(), domain.PrincipalUser, uuid.New()); err == nil { t.Fatal("expected error") }
}

func TestPolicyService_DetachPolicy(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	if err := svc.DetachPolicy(context.Background(), uuid.New(), domain.PrincipalRole, uuid.New()); err != nil { t.Fatal(err) }
}

func TestPolicyService_DetachPolicy_Error(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{detachErr: errors.New(errors.ErrInternal, "x")})
	if err := svc.DetachPolicy(context.Background(), uuid.New(), domain.PrincipalRole, uuid.New()); err == nil { t.Fatal("expected error") }
}
