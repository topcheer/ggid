package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// --- Mock implementations ---

type mockRoleReader struct {
	ancestorChain    map[uuid.UUID][]uuid.UUID // roleID -> ancestor IDs (including self)
	rolePermissions  map[uuid.UUID][]*domain.Permission
	ancestorErr      error
	permissionsErr   error
}

func (m *mockRoleReader) GetAncestorChain(_ context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	if m.ancestorErr != nil {
		return nil, m.ancestorErr
	}
	if chain, ok := m.ancestorChain[roleID]; ok {
		return chain, nil
	}
	return []uuid.UUID{roleID}, nil // default: just the role itself
}

func (m *mockRoleReader) GetRolePermissions(_ context.Context, roleIDs []uuid.UUID) ([]*domain.Permission, error) {
	if m.permissionsErr != nil {
		return nil, m.permissionsErr
	}
	var perms []*domain.Permission
	for _, rid := range roleIDs {
		perms = append(perms, m.rolePermissions[rid]...)
	}
	return perms, nil
}

type mockUserRoleReader struct {
	roleIDs map[uuid.UUID][]uuid.UUID
	err     error
}

func (m *mockUserRoleReader) GetRoleIDsForUser(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.roleIDs[userID], nil
}

type mockPolicyReader struct {
	policies map[uuid.UUID][]*domain.Policy // keyed by userID
	err      error
}

func (m *mockPolicyReader) GetPoliciesForUserAndRoles(_ context.Context, userID uuid.UUID, _ []uuid.UUID) ([]*domain.Policy, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.policies[userID], nil
}

// --- Helpers ---

func newPerm(resourceType, action string) *domain.Permission {
	return &domain.Permission{
		ID:           uuid.New(),
		ResourceType: resourceType,
		Action:       action,
	}
}

func newPolicy(effect domain.Effect, name string, actions, resources []string) *domain.Policy {
	return &domain.Policy{
		ID:       uuid.New(),
		Name:     name,
		Effect:   effect,
		Actions:  actions,
		Resources: resources,
	}
}

func newRequest(userID uuid.UUID, resourceType, action string) *domain.CheckRequest {
	return &domain.CheckRequest{
		UserID:       userID,
		ResourceType: resourceType,
		Action:       action,
	}
}

// --- Evaluator.Check tests ---

func TestCheck_AnonymousUser_Deny(t *testing.T) {
	e := NewEvaluator(&mockRoleReader{}, &mockUserRoleReader{}, &mockPolicyReader{})
	result, err := e.Check(context.Background(), newRequest(uuid.Nil, "users", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("anonymous user should be denied")
	}
}

func TestCheck_NoRoles_Deny(t *testing.T) {
	userID := uuid.New()
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {}}}
	e := NewEvaluator(&mockRoleReader{}, ur, &mockPolicyReader{})
	result, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("user with no roles should be denied")
	}
}

func TestCheck_RBAC_Allow(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {newPerm("users", "read")}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	e := NewEvaluator(rr, ur, &mockPolicyReader{})

	result, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected allow, got deny: %s", result.Reason)
	}
	if result.MatchedBy != "rbac" {
		t.Errorf("expected matchedBy=rbac, got %s", result.MatchedBy)
	}
}

func TestCheck_RBAC_Deny_NoMatchingPermission(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {newPerm("users", "read")}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	e := NewEvaluator(rr, ur, &mockPolicyReader{})

	result, err := e.Check(context.Background(), newRequest(userID, "users", "delete"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected deny for non-matching permission")
	}
}

func TestCheck_RBAC_WildcardAction(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {newPerm("users", "*")}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	e := NewEvaluator(rr, ur, &mockPolicyReader{})

	// Wildcard action should match any action on the same resource type
	for _, action := range []string{"read", "write", "delete"} {
		result, err := e.Check(context.Background(), newRequest(userID, "users", action))
		if err != nil {
			t.Fatalf("unexpected error for action %s: %v", action, err)
		}
		if !result.Allowed {
			t.Errorf("wildcard action should allow %s", action)
		}
	}
}

func TestCheck_RBAC_RoleInheritance(t *testing.T) {
	userID := uuid.New()
	parentRole := uuid.New()
	childRole := uuid.New()

	rr := &mockRoleReader{
		// child inherits from parent
		ancestorChain: map[uuid.UUID][]uuid.UUID{
			childRole: {childRole, parentRole},
			parentRole: {parentRole},
		},
		// Only parent has the permission; child inherits it
		rolePermissions: map[uuid.UUID][]*domain.Permission{
			parentRole: {newPerm("roles", "write")},
			childRole:  {},
		},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {childRole}}}
	e := NewEvaluator(rr, ur, &mockPolicyReader{})

	// User assigned child role should inherit parent's permissions
	result, err := e.Check(context.Background(), newRequest(userID, "roles", "write"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("child role should inherit parent permission")
	}
}

func TestCheck_ABAC_Allow_Policy(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	pr := &mockPolicyReader{
		policies: map[uuid.UUID][]*domain.Policy{
			userID: {newPolicy(domain.EffectAllow, "allow-all-users", []string{"*"}, nil)},
		},
	}
	e := NewEvaluator(rr, ur, pr)

	result, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("ABAC allow policy should grant access")
	}
}

func TestCheck_ABAC_Deny_Overrides_RBAC_Allow(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	rr := &mockRoleReader{
		ancestorChain: map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		// RBAC would allow this
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {newPerm("users", "delete")}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	pr := &mockPolicyReader{
		policies: map[uuid.UUID][]*domain.Policy{
			userID: {newPolicy(domain.EffectDeny, "deny-delete", []string{"*"}, []string{"*"})},
		},
	}
	e := NewEvaluator(rr, ur, pr)

	result, err := e.Check(context.Background(), newRequest(userID, "users", "delete"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("deny policy must override RBAC allow")
	}
	if result.MatchedBy == "" {
		t.Error("deny result should have matchedBy set")
	}
}

func TestCheck_ABAC_Deny_Overrides_ABAC_Allow(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	pr := &mockPolicyReader{
		policies: map[uuid.UUID][]*domain.Policy{
			userID: {
				newPolicy(domain.EffectAllow, "allow-read", []string{"users:read"}, nil),
				newPolicy(domain.EffectDeny, "deny-read", []string{"users:read"}, nil),
			},
		},
	}
	e := NewEvaluator(rr, ur, pr)

	result, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("deny policy must override allow policy")
	}
}

func TestCheck_ABAC_ResourceGlobMatch(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	pr := &mockPolicyReader{
		policies: map[uuid.UUID][]*domain.Policy{
			userID: {
				newPolicy(domain.EffectAllow, "allow-specific-user",
					[]string{"users:read"},
					[]string{"arn:ggid:iam::tenant:user/*"}),
			},
		},
	}
	e := NewEvaluator(rr, ur, pr)

	// Should match: resource starts with the ARN prefix
	result, err := e.Check(context.Background(), &domain.CheckRequest{
		UserID:       userID,
		ResourceType: "users",
		Action:       "users:read",
		Resource:     "arn:ggid:iam::tenant:user/abc-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("resource glob pattern should match")
	}
}

func TestCheck_ABAC_ResourceGlobNoMatch(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	pr := &mockPolicyReader{
		policies: map[uuid.UUID][]*domain.Policy{
			userID: {
				newPolicy(domain.EffectAllow, "allow-org-users-only",
					[]string{"users:read"},
					[]string{"arn:ggid:iam::tenant:org-a:user/*"}),
			},
		},
	}
	e := NewEvaluator(rr, ur, pr)

	// Should NOT match: different org prefix
	result, err := e.Check(context.Background(), &domain.CheckRequest{
		UserID:       userID,
		ResourceType: "users",
		Action:       "users:read",
		Resource:     "arn:ggid:iam::tenant:org-b:user/xyz",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("resource should not match glob pattern for different org")
	}
}

func TestCheck_DefaultDeny_NoMatch(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {newPerm("roles", "read")}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	e := NewEvaluator(rr, ur, &mockPolicyReader{})

	// User has roles:read but requests audit:read — should deny
	result, err := e.Check(context.Background(), newRequest(userID, "audit", "read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("should deny when no permission or policy matches")
	}
}

func TestCheck_Error_GetRoleIDs(t *testing.T) {
	userID := uuid.New()
	dbErr := errors.New("db connection lost")
	ur := &mockUserRoleReader{err: dbErr}
	e := NewEvaluator(&mockRoleReader{}, ur, &mockPolicyReader{})

	_, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err == nil {
		t.Fatal("expected error when GetRoleIDsForUser fails")
	}
}

func TestCheck_Error_GetAncestorChain(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	rr := &mockRoleReader{ancestorErr: errors.New("db error")}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	e := NewEvaluator(rr, ur, &mockPolicyReader{})

	_, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err == nil {
		t.Fatal("expected error when GetAncestorChain fails")
	}
}

func TestCheck_Error_GetRolePermissions(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	rr := &mockRoleReader{permissionsErr: errors.New("db error")}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	e := NewEvaluator(rr, ur, &mockPolicyReader{})

	_, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err == nil {
		t.Fatal("expected error when GetRolePermissions fails")
	}
}

func TestCheck_Error_GetPolicies(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	rr := &mockRoleReader{
		ancestorChain:   map[uuid.UUID][]uuid.UUID{roleID: {roleID}},
		rolePermissions: map[uuid.UUID][]*domain.Permission{roleID: {newPerm("users", "read")}},
	}
	ur := &mockUserRoleReader{roleIDs: map[uuid.UUID][]uuid.UUID{userID: {roleID}}}
	pr := &mockPolicyReader{err: errors.New("db error")}
	e := NewEvaluator(rr, ur, pr)

	_, err := e.Check(context.Background(), newRequest(userID, "users", "read"))
	if err == nil {
		t.Fatal("expected error when GetPoliciesForUserAndRoles fails")
	}
}

// --- matchActions tests ---

func TestMatchActions(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		action   string
		want     bool
	}{
		{"exact match", []string{"users:read"}, "users:read", true},
		{"no match", []string{"users:read"}, "users:write", false},
		{"star wildcard", []string{"*"}, "anything", true},
		{"prefix wildcard match", []string{"iam:users:*"}, "iam:users:read", true},
		{"prefix wildcard no match", []string{"iam:users:*"}, "iam:roles:read", false},
		{"multiple patterns first match", []string{"iam:roles:*", "iam:users:*"}, "iam:users:delete", true},
		{"multiple patterns second match", []string{"iam:roles:*", "iam:users:*"}, "iam:roles:read", true},
		{"empty patterns", []string{}, "anything", false},
		{"prefix wildcard matches exact prefix", []string{"iam:users:*"}, "iam:users:", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchActions(tt.patterns, tt.action)
			if got != tt.want {
				t.Errorf("matchActions(%v, %q) = %v, want %v", tt.patterns, tt.action, got, tt.want)
			}
		})
	}
}

// --- matchResources tests ---

func TestMatchResources(t *testing.T) {
	tests := []struct {
		name      string
		patterns  []string
		resource  string
		want      bool
	}{
		{"exact match", []string{"arn:ggid:iam::t:resource/1"}, "arn:ggid:iam::t:resource/1", true},
		{"no match", []string{"arn:ggid:iam::t:resource/1"}, "arn:ggid:iam::t:resource/2", false},
		{"star wildcard", []string{"*"}, "anything", true},
		{"prefix glob", []string{"arn:ggid:iam::t:user/*"}, "arn:ggid:iam::t:user/abc-123", true},
		{"prefix glob no match", []string{"arn:ggid:iam::t:user/*"}, "arn:ggid:iam::t:role/xyz", false},
		{"suffix glob", []string{"*/admin"}, "arn:ggid:iam::t:role/admin", true},
		{"middle glob", []string{"arn:*:user"}, "arn:ggid:iam::t:user", true},
		{"empty patterns", []string{}, "anything", false},
		{"multiple patterns", []string{"arn:ggid:iam::a:*", "arn:ggid:iam::b:*"}, "arn:ggid:iam::b:res", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchResources(tt.patterns, tt.resource)
			if got != tt.want {
				t.Errorf("matchResources(%v, %q) = %v, want %v", tt.patterns, tt.resource, got, tt.want)
			}
		})
	}
}

// --- matchGlob tests ---

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		s       string
		want    bool
	}{
		{"no wildcard exact match", "hello", "hello", true},
		{"no wildcard no match", "hello", "world", false},
		{"prefix wildcard", "pre*", "prefix", true},
		{"prefix wildcard no match", "pre*", "other", false},
		{"suffix wildcard", "*fix", "suffix", true},
		{"suffix wildcard no match", "*fix", "other", false},
		{"middle wildcard", "a*c", "abc", true},
		{"middle wildcard longer", "a*c", "aXXXc", true},
		{"middle wildcard no match", "a*c", "abd", false},
		{"multiple wildcards", "a*b*c", "aXbXc", true},
		{"multiple wildcards no match", "a*b*c", "aXbXd", false},
		{"double star", "a**b", "aXXb", true},
		{"only star", "*", "anything", true},
		{"star at start and end", "*middle*", "XXmiddleYY", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchGlob(tt.pattern, tt.s)
			if got != tt.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.s, got, tt.want)
			}
		})
	}
}
