package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	pkgerrors "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/ggid/ggid/services/policy/internal/service"
	"github.com/google/uuid"
)

// ===== In-memory mock repositories =====
// These satisfy the unexported interfaces in the service package via
// Go structural typing (all methods are exported).

// testRoleRepo implements service.RoleRepo + service.RoleReader.
type testRoleRepo struct {
	mu             sync.Mutex
	roles          map[uuid.UUID]*domain.Role
	rolePerms      map[uuid.UUID][]*domain.Permission
	getErr         error
	createErr      error
	getRolePermErr error
	grantErr       error
	revokeErr      error
	listErr        error
}

func (m *testRoleRepo) Create(_ context.Context, r *domain.Role) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.roles == nil {
		m.roles = map[uuid.UUID]*domain.Role{}
	}
	r.ID = uuid.New()
	r.CreatedAt = time.Now()
	m.roles[r.ID] = r
	return nil
}

func (m *testRoleRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Role, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if r, ok := m.roles[id]; ok {
		return r, nil
	}
	return nil, pkgerrors.NotFound("role", id.String())
}

func (m *testRoleRepo) ListByTenant(_ context.Context, tid uuid.UUID, limit, offset int) ([]*domain.Role, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var res []*domain.Role
	n := 0
	for _, r := range m.roles {
		if r.TenantID == tid {
			if n >= offset && len(res) < limit {
				res = append(res, r)
			}
			n++
		}
	}
	return res, nil
}

func (m *testRoleRepo) Update(_ context.Context, r *domain.Role) error {
	if m.roles == nil {
		m.roles = map[uuid.UUID]*domain.Role{}
	}
	m.roles[r.ID] = r
	return nil
}

func (m *testRoleRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.roles[id]; !ok {
		return pkgerrors.NotFound("role", id.String())
	}
	delete(m.roles, id)
	return nil
}

func (m *testRoleRepo) GrantPermissions(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ map[string]any) error {
	if m.grantErr != nil {
		return m.grantErr
	}
	return nil
}

func (m *testRoleRepo) RevokePermissions(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}
	return nil
}

func (m *testRoleRepo) GetRolePermissions(_ context.Context, roleIDs []uuid.UUID) ([]*domain.Permission, error) {
	if m.getRolePermErr != nil {
		return nil, m.getRolePermErr
	}
	if m.rolePerms == nil {
		return nil, nil
	}
	var result []*domain.Permission
	for _, rid := range roleIDs {
		result = append(result, m.rolePerms[rid]...)
	}
	return result, nil
}

func (m *testRoleRepo) GetAncestorChain(_ context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	var chain []uuid.UUID
	current := roleID
	for i := 0; i < 100; i++ {
		r, ok := m.roles[current]
		if !ok {
			break
		}
		chain = append(chain, current)
		if r.ParentRoleID == nil {
			break
		}
		current = *r.ParentRoleID
	}
	return chain, nil
}

// testPermRepo implements service.PermRepo.
type testPermRepo struct {
	perms   map[uuid.UUID]*domain.Permission
	listErr error
}

func (m *testPermRepo) Create(_ context.Context, p *domain.Permission) error {
	if m.perms == nil {
		m.perms = map[uuid.UUID]*domain.Permission{}
	}
	p.ID = uuid.New()
	m.perms[p.ID] = p
	return nil
}

func (m *testPermRepo) ListByTenant(_ context.Context, tid uuid.UUID, limit, offset int) ([]*domain.Permission, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var res []*domain.Permission
	n := 0
	for _, p := range m.perms {
		if p.TenantID == tid {
			if n >= offset && len(res) < limit {
				res = append(res, p)
			}
			n++
		}
	}
	return res, nil
}

// testUserRoleRepo implements service.UserRoleRepo.
type testUserRoleRepo struct{}

func (m *testUserRoleRepo) Assign(_ context.Context, _ *domain.UserRole) error      { return nil }
func (m *testUserRoleRepo) Revoke(_ context.Context, _, _ uuid.UUID, _ domain.ScopeType, _ uuid.UUID) error {
	return nil
}
func (m *testUserRoleRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]*domain.UserRole, error) {
	return nil, nil
}

// testPolicyRepo implements service.PolicyRepo.
type testPolicyRepo struct {
	policies  map[uuid.UUID]*domain.Policy
	createErr error
	deleteErr error
	listErr   error
}

func (m *testPolicyRepo) Create(_ context.Context, p *domain.Policy) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.policies == nil {
		m.policies = map[uuid.UUID]*domain.Policy{}
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	m.policies[p.ID] = p
	return nil
}

func (m *testPolicyRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Policy, error) {
	if p, ok := m.policies[id]; ok {
		return p, nil
	}
	return nil, pkgerrors.NotFound("policy", id.String())
}

func (m *testPolicyRepo) ListByTenant(_ context.Context, tid uuid.UUID, limit, offset int) ([]*domain.Policy, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var res []*domain.Policy
	n := 0
	for _, p := range m.policies {
		if p.TenantID == tid {
			if n >= offset && len(res) < limit {
				res = append(res, p)
			}
			n++
		}
	}
	return res, nil
}

func (m *testPolicyRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.policies, id)
	return nil
}

func (m *testPolicyRepo) AttachPolicy(_ context.Context, _ *domain.PolicyAttachment) error { return nil }
func (m *testPolicyRepo) DetachPolicy(_ context.Context, _ uuid.UUID, _ domain.PrincipalType, _ uuid.UUID) error {
	return nil
}

// testUserRoleReader implements service.UserRoleReader for the evaluator.
type testUserRoleReader struct {
	userRoles map[uuid.UUID][]uuid.UUID
}

func (m *testUserRoleReader) GetUserRoles(_ context.Context, userID uuid.UUID) ([]*domain.UserRole, error) {
	ids := m.userRoles[userID]
	var roles []*domain.UserRole
	for _, id := range ids {
		roles = append(roles, &domain.UserRole{RoleID: id})
	}
	return roles, nil
}

// testPolicyReader implements service.PolicyReader for the evaluator.
type testPolicyReader struct{}

func (m *testPolicyReader) GetPoliciesForUserAndRoles(_ context.Context, _ uuid.UUID, _ []uuid.UUID) ([]*domain.Policy, error) {
	return nil, nil
}

// ===== Test harness =====

type testHarness struct {
	srv            *HTTPServer
	roleRepo       *testRoleRepo
	permRepo       *testPermRepo
	policyRepo     *testPolicyRepo
	roleSvc        *service.RoleService
	policySvc      *service.PolicyService
	evaluator      *service.Evaluator
	userRoleReader *testUserRoleReader
	tenantID       uuid.UUID
}

func newTestHarness() *testHarness {
	// Clear global decision log so evaluator tests don't pollute other tests
	clearTestDecisions()

	roleRepo := &testRoleRepo{
		roles:     map[uuid.UUID]*domain.Role{},
		rolePerms: map[uuid.UUID][]*domain.Permission{},
	}
	permRepo := &testPermRepo{perms: map[uuid.UUID]*domain.Permission{}}
	policyRepo := &testPolicyRepo{policies: map[uuid.UUID]*domain.Policy{}}
	userRoleReader := &testUserRoleReader{userRoles: map[uuid.UUID][]uuid.UUID{}}
	policyReader := &testPolicyReader{}

	roleSvc := service.NewRoleService(roleRepo, permRepo, &testUserRoleRepo{})
	policySvc := service.NewPolicyService(policyRepo)
	evaluator := service.NewEvaluator(roleRepo, userRoleReader, policyReader)

	srv := &HTTPServer{
		roleSvc:   roleSvc,
		policySvc: policySvc,
		evaluator: evaluator,
	}
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	testMux = mux

	return &testHarness{
		srv:            srv,
		roleRepo:       roleRepo,
		permRepo:       permRepo,
		policyRepo:     policyRepo,
		roleSvc:        roleSvc,
		policySvc:      policySvc,
		evaluator:      evaluator,
		userRoleReader: userRoleReader,
		tenantID:       uuid.New(),
	}
}

// doReq sends a request with an optional JSON body string.
func doReq(method, path, body string) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	w := httptest.NewRecorder()
	testMux.ServeHTTP(w, req)
	return w
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("expected status %d, got %d (body: %s)", expected, w.Code, w.Body.String())
	}
}

func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v (body: %s)", err, w.Body.String())
	}
	return m
}

// ===== Roles tests =====

func TestHandleRoles_Create(t *testing.T) {
	h := newTestHarness()
	body := `{"tenant_id":"` + h.tenantID.String() + `","key":"admin","name":"Admin","description":"super user"}`
	w := doReq("POST", "/api/v1/roles", body)
	assertStatus(t, w, http.StatusCreated)
	resp := parseJSON(t, w)
	if resp["key"] != "admin" {
		t.Errorf("expected key=admin, got %v", resp["key"])
	}
}

func TestHandleRoles_CreateInvalidJSON(t *testing.T) {
	h := newTestHarness()
	_ = h
	w := doReq("POST", "/api/v1/roles", "not-json")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleRoles_CreateInvalidTenant(t *testing.T) {
	w := doReq("POST", "/api/v1/roles", `{"tenant_id":"bad","key":"k","name":"N"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleRoles_CreateInvalidParent(t *testing.T) {
	h := newTestHarness()
	body := `{"tenant_id":"` + h.tenantID.String() + `","key":"k","name":"N","parent_role_id":"bad"}`
	w := doReq("POST", "/api/v1/roles", body)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleRoles_List(t *testing.T) {
	h := newTestHarness()
	// Pre-create a role
	_, _ = h.roleSvc.CreateRole(context.Background(), h.tenantID, "editor", "Editor", "", nil)

	w := doReq("GET", "/api/v1/roles?tenant_id="+h.tenantID.String(), "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	roles := resp["roles"].([]any)
	if len(roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(roles))
	}
}

func TestHandleRoles_ListMissingTenant(t *testing.T) {
	w := doReq("GET", "/api/v1/roles", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleRoles_ListInvalidTenant(t *testing.T) {
	w := doReq("GET", "/api/v1/roles?tenant_id=bad", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleRoles_MethodNotAllowed(t *testing.T) {
	w := doReq("DELETE", "/api/v1/roles", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandleRoleByID_GetAndDelete(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "admin", "Admin", "", nil)

	// GET
	w := doReq("GET", "/api/v1/roles/"+role.ID.String(), "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["name"] != "Admin" {
		t.Errorf("expected name=Admin, got %v", resp["name"])
	}

	// DELETE
	w = doReq("DELETE", "/api/v1/roles/"+role.ID.String(), "")
	assertStatus(t, w, http.StatusOK)
}

func TestHandleRoleByID_InvalidID(t *testing.T) {
	w := doReq("GET", "/api/v1/roles/bad-uuid", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleRoleByID_NotFound(t *testing.T) {
	w := doReq("GET", "/api/v1/roles/"+uuid.New().String(), "")
	assertStatus(t, w, http.StatusNotFound)
}

func TestHandleRoleByID_MethodNotAllowed(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("PATCH", "/api/v1/roles/"+role.ID.String(), "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandleRoleByID_EmptyID(t *testing.T) {
	// /api/v1/roles/ with nothing after the slash
	w := doReq("GET", "/api/v1/roles/", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleRolePermissions(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "admin", "Admin", "", nil)
	permID := uuid.New()

	// POST grant
	body := `{"permission_ids":["` + permID.String() + `"]}`
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/permissions", body)
	assertStatus(t, w, http.StatusOK)

	// GET
	w = doReq("GET", "/api/v1/roles/"+role.ID.String()+"/permissions", "")
	assertStatus(t, w, http.StatusOK)

	// DELETE revoke
	w = doReq("DELETE", "/api/v1/roles/"+role.ID.String()+"/permissions", body)
	assertStatus(t, w, http.StatusOK)
}

func TestHandleRolePermissions_InvalidJSON(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/permissions", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleRolePermissions_InvalidPermID(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/permissions", `{"permission_ids":["bad"]}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleRolePermissions_MethodNotAllowed(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("PATCH", "/api/v1/roles/"+role.ID.String()+"/permissions", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandleEffectivePermissions(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "admin", "Admin", "", nil)
	perm := &domain.Permission{ID: uuid.New(), Key: "read", Name: "Read", ResourceType: "docs", Action: "read"}
	h.roleRepo.rolePerms = map[uuid.UUID][]*domain.Permission{role.ID: {perm}}

	w := doReq("GET", "/api/v1/roles/"+role.ID.String()+"/effective-permissions", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["total_effective"].(float64) != 1 {
		t.Errorf("expected 1 effective perm, got %v", resp["total_effective"])
	}
}

func TestHandleEffectivePermissions_MethodNotAllowed(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/effective-permissions", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandleSetRoleParent(t *testing.T) {
	h := newTestHarness()
	parent, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "p", "Parent", "", nil)
	child, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "c", "Child", "", nil)

	body := `{"parent_role_id":"` + parent.ID.String() + `"}`
	w := doReq("POST", "/api/v1/roles/"+child.ID.String()+"/parent", body)
	assertStatus(t, w, http.StatusOK)
}

func TestHandleSetRoleParent_ClearParent(t *testing.T) {
	h := newTestHarness()
	parent, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "p", "Parent", "", nil)
	child, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "c", "Child", "", nil)
	_ = h.roleSvc // ensure used
	_, _ = h.roleSvc.SetParent(context.Background(), child.ID, parent.ID)

	w := doReq("POST", "/api/v1/roles/"+child.ID.String()+"/parent", `{"parent_role_id":""}`)
	assertStatus(t, w, http.StatusOK)
}

func TestHandleSetRoleParent_InvalidJSON(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/parent", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleSetRoleParent_InvalidParentID(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/parent", `{"parent_role_id":"bad"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleSetRoleParent_MethodNotAllowed(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("GET", "/api/v1/roles/"+role.ID.String()+"/parent", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandleBulkAssign(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	uid1, uid2 := uuid.New(), uuid.New()
	body := `{"user_ids":["` + uid1.String() + `","` + uid2.String() + `"]}`
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/bulk-assign", body)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["assigned"].(float64) != 2 {
		t.Errorf("expected 2 assigned, got %v", resp["assigned"])
	}
}

func TestHandleBulkAssign_InvalidUUID(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	body := `{"user_ids":["bad-uuid"]}`
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/bulk-assign", body)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["errors"].(float64) != 1 {
		t.Errorf("expected 1 error, got %v", resp["errors"])
	}
}

func TestHandleBulkAssign_EmptyUserIDs(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/bulk-assign", `{"user_ids":[]}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleBulkAssign_InvalidJSON(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/bulk-assign", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleBulkAssign_RoleNotFound(t *testing.T) {
	body := `{"user_ids":["` + uuid.New().String() + `"]}`
	w := doReq("POST", "/api/v1/roles/"+uuid.New().String()+"/bulk-assign", body)
	assertStatus(t, w, http.StatusNotFound)
}

func TestHandleBulkAssign_MethodNotAllowed(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("GET", "/api/v1/roles/"+role.ID.String()+"/bulk-assign", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Permissions tests =====

func TestHandlePermissions_List(t *testing.T) {
	h := newTestHarness()
	_, _ = h.roleSvc.CreatePermission(context.Background(), &domain.Permission{
		TenantID: h.tenantID, Key: "read", Name: "Read", ResourceType: "docs", Action: "read",
	})
	w := doReq("GET", "/api/v1/permissions?tenant_id="+h.tenantID.String(), "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	perms := resp["permissions"].([]any)
	if len(perms) != 1 {
		t.Errorf("expected 1 perm, got %d", len(perms))
	}
}

func TestHandlePermissions_MissingTenant(t *testing.T) {
	w := doReq("GET", "/api/v1/permissions", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePermissions_InvalidTenant(t *testing.T) {
	w := doReq("GET", "/api/v1/permissions?tenant_id=bad", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePermissions_MethodNotAllowed(t *testing.T) {
	w := doReq("POST", "/api/v1/permissions", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Policies tests =====

func TestHandlePolicies_Create(t *testing.T) {
	h := newTestHarness()
	body := `{"tenant_id":"` + h.tenantID.String() + `","name":"DenyAll","effect":"deny","actions":["*"],"resources":["*"]}`
	w := doReq("POST", "/api/v1/policies", body)
	assertStatus(t, w, http.StatusCreated)
	resp := parseJSON(t, w)
	if resp["name"] != "DenyAll" {
		t.Errorf("expected name=DenyAll, got %v", resp["name"])
	}
}

func TestHandlePolicies_CreateInvalidJSON(t *testing.T) {
	w := doReq("POST", "/api/v1/policies", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicies_CreateInvalidTenant(t *testing.T) {
	w := doReq("POST", "/api/v1/policies", `{"tenant_id":"bad","name":"X","effect":"allow"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicies_List(t *testing.T) {
	h := newTestHarness()
	_, _ = h.policySvc.CreatePolicy(context.Background(), &domain.Policy{
		TenantID: h.tenantID, Name: "P1", Effect: domain.EffectAllow,
	})
	w := doReq("GET", "/api/v1/policies?tenant_id="+h.tenantID.String(), "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	policies := resp["policies"].([]any)
	if len(policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(policies))
	}
}

func TestHandlePolicies_ListMissingTenant(t *testing.T) {
	w := doReq("GET", "/api/v1/policies", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicies_MethodNotAllowed(t *testing.T) {
	w := doReq("DELETE", "/api/v1/policies", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandlePolicyByID_Delete(t *testing.T) {
	h := newTestHarness()
	policy, _ := h.policySvc.CreatePolicy(context.Background(), &domain.Policy{
		TenantID: h.tenantID, Name: "P1", Effect: domain.EffectAllow,
	})
	w := doReq("DELETE", "/api/v1/policies/"+policy.ID.String(), "")
	assertStatus(t, w, http.StatusOK)
}

func TestHandlePolicyByID_InvalidID(t *testing.T) {
	w := doReq("DELETE", "/api/v1/policies/bad-uuid", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyByID_MethodNotAllowed(t *testing.T) {
	h := newTestHarness()
	policy, _ := h.policySvc.CreatePolicy(context.Background(), &domain.Policy{
		TenantID: h.tenantID, Name: "P1", Effect: domain.EffectAllow,
	})
	w := doReq("GET", "/api/v1/policies/"+policy.ID.String(), "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Check tests =====

func TestHandleCheck(t *testing.T) {
	newTestHarness()
	uid := uuid.New()
	body := `{"user_id":"` + uid.String() + `","resource_type":"docs","action":"read","resource":"doc:1"}`
	w := doReq("POST", "/api/v1/policies/check", body)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if _, ok := resp["allowed"]; !ok {
		t.Errorf("expected 'allowed' in response, got: %v", resp)
	}
}

func TestHandleCheck_MethodNotAllowed(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/check", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandleCheck_InvalidJSON(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/check", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleCheck_InvalidUserID(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/check", `{"user_id":"bad"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleCheck_AllowedTrue(t *testing.T) {
	h := newTestHarness()
	uid := uuid.New()
	roleID := uuid.New()
	permID := uuid.New()

	// Set up role + permission + user-role mapping
	h.roleRepo.roles[roleID] = &domain.Role{ID: roleID, TenantID: h.tenantID, Key: "admin", Name: "Admin"}
	h.roleRepo.rolePerms[roleID] = []*domain.Permission{
		{ID: permID, Key: "read", Name: "Read", ResourceType: "docs", Action: "read"},
	}
	h.userRoleReader.userRoles[uid] = []uuid.UUID{roleID}

	body := `{"user_id":"` + uid.String() + `","resource_type":"docs","action":"read","resource":"doc:1"}`
	w := doReq("POST", "/api/v1/policies/check", body)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["allowed"] != true {
		t.Errorf("expected allowed=true, got %v", resp["allowed"])
	}
}

// ===== Evaluate tests =====

func TestHandleEvaluate(t *testing.T) {
	h := newTestHarness()
	uid := uuid.New()
	roleID := uuid.New()
	h.roleRepo.roles[roleID] = &domain.Role{ID: roleID, TenantID: h.tenantID, Key: "admin", Name: "Admin"}
	h.roleRepo.rolePerms[roleID] = []*domain.Permission{
		{ID: uuid.New(), Key: "read", Name: "Read", ResourceType: "docs", Action: "read"},
	}
	h.userRoleReader.userRoles[uid] = []uuid.UUID{roleID}

	body := `{"user_id":"` + uid.String() + `","resource_type":"docs","action":"read","attributes":{"dept":"eng"}}`
	w := doReq("POST", "/api/v1/policies/evaluate", body)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["allowed"] != true {
		t.Errorf("expected allowed=true, got %v", resp["allowed"])
	}
}

func TestHandleEvaluate_MethodNotAllowed(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/evaluate", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandleEvaluate_InvalidJSON(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/evaluate", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleEvaluate_MissingUserID(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/evaluate", `{"action":"read"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleEvaluate_InvalidUserID(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/evaluate", `{"user_id":"bad"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

// ===== Policy Templates tests =====

func TestHandlePolicyTemplates_List(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/templates", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["count"].(float64) < 1 {
		t.Errorf("expected templates, got %v", resp["count"])
	}
}

func TestHandlePolicyTemplates_Search(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/templates?search=pci", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["count"].(float64) != 1 {
		t.Errorf("expected 1 template for pci, got %v", resp["count"])
	}
}

func TestHandlePolicyTemplates_SearchNoMatch(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/templates?search=nonexistent", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["count"].(float64) != 0 {
		t.Errorf("expected 0 templates, got %v", resp["count"])
	}
}

func TestHandlePolicyTemplates_MethodNotAllowed(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/templates", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandleFromTemplate(t *testing.T) {
	h := newTestHarness()
	body := `{"tenant_id":"` + h.tenantID.String() + `"}`
	w := doReq("POST", "/api/v1/policies/from-template/pci-dss", body)
	assertStatus(t, w, http.StatusCreated)
	resp := parseJSON(t, w)
	if resp["status"] != "created" {
		t.Errorf("expected status=created, got %v", resp["status"])
	}
}

func TestHandleFromTemplate_NotFound(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/from-template/nonexistent", "")
	assertStatus(t, w, http.StatusNotFound)
}

func TestHandleFromTemplate_MethodNotAllowed(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/from-template/pci-dss", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Default Action tests =====

func TestHandleDefaultAction_Get(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/default-action", "")
	assertStatus(t, w, http.StatusOK)
}

func TestHandleDefaultAction_Put(t *testing.T) {
	w := doReq("PUT", "/api/v1/policies/default-action", `{"default_action":"allow"}`)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["default_action"] != "allow" {
		t.Errorf("expected allow, got %v", resp["default_action"])
	}
	// Reset to deny for other tests
	_ = doReq("PUT", "/api/v1/policies/default-action", `{"default_action":"deny"}`)
}

func TestHandleDefaultAction_PutInvalid(t *testing.T) {
	w := doReq("PUT", "/api/v1/policies/default-action", `{"default_action":"bad"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleDefaultAction_PutInvalidJSON(t *testing.T) {
	w := doReq("PUT", "/api/v1/policies/default-action", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleDefaultAction_MethodNotAllowed(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/default-action", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Time Conditions tests =====

func TestHandleTimeConditions_Get(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/time-conditions", "")
	assertStatus(t, w, http.StatusOK)
}

func TestHandleTimeConditions_Create(t *testing.T) {
	body := `{"name":"business-hours","time_between":"09:00-17:00","days_of_week":[1,2,3,4,5],"timezone":"UTC","effect":"allow"}`
	w := doReq("POST", "/api/v1/policies/time-conditions", body)
	assertStatus(t, w, http.StatusCreated)
	resp := parseJSON(t, w)
	if resp["name"] != "business-hours" {
		t.Errorf("expected name=business-hours, got %v", resp["name"])
	}
}

func TestHandleTimeConditions_MissingName(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/time-conditions", `{"effect":"allow"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleTimeConditions_InvalidJSON(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/time-conditions", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleTimeConditions_MethodNotAllowed(t *testing.T) {
	w := doReq("DELETE", "/api/v1/policies/time-conditions", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Dry-Run tests =====

func TestHandleDryRun(t *testing.T) {
	// Use empty user_id so evaluator.Check returns early (anonymous user)
	// without touching nil readers in the internally-created Evaluator{}.
	body := `{"resource":"docs:123","action":"read","attributes":{"dept":"eng"}}`
	w := doReq("POST", "/api/v1/policies/dry-run", body)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", resp["dry_run"])
	}
}

func TestHandleDryRun_MissingFields(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/dry-run", `{"resource":"x"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleDryRun_InvalidJSON(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/dry-run", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleDryRun_MethodNotAllowed(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/dry-run", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Policy Export/Import tests =====

func TestHandlePolicyExport(t *testing.T) {
	h := newTestHarness()
	_, _ = h.policySvc.CreatePolicy(context.Background(), &domain.Policy{
		TenantID: h.tenantID, Name: "P1", Effect: domain.EffectAllow,
	})
	w := doReq("GET", "/api/v1/policies/export?tenant_id="+h.tenantID.String(), "")
	assertStatus(t, w, http.StatusOK)
	if w.Header().Get("Content-Disposition") == "" {
		t.Error("expected Content-Disposition header")
	}
	resp := parseJSON(t, w)
	if resp["version"] != "1.0" {
		t.Errorf("expected version 1.0, got %v", resp["version"])
	}
}

func TestHandlePolicyExport_MissingTenant(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/export", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyExport_InvalidTenant(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/export?tenant_id=bad", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyExport_MethodNotAllowed(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/export", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestHandlePolicyImport(t *testing.T) {
	h := newTestHarness()
	body := `{"policies":[{"name":"Imported1","effect":"allow","actions":["read"],"resources":["docs"]},{"name":"Imported2","effect":"deny","actions":["*"],"resources":["secrets"]}]}`
	w := doReq("POST", "/api/v1/policies/import?tenant_id="+h.tenantID.String(), body)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["imported"].(float64) != 2 {
		t.Errorf("expected 2 imported, got %v", resp["imported"])
	}
}

func TestHandlePolicyImport_MissingName(t *testing.T) {
	h := newTestHarness()
	body := `{"policies":[{"effect":"allow"}]}`
	w := doReq("POST", "/api/v1/policies/import?tenant_id="+h.tenantID.String(), body)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	errs := resp["errors"].([]any)
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestHandlePolicyImport_MissingTenant(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/import", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyImport_InvalidJSON(t *testing.T) {
	h := newTestHarness()
	w := doReq("POST", "/api/v1/policies/import?tenant_id="+h.tenantID.String(), "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyImport_MethodNotAllowed(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/import", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Analyze tests =====

func TestHandleAnalyze(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "admin", "Admin", "", nil)
	h.roleRepo.rolePerms = map[uuid.UUID][]*domain.Permission{
		role.ID: {
			{ID: uuid.New(), Key: "read", Name: "Read", ResourceType: "docs", Action: "read"},
			{ID: uuid.New(), Key: "write", Name: "Write", ResourceType: "docs", Action: "write"},
		},
	}

	w := doReq("GET", "/api/v1/policies/analyze?role_id="+role.ID.String(), "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["total_direct"].(float64) != 2 {
		t.Errorf("expected 2 direct perms, got %v", resp["total_direct"])
	}
}

func TestHandleAnalyze_MissingRoleID(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/analyze", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleAnalyze_InvalidRoleID(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/analyze?role_id=bad", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleAnalyze_MethodNotAllowed(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/analyze", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Attribute Mapping tests =====

func TestHandleAttributeMapping_Get(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/attribute-mapping", "")
	assertStatus(t, w, http.StatusOK)
}

func TestHandleAttributeMapping_Create(t *testing.T) {
	body := `{"attribute":"department","value":"Engineering","role_id":"` + uuid.New().String() + `","action":"assign_role"}`
	w := doReq("POST", "/api/v1/policies/attribute-mapping", body)
	assertStatus(t, w, http.StatusCreated)
}

func TestHandleAttributeMapping_CreateDefaultAction(t *testing.T) {
	body := `{"attribute":"department","value":"Sales"}`
	w := doReq("POST", "/api/v1/policies/attribute-mapping", body)
	assertStatus(t, w, http.StatusCreated)
	resp := parseJSON(t, w)
	if resp["action"] != "assign_role" {
		t.Errorf("expected default action=assign_role, got %v", resp["action"])
	}
}

func TestHandleAttributeMapping_MissingFields(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/attribute-mapping", `{"attribute":"dept"}`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleAttributeMapping_InvalidJSON(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/attribute-mapping", "bad")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleAttributeMapping_Delete(t *testing.T) {
	w := doReq("DELETE", "/api/v1/policies/attribute-mapping?id=some-id", "")
	assertStatus(t, w, http.StatusOK)
}

func TestHandleAttributeMapping_DeleteMissingID(t *testing.T) {
	w := doReq("DELETE", "/api/v1/policies/attribute-mapping", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleAttributeMapping_MethodNotAllowed(t *testing.T) {
	w := doReq("PATCH", "/api/v1/policies/attribute-mapping", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Policy Versions tests =====

func TestHandlePolicyVersions_GetEmpty(t *testing.T) {
	pid := uuid.New().String()
	w := doReq("GET", "/api/v1/policies/versions?policy_id="+pid, "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["total"].(float64) != 0 {
		t.Errorf("expected 0 versions, got %v", resp["total"])
	}
}

func TestHandlePolicyVersions_MissingPolicyID(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/versions", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyVersions_CreateSnapshot(t *testing.T) {
	h := newTestHarness()
	policy, _ := h.policySvc.CreatePolicy(context.Background(), &domain.Policy{
		TenantID: h.tenantID, Name: "P1", Effect: domain.EffectAllow, Actions: []string{"read"}, Resources: []string{"docs"},
	})
	w := doReq("POST", "/api/v1/policies/versions?policy_id="+policy.ID.String(), "")
	assertStatus(t, w, http.StatusCreated)
	resp := parseJSON(t, w)
	if resp["name"] != "P1" {
		t.Errorf("expected name=P1, got %v", resp["name"])
	}
}

func TestHandlePolicyVersions_Rollback(t *testing.T) {
	h := newTestHarness()
	policy, _ := h.policySvc.CreatePolicy(context.Background(), &domain.Policy{
		TenantID: h.tenantID, Name: "P1", Effect: domain.EffectAllow,
	})
	pid := policy.ID.String()
	// Pre-populate versions
	policyVersions[pid] = []map[string]any{
		{
			"version":    1,
			"name":       "OldPolicy",
			"effect":     "deny",
			"actions":    []string{"read"},
			"resources":  []string{"docs"},
			"created_at": "2024-01-01T00:00:00Z",
		},
	}
	w := doReq("POST", "/api/v1/policies/versions?policy_id="+pid+"&action=rollback&version=1", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if resp["status"] != "rolled_back" {
		t.Errorf("expected status=rolled_back, got %v", resp["status"])
	}
	delete(policyVersions, pid)
}

func TestHandlePolicyVersions_RollbackMissingVersion(t *testing.T) {
	pid := uuid.New().String()
	w := doReq("POST", "/api/v1/policies/versions?policy_id="+pid+"&action=rollback", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyVersions_RollbackInvalidVersion(t *testing.T) {
	pid := uuid.New().String()
	w := doReq("POST", "/api/v1/policies/versions?policy_id="+pid+"&action=rollback&version=99", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyVersions_MethodNotAllowed(t *testing.T) {
	w := doReq("DELETE", "/api/v1/policies/versions?policy_id="+uuid.New().String(), "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Policy Diff tests =====

func TestHandlePolicyDiff(t *testing.T) {
	pid := uuid.New().String()
	policyVersions[pid] = []map[string]any{
		{"name": "V1", "effect": "allow", "actions": []string{"read"}},
		{"name": "V2", "effect": "deny", "actions": []string{"read", "write"}},
	}
	w := doReq("GET", "/api/v1/policies/diff?policy_id="+pid+"&v1=1&v2=2", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	diff := resp["diff"].(map[string]any)
	modified := diff["modified"].([]any)
	if len(modified) < 1 {
		t.Errorf("expected modifications, got %d", len(modified))
	}
	delete(policyVersions, pid)
}

func TestHandlePolicyDiff_MissingPolicyID(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/diff?v1=1&v2=2", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyDiff_MissingVersions(t *testing.T) {
	w := doReq("GET", "/api/v1/policies/diff?policy_id="+uuid.New().String(), "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyDiff_InvalidVersion(t *testing.T) {
	pid := uuid.New().String()
	w := doReq("GET", "/api/v1/policies/diff?policy_id="+pid+"&v1=abc&v2=2", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandlePolicyDiff_VersionOutOfRange(t *testing.T) {
	pid := uuid.New().String()
	policyVersions[pid] = []map[string]any{
		{"name": "V1", "effect": "allow"},
	}
	w := doReq("GET", "/api/v1/policies/diff?policy_id="+pid+"&v1=1&v2=5", "")
	assertStatus(t, w, http.StatusBadRequest)
	delete(policyVersions, pid)
}

func TestHandlePolicyDiff_MethodNotAllowed(t *testing.T) {
	w := doReq("POST", "/api/v1/policies/diff", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

// ===== Helper function tests =====

func TestGetDefaultPolicyAction(t *testing.T) {
	val := GetDefaultPolicyAction()
	if val != "allow" && val != "deny" {
		t.Errorf("expected allow or deny, got %s", val)
	}
}

func TestTernary(t *testing.T) {
	if ternary(true, "a", "b") != "a" {
		t.Error("expected 'a'")
	}
	if ternary(false, "a", "b") != "b" {
		t.Error("expected 'b'")
	}
}

func TestToStringSlice(t *testing.T) {
	if got := toStringSlice([]string{"a", "b"}); len(got) != 2 {
		t.Errorf("expected 2 elements, got %d", len(got))
	}
	if got := toStringSlice([]any{"a", "b"}); len(got) != 2 {
		t.Errorf("expected 2 elements, got %d", len(got))
	}
	if got := toStringSlice("not-a-slice"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// ===== writeServiceError direct tests =====

func TestWriteServiceError_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"not_found", pkgerrors.NotFound("role", "123"), http.StatusNotFound},
		{"already_exists", pkgerrors.AlreadyExists("role", "123"), http.StatusConflict},
		{"invalid_argument", pkgerrors.InvalidArgument("bad input"), http.StatusBadRequest},
		{"permission_denied", pkgerrors.PermissionDenied("forbidden"), http.StatusForbidden},
		{"failed_precondition", pkgerrors.New(pkgerrors.ErrFailedPrecondition, "precondition"), http.StatusPreconditionFailed},
		{"internal_ggid", pkgerrors.New(pkgerrors.ErrInternal, "internal"), http.StatusInternalServerError},
		{"unauthenticated", pkgerrors.New(pkgerrors.ErrUnauthenticated, "no auth"), http.StatusInternalServerError},
		{"resource_exhausted", pkgerrors.New(pkgerrors.ErrResourceExhausted, "exhausted"), http.StatusInternalServerError},
		{"plain_error", fmt.Errorf("plain error"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeServiceError(w, tt.err)
			if w.Code != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, w.Code)
			}
		})
	}
}

// ===== NewHTTPServer test =====

func TestNewHTTPServer(t *testing.T) {
	roleRepo := &testRoleRepo{roles: map[uuid.UUID]*domain.Role{}}
	permRepo := &testPermRepo{perms: map[uuid.UUID]*domain.Permission{}}
	policyRepo := &testPolicyRepo{policies: map[uuid.UUID]*domain.Policy{}}
	roleSvc := service.NewRoleService(roleRepo, permRepo, &testUserRoleRepo{})
	policySvc := service.NewPolicyService(policyRepo)
	evaluator := service.NewEvaluator(roleRepo, &testUserRoleReader{}, &testPolicyReader{})
	srv := NewHTTPServer(roleSvc, policySvc, evaluator)
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

// ===== Error path coverage for handleAnalyze =====

func TestHandleAnalyze_RoleNotFound(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policies/analyze?role_id="+uuid.New().String(), "")
	assertStatus(t, w, http.StatusNotFound)
}

func TestHandleAnalyze_PermissionError(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "admin", "Admin", "", nil)
	h.roleRepo.getRolePermErr = fmt.Errorf("db error")
	w := doReq("GET", "/api/v1/policies/analyze?role_id="+role.ID.String(), "")
	assertStatus(t, w, http.StatusInternalServerError)
}

// ===== Error path coverage for handleRolePermissions =====

func TestHandleRolePermissions_GrantError(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	h.roleRepo.grantErr = fmt.Errorf("grant failed")
	body := `{"permission_ids":["` + uuid.New().String() + `"]}`
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/permissions", body)
	assertStatus(t, w, http.StatusInternalServerError)
}

func TestHandleRolePermissions_RevokeError(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	h.roleRepo.revokeErr = fmt.Errorf("revoke failed")
	body := `{"permission_ids":["` + uuid.New().String() + `"]}`
	w := doReq("DELETE", "/api/v1/roles/"+role.ID.String()+"/permissions", body)
	assertStatus(t, w, http.StatusInternalServerError)
}

func TestHandleRolePermissions_GetError(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	h.roleRepo.getRolePermErr = fmt.Errorf("get failed")
	w := doReq("GET", "/api/v1/roles/"+role.ID.String()+"/permissions", "")
	assertStatus(t, w, http.StatusInternalServerError)
}

// ===== Error path coverage for handleEffectivePermissions =====

func TestHandleEffectivePermissions_RoleNotFound(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/roles/"+uuid.New().String()+"/effective-permissions", "")
	assertStatus(t, w, http.StatusNotFound)
}

func TestHandleEffectivePermissions_EffectiveError(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	h.roleRepo.listErr = fmt.Errorf("list failed")
	w := doReq("GET", "/api/v1/roles/"+role.ID.String()+"/effective-permissions", "")
	assertStatus(t, w, http.StatusInternalServerError)
}

// ===== Error path coverage for handlePolicyByID =====

func TestHandlePolicyByID_DeleteError(t *testing.T) {
	h := newTestHarness()
	policy, _ := h.policySvc.CreatePolicy(context.Background(), &domain.Policy{
		TenantID: h.tenantID, Name: "P1", Effect: domain.EffectAllow,
	})
	h.policyRepo.deleteErr = fmt.Errorf("delete failed")
	w := doReq("DELETE", "/api/v1/policies/"+policy.ID.String(), "")
	assertStatus(t, w, http.StatusInternalServerError)
}

// ===== Error path coverage for listPolicies =====

func TestListPolicies_RepoError(t *testing.T) {
	h := newTestHarness()
	h.policyRepo.listErr = fmt.Errorf("list failed")
	w := doReq("GET", "/api/v1/policies?tenant_id="+h.tenantID.String(), "")
	assertStatus(t, w, http.StatusInternalServerError)
}

// ===== Error path coverage for handleFromTemplate =====

func TestHandleFromTemplate_InvalidJSON(t *testing.T) {
	newTestHarness()
	w := doReq("POST", "/api/v1/policies/from-template/pci-dss", "bad-json")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestHandleFromTemplate_RepoError(t *testing.T) {
	h := newTestHarness()
	h.policyRepo.createErr = fmt.Errorf("create failed")
	body := `{"tenant_id":"` + h.tenantID.String() + `"}`
	w := doReq("POST", "/api/v1/policies/from-template/pci-dss", body)
	assertStatus(t, w, http.StatusInternalServerError)
}

// ===== Error path coverage for handleSetRoleParent =====

func TestHandleSetRoleParent_CycleDetection(t *testing.T) {
	h := newTestHarness()
	roleA, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "a", "A", "", nil)
	roleB, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "b", "B", "", nil)
	// Set B's parent to A
	_, _ = h.roleSvc.SetParent(context.Background(), roleB.ID, roleA.ID)
	// Try to set A's parent to B — creates cycle
	body := `{"parent_role_id":"` + roleB.ID.String() + `"}`
	w := doReq("POST", "/api/v1/roles/"+roleA.ID.String()+"/parent", body)
	assertStatus(t, w, http.StatusPreconditionFailed)
}

func TestHandleSetRoleParent_SameID(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	body := `{"parent_role_id":"` + role.ID.String() + `"}`
	w := doReq("POST", "/api/v1/roles/"+role.ID.String()+"/parent", body)
	assertStatus(t, w, http.StatusBadRequest)
}

// ===== Additional error path coverage =====

func TestHandleRoles_CreateRepoError(t *testing.T) {
	h := newTestHarness()
	h.roleRepo.createErr = fmt.Errorf("create failed")
	body := `{"tenant_id":"` + h.tenantID.String() + `","key":"k","name":"N"}`
	w := doReq("POST", "/api/v1/roles", body)
	assertStatus(t, w, http.StatusInternalServerError)
}

func TestHandleRoles_ListRepoError(t *testing.T) {
	h := newTestHarness()
	h.roleRepo.listErr = fmt.Errorf("list failed")
	w := doReq("GET", "/api/v1/roles?tenant_id="+h.tenantID.String(), "")
	assertStatus(t, w, http.StatusInternalServerError)
}

func TestHandlePolicies_CreateRepoError(t *testing.T) {
	h := newTestHarness()
	h.policyRepo.createErr = fmt.Errorf("create failed")
	body := `{"tenant_id":"` + h.tenantID.String() + `","name":"X","effect":"allow"}`
	w := doReq("POST", "/api/v1/policies", body)
	assertStatus(t, w, http.StatusInternalServerError)
}

func TestHandlePermissions_ListRepoError(t *testing.T) {
	h := newTestHarness()
	h.permRepo.listErr = fmt.Errorf("list failed")
	w := doReq("GET", "/api/v1/permissions?tenant_id="+h.tenantID.String(), "")
	assertStatus(t, w, http.StatusInternalServerError)
}

// ===== handleRoleByID unknown sub-path fallthrough =====

func TestHandleRoleByID_UnknownSubPath(t *testing.T) {
	h := newTestHarness()
	role, _ := h.roleSvc.CreateRole(context.Background(), h.tenantID, "x", "X", "", nil)
	w := doReq("PUT", "/api/v1/roles/"+role.ID.String()+"/unknown", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}
