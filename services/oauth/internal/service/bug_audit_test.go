package service

import (
	"context"
	"strings"
	"testing"
	"time"

	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// BUG #1: ClientCredentials returns wrong scope in response
// The token is issued with filtered scopes (intersection of requested and allowed),
// but the response returns the original requested scopes, misleading the client.
func TestBug_ClientCredentials_WrongScopeInResponse(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	// Create a client with limited scopes
	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "limited-client",
		Name:       "Limited Scope Client",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"client_credentials"},
		Scopes:     []string{"openid", "profile"}, // Only these are allowed
		Enabled:    true,
	}
	secret := generateClientSecret()
	hash, _ := pkgcrypto.HashPassword(secret)
	client.ClientSecretHash = hash
	_ = clientRepo.CreateClient(context.Background(), client)

	// Request MORE scopes than allowed (attempting scope escalation)
	resp, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID:     testTenantID,
		ClientID:     "limited-client",
		ClientSecret: secret,
		Scope:        []string{"openid", "profile", "email", "admin", "delete:all"},
	})

	if err != nil {
		t.Fatalf("ClientCredentials failed: %v", err)
	}

	// Parse the actual token to see what scopes were granted
	claims, err := svc.ParseAccessToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken failed: %v", err)
	}

	tokenScope, _ := claims["scope"].(string)
	responseScope := resp.Scope

	// BUG: Response scope should match the granted token scope
	// But the code at line 1732 returns req.Scope instead of finalScopes
	if responseScope == tokenScope {
		t.Log("RESPONSE SCOPE BUG APPEARS FIXED")
	} else {
		t.Logf("BUG CONFIRMED: Response scope differs from token scope")
		t.Logf("  Response scope: %q", responseScope)
		t.Logf("  Token scope:    %q", tokenScope)
		t.Logf("  Response should be the filtered/granted scope, not the requested scope")
		t.Logf("  Response contains: %v", strings.Fields(responseScope))
		t.Logf("  Token contains:    %v", strings.Fields(tokenScope))

		// The token should only have the client's allowed scopes
		expectedTokenScopes := []string{"openid", "profile"}
		for _, expected := range expectedTokenScopes {
			if !strings.Contains(tokenScope, expected) {
				t.Errorf("Token missing expected scope %q in %q", expected, tokenScope)
			}
		}

		// The response should NOT contain scopes that weren't granted
		if strings.Contains(responseScope, "admin") {
			t.Error("BUG: Response contains 'admin' scope which was not granted in the token")
		}
		if strings.Contains(responseScope, "delete:all") {
			t.Error("BUG: Response contains 'delete:all' scope which was not granted in the token")
		}
	}
}

// BUG #2: intersectScopes falls back to ALL allowed scopes when nothing matches
// This is a scope escalation vulnerability - if a client requests invalid scopes,
// they get ALL the client's scopes instead of an empty set.
func TestBug_IntersectScopes_FallbackToAllAllowed(t *testing.T) {
	allowed := []string{"openid", "profile", "email"}

	// Request completely different scopes - should return empty or requested
	requested := []string{"admin", "delete:all", "platform:write"}

	result := intersectScopes(requested, allowed)

	// BUG: Current implementation returns ALL allowed scopes when nothing matches
	// This means a malicious client could request invalid scopes and still get
	// access to all the client's permissions!
	if len(result) == len(allowed) {
		t.Log("BUG CONFIRMED: intersectScopes returns all allowed scopes when nothing matches")
		t.Logf("  Requested: %v", requested)
		t.Logf("  Allowed:   %v", allowed)
		t.Logf("  Result:    %v", result)
		t.Error("SECURITY ISSUE: Client requested invalid scopes but got ALL allowed scopes instead of empty")
		t.Error("This allows scope escalation by requesting invalid scope names")

		// Correct behavior would be to return empty or only the requested scopes that matched
		t.Log("  Expected: [] (empty - no requested scopes were in allowed)")
	} else if len(result) == 0 {
		t.Log("BUG APPEARS FIXED: Returns empty when nothing matches")
	} else {
		t.Logf("Unexpected result: %v", result)
	}
}

// BUG #3: PasswordGrant does not check if client supports password grant
// Looking at lines 1780-1936, there's no check for client.SupportsGrantType("password")
func TestBug_PasswordGrant_NoGrantTypeCheck(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()
	userID := uuid.New()
	hash, _ := pkgcrypto.HashPassword("correct-pass-123")
	svc.SetPool(&fakePool{userID: userID, credHash: hash})

	// Create a client that does NOT support password grant
	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "no-password-grant",
		Name:       "Auth Code Only",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code", "refresh_token"}, // No "password"!
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	// Try to use password grant with this client
	resp, err := svc.PasswordGrant(context.Background(), &PasswordGrantRequest{
		TenantID: testTenantID,
		Username: "admin",
		Password: "correct-pass-123",
		ClientID: "no-password-grant",
		Scope:    []string{"openid"},
	})

	if err == nil && resp != nil && resp.AccessToken != "" {
		t.Log("BUG CONFIRMED: PasswordGrant succeeds even though client doesn't support it")
		t.Error("SECURITY ISSUE: Client registered without 'password' grant type can still use password grant")
		t.Error("This violates OAuth2 client registration and grant type enforcement")

		// This is a P0 security bug - grant type enforcement is critical
	} else {
		t.Log("BUG APPEARS FIXED: PasswordGrant rejects unsupported grant types")
	}
}

// BUG #4: filterSafeScopes allows bypass via scope patterns
// Test if scope patterns like "openid.*" can bypass the filter
func TestBug_FilterSafeScopes_PatternBypass(t *testing.T) {
	testCases := []struct {
		name     string
		scopes   []string
		shouldPass bool
	}{
		{"Valid OAuth scopes", []string{"openid", "profile", "email", "offline_access"}, true},
		{"Admin scope attempt", []string{"platform:admin", "tenant:write"}, false},
		{"Pattern attempt 1", []string{"openid admin"}, false}, // Should not parse as separate scopes
		{"Pattern attempt 2", []string{"openid,admin"}, false}, // Comma separator
		{"Exact admin scope", []string{"platform:admin"}, false},
		{"Mixed valid and invalid", []string{"openid", "profile", "platform:admin"}, true}, // Should filter out admin
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterSafeScopes(tc.scopes)

			if tc.shouldPass {
				if len(result) == 0 && len(tc.scopes) > 0 {
					t.Log("No scopes passed filter")
				}
			} else {
				// Should have filtered out all admin scopes
				for _, scope := range result {
					if strings.Contains(scope, "platform:") || strings.Contains(scope, "tenant:") {
						t.Errorf("BUG: Admin scope %q passed filterSafeScopes", scope)
					}
				}
			}
		})
	}
}

// BUG #5: RevokeToken cascade doesn't check if the revoked token is actually a refresh token
// It cascades to ALL refresh tokens for the user regardless of which token type was revoked
func TestBug_RevokeToken_CascadesToAllRefreshTokens(t *testing.T) {
	// This is more of a design issue - when revoking an access token,
	// should ALL refresh tokens be revoked?
	// Looking at lines 1437-1451, it does cascade to ALL refresh tokens for the user.

	t.Log("DESIGN CONSIDERATION: RevokeToken cascades to ALL refresh tokens for the user")
	t.Log("  Line 1450: UPDATE oidc_refresh_tokens SET revoked = true WHERE tenant_id = $1 AND user_id = $2")
	t.Log("  This means revoking ONE access token revokes ALL refresh tokens for that user")
	t.Log("  This might be overly aggressive - should only cascade if the access token was")
	t.Log("  issued from a refresh token, or if we want full session revocation on logout.")
	t.Log("  Current behavior: Session-level revocation (all tokens for user)")
	t.Log("  Alternative: Token-level revocation (only related tokens)")
}

// BUG #6: RefreshToken response scope issue (similar to ClientCredentials)
func TestBug_RefreshToken_ScopeResponse(t *testing.T) {
	svc, clientRepo, tokenRepo, _ := newFamilyTestService()
	client := addRefreshClient(t, clientRepo, "scope-test-client")

	userID := uuid.New()
	// Create a refresh token with specific scopes
	seed := &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  client.ID,
		UserID:    userID,
		TokenHash: hashTokenSHA256("seed-rt"),
		Scope:     []string{"openid", "profile"}, // Only these scopes
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = tokenRepo.StoreRefreshToken(context.Background(), seed)

	// Refresh with additional scope request
	resp, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: "seed-rt",
		ClientID:     "scope-test-client",
		Scope:        []string{"openid", "profile", "email", "admin"}, // Request more
	})

	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	// Check what's in the response vs the token
	claims, err := svc.ParseAccessToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken failed: %v", err)
	}

	tokenScope, _ := claims["scope"].(string)
	responseScope := resp.Scope

	// The response scope should match what was actually granted
	if responseScope != tokenScope {
		t.Log("POTENTIAL BUG: RefreshToken response scope differs from token scope")
		t.Logf("  Response: %q", responseScope)
		t.Logf("  Token:    %q", tokenScope)

		// Check if admin scope leaked into response
		if strings.Contains(responseScope, "admin") {
			t.Error("BUG: Response contains 'admin' which should have been filtered")
		}
	}
}

// SECURITY TEST: Verify scope escalation is prevented across all grant types
func TestSecurity_NoScopeEscalation(t *testing.T) {
	t.Run("PasswordGrant", func(t *testing.T) {
		svc, clientRepo, _, _ := newTestOAuthService()
		userID := uuid.New()
		hash, _ := pkgcrypto.HashPassword("pass")
		svc.SetPool(&fakePool{userID: userID, credHash: hash})

		client := &domain.OAuthClient{
			ID:         uuid.New(),
			TenantID:   testTenantID,
			ClientID:   "test",
			Name:       "Test",
			Type:       domain.ClientTypePublic,
			GrantTypes: []string{"password", "refresh_token"},
			Enabled:    true,
		}
		_ = clientRepo.CreateClient(context.Background(), client)

		resp, err := svc.PasswordGrant(context.Background(), &PasswordGrantRequest{
			TenantID: testTenantID,
			Username: "admin",
			Password: "pass",
			ClientID: "test",
			Scope:    []string{"openid", "platform:admin", "tenant:write"},
		})

		if err != nil {
			t.Fatalf("PasswordGrant failed: %v", err)
		}

		claims, _ := svc.ParseAccessToken(resp.AccessToken)
		scope, _ := claims["scope"].(string)

		// Admin scopes should NOT be present - they come from DB role keys only
		if strings.Contains(scope, "platform:admin") || strings.Contains(scope, "tenant:write") {
			t.Error("SCOPE ESCALATION: Admin scopes from request leaked into token")
		}
	})
}
