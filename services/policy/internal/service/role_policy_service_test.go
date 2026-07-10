package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// --- Mock repos for RoleService and PolicyService ---

type mockRoleRepo struct {
	roles      map[uuid.UUID]*domain.Role
	createErr  error
	updateErr  error
	deleteErr  error
	grantErr   error
	revokeErr  error
	listResult []*domain.Role
	listErr    error
}

func (m *mockRoleRepo) Create(_ context.Context, role *domain.Role) error {
	if m.createErr != nil {
		return m.createErr
	}
	role.ID = uuid.New()
	if m.roles == nil {
		m.roles = make(map[uuid.UUID]*domain.Role)
	}
	m.roles[role.ID] = role
	return nil
}

func (m *mockRoleRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Role, error) {
	if r, ok := m.roles[id]; ok {
		return r, nil
	}
	return nil, &testErr{"role not found"}
}

func (m *mockRoleRepo) ListByTenant(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.Role, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.listResult != nil {
		return m.listResult, nil
	}
	var result []*domain.Role
	for _, r := range m.roles {
		result = append(result, r)
	}
	return result, nil
}

func (m *mockRoleRepo) Update(_ context.Context, role *domain.Role) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.roles[role.ID] = role
	return nil
}

func (m *mockRoleRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
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
	perms     []*domain.Permission
	createErr error
	listErr   error
}

func (m *mockPermRepo) Create(_ context.Context, perm *domain.Permission) error {
	if m.createErr != nil {
		return m.createErr
	}
	perm.ID = uuid.New()
	m.perms = append(m.perms, perm)
	return nil
}

func (m *mockPermRepo) ListByTenant(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.Permission, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.perms, nil
}

type mockUserRoleRepo struct {
	assignments map[uuid.UUID][]*domain.UserRole
	assignErr   error
	revokeErr   error
}

func (m *mockUserRoleRepo) Assign(_ context.Context, ur *domain.UserRole) error {
	if m.assignErr != nil {
		return m.assignErr
	}
	if m.assignments == nil {
		m.assignments = make(map[uuid.UUID][]*domain.UserRole)
	}
	m.assignments[ur.UserID] = append(m.assignments[ur.UserID], ur)
	return nil
}

func (m *mockUserRoleRepo) Revoke(_ context.Context, _, _ uuid.UUID, _ domain.ScopeType, _ uuid.UUID) error {
	return m.revokeErr
}

func (m *mockUserRoleRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*domain.UserRole, error) {
	return m.assignments[userID], nil
}

type mockPolicyRepo struct {
	policies   map[uuid.UUID]*domain.Policy
	createErr  error
	deleteErr  error
	attachErr  error
	detachErr  error
	listResult []*domain.Policy
	listErr    error
}

func (m *mockPolicyRepo) Create(_ context.Context, p *domain.Policy) error {
	if m.createErr != nil {
		return m.createErr
	}
	p.ID = uuid.New()
	if m.policies == nil {
		m.policies = make(map[uuid.UUID]*domain.Policy)
	}
	m.policies[p.ID] = p
	return nil
}

func (m *mockPolicyRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Policy, error) {
	if p, ok := m.policies[id]; ok {
		return p, nil
	}
	return nil, &testErr{"policy not found"}
}

func (m *mockPolicyRepo) ListByTenant(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.Policy, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.listResult != nil {
		return m.listResult, nil
	}
	var result []*domain.Policy
	for _, p := range m.policies {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockPolicyRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.policies, id)
	return nil
}

func (m *mockPolicyRepo) AttachPolicy(_ context.Context, _ *domain.PolicyAttachment) error {
	return m.attachErr
}

func (m *mockPolicyRepo) DetachPolicy(_ context.Context, _ uuid.UUID, _ domain.PrincipalType, _ uuid.UUID) error {
	return m.detachErr
}

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }

// --- RoleService Tests ---

func TestRoleService_CreateRole(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	role, err := svc.CreateRole(context.Background(), uuid.New(), "editor", "Editor", "Edit role", nil)
	if err != nil || role.Key != "editor" {
		t.Fatalf("CreateRole: err=%v key=%s", err, role.Key)
	}
}

func TestRoleService_CreateRole_Error(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{createErr: &testErr{"db"}}, &mockPermRepo{}, &mockUserRoleRepo{})
	_, err := svc.CreateRole(context.Background(), uuid.New(), "x", "X", "", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRoleService_GetRole(t *testing.T) {
	id := uuid.New()
	repo := &mockRoleRepo{roles: map[uuid.UUID]*domain.Role{id: {ID: id, Key: "admin"}}}
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	role, err := svc.GetRole(context.Background(), id)
	if err != nil || role.Key != "admin" {
		t.Fatalf("GetRole: err=%v", err)
	}
}

func TestRoleService_ListRoles(t *testing.T) {
	repo := &mockRoleRepo{listResult: []*domain.Role{{ID: uuid.New()}, {ID: uuid.New()}}}
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	roles, err := svc.ListRoles(context.Background(), uuid.New(), 1, 10)
	if err != nil || len(roles) != 2 {
		t.Fatalf("ListRoles: err=%v len=%d", err, len(roles))
	}
}

func TestRoleService_ListRoles_PageSizeClamp(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{listResult: []*domain.Role{}}, &mockPermRepo{}, &mockUserRoleRepo{})
	for _, ps := range []int{0, 999} {
		if _, err := svc.ListRoles(context.Background(), uuid.New(), 1, ps); err != nil {
			t.Errorf("pageSize=%d: %v", ps, err)
		}
	}
}

func TestRoleService_UpdateRole(t *testing.T) {
	id := uuid.New()
	repo := &mockRoleRepo{roles: map[uuid.UUID]*domain.Role{id: {ID: id, Name: "Dev"}}}
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	name := "Senior Dev"
	role, err := svc.UpdateRole(context.Background(), id, &name, nil, nil)
	if err != nil || role.Name != "Senior Dev" {
		t.Fatalf("UpdateRole: err=%v", err)
	}
}

func TestRoleService_UpdateRole_NotFound(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{roles: map[uuid.UUID]*domain.Role{}}, &mockPermRepo{}, &mockUserRoleRepo{})
	n := "x"
	_, err := svc.UpdateRole(context.Background(), uuid.New(), &n, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRoleService_DeleteRole(t *testing.T) {
	id := uuid.New()
	repo := &mockRoleRepo{roles: map[uuid.UUID]*domain.Role{id: {ID: id}}}
	svc := NewRoleService(repo, &mockPermRepo{}, &mockUserRoleRepo{})
	if err := svc.DeleteRole(context.Background(), id); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
}

func TestRoleService_AssignRevokeList(t *testing.T) {
	rid := uuid.New()
	roleRepo := &mockRoleRepo{roles: map[uuid.UUID]*domain.Role{rid: {ID: rid, Key: "dev"}}}
	urRepo := &mockUserRoleRepo{}
	svc := NewRoleService(roleRepo, &mockPermRepo{}, urRepo)
	uid := uuid.New()

	if err := svc.AssignRole(context.Background(), uid, rid, domain.ScopeGlobal, uuid.Nil, uuid.Nil, nil); err != nil {
		t.Fatalf("AssignRole: %v", err)
	}
	roles, _ := svc.ListUserRoles(context.Background(), uid)
	if len(roles) != 1 {
		t.Errorf("expected 1, got %d", len(roles))
	}
	if err := svc.RevokeRole(context.Background(), uid, rid, domain.ScopeGlobal, uuid.Nil); err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}
}

func TestRoleService_AssignRole_Error(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{assignErr: &testErr{"db"}})
	if err := svc.AssignRole(context.Background(), uuid.New(), uuid.New(), domain.ScopeGlobal, uuid.Nil, uuid.Nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestRoleService_CreatePermission(t *testing.T) {
	pRepo := &mockPermRepo{}
	svc := NewRoleService(&mockRoleRepo{}, pRepo, &mockUserRoleRepo{})

	perm, err := svc.CreatePermission(context.Background(), &domain.Permission{
		TenantID: uuid.New(), Key: "users:read", Name: "Read", ResourceType: "user", Action: "read",
	})
	if err != nil || perm.Key != "users:read" {
		t.Fatalf("CreatePermission: err=%v", err)
	}
}

func TestRoleService_ListPermissions(t *testing.T) {
	pRepo := &mockPermRepo{perms: []*domain.Permission{{ID: uuid.New(), Key: "x"}}}
	svc := NewRoleService(&mockRoleRepo{}, pRepo, &mockUserRoleRepo{})
	perms, err := svc.ListPermissions(context.Background(), uuid.New(), 1, 50)
	if err != nil || len(perms) != 1 {
		t.Fatalf("ListPermissions: err=%v len=%d", err, len(perms))
	}
}

func TestRoleService_GrantRevoke(t *testing.T) {
	svc := NewRoleService(&mockRoleRepo{}, &mockPermRepo{}, &mockUserRoleRepo{})
	ids := []uuid.UUID{uuid.New()}
	if err := svc.GrantPermissionsToRole(context.Background(), uuid.New(), ids); err != nil {
		t.Fatalf("Grant: %v", err)
	}
	if err := svc.RevokePermissionsFromRole(context.Background(), uuid.New(), ids); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
}

// --- PolicyService Tests ---

func TestPolicyService_CreatePolicy_Allow(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	p, err := svc.CreatePolicy(context.Background(), &domain.Policy{
		TenantID: uuid.New(), Name: "test", Effect: domain.EffectAllow,
		Actions: []string{"read"}, Resources: []string{"*"},
	})
	if err != nil || p.ID == uuid.Nil {
		t.Fatalf("CreatePolicy allow: err=%v", err)
	}
}

func TestPolicyService_CreatePolicy_DenyPriority(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	p, err := svc.CreatePolicy(context.Background(), &domain.Policy{Name: "deny", Effect: domain.EffectDeny})
	if err != nil || p.Priority != 100 {
		t.Fatalf("deny priority: err=%v priority=%d", err, p.Priority)
	}
}

func TestPolicyService_CreatePolicy_InvalidEffect(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	_, err := svc.CreatePolicy(context.Background(), &domain.Policy{Effect: "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid effect")
	}
}

func TestPolicyService_GetPolicy(t *testing.T) {
	id := uuid.New()
	repo := &mockPolicyRepo{policies: map[uuid.UUID]*domain.Policy{id: {ID: id, Name: "p1"}}}
	svc := NewPolicyService(repo)
	p, err := svc.GetPolicy(context.Background(), id)
	if err != nil || p.Name != "p1" {
		t.Fatalf("GetPolicy: err=%v", err)
	}
}

func TestPolicyService_ListPolicies(t *testing.T) {
	repo := &mockPolicyRepo{listResult: []*domain.Policy{{ID: uuid.New()}, {ID: uuid.New()}}}
	svc := NewPolicyService(repo)
	policies, err := svc.ListPolicies(context.Background(), uuid.New(), 1, 10)
	if err != nil || len(policies) != 2 {
		t.Fatalf("ListPolicies: err=%v len=%d", err, len(policies))
	}
}

func TestPolicyService_DeletePolicy(t *testing.T) {
	id := uuid.New()
	repo := &mockPolicyRepo{policies: map[uuid.UUID]*domain.Policy{id: {ID: id}}}
	svc := NewPolicyService(repo)
	if err := svc.DeletePolicy(context.Background(), id); err != nil {
		t.Fatalf("DeletePolicy: %v", err)
	}
}

func TestPolicyService_AttachPolicy(t *testing.T) {
	id := uuid.New()
	repo := &mockPolicyRepo{policies: map[uuid.UUID]*domain.Policy{id: {ID: id}}}
	svc := NewPolicyService(repo)
	if err := svc.AttachPolicy(context.Background(), id, domain.PrincipalUser, uuid.New()); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}
}

func TestPolicyService_AttachPolicy_NotFound(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{policies: map[uuid.UUID]*domain.Policy{}})
	if err := svc.AttachPolicy(context.Background(), uuid.New(), domain.PrincipalUser, uuid.New()); err == nil {
		t.Fatal("expected error")
	}
}

func TestPolicyService_DetachPolicy(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{})
	if err := svc.DetachPolicy(context.Background(), uuid.New(), domain.PrincipalUser, uuid.New()); err != nil {
		t.Fatalf("DetachPolicy: %v", err)
	}
}
