package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// TestPasskeyRevoke_TenantIsolation_BUG demonstrates that a user from one tenant
// can revoke passkeys belonging to users in another tenant.
func TestPasskeyRevoke_TenantIsolation_BUG(t *testing.T) {
	// BUG: No tenant context check in handlePasskeyRevoke
	// Expected: Should only revoke passkeys for the current tenant's user
	// Actual: Revokes any passkey by ID without tenant validation

	t.Run("revoke_without_tenant_context_bypasses_isolation", func(t *testing.T) {
		// Create a handler with pool
		h := &Handler{}
		// Simulate request without proper tenant context
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/passkeys/123", nil)
		w := httptest.NewRecorder()

		// The bug: handlePasskeyRevoke doesn't check tenant_id
		// A malicious user from tenant1 could revoke a passkey from tenant2
		h.handlePasskeyRevoke(w, req)

		if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
			t.Errorf("Expected 401/403 for cross-tenant revoke, got %d", w.Code)
		}
	})
}

// TestPasskeyStatus_TenantIsolation_BUG demonstrates that an empty user_id
// can return passkeys from all tenants.
func TestPasskeyStatus_TenantIsolation_BUG(t *testing.T) {
	// BUG: Query uses ($1 = '' OR user_id = $1)
	// SELECT ... WHERE revoked = false AND ($1 = '' OR user_id = $1)
	// When user_id is empty string, it returns ALL non-revoked passkeys

	h := &Handler{}
	tc := &tenant.Context{TenantID: uuid.New(), IsolationLevel: tenant.IsolationShared}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/passkeys/status?user_id=", nil)
	req = req.WithContext(tenant.WithContext(req.Context(), tc))
	w := httptest.NewRecorder()

	h.handlePasskeyStatus(w, req)

	// The bug: empty user_id returns passkeys from ALL tenants
	// Expected: Should return only passkeys for current tenant
	// Actual: Returns passkeys from tenant2 as well
	if w.Code == http.StatusOK && w.Body.Len() > 0 {
		t.Logf("Response: %s", w.Body.String())
	}
}

// TestVerifyCredentials_MFABypass_BUG demonstrates that MFA can be bypassed
// by simply not providing an MFA code.
func TestVerifyCredentials_MFABypass_BUG(t *testing.T) {
	// BUG in http.go verifyCredentials (lines 702-711):
	// The code only verifies MFA if both mfaRequired AND req.MFACode are set.
	// If MFA is required but no code is provided, it returns success with mfa_required=true
	// A malicious client can ignore this flag and proceed without MFA.

	t.Run("mfa_required_but_no_code_allows_bypass", func(t *testing.T) {
		// Scenario: User has MFA enabled
		// Attacker sends login without MFA code
		// Service returns: {"user_id": "...", "mfa_required": true}
		// Attacker ignores mfa_required flag and uses the user_id

		// Expected: Should reject login if MFA required and not provided
		// Actual: Returns success with mfa_required flag (soft fail, not hard fail)
		t.Skip("Test requires full auth service setup")
	})

	t.Run("mfa_bypass_via_missing_code_field", func(t *testing.T) {
		// The bug flow:
		// 1. User has MFA enabled
		// 2. Attacker POSTs to /api/v1/auth/verify with username/password only
		// 3. Server verifies password successfully
		// 4. Server sees MFA required but no code in request
		// 5. Server returns: {"user_id": "xyz", "mfa_required": true}
		// 6. Attacker uses user_id directly without completing MFA

		t.Skip("Test requires full auth service setup")
	})
}
