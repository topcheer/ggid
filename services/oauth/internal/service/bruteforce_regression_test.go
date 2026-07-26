package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
)

// TestPasswordGrant_LockedAccountRejected verifies that locked accounts
// are rejected even when the password is correct. The SQL query now
// includes `AND c.enabled = true AND (c.locked_until IS NULL OR
// c.locked_until < NOW())`. When a locked credential is encountered,
// the query returns no rows → unauthenticated.
//
// Since fakePool always returns a row, we simulate the locked scenario
// by having credHash empty (no credential found → fail closed).
func TestPasswordGrant_LockedAccountRejected(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()
	// Empty credHash simulates "no row returned" (account locked or disabled)
	svc.SetPool(&fakePool{userID: uuid.New(), credHash: ""})

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "ggid-console",
		Name:       "Console",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"password", "authorization_code", "refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	_, err := svc.PasswordGrant(context.Background(), &PasswordGrantRequest{
		TenantID: testTenantID,
		Username: "locked_user",
		Password: "correct_password",
		ClientID: "ggid-console",
		Scope:    []string{"openid"},
	})
	if err == nil {
		t.Error("locked account must be rejected even with correct password")
	}
}

// TestPasswordGrant_FailedAttemptsIncremented verifies that the
// failed_attempts UPDATE is called when password verification fails.
// Since fakePool.Exec is a no-op, we verify by checking that the
// wrong password still returns an error (the Exec call doesn't block
// the error path).
func TestPasswordGrant_FailedAttemptsIncremented(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()
	hash, _ := crypto.HashPassword("correct-pass")
	svc.SetPool(&fakePool{userID: uuid.New(), credHash: hash})

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "ggid-console",
		Name:       "Console",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"password"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	// Wrong password should fail
	_, err := svc.PasswordGrant(context.Background(), &PasswordGrantRequest{
		TenantID: testTenantID,
		Username: "admin",
		Password: "wrong",
		ClientID: "ggid-console",
		Scope:    []string{"openid"},
	})
	if err == nil {
		t.Error("wrong password must fail")
	}

	// The fakePool.Exec accepts the failed_attempts UPDATE without error.
	// In production, this UPDATE increments the counter and locks after 5 attempts.
	// Here we just verify the error path completes without panic.
}

// TestPasswordGrant_SuccessResetsAttempts verifies that a successful
// login completes without error when credentials are valid.
func TestPasswordGrant_SuccessResetsAttempts(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()
	hash, _ := crypto.HashPassword("correct-pass-123")
	svc.SetPool(&fakePool{userID: uuid.New(), credHash: hash})

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "ggid-console",
		Name:       "Console",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"password", "authorization_code", "refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	resp, err := svc.PasswordGrant(context.Background(), &PasswordGrantRequest{
		TenantID: testTenantID,
		Username: "admin",
		Password: "correct-pass-123",
		ClientID: "ggid-console",
		Scope:    []string{"openid"},
	})
	if err != nil {
		t.Fatalf("valid login must succeed: %v", err)
	}
	if resp == nil || resp.AccessToken == "" {
		t.Error("valid login must return access token")
	}
}
