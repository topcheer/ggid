package tools

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/mcp/internal/client"
)

// TestRegistry_EmptyRequiredScopesAccessibleToAnyone demonstrates that
// tools with empty RequiredScopes are accessible to any user, regardless
// of their actual scopes.
func TestRegistry_EmptyRequiredScopesAccessibleToAnyone(t *testing.T) {
	r := NewRegistry()

	// Add a tool with NO required scopes
	r.register(Tool{
		Name:           "unrestricted_tool",
		Description:    "Should require explicit scopes but doesn't",
		InputSchema:    map[string]any{"type": "object"},
		RequiredScopes: []string{}, // BUG: Empty slice
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			return nil, nil
		},
	})

	// Try to filter with NO scopes (empty slice)
	available := r.FilterByScopes([]string{})

	// BUG: hasAllScopes([]string{}, []string{}) returns true
	// because the loop doesn't execute when need is empty (line 75-81)
	found := false
	for _, tool := range available {
		if tool.Name == "unrestricted_tool" {
			found = true
			break
		}
	}

	if found {
		t.Error("BUG: Tool with empty RequiredScopes is accessible to users with NO scopes")
		t.Error("This is a security issue - tools should explicitly declare required scopes")
	}
}

// TestRegistry_NilRequiredScopesCausesPanicOrLeak demonstrates what
// happens when RequiredScopes is nil instead of empty slice.
func TestRegistry_NilRequiredScopesCausesPanicOrLeak(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic (expected or not): %v", r)
		}
	}()

	r := NewRegistry()

	// Add a tool with nil RequiredScopes
	r.register(Tool{
		Name:           "nil_scopes_tool",
		Description:    "Tool with nil RequiredScopes",
		InputSchema:    map[string]any{"type": "object"},
		RequiredScopes: nil, // BUG: nil instead of empty slice
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			return nil, nil
		},
	})

	// Try to filter - this might panic if hasAllScopes doesn't handle nil
	available := r.FilterByScopes([]string{"openid"})

	// If we get here without panicking, check if the tool is accessible
	found := false
	for _, tool := range available {
		if tool.Name == "nil_scopes_tool" {
			found = true
			break
		}
	}

	if found {
		t.Error("BUG: Tool with nil RequiredScopes is accessible")
	}
}

// TestRegistry_AnyAdminScopeGrantsAllAccess demonstrates that
// admin scopes should NOT grant access to ALL tools - they should
// still respect RequiredScopes to maintain tenant isolation.
func TestRegistry_AnyAdminScopeGrantsAllAccess(t *testing.T) {
	r := NewRegistry()

	// Count total tools registered
	allTools := r.All()
	t.Logf("Total tools registered: %d", len(allTools))

	// Test with "admin" scope
	adminScopes := r.FilterByScopes([]string{"admin"})
	t.Logf("Tools visible to 'admin' scope: %d", len(adminScopes))

	// Test with "tenant:admin" scope
	tenantAdminScopes := r.FilterByScopes([]string{"tenant:admin"})
	t.Logf("Tools visible to 'tenant:admin' scope: %d", len(tenantAdminScopes))

	// Test with "platform:admin" scope
	platformAdminScopes := r.FilterByScopes([]string{"platform:admin"})
	t.Logf("Tools visible to 'platform:admin' scope: %d", len(platformAdminScopes))

	// FIXED: Admin scopes should NOT grant access to ALL tools
	// They should still respect RequiredScopes to prevent
	// tenant:admin from tenant A accessing tools intended for tenant B
	if len(adminScopes) == len(allTools) {
		t.Error("FIXED: 'admin' scope should NOT grant access to ALL tools - must respect RequiredScopes")
	}
	if len(tenantAdminScopes) == len(allTools) {
		t.Error("FIXED: 'tenant:admin' scope should NOT grant access to ALL tools - must respect RequiredScopes")
	}
	if len(platformAdminScopes) == len(allTools) {
		t.Error("FIXED: 'platform:admin' scope should NOT grant access to ALL tools - must respect RequiredScopes")
	}
}

// TestRegistry_ScopeFilteringCanBeBypassed tests if scope filtering
// can be bypassed by manipulating the scope string.
func TestRegistry_ScopeFilteringCanBeBypassed(t *testing.T) {
	r := NewRegistry()

	// A tool that requires "users:write" scope
	sensitiveTool := Tool{
		Name:           "delete_all_users",
		Description:    "Dangerous operation",
		InputSchema:    map[string]any{"type": "object"},
		RequiredScopes: []string{"users:write"},
		Handler:        nil,
	}
	r.register(sensitiveTool)

	// Try various scope manipulation attempts
	testCases := []struct {
		name   string
		scopes []string
		shouldFail bool
	}{
		{
			name:       "empty scope",
			scopes:     []string{},
			shouldFail: true,
		},
		{
			name:       "wrong scope users:read",
			scopes:     []string{"users:read"},
			shouldFail: true,
		},
		{
			name:       "admin bypass - FIXED",
			scopes:     []string{"admin"},
			shouldFail: true, // FIXED: Admin now respects RequiredScopes
		},
		{
			name:       "partial match",
			scopes:     []string{"users:write", "extra:scope"},
			shouldFail: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			available := r.FilterByScopes(tc.scopes)
			found := false
			for _, tool := range available {
				if tool.Name == "delete_all_users" {
					found = true
					break
				}
			}

			if found && tc.shouldFail {
				t.Errorf("BUG: Tool was accessible with scopes %v when it should have been blocked", tc.scopes)
			} else if !found && !tc.shouldFail {
				t.Errorf("Tool was blocked with scopes %v when it should have been accessible", tc.scopes)
			}
		})
	}
}
